package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	aigovdrift "github.com/clario360/platform/internal/aigovernance/drift"
	aigovhandler "github.com/clario360/platform/internal/aigovernance/handler"
	aigovrepo "github.com/clario360/platform/internal/aigovernance/repository"
	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	aigovshadow "github.com/clario360/platform/internal/aigovernance/shadow"
	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
	filemetrics "github.com/clario360/platform/internal/filemanager/metrics"
	filerepo "github.com/clario360/platform/internal/filemanager/repository"
	fileservice "github.com/clario360/platform/internal/filemanager/service"
	iamhandler "github.com/clario360/platform/internal/iam/handler"
	iamrepo "github.com/clario360/platform/internal/iam/repository"
	iamservice "github.com/clario360/platform/internal/iam/service"
	"github.com/clario360/platform/internal/middleware"
	notebookconsumer "github.com/clario360/platform/internal/notebook/consumer"
	notebookhandler "github.com/clario360/platform/internal/notebook/handler"
	notebookservice "github.com/clario360/platform/internal/notebook/service"
	notifchannel "github.com/clario360/platform/internal/notification/channel"
	notifcfg "github.com/clario360/platform/internal/notification/config"
	notifservice "github.com/clario360/platform/internal/notification/service"
	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	onboardinghandler "github.com/clario360/platform/internal/onboarding/handler"
	onboardingmiddleware "github.com/clario360/platform/internal/onboarding/middleware"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
	onboardingsvc "github.com/clario360/platform/internal/onboarding/service"
	"github.com/clario360/platform/internal/security"
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
		AdminPort:   9081,
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
	// Monitoring alert metric (used by clario360-alerts.yaml for brute-force detection).
	svc.Metrics.Counter("clario360_auth_login_failures_total", "Failed login attempts by source IP", []string{"ip"})

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
	oauthClients := []iamservice.OAuthClient{
		{
			ClientID:     "jupyterhub",
			ClientSecret: envOrDefault("JUPYTERHUB_OAUTH_CLIENT_SECRET", ""),
			RedirectURIs: splitCSV(
				envOrDefault("JUPYTERHUB_OAUTH_CALLBACK_URL", "https://notebooks.clario360.sa/hub/oauth_callback"),
			),
			Scopes:      []string{"openid", "profile", "email", "roles"},
			RequirePKCE: true,
		},
	}
	oauthClients = append(oauthClients, loadAdditionalOAuthClientsFromEnv(svc.Logger)...)
	oauthSvc := iamservice.NewOAuthService(
		jwtMgr,
		authSvc,
		userRepo,
		tenantRepo,
		svc.Redis,
		envOrDefault("CLARIO360_PUBLIC_URL", "http://localhost:8080"),
		envOrDefault("NOTEBOOK_LOGIN_URL", envOrDefault("CLARIO360_APP_URL", "http://localhost:3000")+"/login"),
		oauthClients,
		svc.Logger,
	)
	notebookMetrics := security.NewNotebookMetrics(svc.Metrics.Registry())
	notebookSvc := notebookservice.NewNotebookService(
		envOrDefault("JUPYTERHUB_API_URL", "http://jupyterhub.jupyterhub.svc:8081/hub/api"),
		envOrDefault("JUPYTERHUB_BASE_URL", "https://notebooks.clario360.sa"),
		envOrDefault("JUPYTERHUB_ADMIN_TOKEN", ""),
		nil,
		producer,
		notebookMetrics,
		svc.Logger,
	)

	// Handlers.
	authHandler := iamhandler.NewAuthHandler(authSvc, svc.Logger)
	userHandler := iamhandler.NewUserHandler(userSvc, svc.Logger)
	roleHandler := iamhandler.NewRoleHandler(roleSvc, svc.Logger)
	tenantHandler := iamhandler.NewTenantHandler(tenantSvc, svc.Logger)
	apiKeyHandler := iamhandler.NewAPIKeyHandler(apiKeySvc, svc.Logger)
	oauthHandler := iamhandler.NewOAuthHandler(oauthSvc, svc.Logger)
	notebookHandler := notebookhandler.NewNotebookHandler(notebookSvc, svc.Logger)

	// AI governance control plane.
	aiMetrics := aigovservice.NewMetrics(svc.Metrics.Registry())
	aiRegistryRepo := aigovrepo.NewModelRegistryRepository(svc.DBPool, svc.Logger)
	aiPredictionRepo := aigovrepo.NewPredictionLogRepository(svc.DBPool, svc.Logger)
	aiShadowRepo := aigovrepo.NewShadowComparisonRepository(svc.DBPool, svc.Logger)
	aiDriftRepo := aigovrepo.NewDriftReportRepository(svc.DBPool, svc.Logger)
	aiValidationRepo := aigovrepo.NewValidationResultRepository(svc.DBPool, svc.Logger)
	aiExplanationSvc := aigovservice.NewExplanationService(svc.Logger)
	aiRegistrySvc := aigovservice.NewRegistryService(aiRegistryRepo, producer, aiMetrics, svc.Logger)
	aiPredictionSvc := aigovservice.NewPredictionService(aiPredictionRepo, aiRegistryRepo, producer, aiMetrics, svc.Logger)
	aiComparisonSvc := aigovservice.NewComparisonService(aiRegistryRepo, aiPredictionRepo, aiShadowRepo, producer, aiMetrics, svc.Logger)
	aiShadowSvc := aigovservice.NewShadowService(aiRegistryRepo, aiShadowRepo, aiPredictionRepo, producer, aiMetrics, svc.Logger)
	aiLifecycleSvc := aigovservice.NewLifecycleService(aiRegistryRepo, aiShadowRepo, producer, aiMetrics, svc.Logger)
	aiDriftSvc := aigovservice.NewDriftService(aiRegistryRepo, aiPredictionRepo, aiDriftRepo, producer, aiMetrics, svc.Logger)
	aiValidationSvc := aigovservice.NewValidationService(aiRegistryRepo, aiPredictionRepo, aiValidationRepo, producer, aiMetrics, nil, svc.Logger)
	aiDashboardSvc := aigovservice.NewDashboardService(aiRegistryRepo, aiPredictionRepo, aiDriftRepo, svc.Logger)
	aiServices := aigovhandler.Services{
		Registry:     aiRegistrySvc,
		Predictions:  aiPredictionSvc,
		Explanations: aiExplanationSvc,
		Shadow:       aiShadowSvc,
		Lifecycle:    aiLifecycleSvc,
		Drift:        aiDriftSvc,
		Validation:   aiValidationSvc,
		Dashboard:    aiDashboardSvc,
	}
	go func() {
		_ = aigovshadow.NewScheduler(aiComparisonSvc, time.Hour, svc.Logger).Run(ctx)
	}()
	go func() {
		_ = aigovdrift.NewScheduler(aiDriftSvc, 24*time.Hour, svc.Logger).Run(ctx)
	}()

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
	var brandingUploader onboardingsvc.BrandingAssetUploader
	if storageClient != nil {
		fileRepository := filerepo.NewFileRepository(svc.DBPool, svc.Logger)
		fileService := fileservice.NewFileService(
			fileRepository,
			storageClient,
			nil,
			producer,
			filemetrics.NewFileMetrics(svc.Metrics.Registry()),
			svc.Logger,
			"clario360",
			"clario360-quarantine",
			15*time.Minute,
		)
		brandingUploader = onboardingsvc.NewBrandingAssetUploader(fileService)
	}

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
		provisioningRepository,
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
		brandingUploader,
		provisioningRepository,
		svc.Logger,
	)

	// Security headers on all responses.
	svc.Router.Use(middleware.SecurityHeaders())
	svc.Router.Get("/.well-known/openid-configuration", oauthHandler.Discovery)
	svc.Router.Get("/.well-known/jwks.json", oauthHandler.JWKS)

	// Routes.
	svc.Router.Route("/api/v1", func(r chi.Router) {
		r.Get("/internal/users/by-role", roleHandler.InternalUserIDsByRole)
		r.Get("/internal/users/by-email", userHandler.InternalGetByEmail)
		r.Get("/internal/users/{id}/email", userHandler.InternalGetEmail)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(svc.Redis, middleware.RateLimitConfig{
				RequestsPerWindow: 20,
				Window:            1 * time.Minute,
				KeyPrefix:         "ratelimit:auth",
			}))
			r.Mount("/auth", authHandler.Routes())
			r.Mount("/auth/oauth", oauthHandler.Routes())
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
			r.Mount("/notebooks", notebookHandler.Routes())
			aigovhandler.RegisterRoutes(r, aiServices, svc.Logger)

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

	if producer != nil {
		notebookConsumerCfg := legacyCfg.Kafka
		notebookConsumerCfg.GroupID = "iam-service-notebook-consumer"
		kafkaConsumer, err := events.NewConsumer(notebookConsumerCfg, svc.Logger)
		if err != nil {
			svc.Logger.Warn().Err(err).Msg("notebook audit consumer unavailable")
		} else {
			kafkaConsumer.Subscribe(events.Topics.NotebookEvents, notebookconsumer.NewNotebookConsumer(producer, svc.Logger))
			go func() {
				if err := kafkaConsumer.Start(ctx); err != nil && err != context.Canceled {
					svc.Logger.Error().Err(err).Msg("notebook audit consumer exited")
				}
			}()
			defer kafkaConsumer.Close()
		}
	}

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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func loadAdditionalOAuthClientsFromEnv(logger zerolog.Logger) []iamservice.OAuthClient {
	raw := strings.TrimSpace(os.Getenv("OAUTH_ADDITIONAL_CLIENTS_JSON"))
	if raw == "" {
		return nil
	}

	var clients []iamservice.OAuthClient
	if err := json.Unmarshal([]byte(raw), &clients); err != nil {
		logger.Fatal().Err(err).Msg("failed to parse OAUTH_ADDITIONAL_CLIENTS_JSON")
	}
	for i := range clients {
		if clients[i].ClientID == "" || len(clients[i].RedirectURIs) == 0 {
			logger.Fatal().Int("index", i).Msg("oauth additional clients must define client_id and redirect_uris")
		}
		if len(clients[i].Scopes) == 0 {
			clients[i].Scopes = []string{"openid", "profile", "email", "roles"}
		}
		clients[i].RequirePKCE = true
	}
	return clients
}

func buildOnboardingDBPools(ctx context.Context, cfg *config.Config, logger zerolog.Logger) (map[string]*pgxpool.Pool, map[string]string, error) {
	dsns := map[string]string{
		"platform_core": envOrDefault("PLATFORM_DB_URL", buildPostgresURL(cfg.Database, "platform_core")),
		"cyber_db":      envOrDefault("CYBER_DB_URL", buildPostgresURL(cfg.Database, "cyber_db")),
		"data_db":       envOrDefault("DATA_DB_URL", buildPostgresURL(cfg.Database, "data_db")),
		"acta_db":       envOrDefault("ACTA_DB_URL", buildPostgresURL(cfg.Database, "acta_db")),
		"lex_db":        envOrDefault("LEX_DB_URL", buildPostgresURL(cfg.Database, "lex_db")),
		"visus_db":      envOrDefault("VISUS_DB_URL", buildPostgresURL(cfg.Database, "visus_db")),
	}

	pools := make(map[string]*pgxpool.Pool, len(dsns)-1)
	for name, dsn := range dsns {
		if name == "platform_core" {
			continue
		}

		pool, err := newPGXPool(ctx, dsn, cfg.Database.MaxIdleConns, cfg.Database.MaxOpenConns)
		if err != nil {
			return nil, nil, fmt.Errorf("connect %s: %w", name, err)
		}
		pools[name] = pool
		logger.Info().Str("database", name).Msg("onboarding database pool established")
	}

	return pools, dsns, nil
}

func newPGXPool(ctx context.Context, dsn string, minConns, maxConns int) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	if minConns < 1 {
		minConns = 1
	}
	if maxConns < minConns {
		maxConns = minConns
	}

	poolCfg.MinConns = int32(minConns)
	poolCfg.MaxConns = int32(maxConns)
	poolCfg.MaxConnLifetime = 5 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres pool: %w", err)
	}
	return pool, nil
}

