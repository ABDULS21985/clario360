package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	lexapp "github.com/clario360/platform/internal/lex"
	lexconfig "github.com/clario360/platform/internal/lex/config"
	lexconsumer "github.com/clario360/platform/internal/lex/consumer"
	lexhealth "github.com/clario360/platform/internal/lex/health"
	lexmonitor "github.com/clario360/platform/internal/lex/monitor"
	sharedmw "github.com/clario360/platform/internal/middleware"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
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
	lexCfg, err := lexconfig.Load(baseCfg)
	if err != nil {
		os.Stderr.WriteString("loading lex config: " + err.Error() + "\n")
		os.Exit(1)
	}

	bootstrapCfg := buildBootstrapConfig(baseCfg, lexCfg)
	svc, err := bootstrap.Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		os.Stderr.WriteString("bootstrapping lex-service: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := svc.Logger

	if err := runMigrations(lexCfg.DBURL); err != nil {
		logger.Fatal().Err(err).Msg("failed to run lex migrations")
	}
	if err := workflowrepo.RunMigration(ctx, svc.DBPool); err != nil {
		logger.Fatal().Err(err).Msg("failed to run workflow schema migration")
	}

	if lexCfg.JWTPublicKeyPath != "" {
		publicKeyPEM, err := os.ReadFile(lexCfg.JWTPublicKeyPath)
		if err != nil {
			logger.Fatal().Err(err).Str("path", lexCfg.JWTPublicKeyPath).Msg("failed to read LEX_JWT_PUBLIC_KEY_PATH")
		}
		baseCfg.Auth.RSAPublicKeyPEM = string(publicKeyPEM)
	}
	jwtMgr, err := auth.NewJWTManager(baseCfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	var producer *events.Producer
	if len(lexCfg.KafkaBrokers) > 0 {
		producer, err = events.NewProducer(appconfig.KafkaConfig{
			Brokers: lexCfg.KafkaBrokers,
			GroupID: lexCfg.KafkaGroupID,
		}, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka producer unavailable; lex events will not be published")
		} else {
			defer producer.Close()
		}
	}

	app, err := lexapp.NewApplication(lexapp.Dependencies{
		DB:                svc.DBPool,
		Redis:             svc.Redis,
		Publisher:         producer,
		Logger:            logger,
		Registerer:        svc.Metrics.Registry(),
		WorkflowDefRepo:   workflowrepo.NewDefinitionRepository(svc.DBPool),
		WorkflowInstRepo:  workflowrepo.NewInstanceRepository(svc.DBPool),
		WorkflowTaskRepo:  workflowrepo.NewTaskRepository(svc.DBPool),
		Config:            lexCfg,
		DashboardCacheTTL: lexCfg.DashboardCacheTTL,
		OrgJurisdiction:   lexCfg.OrgJurisdiction,
		KafkaTopic:        lexCfg.KafkaTopic,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize lex application")
	}
	if lexCfg.SeedDemoData {
		if _, err := lexapp.SeedDemoData(ctx, app, logger); err != nil {
			logger.Fatal().Err(err).Msg("failed to seed lex demo data")
		}
	}

	svc.Router.Use(sharedmw.SecurityHeaders())
	lexhealth.Register(svc.Router, svc.Health, bootstrapCfg.Name, bootstrapCfg.Version)
	app.RegisterRoutes(svc.Router, jwtMgr, svc.Redis, lexCfg.RateLimitPerMinute)
	dlqTracker := events.NewDLQTracker(svc.Redis)
	crossSuiteMetrics := events.NewCrossSuiteMetrics(svc.Metrics.Registry())
	svc.Router.Get("/api/v1/admin/dlq/count", events.DLQCountHandler("lex-service", dlqTracker, logger))

	expiryMonitor := lexmonitor.NewExpiryMonitor(
		svc.DBPool,
		app.Store.Contracts,
		app.Store.Alerts,
		app.ContractService,
		app.Metrics,
		producer,
		lexCfg.KafkaTopic,
		lexCfg.ExpiryMonitorInterval,
		logger,
	)
	complianceMonitor := lexmonitor.NewComplianceMonitor(
		app.Store.Contracts,
		app.ComplianceService,
		lexCfg.ComplianceMonitorInterval,
		logger,
	)
	renewalReminder := lexmonitor.NewRenewalReminder(
		svc.DBPool,
		app.Store.Contracts,
		app.Store.Alerts,
		producer,
		lexCfg.KafkaTopic,
		lexCfg.RenewalReminderInterval,
		logger,
	)

	go runBackground(ctx, logger, "lex-expiry-monitor", expiryMonitor.Run)
	go runBackground(ctx, logger, "lex-compliance-monitor", complianceMonitor.Run)
	go runBackground(ctx, logger, "lex-renewal-reminder", renewalReminder.Run)

	if len(lexCfg.KafkaBrokers) > 0 {
		kafkaConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
			Brokers:         lexCfg.KafkaBrokers,
			GroupID:         lexCfg.KafkaGroupID + "-consumer",
			AutoOffsetReset: baseCfg.Kafka.AutoOffsetReset,
		}, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka consumer unavailable; lex cross-suite sync disabled")
		} else {
			defer kafkaConsumer.Close()
			kafkaConsumer.SetDeadLetterProducer(producer)
			kafkaConsumer.SetCrossSuiteMetrics(crossSuiteMetrics)
			kafkaConsumer.SetDLQTracker(dlqTracker, "lex-service")
			handler := lexconsumer.NewLexConsumer(app.ComplianceService, app.WorkflowService, kafkaConsumer, logger)
			go runBackground(ctx, logger, "lex-consumer", handler.Start)
		}
	}

	logger.Info().Int("port", bootstrapCfg.Port).Msg("lex-service starting")
	if err := svc.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("lex-service failed")
	}
}

