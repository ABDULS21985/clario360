//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	tc "github.com/testcontainers/testcontainers-go"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"
	redpandamod "github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	filemetrics "github.com/clario360/platform/internal/filemanager/metrics"
	filerepo "github.com/clario360/platform/internal/filemanager/repository"
	fileservice "github.com/clario360/platform/internal/filemanager/service"
	iamhandler "github.com/clario360/platform/internal/iam/handler"
	iamrepo "github.com/clario360/platform/internal/iam/repository"
	iamservice "github.com/clario360/platform/internal/iam/service"
	"github.com/clario360/platform/internal/middleware"
	obsmetrics "github.com/clario360/platform/internal/observability/metrics"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardinghandler "github.com/clario360/platform/internal/onboarding/handler"
	onboardingmiddleware "github.com/clario360/platform/internal/onboarding/middleware"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
	onboardingsvc "github.com/clario360/platform/internal/onboarding/service"
	"github.com/clario360/platform/pkg/storage"
)

var (
	sharedEnvOnce sync.Once
	sharedEnv     *onboardingIntegrationEnv
	sharedEnvErr  error

	testSequence uint64
)

type onboardingIntegrationEnv struct {
	logger zerolog.Logger

	postgres *postgresmod.PostgresContainer
	redpanda *redpandamod.Container
	redis    tc.Container
	minio    tc.Container

	platformPool *pgxpool.Pool
	dbPools      map[string]*pgxpool.Pool
	dbDSNs       map[string]string
	rdb          *redis.Client
	jwtMgr       *auth.JWTManager
	producer     *events.Producer
	emailSender  *recordingEmailSender
	storage      *storage.MinIOStorage

	onboardingRepo   *onboardingrepo.OnboardingRepository
	provisioningRepo *onboardingrepo.ProvisioningRepository

	server *httptest.Server
}

type onboardingHarness struct {
	env      *onboardingIntegrationEnv
	client   *http.Client
	sequence uint64
	ip       string
}

type tenantFixture struct {
	TenantID         uuid.UUID
	OrganizationName string
	AdminEmail       string
	AdminFirstName   string
	AdminLastName    string
	Password         string
	OTP              string
	AccessToken      string
	RefreshToken     string
}

