//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
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
	lexapp "github.com/clario360/platform/internal/lex"
	lexconfig "github.com/clario360/platform/internal/lex/config"
	"github.com/clario360/platform/internal/lex/model"
	lexmonitor "github.com/clario360/platform/internal/lex/monitor"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

var (
	sharedEnvOnce sync.Once
	sharedEnv     *lexIntegrationEnv
	sharedEnvErr  error

	demoSeedOnce sync.Once
	demoSeedErr  error
)

type lexIntegrationEnv struct {
	logger zerolog.Logger

	postgres *postgresmod.PostgresContainer
	redpanda *redpandamod.Container
	redis    tc.Container

	db       *pgxpool.Pool
	rdb      *redis.Client
	jwt      *auth.JWTManager
	app      *lexapp.Application
	producer *events.Producer
	brokers  []string
	server   *httptest.Server
}

type lexHarness struct {
	env      *lexIntegrationEnv
	client   *http.Client
	tenantID uuid.UUID
	userID   uuid.UUID
	token    string
}

type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

type paginatedEnvelope[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		Page       int `json:"page"`
		PerPage    int `json:"per_page"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		if err := sharedEnv.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "lex integration cleanup failed: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

func newLexHarness(t *testing.T) *lexHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	env := mustSharedEnv(t)
	tenantID := uuid.New()
	userID := uuid.New()
	h := &lexHarness{
		env:      env,
		client:   env.server.Client(),
		tenantID: tenantID,
		userID:   userID,
		token:    env.mustToken(t, tenantID, userID, "tenant_admin"),
	}
	t.Cleanup(func() {
		h.cleanupTenant(t)
	})
	return h
}

func newDemoHarness(t *testing.T) *lexHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	env := mustSharedEnv(t)
	demoSeedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_, demoSeedErr = lexapp.SeedDemoData(ctx, env.app, env.logger)
	})
	if demoSeedErr != nil {
		t.Fatalf("seed demo dataset: %v", demoSeedErr)
	}

	tenantID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222201")
	return &lexHarness{
		env:      env,
		client:   env.server.Client(),
		tenantID: tenantID,
		userID:   userID,
		token:    env.mustToken(t, tenantID, userID, "tenant_admin"),
	}
}

func mustSharedEnv(t *testing.T) *lexIntegrationEnv {
	t.Helper()
	sharedEnvOnce.Do(func() {
		sharedEnv, sharedEnvErr = startSharedEnv()
	})
	if sharedEnvErr != nil {
		t.Fatalf("start lex integration environment: %v", sharedEnvErr)
	}
	return sharedEnv
}

func startSharedEnv() (*lexIntegrationEnv, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger := zerolog.New(io.Discard)

	postgresContainer, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("lex_service"),
		postgresmod.WithUsername("lex"),
		postgresmod.WithPassword("lex"),
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

	dbURL := postgresContainer.MustConnectionString(ctx, "sslmode=disable")
	if err := database.RunMigrations(dbURL, lexMigrationsPath()); err != nil {
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("run lex migrations: %w", err)
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("open lex postgres pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("ping lex postgres: %w", err)
	}
	if err := workflowrepo.RunMigration(ctx, db); err != nil {
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("run workflow migration: %w", err)
	}

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("redis host: %w", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("redis port: %w", err)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	seedBroker, err := redpandaContainer.KafkaSeedBroker(ctx)
	if err != nil {
		_ = rdb.Close()
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("resolve redpanda broker: %w", err)
	}
	brokers := []string{seedBroker}

	producer, err := events.NewProducer(appconfig.KafkaConfig{
		Brokers: brokers,
		GroupID: "lex-integration-producer",
	}, logger)
	if err != nil {
		_ = rdb.Close()
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}

	jwtMgr, err := auth.NewJWTManager(appconfig.AuthConfig{
		JWTIssuer:       "lex-integration",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		_ = producer.Close()
		_ = rdb.Close()
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("create jwt manager: %w", err)
	}

	lexCfg := lexconfig.Default()
	lexCfg.KafkaBrokers = brokers
	app, err := lexapp.NewApplication(lexapp.Dependencies{
		DB:                db,
		Redis:             rdb,
		Publisher:         producer,
		Logger:            logger,
		Registerer:        prometheus.NewRegistry(),
		WorkflowDefRepo:   workflowrepo.NewDefinitionRepository(db),
		WorkflowInstRepo:  workflowrepo.NewInstanceRepository(db),
		WorkflowTaskRepo:  workflowrepo.NewTaskRepository(db),
		Config:            lexCfg,
		DashboardCacheTTL: 60 * time.Second,
		OrgJurisdiction:   "Saudi Arabia",
		KafkaTopic:        events.Topics.LexEvents,
	})
	if err != nil {
		_ = producer.Close()
		_ = rdb.Close()
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("create lex app: %w", err)
	}

	router := chi.NewRouter()
	app.RegisterRoutes(router, jwtMgr, rdb, 10000)
	server := httptest.NewServer(router)

	return &lexIntegrationEnv{
		logger:   logger,
		postgres: postgresContainer,
		redpanda: redpandaContainer,
		redis:    redisContainer,
		db:       db,
		rdb:      rdb,
		jwt:      jwtMgr,
		app:      app,
		producer: producer,
		brokers:  brokers,
		server:   server,
	}, nil
}

func (e *lexIntegrationEnv) Close() error {
	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var errs []error
	if e.server != nil {
		e.server.Close()
	}
	if e.producer != nil {
		if err := e.producer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if e.rdb != nil {
		if err := e.rdb.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if e.db != nil {
		e.db.Close()
	}
	if e.redpanda != nil {
		if err := e.redpanda.Terminate(closeCtx); err != nil {
			errs = append(errs, err)
		}
	}
	if e.redis != nil {
		if err := e.redis.Terminate(closeCtx); err != nil {
			errs = append(errs, err)
		}
	}
	if e.postgres != nil {
		if err := e.postgres.Terminate(closeCtx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("cleanup errors: %v", errs)
}

func (e *lexIntegrationEnv) mustToken(t *testing.T, tenantID, userID uuid.UUID, roles ...string) string {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"tenant_admin"}
	}
	tokenPair, err := e.jwt.GenerateTokenPair(userID.String(), tenantID.String(), fmt.Sprintf("%s@lex.integration", userID.String()), roles)
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}
	return tokenPair.AccessToken
}

func (h *lexHarness) doJSON(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body %T: %v", body, err)
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, h.env.server.URL+path, payload)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("execute request %s %s: %v", method, path, err)
	}
	return resp
}

func (h *lexHarness) cleanupTenant(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statements := []string{
		`DELETE FROM workflow_tasks WHERE tenant_id = $1`,
		`DELETE FROM workflow_timers WHERE instance_id IN (SELECT id FROM workflow_instances WHERE tenant_id = $1)`,
		`DELETE FROM workflow_step_executions WHERE instance_id IN (SELECT id FROM workflow_instances WHERE tenant_id = $1)`,
		`DELETE FROM workflow_instances WHERE tenant_id = $1`,
		`DELETE FROM workflow_definitions WHERE tenant_id = $1`,
		`DELETE FROM expiry_notifications WHERE tenant_id = $1`,
		`DELETE FROM compliance_alerts WHERE tenant_id = $1`,
		`DELETE FROM compliance_rules WHERE tenant_id = $1`,
		`DELETE FROM legal_documents WHERE tenant_id = $1`,
		`DELETE FROM contracts WHERE tenant_id = $1`,
	}
	for _, stmt := range statements {
		if _, err := h.env.db.Exec(ctx, stmt, h.tenantID); err != nil {
			t.Fatalf("cleanup tenant %s: %v", h.tenantID, err)
		}
	}
}

func mustData[T any](t *testing.T, resp *http.Response, wantStatus int) T {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("response status = %d, want %d, body=%s", resp.StatusCode, wantStatus, readBody(t, resp.Body))
	}
	var envelope dataEnvelope[T]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func mustPaginated[T any](t *testing.T, resp *http.Response, wantStatus int) paginatedEnvelope[T] {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("response status = %d, want %d, body=%s", resp.StatusCode, wantStatus, readBody(t, resp.Body))
	}
	var envelope paginatedEnvelope[T]
	decodeBody(t, resp.Body, &envelope)
	return envelope
}

func mustError(t *testing.T, resp *http.Response, wantStatus int) errorEnvelope {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("response status = %d, want %d, body=%s", resp.StatusCode, wantStatus, readBody(t, resp.Body))
	}
	var envelope errorEnvelope
	decodeBody(t, resp.Body, &envelope)
	return envelope
}

func readBody(t *testing.T, body io.Reader) string {
	t.Helper()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(raw)
}

func decodeBody(t *testing.T, body io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func lexMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "lex_db")
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)
}

func createActiveContractForMonitor(t *testing.T, h *lexHarness, title string, expiryDate time.Time, autoRenew bool) model.Contract {
	t.Helper()
	effectiveDate := expiryDate.AddDate(0, -6, 0)
	req := map[string]any{
		"title":               title,
		"type":                "service_agreement",
		"description":         "Monitor fixture contract",
		"party_a_name":        "Clario Holdings Limited",
		"party_b_name":        "Counterparty Ltd.",
		"total_value":         250000,
		"currency":            "SAR",
		"payment_terms":       "net_30",
		"effective_date":      effectiveDate.Format(time.RFC3339),
		"expiry_date":         expiryDate.Format(time.RFC3339),
		"auto_renew":          autoRenew,
		"renewal_notice_days": 30,
		"owner_user_id":       h.userID,
		"owner_name":          "Integration Owner",
		"document": map[string]any{
			"file_id":         uuid.New(),
			"file_name":       "fixture.txt",
			"file_size_bytes": 64,
			"content_hash":    uuid.NewString(),
			"extracted_text":  "Section 1 Termination\nEither party may terminate for material breach after notice.",
			"change_summary":  "initial fixture",
		},
	}
	contract := mustData[model.Contract](t, h.doJSON(t, http.MethodPost, "/api/v1/lex/contracts", req), http.StatusCreated)
	for _, status := range []string{"internal_review", "legal_review", "negotiation", "pending_signature", "active"} {
		contract = mustData[model.Contract](t, h.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/lex/contracts/%s/status", contract.ID), map[string]any{"status": status}), http.StatusOK)
	}
	return contract
}

func runExpiryMonitor(t *testing.T, h *lexHarness) {
	t.Helper()
	monitor := lexmonitor.NewExpiryMonitor(
		h.env.db,
		h.env.app.Store.Contracts,
		h.env.app.Store.Alerts,
		h.env.app.ContractService,
		h.env.app.Metrics,
		h.env.producer,
		events.Topics.LexEvents,
		time.Hour,
		h.env.logger,
	)
	if err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("expiry monitor RunOnce(): %v", err)
	}
}