func buildBootstrapConfig(baseCfg *appconfig.Config, lexCfg *lexconfig.Config) *bootstrap.ServiceConfig {
	env := envOr("ENVIRONMENT", "development")
	cfg := &bootstrap.ServiceConfig{
		Name:        "lex-service",
		Version:     serviceVersion,
		Environment: env,
		Port:        lexCfg.HTTPPort,
		AdminPort:   lexCfg.AdminPort,
		LogLevel:    baseCfg.Observability.LogLevel,
		DB: &bootstrap.DBConfig{
			URL:               lexCfg.DBURL,
			MinConns:          lexCfg.DBMinConns,
			MaxConns:          lexCfg.DBMaxConns,
			MaxConnLife:       baseCfg.Database.ConnMaxLifetime,
			MaxConnIdle:       5 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr:     lexCfg.RedisAddr,
			Password: lexCfg.RedisPassword,
			DB:       lexCfg.RedisDB,
		},
		Tracing: tracing.TracerConfig{
			Enabled:     baseCfg.Observability.OTLPEndpoint != "",
			Endpoint:    baseCfg.Observability.OTLPEndpoint,
			ServiceName: "lex-service",
			Version:     serviceVersion,
			Environment: env,
			SampleRate:  0.1,
			Insecure:    true,
		},
		ShutdownTimeout: baseCfg.Server.ShutdownTimeout,
		ReadTimeout:     baseCfg.Server.ReadTimeout,
		WriteTimeout:    baseCfg.Server.WriteTimeout,
	}
	if len(lexCfg.KafkaBrokers) > 0 {
		cfg.Kafka = &bootstrap.KafkaConfig{
			Brokers: lexCfg.KafkaBrokers,
			GroupID: lexCfg.KafkaGroupID,
		}
	}
	return cfg
}

func runMigrations(dsn string) error {
	migrationsPath := envOr("LEX_MIGRATIONS_PATH", filepath.Join("migrations", "lex_db"))
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "lex_db")
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
