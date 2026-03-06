package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
	fmConfig "github.com/clario360/platform/internal/filemanager/config"
	"github.com/clario360/platform/internal/filemanager/consumer"
	fmHandler "github.com/clario360/platform/internal/filemanager/handler"
	fmHealth "github.com/clario360/platform/internal/filemanager/health"
	fmMetrics "github.com/clario360/platform/internal/filemanager/metrics"
	fmMiddleware "github.com/clario360/platform/internal/filemanager/middleware"
	"github.com/clario360/platform/internal/filemanager/repository"
	"github.com/clario360/platform/internal/filemanager/service"
	mw "github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/health"
	"github.com/clario360/platform/internal/observability/tracing"
	"github.com/clario360/platform/pkg/storage"
)

func main() {
	// 1. Load config
	cfg, err := fmConfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// 2. Bootstrap observability (logger, metrics, tracer, DB, Redis)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := bootstrap.Bootstrap(ctx, cfg.ServiceConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap error: %v\n", err)
		os.Exit(1)
	}

	logger := svc.Logger

	// Initialize JWT manager for auth middleware
	jwtPublicKeyPEM, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		logger.Fatal().Err(err).Str("path", cfg.JWTPublicKeyPath).Msg("failed to read JWT public key")
	}

	jwtMgr, err := auth.NewJWTManager(config.AuthConfig{
		RSAPublicKeyPEM: string(jwtPublicKeyPEM),
		JWTIssuer:       "clario360",
		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// 3. Initialize MinIO client
	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
		Region: cfg.MinIORegion,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create MinIO client")
	}

	// Verify MinIO connectivity
	if _, err = minioClient.ListBuckets(ctx); err != nil {
		logger.Fatal().Err(err).Msg("MinIO unreachable — cannot start")
	}
	logger.Info().Str("endpoint", cfg.MinIOEndpoint).Msg("MinIO connected")

	// 4. Initialize Storage
	store, err := storage.NewStorage(storage.Config{
		Backend:          "minio",
		Endpoint:         cfg.MinIOEndpoint,
		AccessKey:        cfg.MinIOAccessKey,
		SecretKey:        cfg.MinIOSecretKey,
		UseSSL:           cfg.MinIOUseSSL,
		Region:           cfg.MinIORegion,
		BucketPrefix:     cfg.BucketPrefix,
		QuarantineBucket: cfg.QuarantineBucket,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create storage")
	}

	// Ensure quarantine bucket exists
	if err := store.EnsureBucket(ctx, cfg.QuarantineBucket); err != nil {
		logger.Error().Err(err).Msg("failed to ensure quarantine bucket")
	}

	// 5. Initialize Encryptor
	encryptor, err := storage.NewEncryptor(cfg.EncryptionMasterKey, cfg.EncryptionKeyID)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create encryptor")
	}

	// 6. Initialize VirusScanner (graceful degradation)
	scanner := storage.NewVirusScanner(cfg.ClamAVAddress, cfg.ClamAVTimeout, cfg.ClamAVMaxSizeMB)
	if err := scanner.Ping(); err != nil {
		logger.Warn().Err(err).Str("address", cfg.ClamAVAddress).Msg("ClamAV not available — scans will be deferred")
	} else {
		logger.Info().Str("address", cfg.ClamAVAddress).Msg("ClamAV connected")
	}

	// Initialize Kafka producer
	kafkaCfg := config.KafkaConfig{
		Brokers:         cfg.ServiceConfig.Kafka.Brokers,
		GroupID:         cfg.ServiceConfig.Kafka.GroupID,
		AutoOffsetReset: "earliest",
	}
	producer, err := events.NewProducer(kafkaCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create Kafka producer")
	}

	// Initialize file-specific metrics
	fileMetrics := fmMetrics.NewFileMetrics(svc.Metrics.Registry())

	// Initialize repository
	repo := repository.NewFileRepository(svc.DBPool, logger)

	// Initialize services
	fileSvc := service.NewFileService(
		repo, store, encryptor, producer, fileMetrics, logger,
		cfg.BucketPrefix, cfg.QuarantineBucket, cfg.PresignedURLExpiry,
	)

	scanSvc := service.NewScanService(
		repo, store, scanner, encryptor, producer, fileMetrics, logger,
		cfg.QuarantineBucket,
	)

	lifecycleSvc := service.NewLifecycleService(
		repo, store, producer, fileMetrics, logger,
		cfg.QuarantineBucket,
	)

	// Register additional health checkers for MinIO + ClamAV
	fileCheckers := []health.HealthChecker{
		fmHealth.NewMinIOChecker(minioClient),
		fmHealth.NewClamAVChecker(scanner),
	}
	_ = health.NewCompositeHealthChecker(5*time.Second, fileCheckers...)

	// 7. Initialize Kafka consumer
	kafkaConsumer, err := events.NewConsumer(kafkaCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create Kafka consumer")
	}

	scanConsumer := consumer.NewScanConsumer(scanSvc, logger)
	kafkaConsumer.Subscribe("platform.file.events", scanConsumer)

	// 9. Register HTTP handlers
	fileHandler := fmHandler.NewFileHandler(fileSvc, logger)
	presignedHandler := fmHandler.NewPresignedHandler(fileSvc, logger)
	adminHandler := fmHandler.NewAdminHandler(fileSvc, logger)

	svc.Router.Route("/api/v1/files", func(r chi.Router) {
		r.Use(mw.Auth(jwtMgr))
		r.Use(mw.Tenant)
		r.Use(fmMiddleware.UploadGuard(cfg.MaxUploadSizeMB))
		r.Use(fmMiddleware.RateLimiter(svc.Redis, 60))

		r.Post("/upload", fileHandler.Upload)
		r.Post("/upload/presigned", presignedHandler.GenerateUploadURL)
		r.Post("/upload/confirm", presignedHandler.ConfirmUpload)
		r.Get("/", fileHandler.ListFiles)
		r.Get("/{id}", fileHandler.GetFile)
		r.Get("/{id}/download", fileHandler.Download)
		r.Get("/{id}/presigned", presignedHandler.GenerateDownloadURL)
		r.Delete("/{id}", fileHandler.DeleteFile)
		r.Get("/{id}/versions", fileHandler.GetVersions)
		r.Get("/{id}/access-log", fileHandler.GetAccessLog)

		// Admin endpoints
		r.Get("/quarantine", adminHandler.ListQuarantined)
		r.Post("/quarantine/{id}/resolve", adminHandler.ResolveQuarantine)
		r.Get("/stats", adminHandler.GetStats)
		r.Post("/{id}/rescan", adminHandler.Rescan)
	})

	// Add tracing span enricher
	svc.Router.Use(tracing.SpanEnricher())

	// 10. Start services
	g, gCtx := errgroup.WithContext(ctx)

	// Start Kafka consumer
	g.Go(func() error {
		logger.Info().Msg("starting scan consumer")
		return kafkaConsumer.Start(gCtx)
	})

	// Start lifecycle cleanup ticker (daily at 03:00 UTC)
	g.Go(func() error {
		return runLifecycleTicker(gCtx, lifecycleSvc, logger)
	})

	// Start HTTP servers
	g.Go(func() error {
		return svc.Run(gCtx)
	})

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
		cancel()
	case <-gCtx.Done():
	}

	// Wait for goroutines
	if err := g.Wait(); err != nil && err != context.Canceled {
		logger.Error().Err(err).Msg("service exited with error")
	}

	// Ordered shutdown
	kafkaConsumer.Close()
	producer.Close()

	logger.Info().Msg("file-service stopped")
}

// runLifecycleTicker runs lifecycle cleanup daily at 03:00 UTC.
func runLifecycleTicker(ctx context.Context, svc *service.LifecycleService, logger zerolog.Logger) error {
	// Run initial cleanup
	svc.RunCleanup(ctx)

	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, time.UTC)
		if now.Hour() < 3 {
			next = time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)
		}

		logger.Info().Time("next_run", next).Msg("lifecycle cleanup scheduled")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Until(next)):
			svc.RunCleanup(ctx)
		}
	}
}
