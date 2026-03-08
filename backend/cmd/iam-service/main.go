package main

import (
	"context"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/handler"
	"github.com/clario360/platform/internal/iam/repository"
	"github.com/clario360/platform/internal/iam/service"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
)

func main() {
	ctx := context.Background()

	// Load legacy config for auth and Kafka settings.
	legacyCfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("loading config")
	}

	env := envOrDefault("ENVIRONMENT", "development")

	cfg := &bootstrap.ServiceConfig{
		Name:        "iam-service",
		Version:     "1.0.0",
		Environment: env,
		Port:        8081,
		AdminPort:   9091,
		LogLevel:    legacyCfg.Observability.LogLevel,
		DB: &bootstrap.DBConfig{
			URL:               "postgres://" + legacyCfg.Database.User + ":" + legacyCfg.Database.Password + "@" + legacyCfg.Database.Host + ":" + intToStr(legacyCfg.Database.Port) + "/platform_core?sslmode=" + legacyCfg.Database.SSLMode,
			MinConns:          legacyCfg.Database.MaxIdleConns,
			MaxConns:          legacyCfg.Database.MaxOpenConns,
			MaxConnLife:       legacyCfg.Database.ConnMaxLifetime,
			MaxConnIdle:       5 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr:     legacyCfg.Redis.Addr(),
			Password: legacyCfg.Redis.Password,
			DB:       legacyCfg.Redis.DB,
		},
		Kafka: &bootstrap.KafkaConfig{
			Brokers: legacyCfg.Kafka.Brokers,
			GroupID: "iam-service",
		},
		Tracing: tracing.TracerConfig{
			Enabled:     legacyCfg.Observability.OTLPEndpoint != "",
			Endpoint:    legacyCfg.Observability.OTLPEndpoint,
			ServiceName: "iam-service",
			Version:     "1.0.0",
			Environment: env,
			SampleRate:  0.1,
			Insecure:    true,
		},
		ShutdownTimeout: legacyCfg.Server.ShutdownTimeout,
		ReadTimeout:     legacyCfg.Server.ReadTimeout,
		WriteTimeout:    legacyCfg.Server.WriteTimeout,
	}

	svc, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to bootstrap iam-service")
	}

	// Register IAM-specific metrics.
	svc.Metrics.Counter("iam_logins_total", "Total login attempts", []string{"status", "method"})
	svc.Metrics.Counter("iam_tokens_issued_total", "Total tokens issued", []string{"grant_type"})

	// Initialize Kafka producer (optional — graceful degradation if unavailable).
	var producer *events.Producer
	kafkaProducer, producerErr := events.NewProducer(legacyCfg.Kafka, svc.Logger)
	if producerErr != nil {
		svc.Logger.Warn().Err(producerErr).Msg("kafka producer unavailable — events will not be published")
	} else {
		producer = kafkaProducer
		defer producer.Close()
	}

	// JWT Manager.
	jwtMgr, err := auth.NewJWTManager(legacyCfg.Auth)
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// Repositories (using raw pool for backward compatibility with existing repos).
	userRepo := repository.NewUserRepository(svc.DBPool)
	roleRepo := repository.NewRoleRepository(svc.DBPool)
	sessionRepo := repository.NewSessionRepository(svc.DBPool)
	tenantRepo := repository.NewTenantRepository(svc.DBPool)
	apiKeyRepo := repository.NewAPIKeyRepository(svc.DBPool)

	// Services.
	authSvc := service.NewAuthService(
		userRepo, sessionRepo, roleRepo, tenantRepo,
		jwtMgr, svc.Redis, producer, svc.Logger,
		legacyCfg.Auth.BcryptCost, legacyCfg.Auth.RefreshTokenTTL,
	)
	userSvc := service.NewUserService(userRepo, roleRepo, sessionRepo, svc.Redis, producer, svc.Logger, legacyCfg.Auth.BcryptCost)
	roleSvc := service.NewRoleService(roleRepo, userRepo, producer, svc.Logger)
	tenantSvc := service.NewTenantService(tenantRepo, roleRepo, producer, svc.Logger)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, producer, svc.Logger)

	// Handlers.
	authHandler := handler.NewAuthHandler(authSvc, svc.Logger)
	userHandler := handler.NewUserHandler(userSvc, svc.Logger)
	roleHandler := handler.NewRoleHandler(roleSvc, svc.Logger)
	tenantHandler := handler.NewTenantHandler(tenantSvc, svc.Logger)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc, svc.Logger)

	// Security headers on all responses.
	svc.Router.Use(middleware.SecurityHeaders())

	// Routes.
	svc.Router.Route("/api/v1", func(r chi.Router) {
		r.Get("/internal/users/by-role", roleHandler.InternalUserIDsByRole)
		r.Get("/internal/users/{id}/email", userHandler.InternalGetEmail)

		// Public auth routes (no auth middleware).
		r.Mount("/auth", authHandler.Routes())

		// Login rate limiting.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(svc.Redis, middleware.RateLimitConfig{
				RequestsPerWindow: 20,
				Window:            1 * time.Minute,
				KeyPrefix:         "ratelimit:auth",
			}))
		})

		// Protected routes.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtMgr))
			r.Use(middleware.RateLimit(svc.Redis, middleware.DefaultRateLimitConfig()))
			r.Use(middleware.Tenant)
			r.Use(tracing.SpanEnricher())

			r.Mount("/users", userHandler.Routes())
			r.Mount("/roles", roleHandler.Routes())
			r.Mount("/tenants", tenantHandler.Routes())
			r.Mount("/api-keys", apiKeyHandler.Routes())

			r.Route("/users/{id}/roles", func(r chi.Router) {
				r.Get("/", roleHandler.GetUserRoles)
				r.Post("/", roleHandler.AssignRole)
				r.Delete("/{roleId}", roleHandler.RemoveRole)
			})
		})
	})

	svc.Logger.Info().Int("port", cfg.Port).Msg("iam-service starting")
	if err := svc.Run(ctx); err != nil {
		svc.Logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func intToStr(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
