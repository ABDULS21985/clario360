package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	auditcfg "github.com/clario360/platform/internal/audit/config"
	"github.com/clario360/platform/internal/audit/consumer"
	"github.com/clario360/platform/internal/audit/handler"
	"github.com/clario360/platform/internal/audit/health"
	auditmw "github.com/clario360/platform/internal/audit/middleware"
	"github.com/clario360/platform/internal/audit/repository"
	"github.com/clario360/platform/internal/audit/service"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"
	"github.com/rs/zerolog"
)

func main() {
	// 1. Load platform config
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}

	// Load audit-specific config from env
	auditCfg := auditcfg.LoadFromEnv()
	if err := auditCfg.Validate(); err != nil {
		panic("invalid audit config: " + err.Error())
	}
	cfg.Server.Port = auditCfg.HTTPPort

	// 2. Initialize structured logger
	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"audit-service",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize tracer
	shutdownTracer, err := observability.InitTracer(ctx, "audit-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	// 4. Connect PostgreSQL
	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// 5. Connect Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("redis connection failed — continuing with degraded functionality")
	}

	// 6. Create HTTP server with middleware stack
	srv, err := server.New(cfg, db, rdb, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	// 7. Initialize Kafka producer (for DLQ)
	var producer *events.Producer
	kafkaProducer, err := events.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("kafka producer unavailable — DLQ events will not be published")
	} else {
		producer = kafkaProducer
		defer producer.Close()
	}

	// 8. Initialize repositories
	auditRepo := repository.NewAuditRepository(db, logger)
	partitionMgr := repository.NewPartitionManager(db, logger)

	// 9. Run partition ensure on startup
	created, err := partitionMgr.EnsurePartitions(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to ensure partitions on startup")
	} else if len(created) > 0 {
		logger.Info().Strs("created", created).Msg("partitions created on startup")
	}

	// 10. Initialize services
	maskingSvc := service.NewMaskingService()
	auditSvc := service.NewAuditService(auditRepo, rdb, logger, auditCfg.BatchSize, auditCfg.BatchWindow)
	querySvc := service.NewQueryService(auditRepo, maskingSvc, logger)
	integritySvc := service.NewIntegrityService(auditRepo, logger)
	exportSvc := service.NewExportService(auditRepo, maskingSvc, logger, auditCfg.ExportAsyncThreshold)

	// Start audit service batch flusher
	auditSvc.Start(ctx)

	// 11. Initialize handlers
	auditHandler := handler.NewAuditHandler(querySvc, logger)
	exportHandler := handler.NewExportHandler(exportSvc, logger)
	adminHandler := handler.NewAdminHandler(partitionMgr, integritySvc, logger)

	// 12. Initialize health checker
	healthChecker := health.NewChecker(db, rdb, cfg.Kafka.Brokers, logger)

	// Override health endpoints with audit-specific checks
	srv.Router.Get("/healthz", health.LivenessHandler())
	srv.Router.Get("/readyz", healthChecker.ReadinessHandler())

	// 13. Register routes
	srv.Router.Route("/api/v1/audit", func(r chi.Router) {
		r.Use(middleware.Auth(srv.JWTManager))
		r.Use(auditmw.TenantGuard)
		r.Use(auditmw.RateLimiter(rdb, auditCfg.RateLimitPerMinute, logger))

		// Query endpoints
		r.Get("/logs", auditHandler.ListLogs)
		r.Get("/logs/stats", auditHandler.GetStats)
		r.Get("/logs/export", exportHandler.Export)
		r.Get("/logs/timeline/{resourceId}", auditHandler.GetTimeline)
		r.Get("/logs/{id}", auditHandler.GetLog)

		// Admin endpoints
		r.Post("/verify", adminHandler.VerifyChain)
		r.Get("/partitions", adminHandler.ListPartitions)
		r.Post("/partitions", adminHandler.CreatePartition)
		r.Post("/partitions/{name}/archive", adminHandler.ArchivePartition)
		r.Delete("/partitions/{name}", adminHandler.DeletePartition)
	})

	// 14. Initialize Kafka consumer
	var auditConsumer *consumer.AuditConsumer
	kafkaConsumer, err := events.NewConsumer(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("kafka consumer unavailable — audit event ingestion disabled")
	} else {
		var dlq *consumer.DeadLetterProducer
		if producer != nil {
			dlq = consumer.NewDeadLetterProducer(producer, logger)
		}
		auditConsumer = consumer.NewAuditConsumer(kafkaConsumer, auditSvc, dlq, logger)
	}

	// 15. Start all components via errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// HTTP server
	g.Go(func() error {
		logger.Info().Int("port", cfg.Server.Port).Msg("audit-service HTTP server starting")
		if err := srv.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Kafka consumer
	if auditConsumer != nil {
		g.Go(func() error {
			return auditConsumer.Start(gCtx)
		})
	}

	// Partition maintenance ticker (daily)
	g.Go(func() error {
		return runPartitionTicker(gCtx, partitionMgr, logger)
	})

	// 16. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case <-gCtx.Done():
		logger.Info().Msg("context cancelled")
	}

	// 17. Graceful shutdown sequence
	cancel()

	// Flush audit buffer
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer flushCancel()
	if err := auditSvc.Stop(flushCtx); err != nil {
		logger.Error().Err(err).Msg("failed to flush audit buffer on shutdown")
	}

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.HTTPServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	// Stop Kafka consumer
	if auditConsumer != nil {
		if err := auditConsumer.Stop(); err != nil {
			logger.Error().Err(err).Msg("kafka consumer shutdown error")
		}
	}

	if err := g.Wait(); err != nil {
		logger.Error().Err(err).Msg("errgroup finished with error")
	}

	logger.Info().Msg("audit-service stopped")
}

// runPartitionTicker runs daily partition maintenance.
func runPartitionTicker(ctx context.Context, pm *repository.PartitionManager, logger zerolog.Logger) error {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if created, err := pm.EnsurePartitions(ctx); err != nil {
				logger.Warn().Err(err).Msg("partition maintenance failed")
			} else if len(created) > 0 {
				logger.Info().Strs("created", created).Msg("partitions created by daily ticker")
			}
		}
	}
}
