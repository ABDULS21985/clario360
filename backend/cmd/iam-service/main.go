package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
	iamhandler "github.com/clario360/platform/internal/iam/handler"
	iamrepo "github.com/clario360/platform/internal/iam/repository"
	iamservice "github.com/clario360/platform/internal/iam/service"
	"github.com/clario360/platform/internal/middleware"
	notifchannel "github.com/clario360/platform/internal/notification/channel"
	notifcfg "github.com/clario360/platform/internal/notification/config"
	notifservice "github.com/clario360/platform/internal/notification/service"
	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	onboardinghandler "github.com/clario360/platform/internal/onboarding/handler"
	onboardingmiddleware "github.com/clario360/platform/internal/onboarding/middleware"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
	onboardingsvc "github.com/clario360/platform/internal/onboarding/service"
	"github.com/clario360/platform/pkg/storage"
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
	userRepo := iamrepo.NewUserRepository(svc.DBPool)
	roleRepo := iamrepo.NewRoleRepository(svc.DBPool)
	sessionRepo := iamrepo.NewSessionRepository(svc.DBPool)
	tenantRepo := iamrepo.NewTenantRepository(svc.DBPool)
	apiKeyRepo := iamrepo.NewAPIKeyRepository(svc.DBPool)

	// Services.
	authSvc := iamservice.NewAuthService(
		userRepo, sessionRepo, roleRepo, tenantRepo,
		jwtMgr, svc.Redis, producer, svc.Logger,
		legacyCfg.Auth.BcryptCost, legacyCfg.Auth.RefreshTokenTTL,
	)
	userSvc := iamservice.NewUserService(userRepo, roleRepo, sessionRepo, svc.Redis, producer, svc.Logger, legacyCfg.Auth.BcryptCost)
	roleSvc := iamservice.NewRoleService(roleRepo, userRepo, producer, svc.Logger)
	tenantSvc := iamservice.NewTenantService(tenantRepo, roleRepo, producer, svc.Logger)
	apiKeySvc := iamservice.NewAPIKeyService(apiKeyRepo, producer, svc.Logger)

	// Handlers.
	authHandler := iamhandler.NewAuthHandler(authSvc, svc.Logger)
	userHandler := iamhandler.NewUserHandler(userSvc, svc.Logger)
	roleHandler := iamhandler.NewRoleHandler(roleSvc, svc.Logger)
	tenantHandler := iamhandler.NewTenantHandler(tenantSvc, svc.Logger)
	apiKeyHandler := iamhandler.NewAPIKeyHandler(apiKeySvc, svc.Logger)

	// Onboarding dependencies.
	onboardingMetrics := onboardingsvc.NewMetrics(svc.Metrics)
	dbPools, dbDSNs, err := buildOnboardingDBPools(ctx, legacyCfg, svc.Logger)
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to initialize onboarding database pools")
	}
	for _, pool := range dbPools {
		defer pool.Close()
	}

	storageClient := buildOnboardingStorage(ctx, legacyCfg, svc.Logger)
	emailSender := buildOnboardingEmailSender(svc.Logger)
	migrationsBasePath := resolveMigrationsBasePath()

	onboardingRepository := onboardingrepo.NewOnboardingRepository(svc.DBPool)
	invitationRepository := onboardingrepo.NewInvitationRepository(svc.DBPool)
	provisioningRepository := onboardingrepo.NewProvisioningRepository(svc.DBPool)

	provisioner := onboardingsvc.NewTenantProvisioner(
		svc.DBPool,
		dbPools,
		dbDSNs,
		migrationsBasePath,
		onboardingRepository,
		provisioningRepository,
		storageClient,
		emailSender,
		producer,
		svc.Logger,
		onboardingMetrics,
	)
	registrationService := onboardingsvc.NewRegistrationService(
		onboardingRepository,
		userRepo,
		roleRepo,
		sessionRepo,
		jwtMgr,
		svc.Redis,
		producer,
		emailSender,
		provisioner,
		svc.Logger,
		onboardingMetrics,
		legacyCfg.Auth.BcryptCost,
		legacyCfg.Auth.RefreshTokenTTL,
	)
	invitationService := onboardingsvc.NewInvitationService(
		invitationRepository,
		onboardingRepository,
		userRepo,
		roleRepo,
		sessionRepo,
		jwtMgr,
		producer,
		emailSender,
		svc.Logger,
		onboardingMetrics,
		legacyCfg.Auth.BcryptCost,
		legacyCfg.Auth.RefreshTokenTTL,
	)
	wizardService := onboardingsvc.NewWizardService(
		onboardingRepository,
		invitationService,
		producer,
		svc.Logger,
		onboardingMetrics,
	)
	deprovisioner := onboardingsvc.NewTenantDeprovisioner(
		svc.DBPool,
		dbPools,
		onboardingRepository,
		storageClient,
		svc.Redis,
		producer,
		svc.Logger,
		onboardingMetrics,
	)
	onboardingHandler := onboardinghandler.New(
		registrationService,
		wizardService,
		invitationService,
		provisioner,
		deprovisioner,
		provisioningRepository,
		svc.Logger,
	)

	// Security headers on all responses.
	svc.Router.Use(middleware.SecurityHeaders())

	// Routes.
	svc.Router.Route("/api/v1", func(r chi.Router) {
		r.Get("/internal/users/by-role", roleHandler.InternalUserIDsByRole)
		r.Get("/internal/users/{id}/email", userHandler.InternalGetEmail)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(svc.Redis, middleware.RateLimitConfig{
				RequestsPerWindow: 20,
				Window:            1 * time.Minute,
				KeyPrefix:         "ratelimit:auth",
			}))
			r.Mount("/auth", authHandler.Routes())
		})

		r.Route("/onboarding", func(r chi.Router) {
			r.With(onboardingmiddleware.NewPublicRateLimiter(svc.Redis, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 5,
				Window:            time.Hour,
				KeyPrefix:         "ratelimit:onboarding:register",
			})).Post("/register", onboardingHandler.Register)
			r.With(onboardingmiddleware.NewPublicRateLimiter(svc.Redis, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 20,
				Window:            10 * time.Minute,
				KeyPrefix:         "ratelimit:onboarding:verify-email",
			})).Post("/verify-email", onboardingHandler.VerifyEmail)
			r.With(onboardingmiddleware.NewPublicRateLimiter(svc.Redis, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 1,
				Window:            time.Minute,
				KeyPrefix:         "ratelimit:onboarding:resend-otp",
			})).Post("/resend-otp", onboardingHandler.ResendOTP)
			r.With(
				middleware.OptionalAuth(jwtMgr),
				onboardingmiddleware.NewPublicRateLimiter(svc.Redis, onboardingmiddleware.PublicRateLimitConfig{
					RequestsPerWindow: 120,
					Window:            time.Minute,
					KeyPrefix:         "ratelimit:onboarding:status",
				}),
			).Get("/status/{tenantId}", onboardingHandler.GetOnboardingStatus)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(jwtMgr))
				r.Use(middleware.RateLimit(svc.Redis, middleware.DefaultRateLimitConfig()))
				r.Use(middleware.Tenant)
				r.Use(tracing.SpanEnricher())

				r.Get("/wizard", onboardingHandler.GetWizardProgress)
				r.Post("/wizard/organization", onboardingHandler.SaveOrganization)
				r.Post("/wizard/branding", onboardingHandler.SaveBranding)
				r.Post("/wizard/team", onboardingHandler.SaveTeam)
				r.Post("/wizard/suites", onboardingHandler.SaveSuites)
				r.Post("/wizard/complete", onboardingHandler.CompleteWizard)
			})
		})

		r.Route("/invitations", func(r chi.Router) {
			r.With(onboardingmiddleware.NewPublicRateLimiter(svc.Redis, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 60,
				Window:            time.Minute,
				KeyPrefix:         "ratelimit:onboarding:invite-validate",
			})).Get("/validate", onboardingHandler.ValidateInviteToken)
			r.With(onboardingmiddleware.NewPublicRateLimiter(svc.Redis, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 10,
				Window:            15 * time.Minute,
				KeyPrefix:         "ratelimit:onboarding:invite-accept",
			})).Post("/accept", onboardingHandler.AcceptInvitation)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(jwtMgr))
				r.Use(middleware.RateLimit(svc.Redis, middleware.DefaultRateLimitConfig()))
				r.Use(middleware.Tenant)
				r.Use(tracing.SpanEnricher())

				r.Get("/", onboardingHandler.ListInvitations)
				r.Post("/", onboardingHandler.CreateBatchInvitations)
				r.Delete("/{id}", onboardingHandler.CancelInvitation)
				r.Post("/resend/{id}", onboardingHandler.ResendInvitation)
			})
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

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.Auth(jwtMgr))
			r.Use(middleware.RateLimit(svc.Redis, middleware.DefaultRateLimitConfig()))
			r.Use(tracing.SpanEnricher())

			r.Post("/tenants/provision", onboardingHandler.AdminProvision)
			r.Get("/tenants/{id}/provision-status", onboardingHandler.AdminGetProvisionStatus)
			r.Post("/tenants/{id}/deprovision", onboardingHandler.AdminDeprovision)
			r.Post("/tenants/{id}/reprovision", onboardingHandler.AdminReprovision)
			r.Post("/tenants/{id}/reactivate", onboardingHandler.AdminReactivate)
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
