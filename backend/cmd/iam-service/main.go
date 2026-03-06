package main

import (
	"context"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/handler"
	"github.com/clario360/platform/internal/iam/repository"
	"github.com/clario360/platform/internal/iam/service"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}
	cfg.Server.Port = 8081
	cfg.Database.Name = "platform_core"

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"iam-service",
	)

	ctx := context.Background()

	shutdownTracer, err := observability.InitTracer(ctx, "iam-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Initialize Kafka producer (optional — graceful degradation if unavailable)
	var producer *events.Producer
	kafkaProducer, err := events.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("kafka producer unavailable — events will not be published")
	} else {
		producer = kafkaProducer
		defer producer.Close()
	}

	// ---- Repositories ----
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	// ---- Services ----
	srv := server.New(cfg, db, rdb, logger)

	authSvc := service.NewAuthService(
		userRepo, sessionRepo, roleRepo, tenantRepo,
		srv.JWTManager, rdb, producer, logger,
		cfg.Auth.BcryptCost, cfg.Auth.RefreshTokenTTL,
	)
	userSvc := service.NewUserService(userRepo, roleRepo, sessionRepo, rdb, producer, logger, cfg.Auth.BcryptCost)
	roleSvc := service.NewRoleService(roleRepo, userRepo, producer, logger)
	tenantSvc := service.NewTenantService(tenantRepo, roleRepo, producer, logger)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, producer, logger)

	// ---- Handlers ----
	authHandler := handler.NewAuthHandler(authSvc, logger)
	userHandler := handler.NewUserHandler(userSvc, logger)
	roleHandler := handler.NewRoleHandler(roleSvc, logger)
	tenantHandler := handler.NewTenantHandler(tenantSvc, logger)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc, logger)

	// ---- Routes ----
	srv.Router.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no auth middleware)
		r.Mount("/auth", authHandler.Routes())

		// Login rate limiting on auth routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(rdb, middleware.RateLimitConfig{
				RequestsPerWindow: 20,
				Window:            1 * time.Minute,
				KeyPrefix:         "ratelimit:auth",
			}))
		})

		// Protected routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(srv.JWTManager))
			r.Use(middleware.RateLimit(rdb, middleware.DefaultRateLimitConfig()))
			r.Use(middleware.Tenant)

			r.Mount("/users", userHandler.Routes())
			r.Mount("/roles", roleHandler.Routes())
			r.Mount("/tenants", tenantHandler.Routes())
			r.Mount("/api-keys", apiKeyHandler.Routes())

			// Nested user role routes: POST /users/{id}/roles, DELETE /users/{id}/roles/{roleId}
			r.Route("/users/{id}/roles", func(r chi.Router) {
				r.Post("/", roleHandler.AssignRole)
				r.Delete("/{roleId}", roleHandler.RemoveRole)
			})
		})
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("iam-service starting")
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}
