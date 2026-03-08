package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	actaapp "github.com/clario360/platform/internal/acta"
	actaconfig "github.com/clario360/platform/internal/acta/config"
	actaconsumer "github.com/clario360/platform/internal/acta/consumer"
	actahealth "github.com/clario360/platform/internal/acta/health"
	actascheduler "github.com/clario360/platform/internal/acta/scheduler"
	actasvc "github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	sharedmw "github.com/clario360/platform/internal/middleware"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
	"github.com/rs/zerolog"
)

const serviceVersion = "1.0.0"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	baseCfg, err := appconfig.Load()
	if err != nil {
		os.Stderr.WriteString("loading platform config: " + err.Error() + "\n")
		os.Exit(1)
	}
	actaCfg, err := actaconfig.Load(baseCfg)
	if err != nil {
		os.Stderr.WriteString("loading acta config: " + err.Error() + "\n")
		os.Exit(1)
	}
	bootstrapCfg := buildBootstrapConfig(baseCfg, actaCfg)
	svc, err := bootstrap.Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		os.Stderr.WriteString("bootstrapping acta-service: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := svc.Logger

	if err := runMigrations(actaCfg.DBURL); err != nil {
		logger.Fatal().Err(err).Msg("failed to run acta migrations")
	}
	if err := workflowrepo.RunMigration(ctx, svc.DBPool); err != nil {
		logger.Fatal().Err(err).Msg("failed to run workflow schema migration")
	}

	if actaCfg.JWTPublicKeyPath != "" {
		publicKeyPEM, err := os.ReadFile(actaCfg.JWTPublicKeyPath)
		if err != nil {
			logger.Fatal().Err(err).Str("path", actaCfg.JWTPublicKeyPath).Msg("failed to read ACTA_JWT_PUBLIC_KEY_PATH")
		}
		baseCfg.Auth.RSAPublicKeyPEM = string(publicKeyPEM)
	}
	jwtMgr, err := auth.NewJWTManager(baseCfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	var (
		producer  *events.Producer
		publisher actasvc.Publisher
	)
	if len(actaCfg.KafkaBrokers) > 0 {
		producer, err = events.NewProducer(appconfig.KafkaConfig{
			Brokers: actaCfg.KafkaBrokers,
			GroupID: actaCfg.KafkaGroupID,
		}, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka producer unavailable; acta events will not be published")
		} else {
			publisher = producer
			defer producer.Close()
		}
	}

	app, err := actaapp.NewApplication(actaapp.Dependencies{
		DB:                svc.DBPool,
		Redis:             svc.Redis,
		Publisher:         publisher,
		Logger:            logger,
		Registerer:        svc.Metrics.Registry(),
		DashboardCacheTTL: actaCfg.DashboardCacheTTL,
		KafkaTopic:        actaCfg.KafkaTopic,
		WorkflowDefRepo:   workflowrepo.NewDefinitionRepository(svc.DBPool),
		WorkflowInstRepo:  workflowrepo.NewInstanceRepository(svc.DBPool),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize acta application")
	}
	if actaCfg.SeedDemoData {
		tenantID, err := actaapp.SeedDemoData(ctx, app.Store, logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to seed acta demo data")
		}
		if _, err := app.ComplianceService.RunChecks(ctx, tenantID); err != nil {
			logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("failed to compute initial compliance results for demo seed")
		}
	}

	svc.Router.Use(sharedmw.SecurityHeaders())
	actahealth.Register(svc.Router, svc.Health, bootstrapCfg.Name, bootstrapCfg.Version)
	app.RegisterRoutes(svc.Router, jwtMgr, svc.Redis, actaCfg.RateLimitPerMinute)
	dlqTracker := events.NewDLQTracker(svc.Redis)
	crossSuiteMetrics := events.NewCrossSuiteMetrics(svc.Metrics.Registry())
	svc.Router.Get("/api/v1/admin/dlq/count", events.DLQCountHandler("acta-service", dlqTracker, logger))

	var kafkaConsumer *events.Consumer
	if len(actaCfg.KafkaBrokers) > 0 {
		kafkaConsumer, err = events.NewConsumer(appconfig.KafkaConfig{
			Brokers:         actaCfg.KafkaBrokers,
			GroupID:         actaCfg.KafkaGroupID + "-consumer",
			AutoOffsetReset: baseCfg.Kafka.AutoOffsetReset,
		}, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka consumer unavailable; cross-suite acta sync disabled")
		} else {
			defer kafkaConsumer.Close()
			kafkaConsumer.SetDeadLetterProducer(producer)
			kafkaConsumer.SetCrossSuiteMetrics(crossSuiteMetrics)
			kafkaConsumer.SetDLQTracker(dlqTracker, "acta-service")
			actaConsumer := actaconsumer.NewActaConsumer(app.Store, kafkaConsumer, logger)
			go runBackground(ctx, logger, "acta-consumer", actaConsumer.Start)
		}
	}

	go runBackground(ctx, logger, "acta-overdue-checker", actascheduler.NewOverdueChecker(app.ActionItemService, actaCfg.OverdueCheckInterval, logger).Run)
	go runBackground(ctx, logger, "acta-meeting-reminder", actascheduler.NewMeetingReminder(app.Store, publisher, actaCfg.MeetingReminderInterval, logger).Run)
	go runBackground(ctx, logger, "acta-compliance-scheduler", actascheduler.NewComplianceScheduler(app.Store, app.ComplianceService, actaCfg.ComplianceCheckInterval, actaCfg.ComplianceCheckHourUTC, logger).Run)

	logger.Info().Int("port", bootstrapCfg.Port).Msg("acta-service starting")
	if err := svc.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("acta-service failed")
	}
}

func buildBootstrapConfig(baseCfg *appconfig.Config, actaCfg *actaconfig.Config) *bootstrap.ServiceConfig {
	env := envOr("ENVIRONMENT", "development")
	cfg := &bootstrap.ServiceConfig{
		Name:        "acta-service",
		Version:     serviceVersion,
		Environment: env,
		Port:        actaCfg.HTTPPort,
		AdminPort:   actaCfg.AdminPort,
		LogLevel:    baseCfg.Observability.LogLevel,
		DB: &bootstrap.DBConfig{
			URL:               actaCfg.DBURL,
			MinConns:          actaCfg.DBMinConns,
			MaxConns:          actaCfg.DBMaxConns,
			MaxConnLife:       baseCfg.Database.ConnMaxLifetime,
			MaxConnIdle:       5 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr:     actaCfg.RedisAddr,
			Password: actaCfg.RedisPassword,
			DB:       actaCfg.RedisDB,
		},
		Tracing: tracing.TracerConfig{
			Enabled:     baseCfg.Observability.OTLPEndpoint != "",
			Endpoint:    baseCfg.Observability.OTLPEndpoint,
			ServiceName: "acta-service",
			Version:     serviceVersion,
			Environment: env,
			SampleRate:  0.1,
			Insecure:    true,
		},
		ShutdownTimeout: baseCfg.Server.ShutdownTimeout,
		ReadTimeout:     baseCfg.Server.ReadTimeout,
		WriteTimeout:    baseCfg.Server.WriteTimeout,
	}
	if len(actaCfg.KafkaBrokers) > 0 {
		cfg.Kafka = &bootstrap.KafkaConfig{
			Brokers: actaCfg.KafkaBrokers,
			GroupID: actaCfg.KafkaGroupID,
		}
	}
	return cfg
}

func runMigrations(dsn string) error {
	migrationsPath := "migrations/acta_db"
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "acta_db")
	}
	return database.RunMigrations(dsn, migrationsPath)
}

func runBackground(ctx context.Context, logger zerolog.Logger, name string, fn func(context.Context) error) {
	if err := fn(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error().Err(err).Str("job", name).Msg("background job exited")
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