func buildOnboardingStorage(ctx context.Context, cfg *config.Config, logger zerolog.Logger) *storage.MinIOStorage {
	storageClient, err := storage.NewMinIOStorage(storage.Config{
		Backend:      "minio",
		Endpoint:     cfg.MinIO.Endpoint,
		AccessKey:    cfg.MinIO.AccessKey,
		SecretKey:    cfg.MinIO.SecretKey,
		UseSSL:       cfg.MinIO.UseSSL,
		BucketPrefix: "clario360",
	})
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize onboarding storage client")
		return nil
	}

	if _, err := storageClient.Client().ListBuckets(ctx); err != nil {
		logger.Warn().Err(err).Msg("minio connectivity check failed for onboarding")
	}

	return storageClient
}

func buildOnboardingEmailSender(logger zerolog.Logger) onboardingsvc.EmailSender {
	notifCfg := notifcfg.LoadFromEnv()
	templateService := notifservice.NewTemplateService(logger)
	emailChannel := notifchannel.NewEmailChannel(notifchannel.EmailConfig{
		Provider:       notifCfg.EmailProvider,
		SMTPHost:       notifCfg.SMTPHost,
		SMTPPort:       notifCfg.SMTPPort,
		SMTPUser:       notifCfg.SMTPUsername,
		SMTPPass:       notifCfg.SMTPPassword,
		SMTPFrom:       notifCfg.SMTPFrom,
		TLSEnabled:     notifCfg.SMTPTLSEnabled,
		SendGridAPIKey: notifCfg.SendGridAPIKey,
		SendGridFrom:   notifCfg.SendGridFrom,
	}, templateService, logger)

	return onboardingsvc.NewChannelEmailSender(
		envOrDefault("CLARIO360_APP_URL", "http://localhost:3000"),
		emailChannel,
		logger,
	)
}

func resolveMigrationsBasePath() string {
	candidates := []string{
		envOrDefault("ONBOARDING_MIGRATIONS_BASE_PATH", ""),
		"migrations",
		filepath.Join("backend", "migrations"),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(candidate, "platform_core"))
		if err == nil && info.IsDir() {
			return candidate
		}
	}

	return "migrations"
}

func buildPostgresURL(cfg config.DatabaseConfig, dbName string) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   dbName,
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
