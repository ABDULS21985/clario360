package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	dataconfig "github.com/clario360/platform/internal/data/config"
	dataconsumer "github.com/clario360/platform/internal/data/consumer"
	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/handler"
	datahealth "github.com/clario360/platform/internal/data/health"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("loading platform config: " + err.Error() + "\n")
		os.Exit(1)
	}

	dataCfg, err := dataconfig.Load()
	if err != nil {
		os.Stderr.WriteString("loading data config: " + err.Error() + "\n")
		os.Exit(1)
	}

	if port, err := strconv.Atoi(dataCfg.HTTPPort); err == nil {
		cfg.Server.Port = port
	}
	cfg.Kafka.Brokers = dataCfg.KafkaBrokers
	cfg.Kafka.GroupID = dataCfg.KafkaGroupID
	publicKeyPEM, err := os.ReadFile(dataCfg.JWTPublicKeyPath)
	if err != nil {
		os.Stderr.WriteString("reading DATA_JWT_PUBLIC_KEY_PATH: " + err.Error() + "\n")
		os.Exit(1)
	}
	cfg.Auth.RSAPublicKeyPEM = string(publicKeyPEM)

	bootstrapCfg, err := buildBootstrapConfig(cfg, dataCfg)
	if err != nil {
		os.Stderr.WriteString("building bootstrap config: " + err.Error() + "\n")
		os.Exit(1)
	}
	svc, err := bootstrap.Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		os.Stderr.WriteString("bootstrapping data-service: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := svc.Logger

	migrationsPath := envOr("DATA_MIGRATIONS_PATH", filepath.Join("migrations", "data_db"))
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "data_db")
	}
	if err := database.RunMigrations(dataCfg.DBURL, migrationsPath); err != nil {
		logger.Fatal().Err(err).Str("path", migrationsPath).Msg("failed to run data migrations")
	}

	var producer *events.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer, err = events.NewProducer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka producer unavailable — events will not be published")
		}
	}

	sourceRepo := repository.NewSourceRepository(svc.DBPool, logger)
	modelRepo := repository.NewModelRepository(svc.DBPool, logger)
	syncRepo := repository.NewSyncRepository(svc.DBPool, logger)
	dataMetrics := datametrics.New(svc.Metrics.Registry())

	registry := connector.NewConnectorRegistry(connector.ConnectorLimits{
		MaxPoolSize:      dataCfg.ConnectorMaxPoolSize,
		StatementTimeout: dataCfg.ConnectorStatementTimeout,
		ConnectTimeout:   dataCfg.ConnectorConnectTimeout,
		MaxSampleRows:    dataCfg.ConnectorMaxSampleRows,
		MaxTables:        dataCfg.ConnectorMaxTables,
		APIRateLimit:     dataCfg.ConnectorAPIRateLimit,
	}, logger)
	discoveryOpts := connector.DiscoveryOptions{
		MaxTables:    dataCfg.ConnectorMaxTables,
		MaxColumns:   dataCfg.DiscoveryMaxColumns,
		SampleValues: dataCfg.DiscoverySampleValues,
		MaxSamples:   dataCfg.ConnectorMaxSampleRows,
		IncludeViews: true,
	}
	encryptor, err := service.NewConfigEncryptorFromBytes(dataCfg.EncryptionKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize connection config encryptor")
	}

	tester := service.NewConnectionTester(registry, dataMetrics)
	discoverySvc := service.NewSchemaDiscoveryService(registry, discoveryOpts, dataMetrics)
	ingestionSvc := service.NewIngestionService(registry, sourceRepo, syncRepo, discoveryOpts, dataMetrics, logger)
	sourceSvc := service.NewSourceService(dataCfg, sourceRepo, syncRepo, tester, discoverySvc, ingestionSvc, encryptor, producer, dataMetrics, logger)
	modelSvc := service.NewModelService(modelRepo, sourceRepo, producer, dataMetrics, logger)

	jwtMgr, err := auth.NewJWTManager(cfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	sourceHandler := handler.NewSourceHandler(sourceSvc, logger)
	modelHandler := handler.NewModelHandler(modelSvc, logger)

	datahealth.Register(svc.Router, svc.Health, "data-service", cfg.Observability.ServiceName)
	handler.RegisterRoutes(svc.Router, sourceHandler, modelHandler, jwtMgr, svc.Redis)

	var dataConsumer *dataconsumer.DataConsumer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaConsumer, err := events.NewConsumer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka consumer unavailable — background discovery disabled")
		} else {
			dataConsumer = dataconsumer.NewDataConsumer(sourceSvc, kafkaConsumer, logger)
		}
	}

	g, gCtx := errgroup.WithContext(ctx)
	if dataConsumer != nil {
		g.Go(func() error {
			err := dataConsumer.Start(gCtx)
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		})
	}

	logger.Info().Int("port", bootstrapCfg.Port).Msg("data-service starting")
	runErr := svc.Run(ctx)
	cancel()
	if waitErr := g.Wait(); waitErr != nil {
		logger.Error().Err(waitErr).Msg("data-service background components stopped with error")
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error().Err(runErr).Msg("data-service stopped with error")
	}
	if dataConsumer != nil {
		_ = dataConsumer.Stop()
	}
	if producer != nil {
		_ = producer.Close()
	}
}

func buildBootstrapConfig(cfg *config.Config, dataCfg *dataconfig.Config) (*bootstrap.ServiceConfig, error) {
	redisURL, err := url.Parse(dataCfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	redisPassword, _ := redisURL.User.Password()
	redisDB := 0
	if dbSegment := strings.TrimPrefix(redisURL.Path, "/"); dbSegment != "" {
		if parsed, parseErr := strconv.Atoi(dbSegment); parseErr == nil {
			redisDB = parsed
		}
	}

	return &bootstrap.ServiceConfig{
		Name:            "data-service",
		Version:         cfg.Observability.ServiceName,
		Environment:     envOr("ENVIRONMENT", "development"),
		Port:            mustParsePort(dataCfg.HTTPPort, 8091),
		AdminPort:       cfg.Observability.MetricsPort,
		LogLevel:        cfg.Observability.LogLevel,
		DebugSampleRate: 100,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		Tracing: tracing.TracerConfig{
			Enabled:     cfg.Observability.OTLPEndpoint != "",
			Endpoint:    cfg.Observability.OTLPEndpoint,
			ServiceName: "data-service",
			Version:     cfg.Observability.ServiceName,
			Environment: envOr("ENVIRONMENT", "development"),
			SampleRate:  1,
			Insecure:    true,
		},
		DB: &bootstrap.DBConfig{
			URL:               dataCfg.DBURL,
			MinConns:          dataCfg.DBMinConns,
			MaxConns:          dataCfg.DBMaxConns,
			MaxConnLife:       time.Hour,
			MaxConnIdle:       30 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr:     redisURL.Host,
			Password: redisPassword,
			DB:       redisDB,
		},
		Kafka: &bootstrap.KafkaConfig{
			Brokers: dataCfg.KafkaBrokers,
			GroupID: dataCfg.KafkaGroupID,
		},
	}, nil
}

func mustParsePort(raw string, fallback int) int {
	if port, err := strconv.Atoi(raw); err == nil {
		return port
	}
	return fallback
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