type multipartFileInput struct {
	FieldName   string
	FileName    string
	ContentType string
	Content     []byte
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

type recordingEmailSender struct {
	mu                sync.RWMutex
	verificationCodes map[string]string
	invitationTokens  map[string]string
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		if err := sharedEnv.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "onboarding integration cleanup failed: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

func newHarness(t *testing.T) *onboardingHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	seq := atomic.AddUint64(&testSequence, 1)
	env := mustSharedEnv(t)
	return &onboardingHarness{
		env:      env,
		client:   env.server.Client(),
		sequence: seq,
		ip:       fmt.Sprintf("203.0.113.%d", (seq%250)+1),
	}
}

func mustSharedEnv(t *testing.T) *onboardingIntegrationEnv {
	t.Helper()
	sharedEnvOnce.Do(func() {
		sharedEnv, sharedEnvErr = startSharedEnv()
	})
	if sharedEnvErr != nil {
		t.Fatalf("start onboarding integration environment: %v", sharedEnvErr)
	}
	return sharedEnv
}

func startSharedEnv() (*onboardingIntegrationEnv, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	postgresContainer, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("platform_core"),
		postgresmod.WithUsername("clario"),
		postgresmod.WithPassword("clario"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	redisContainer, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			Cmd:          []string{"redis-server", "--save", "", "--appendonly", "no", "--bind", "0.0.0.0"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("start redis container: %w", err)
	}

	redpandaContainer, err := redpandamod.Run(ctx, "docker.redpanda.com/redpandadata/redpanda:v24.1.8", redpandamod.WithAutoCreateTopics())
	if err != nil {
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("start redpanda container: %w", err)
	}

	minioContainer, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "minio/minio:latest",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForListeningPort("9000/tcp"),
		},
		Started: true,
	})
	if err != nil {
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("start minio container: %w", err)
	}

	adminDSN := postgresContainer.MustConnectionString(ctx, "sslmode=disable")
	for _, databaseName := range []string{"cyber_db", "data_db", "acta_db", "lex_db", "visus_db"} {
		if err := createDatabase(ctx, adminDSN, databaseName); err != nil {
			_ = minioContainer.Terminate(context.Background())
			_ = redpandaContainer.Terminate(context.Background())
			_ = redisContainer.Terminate(context.Background())
			_ = postgresContainer.Terminate(context.Background())
			return nil, fmt.Errorf("create database %s: %w", databaseName, err)
		}
	}

	postgresHost, err := postgresContainer.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres host: %w", err)
	}
	postgresPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, fmt.Errorf("postgres port: %w", err)
	}

	dbDSNs := map[string]string{
		"platform_core": fmt.Sprintf("postgres://clario:clario@%s:%s/platform_core?sslmode=disable", postgresHost, postgresPort.Port()),
		"cyber_db":      fmt.Sprintf("postgres://clario:clario@%s:%s/cyber_db?sslmode=disable", postgresHost, postgresPort.Port()),
		"data_db":       fmt.Sprintf("postgres://clario:clario@%s:%s/data_db?sslmode=disable", postgresHost, postgresPort.Port()),
		"acta_db":       fmt.Sprintf("postgres://clario:clario@%s:%s/acta_db?sslmode=disable", postgresHost, postgresPort.Port()),
		"lex_db":        fmt.Sprintf("postgres://clario:clario@%s:%s/lex_db?sslmode=disable", postgresHost, postgresPort.Port()),
		"visus_db":      fmt.Sprintf("postgres://clario:clario@%s:%s/visus_db?sslmode=disable", postgresHost, postgresPort.Port()),
	}

	for dbName, dsn := range dbDSNs {
		if err := database.RunMigrations(dsn, filepath.Join(onboardingMigrationsBasePath(), dbName)); err != nil {
			return nil, fmt.Errorf("run migrations for %s: %w", dbName, err)
		}
	}

	platformPool, err := pgxpool.New(ctx, dbDSNs["platform_core"])
	if err != nil {
		return nil, fmt.Errorf("open platform pool: %w", err)
	}
	if err := platformPool.Ping(ctx); err != nil {
		platformPool.Close()
		return nil, fmt.Errorf("ping platform pool: %w", err)
	}

	dbPools := make(map[string]*pgxpool.Pool, 5)
	for _, dbName := range []string{"cyber_db", "data_db", "acta_db", "lex_db", "visus_db"} {
		pool, openErr := pgxpool.New(ctx, dbDSNs[dbName])
		if openErr != nil {
			platformPool.Close()
			return nil, fmt.Errorf("open %s pool: %w", dbName, openErr)
		}
		if pingErr := pool.Ping(ctx); pingErr != nil {
			pool.Close()
			platformPool.Close()
			return nil, fmt.Errorf("ping %s pool: %w", dbName, pingErr)
		}
		dbPools[dbName] = pool
	}

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("redis host: %w", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return nil, fmt.Errorf("redis port: %w", err)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	var producer *events.Producer

	minioHost, err := minioContainer.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("minio host: %w", err)
	}
	minioPort, err := minioContainer.MappedPort(ctx, "9000/tcp")
	if err != nil {
		return nil, fmt.Errorf("minio port: %w", err)
	}
	storageClient, err := storage.NewMinIOStorage(storage.Config{
		Backend:      "minio",
		Endpoint:     fmt.Sprintf("%s:%s", minioHost, minioPort.Port()),
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		BucketPrefix: "clario360",
	})
	if err != nil {
		return nil, fmt.Errorf("create onboarding storage client: %w", err)
	}
	if _, err := storageClient.Client().ListBuckets(ctx); err != nil {
		return nil, fmt.Errorf("list minio buckets: %w", err)
	}

	jwtMgr, err := auth.NewJWTManager(appconfig.AuthConfig{
		JWTIssuer:       "clario360-integration",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
		BcryptCost:      4,
	})
	if err != nil {
		return nil, fmt.Errorf("create jwt manager: %w", err)
	}

	emailSender := newRecordingEmailSender()
	metrics := obsmetrics.NewMetrics("onboarding_it")
	onboardingMetrics := onboardingsvc.NewMetrics(metrics)

	userRepo := iamrepo.NewUserRepository(platformPool)
	roleRepo := iamrepo.NewRoleRepository(platformPool)
	sessionRepo := iamrepo.NewSessionRepository(platformPool)
	tenantRepo := iamrepo.NewTenantRepository(platformPool)

	authSvc := iamservice.NewAuthService(
		userRepo,
		sessionRepo,
		roleRepo,
		tenantRepo,
		jwtMgr,
		rdb,
		producer,
		logger,
		4,
		24*time.Hour,
	)
	authHandler := iamhandler.NewAuthHandler(authSvc, logger)

	onboardingRepository := onboardingrepo.NewOnboardingRepository(platformPool)
	invitationRepository := onboardingrepo.NewInvitationRepository(platformPool)
	provisioningRepository := onboardingrepo.NewProvisioningRepository(platformPool)

	fileRepository := filerepo.NewFileRepository(platformPool, logger)
	fileService := fileservice.NewFileService(
		fileRepository,
		storageClient,
		nil,
		producer,
		filemetrics.NewFileMetrics(metrics.Registry()),
		logger,
		"clario360",
		"clario360-quarantine",
		15*time.Minute,
	)
	brandingUploader := onboardingsvc.NewBrandingAssetUploader(fileService)

	provisioner := onboardingsvc.NewTenantProvisioner(
		platformPool,
		dbPools,
		dbDSNs,
		onboardingMigrationsBasePath(),
		onboardingRepository,
		provisioningRepository,
		storageClient,
		emailSender,
		producer,
		logger,
		onboardingMetrics,
	)
	registrationService := onboardingsvc.NewRegistrationService(
		onboardingRepository,
		userRepo,
		roleRepo,
		sessionRepo,
		jwtMgr,
		rdb,
		producer,
		emailSender,
		provisioner,
		logger,
		onboardingMetrics,
		4,
		24*time.Hour,
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
		logger,
		onboardingMetrics,
		4,
		24*time.Hour,
	)
	wizardService := onboardingsvc.NewWizardService(
		onboardingRepository,
		invitationService,
		producer,
		logger,
		onboardingMetrics,
	)
	deprovisioner := onboardingsvc.NewTenantDeprovisioner(
		platformPool,
		dbPools,
		onboardingRepository,
		storageClient,
		rdb,
		producer,
		logger,
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
		logger,
	)

	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", authHandler.Routes())

		r.Route("/onboarding", func(r chi.Router) {
			r.With(onboardingmiddleware.NewPublicRateLimiter(rdb, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 5,
				Window:            time.Hour,
				KeyPrefix:         "ratelimit:onboarding:register",
			})).Post("/register", onboardingHandler.Register)
			r.With(onboardingmiddleware.NewPublicRateLimiter(rdb, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 20,
				Window:            10 * time.Minute,
				KeyPrefix:         "ratelimit:onboarding:verify-email",
			})).Post("/verify-email", onboardingHandler.VerifyEmail)
			r.With(onboardingmiddleware.NewPublicRateLimiter(rdb, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 1,
				Window:            time.Minute,
				KeyPrefix:         "ratelimit:onboarding:resend-otp",
			})).Post("/resend-otp", onboardingHandler.ResendOTP)
			r.With(
				middleware.OptionalAuth(jwtMgr),
				onboardingmiddleware.NewPublicRateLimiter(rdb, onboardingmiddleware.PublicRateLimitConfig{
					RequestsPerWindow: 120,
					Window:            time.Minute,
					KeyPrefix:         "ratelimit:onboarding:status",
				}),
			).Get("/status/{tenantId}", onboardingHandler.GetOnboardingStatus)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(jwtMgr))
				r.Use(middleware.RateLimit(rdb, middleware.DefaultRateLimitConfig()))
				r.Use(middleware.Tenant)

				r.Get("/wizard", onboardingHandler.GetWizardProgress)
				r.Post("/wizard/organization", onboardingHandler.SaveOrganization)
				r.Post("/wizard/branding", onboardingHandler.SaveBranding)
				r.Post("/wizard/team", onboardingHandler.SaveTeam)
				r.Post("/wizard/suites", onboardingHandler.SaveSuites)
				r.Post("/wizard/complete", onboardingHandler.CompleteWizard)
			})
		})

		r.Route("/invitations", func(r chi.Router) {
			r.With(onboardingmiddleware.NewPublicRateLimiter(rdb, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 60,
				Window:            time.Minute,
				KeyPrefix:         "ratelimit:onboarding:invite-validate",
			})).Get("/validate", onboardingHandler.ValidateInviteToken)
			r.With(onboardingmiddleware.NewPublicRateLimiter(rdb, onboardingmiddleware.PublicRateLimitConfig{
				RequestsPerWindow: 10,
				Window:            15 * time.Minute,
				KeyPrefix:         "ratelimit:onboarding:invite-accept",
			})).Post("/accept", onboardingHandler.AcceptInvitation)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(jwtMgr))
				r.Use(middleware.RateLimit(rdb, middleware.DefaultRateLimitConfig()))
				r.Use(middleware.Tenant)

				r.Get("/", onboardingHandler.ListInvitations)
				r.Post("/", onboardingHandler.CreateBatchInvitations)
				r.Delete("/{id}", onboardingHandler.CancelInvitation)
				r.Post("/resend/{id}", onboardingHandler.ResendInvitation)
			})
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.Auth(jwtMgr))
			r.Use(middleware.RateLimit(rdb, middleware.DefaultRateLimitConfig()))

			r.Post("/tenants/provision", onboardingHandler.AdminProvision)
			r.Get("/tenants/{id}/provision-status", onboardingHandler.AdminGetProvisionStatus)
			r.Post("/tenants/{id}/deprovision", onboardingHandler.AdminDeprovision)
			r.Post("/tenants/{id}/reprovision", onboardingHandler.AdminReprovision)
			r.Post("/tenants/{id}/reactivate", onboardingHandler.AdminReactivate)
		})
	})

	server := httptest.NewServer(router)

	return &onboardingIntegrationEnv{
		logger:           logger,
		postgres:         postgresContainer,
		redpanda:         redpandaContainer,
		redis:            redisContainer,
		minio:            minioContainer,
		platformPool:     platformPool,
		dbPools:          dbPools,
		dbDSNs:           dbDSNs,
		rdb:              rdb,
		jwtMgr:           jwtMgr,
		producer:         producer,
		emailSender:      emailSender,
		storage:          storageClient,
		onboardingRepo:   onboardingRepository,
		provisioningRepo: provisioningRepository,
		server:           server,
	}, nil
}

