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
	sharedmw "github.com/clario360/platform/internal/middleware"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	visusapp "github.com/clario360/platform/internal/visus"
	visusconfig "github.com/clario360/platform/internal/visus/config"
	visushealth "github.com/clario360/platform/internal/visus/health"
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
	visusCfg, err := visusconfig.Load(baseCfg)
	if err != nil {
		os.Stderr.WriteString("loading visus config: " + err.Error() + "\n")
		os.Exit(1)
	}

	bootstrapCfg := buildBootstrapConfig(baseCfg, visusCfg)
	svc, err := bootstrap.Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		os.Stderr.WriteString("bootstrapping visus-service: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := svc.Logger

	if err := runMigrations(visusCfg.DBURL); err != nil {
		logger.Fatal().Err(err).Msg("failed to run visus migrations")
	}

	if visusCfg.JWTPublicKeyPath != "" {
		publicKeyPEM, err := os.ReadFile(visusCfg.JWTPublicKeyPath)
		if err != nil {
			logger.Fatal().Err(err).Str("path", visusCfg.JWTPublicKeyPath).Msg("failed to read VISUS_JWT_PUBLIC_KEY_PATH")
		}
		baseCfg.Auth.RSAPublicKeyPEM = string(publicKeyPEM)
	}
	if visusCfg.JWTPrivateKeyPath != "" {
		privateKeyPEM, err := os.ReadFile(visusCfg.JWTPrivateKeyPath)
		if err != nil {
			logger.Fatal().Err(err).Str("path", visusCfg.JWTPrivateKeyPath).Msg("failed to read VISUS_JWT_PRIVATE_KEY_PATH")
		}
		baseCfg.Auth.RSAPrivateKeyPEM = string(privateKeyPEM)
	}
	jwtMgr, err := auth.NewJWTManager(baseCfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	var producer *events.Producer
	if len(visusCfg.KafkaBrokers) > 0 {
		producer, err = events.NewProducer(appconfig.KafkaConfig{
			Brokers: visusCfg.KafkaBrokers,
			GroupID: visusCfg.KafkaGroupID,
		}, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka producer unavailable; visus events will not be published")
		} else {
			defer producer.Close()
		}
	}

	app, err := visusapp.NewApplication(visusapp.Dependencies{
		DB:         svc.DBPool,
		Redis:      svc.Redis,
		Publisher:  producer,
		Logger:     logger,
		Registerer: svc.Metrics.Registry(),
		Config:     visusCfg,
		JWTManager: jwtMgr,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize visus application")
	}

	if visusCfg.SeedDemoData {
		tenantID, err := visusapp.SeedDemoData(ctx, app, visusCfg, logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to seed visus demo data")
		}
		logger.Info().Str("tenant_id", tenantID.String()).Msg("visus demo data ready")
	}

	svc.Router.Use(sharedmw.SecurityHeaders())
	visushealth.Register(svc.Router, svc.Health, bootstrapCfg.Name, bootstrapCfg.Version)
	app.RegisterRoutes(svc.Router, jwtMgr, svc.Redis, visusCfg.RateLimitPerMinute)

	go runBackground(ctx, logger, "visus-kpi-scheduler", app.KPIScheduler.Run)
	go runBackground(ctx, logger, "visus-report-scheduler", app.ReportScheduler.Run)

	if len(visusCfg.KafkaBrokers) > 0 {
		kafkaConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
			Brokers:         visusCfg.KafkaBrokers,
			GroupID:         visusCfg.KafkaGroupID + "-consumer",
			AutoOffsetReset: baseCfg.Kafka.AutoOffsetReset,
		}, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka consumer unavailable; visus cross-suite sync disabled")
		} else {
			defer kafkaConsumer.Close()
			app.Consumer.Register(kafkaConsumer)
			go runBackground(ctx, logger, "visus-consumer", kafkaConsumer.Start)
		}
	}

	logger.Info().Int("port", bootstrapCfg.Port).Msg("visus-service starting")
	if err := svc.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("visus-service failed")
	}
}

func buildBootstrapConfig(baseCfg *appconfig.Config, visusCfg *visusconfig.Config) *bootstrap.ServiceConfig {
	env := envOr("ENVIRONMENT", "development")
	cfg := &bootstrap.ServiceConfig{
		Name:        "visus-service",
		Version:     serviceVersion,
		Environment: env,
		Port:        visusCfg.HTTPPort,
		AdminPort:   visusCfg.AdminPort,
		LogLevel:    baseCfg.Observability.LogLevel,
		DB: &bootstrap.DBConfig{
			URL:               visusCfg.DBURL,
			MinConns:          visusCfg.DBMinConns,
			MaxConns:          visusCfg.DBMaxConns,
			MaxConnLife:       baseCfg.Database.ConnMaxLifetime,
			MaxConnIdle:       5 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr:     visusCfg.RedisAddr,
			Password: visusCfg.RedisPassword,
			DB:       visusCfg.RedisDB,
		},
		Tracing: tracing.TracerConfig{
			Enabled:     baseCfg.Observability.OTLPEndpoint != "",
			Endpoint:    baseCfg.Observability.OTLPEndpoint,
			ServiceName: "visus-service",
			Version:     serviceVersion,
			Environment: env,
			SampleRate:  0.1,
			Insecure:    true,
		},
		ShutdownTimeout: baseCfg.Server.ShutdownTimeout,
		ReadTimeout:     baseCfg.Server.ReadTimeout,
		WriteTimeout:    baseCfg.Server.WriteTimeout,
	}
	if len(visusCfg.KafkaBrokers) > 0 {
		cfg.Kafka = &bootstrap.KafkaConfig{
			Brokers: visusCfg.KafkaBrokers,
			GroupID: visusCfg.KafkaGroupID,
		}
	}
	return cfg
}

func runMigrations(dsn string) error {
	migrationsPath := envOr("VISUS_MIGRATIONS_PATH", filepath.Join("migrations", "visus_db"))
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "visus_db")
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