func (e *onboardingIntegrationEnv) Close() error {
	var errs []string

	if e.server != nil {
		e.server.Close()
	}
	if e.producer != nil {
		if err := e.producer.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.rdb != nil {
		if err := e.rdb.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	for _, pool := range e.dbPools {
		if pool != nil {
			pool.Close()
		}
	}
	if e.platformPool != nil {
		e.platformPool.Close()
	}
	if e.minio != nil {
		if err := e.minio.Terminate(context.Background()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.redpanda != nil {
		if err := e.redpanda.Terminate(context.Background()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.redis != nil {
		if err := e.redis.Terminate(context.Background()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.postgres != nil {
		if err := e.postgres.Terminate(context.Background()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func newRecordingEmailSender() *recordingEmailSender {
	return &recordingEmailSender{
		verificationCodes: make(map[string]string),
		invitationTokens:  make(map[string]string),
	}
}

func (s *recordingEmailSender) SendVerificationEmail(ctx context.Context, email, orgName, adminName, otp string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verificationCodes[normalizeEmail(email)] = otp
	return nil
}

func (s *recordingEmailSender) SendInvitationEmail(ctx context.Context, email, organizationName, inviterName, roleName, rawToken string, message *string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invitationTokens[normalizeEmail(email)] = rawToken
	return nil
}

func (s *recordingEmailSender) SendWelcomeEmail(ctx context.Context, email, organizationName, firstName string) error {
	return nil
}

func (s *recordingEmailSender) verificationCode(email string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.verificationCodes[normalizeEmail(email)]
	return code, ok
}

func (s *recordingEmailSender) invitationToken(email string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.invitationTokens[normalizeEmail(email)]
	return token, ok
}

func (h *onboardingHarness) registerTenant(t *testing.T) tenantFixture {
	t.Helper()

	seq := atomic.AddUint64(&testSequence, 1)
	fixture := tenantFixture{
		OrganizationName: fmt.Sprintf("Onboarding Integration %d", seq),
		AdminEmail:       fmt.Sprintf("admin-%d@example.com", seq),
		AdminFirstName:   "Amina",
		AdminLastName:    "Tester",
		Password:         fmt.Sprintf("Clario360!Admin%d", seq),
	}

	body := h.postJSON(t, "/api/v1/onboarding/register", onboardingdto.RegisterRequest{
		OrganizationName: fixture.OrganizationName,
		AdminEmail:       fixture.AdminEmail,
		AdminFirstName:   fixture.AdminFirstName,
		AdminLastName:    fixture.AdminLastName,
		AdminPassword:    fixture.Password,
		Country:          "SA",
		Industry:         "financial",
	}, "", http.StatusCreated)

	var response onboardingdto.RegisterResponse
	mustDecode(t, body, &response)
	fixture.TenantID = uuid.MustParse(response.TenantID)

	otp, ok := h.env.emailSender.verificationCode(fixture.AdminEmail)
	if !ok {
		t.Fatalf("verification OTP not recorded for %s", fixture.AdminEmail)
	}
	fixture.OTP = otp
	return fixture
}

func (h *onboardingHarness) verifyTenant(t *testing.T, fixture tenantFixture, otp string, expectedStatus int) []byte {
	t.Helper()
	return h.postJSON(t, "/api/v1/onboarding/verify-email", onboardingdto.VerifyEmailRequest{
		Email: fixture.AdminEmail,
		OTP:   otp,
	}, "", expectedStatus)
}

func (h *onboardingHarness) registerAndVerifyTenant(t *testing.T) tenantFixture {
	t.Helper()
	fixture := h.registerTenant(t)
	body := h.verifyTenant(t, fixture, fixture.OTP, http.StatusOK)

	var response onboardingdto.VerifyEmailResponse
	mustDecode(t, body, &response)
	fixture.AccessToken = response.AccessToken
	fixture.RefreshToken = response.RefreshToken
	return fixture
}

func (h *onboardingHarness) waitForProvisioning(t *testing.T, tenantID uuid.UUID) *onboardingmodel.ProvisioningStatus {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		status, err := h.env.provisioningRepo.GetStatus(context.Background(), tenantID)
		if err == nil {
			if status.Status == onboardingmodel.OnboardingProvisioningCompleted {
				return status
			}
			if status.Status == onboardingmodel.OnboardingProvisioningFailed {
				errMsg := ""
				if status.Error != nil {
					errMsg = *status.Error
				}
				stepStates := make([]string, 0, len(status.Steps))
				for _, step := range status.Steps {
					detail := ""
					if step.ErrorMessage != nil {
						detail = ": " + *step.ErrorMessage
					}
					stepStates = append(stepStates, fmt.Sprintf("%d=%s%s", step.StepNumber, step.Status, detail))
				}
				t.Fatalf("provisioning failed for %s: %s | steps=%s", tenantID, errMsg, strings.Join(stepStates, ", "))
			}
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("wait for provisioning: %v", lastErr)
	}
	t.Fatalf("provisioning did not complete before timeout for %s", tenantID)
	return nil
}

func (h *onboardingHarness) newContext() context.Context {
	return context.Background()
}

func (h *onboardingHarness) superAdminToken(t *testing.T) string {
	t.Helper()
	pair, err := h.env.jwtMgr.GenerateTokenPair(uuid.NewString(), uuid.Nil.String(), "super-admin@example.com", []string{"super-admin"})
	if err != nil {
		t.Fatalf("generate super admin token: %v", err)
	}
	return pair.AccessToken
}

func (h *onboardingHarness) postJSON(t *testing.T, path string, payload any, token string, expectedStatus int) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request for %s: %v", path, err)
	}
	req := h.newRequest(t, http.MethodPost, path, bytes.NewReader(body), "application/json", token)
	return h.do(t, req, expectedStatus)
}

func (h *onboardingHarness) get(t *testing.T, path, token string, expectedStatus int) []byte {
	t.Helper()
	req := h.newRequest(t, http.MethodGet, path, nil, "", token)
	return h.do(t, req, expectedStatus)
}

func (h *onboardingHarness) delete(t *testing.T, path, token string, expectedStatus int) []byte {
	t.Helper()
	req := h.newRequest(t, http.MethodDelete, path, nil, "", token)
	return h.do(t, req, expectedStatus)
}

func (h *onboardingHarness) postMultipart(t *testing.T, path string, fields map[string]string, files []multipartFileInput, token string, expectedStatus int) []byte {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write multipart field %s: %v", key, err)
		}
	}
	for _, file := range files {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, file.FieldName, file.FileName))
		if file.ContentType != "" {
			header.Set("Content-Type", file.ContentType)
		}
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatalf("create multipart part %s: %v", file.FieldName, err)
		}
		if _, err := part.Write(file.Content); err != nil {
			t.Fatalf("write multipart content %s: %v", file.FieldName, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := h.newRequest(t, http.MethodPost, path, &body, writer.FormDataContentType(), token)
	return h.do(t, req, expectedStatus)
}

func (h *onboardingHarness) newRequest(t *testing.T, method, path string, body io.Reader, contentType, token string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, h.env.server.URL+path, body)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Forwarded-For", h.ip)
	return req
}

func (h *onboardingHarness) do(t *testing.T, req *http.Request, expectedStatus int) []byte {
	t.Helper()

	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", req.Method, req.URL.Path, err)
	}

	if resp.StatusCode != expectedStatus {
		t.Fatalf("%s %s returned %d, expected %d: %s", req.Method, req.URL.Path, resp.StatusCode, expectedStatus, string(body))
	}
	return body
}

func (h *onboardingHarness) countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows failed: %v", err)
	}
	return count
}

func (h *onboardingHarness) tenantSlug(t *testing.T, tenantID uuid.UUID) string {
	t.Helper()
	var slug string
	if err := h.env.platformPool.QueryRow(context.Background(), `SELECT slug FROM tenants WHERE id = $1`, tenantID).Scan(&slug); err != nil {
		t.Fatalf("load tenant slug: %v", err)
	}
	return slug
}

func (h *onboardingHarness) loadLogoStorageRecord(t *testing.T, fileID string) (bucket, storageKey, detectedType string) {
	t.Helper()
	if err := h.env.platformPool.QueryRow(context.Background(), `
		SELECT bucket, storage_key, detected_content_type
		FROM files
		WHERE id = $1`,
		fileID,
	).Scan(&bucket, &storageKey, &detectedType); err != nil {
		t.Fatalf("load logo storage record: %v", err)
	}
	return bucket, storageKey, detectedType
}

func (h *onboardingHarness) bucketExists(t *testing.T, bucket string) bool {
	t.Helper()
	exists, err := h.env.storage.Client().BucketExists(context.Background(), bucket)
	if err != nil {
		t.Fatalf("check bucket %s: %v", bucket, err)
	}
	return exists
}

func (h *onboardingHarness) bucketTags(t *testing.T, bucket string) map[string]string {
	t.Helper()
	tagSet, err := h.env.storage.Client().GetBucketTagging(context.Background(), bucket)
	if err != nil {
		t.Fatalf("get bucket tags for %s: %v", bucket, err)
	}
	return tagSet.ToMap()
}

func mustDecode(t *testing.T, body []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(body, dst); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(body))
	}
}

func createDatabase(ctx context.Context, dsn, dbName string) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	return err
}

func onboardingMigrationsBasePath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations")
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)
}
