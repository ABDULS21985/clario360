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

	"github.com/IBM/sarama"
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

	actaapp "github.com/clario360/platform/internal/acta"
	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

var (
	sharedEnvOnce sync.Once
	sharedEnv     *actaIntegrationEnv
	sharedEnvErr  error

	demoSeedOnce sync.Once
	demoSeedErr  error
)

type actaIntegrationEnv struct {
	logger zerolog.Logger

	postgres *postgresmod.PostgresContainer
	redpanda *redpandamod.Container
	redis    tc.Container

	db    *pgxpool.Pool
	rdb   *redis.Client
	jwt   *auth.JWTManager
	app   *actaapp.Application
	store *repository.Store

	producer *events.Producer
	brokers  []string
	server   *httptest.Server
}

type actaHarness struct {
	env      *actaIntegrationEnv
	client   *http.Client
	tenantID uuid.UUID
	userID   uuid.UUID
	token    string
}

type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

type paginatedEnvelope[T any] struct {
	Data []T `json:"data"`
	Meta struct {
		Page       int `json:"page"`
		PerPage    int `json:"per_page"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"meta"`
}

type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type committeeFixture struct {
	Committee model.Committee
	Members   []userFixture
}

type userFixture struct {
	ID    uuid.UUID
	Name  string
	Email string
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		if err := sharedEnv.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "acta integration cleanup failed: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

func newActaHarness(t *testing.T) *actaHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	env := mustSharedEnv(t)
	tenantID := uuid.New()
	userID := uuid.New()

	h := &actaHarness{
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

func newDemoHarness(t *testing.T) *actaHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	env := mustSharedEnv(t)
	demoSeedOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		tenantID, err := actaapp.SeedDemoData(ctx, env.store, env.logger)
		if err != nil {
			demoSeedErr = fmt.Errorf("seed demo data: %w", err)
			return
		}
		if _, err := env.app.ComplianceService.RunChecks(ctx, tenantID); err != nil {
			demoSeedErr = fmt.Errorf("run demo compliance: %w", err)
		}
	})
	if demoSeedErr != nil {
		t.Fatalf("seed demo dataset: %v", demoSeedErr)
	}

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111001")
	return &actaHarness{
		env:      env,
		client:   env.server.Client(),
		tenantID: tenantID,
		userID:   userID,
		token:    env.mustToken(t, tenantID, userID, "tenant_admin"),
	}
}

func mustSharedEnv(t *testing.T) *actaIntegrationEnv {
	t.Helper()

	sharedEnvOnce.Do(func() {
		sharedEnv, sharedEnvErr = startSharedEnv()
	})
	if sharedEnvErr != nil {
		t.Fatalf("start acta integration environment: %v", sharedEnvErr)
	}
	return sharedEnv
}

func startSharedEnv() (*actaIntegrationEnv, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger := zerolog.New(io.Discard)

	postgresContainer, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("acta_service"),
		postgresmod.WithUsername("acta"),
		postgresmod.WithPassword("acta"),
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
	if err := database.RunMigrations(dbURL, actaMigrationsPath()); err != nil {
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("run acta migrations: %w", err)
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("open acta postgres pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("ping acta postgres: %w", err)
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
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
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
		GroupID: "acta-integration-producer",
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
		JWTIssuer:       "acta-integration",
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

	app, err := actaapp.NewApplication(actaapp.Dependencies{
		DB:                db,
		Redis:             rdb,
		Publisher:         producer,
		Logger:            logger,
		Registerer:        prometheus.NewRegistry(),
		DashboardCacheTTL: 60 * time.Second,
		KafkaTopic:        events.Topics.ActaEvents,
		WorkflowDefRepo:   workflowrepo.NewDefinitionRepository(db),
		WorkflowInstRepo:  workflowrepo.NewInstanceRepository(db),
	})
	if err != nil {
		_ = producer.Close()
		_ = rdb.Close()
		db.Close()
		_ = redpandaContainer.Terminate(context.Background())
		_ = redisContainer.Terminate(context.Background())
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("create acta app: %w", err)
	}

	router := chi.NewRouter()
	app.RegisterRoutes(router, jwtMgr, rdb, 10000)
	server := httptest.NewServer(router)

	return &actaIntegrationEnv{
		logger:   logger,
		postgres: postgresContainer,
		redpanda: redpandaContainer,
		redis:    redisContainer,
		db:       db,
		rdb:      rdb,
		jwt:      jwtMgr,
		app:      app,
		store:    app.Store,
		producer: producer,
		brokers:  brokers,
		server:   server,
	}, nil
}

func (e *actaIntegrationEnv) Close() error {
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

func (e *actaIntegrationEnv) mustToken(t *testing.T, tenantID, userID uuid.UUID, roles ...string) string {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"tenant_admin"}
	}
	tokenPair, err := e.jwt.GenerateTokenPair(userID.String(), tenantID.String(), fmt.Sprintf("%s@acta.integration", userID.String()), roles)
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}
	return tokenPair.AccessToken
}

func (h *actaHarness) tokenForUser(t *testing.T, userID uuid.UUID, roles ...string) string {
	t.Helper()
	return h.env.mustToken(t, h.tenantID, userID, roles...)
}

func (h *actaHarness) cleanupTenant(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statements := []string{
		`DELETE FROM workflow_tasks WHERE tenant_id = $1`,
		`DELETE FROM workflow_timers WHERE instance_id IN (SELECT id FROM workflow_instances WHERE tenant_id = $1)`,
		`DELETE FROM workflow_step_executions WHERE instance_id IN (SELECT id FROM workflow_instances WHERE tenant_id = $1)`,
		`DELETE FROM workflow_instances WHERE tenant_id = $1`,
		`DELETE FROM workflow_definitions WHERE tenant_id = $1`,
		`DELETE FROM compliance_checks WHERE tenant_id = $1`,
		`DELETE FROM action_items WHERE tenant_id = $1`,
		`DELETE FROM meeting_minutes WHERE tenant_id = $1`,
		`DELETE FROM meeting_attendance WHERE tenant_id = $1`,
		`DELETE FROM agenda_items WHERE tenant_id = $1`,
		`DELETE FROM meetings WHERE tenant_id = $1`,
		`DELETE FROM committee_members WHERE tenant_id = $1`,
		`DELETE FROM committees WHERE tenant_id = $1`,
	}
	for _, stmt := range statements {
		if _, err := h.env.db.Exec(ctx, stmt, h.tenantID); err != nil {
			t.Fatalf("cleanup tenant %s: %v", h.tenantID, err)
		}
	}
}

func (h *actaHarness) doJSON(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	return h.doJSONWithToken(t, h.token, method, path, body)
}

func (h *actaHarness) doJSONWithToken(t *testing.T, token, method, path string, body any) *http.Response {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("execute request %s %s: %v", method, path, err)
	}
	return resp
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

func (h *actaHarness) createCommittee(t *testing.T, name string, memberCount int) committeeFixture {
	t.Helper()

	chair := userFixture{
		ID:    h.userID,
		Name:  "Chair User",
		Email: "chair.user@acta.integration",
	}
	committee := mustData[model.Committee](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/committees", map[string]any{
		"name":              name,
		"type":              "audit",
		"description":       "Integration test committee",
		"chair_user_id":     chair.ID,
		"chair_name":        chair.Name,
		"chair_email":       chair.Email,
		"meeting_frequency": "monthly",
		"quorum_percentage": 51,
		"quorum_type":       "percentage",
		"tags":              []string{"integration"},
		"metadata": map[string]any{
			"charter_reviewed_at": time.Now().UTC().Format(time.RFC3339),
		},
	}), http.StatusCreated)

	members := []userFixture{chair}
	for idx := 1; idx < memberCount; idx++ {
		member := userFixture{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("Member %02d", idx),
			Email: fmt.Sprintf("member%02d@acta.integration", idx),
		}
		committee = mustData[model.Committee](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/committees/%s/members", committee.ID), map[string]any{
			"user_id":    member.ID,
			"user_name":  member.Name,
			"user_email": member.Email,
			"role":       "member",
		}), http.StatusOK)
		members = append(members, member)
	}

	return committeeFixture{
		Committee: committee,
		Members:   members,
	}
}

func (h *actaHarness) waitForEvent(t *testing.T, match func(*events.Event) bool) *events.Event {
	t.Helper()

	cfg := sarama.NewConfig()
	cfg.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumer(h.env.brokers, cfg)
	if err != nil {
		t.Fatalf("create sarama consumer: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	partitions, err := consumer.Partitions(events.Topics.ActaEvents)
	if err != nil {
		t.Fatalf("list topic partitions: %v", err)
	}

	type eventResult struct {
		event *events.Event
		err   error
	}
	resultCh := make(chan eventResult, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, partition := range partitions {
		partitionConsumer, err := consumer.ConsumePartition(events.Topics.ActaEvents, partition, sarama.OffsetOldest)
		if err != nil {
			t.Fatalf("consume partition %d: %v", partition, err)
		}
		defer func(pc sarama.PartitionConsumer) { _ = pc.Close() }(partitionConsumer)

		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case err := <-pc.Errors():
					if err != nil {
						select {
						case resultCh <- eventResult{err: err}:
						default:
						}
						return
					}
				case msg := <-pc.Messages():
					if msg == nil {
						continue
					}
					var event events.Event
					if err := json.Unmarshal(msg.Value, &event); err != nil {
						continue
					}
					if match(&event) {
						select {
						case resultCh <- eventResult{event: &event}:
						default:
						}
						return
					}
				}
			}
		}(partitionConsumer)
	}

	select {
	case result := <-resultCh:
		cancel()
		wg.Wait()
		if result.err != nil {
			t.Fatalf("consume kafka event: %v", result.err)
		}
		return result.event
	case <-ctx.Done():
		wg.Wait()
		t.Fatalf("timed out waiting for kafka event")
		return nil
	}
}

func (h *actaHarness) waitForTenantEventType(t *testing.T, eventType string) *events.Event {
	t.Helper()
	return h.waitForEvent(t, func(event *events.Event) bool {
		return event.TenantID == h.tenantID.String() && event.Type == eventType
	})
}

func (h *actaHarness) getMeeting(t *testing.T, meetingID uuid.UUID) model.Meeting {
	t.Helper()
	return mustData[model.Meeting](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/acta/meetings/%s", meetingID), nil), http.StatusOK)
}

func (h *actaHarness) getActionItem(t *testing.T, actionItemID uuid.UUID) model.ActionItem {
	t.Helper()
	return mustData[model.ActionItem](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/acta/action-items/%s", actionItemID), nil), http.StatusOK)
}

func ensureKafkaTopic(ctx context.Context, brokers []string, topic string) error {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_5_0_0
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		admin, err := sarama.NewClusterAdmin(brokers, cfg)
		if err != nil {
			lastErr = fmt.Errorf("create cluster admin: %w", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}

		detail := &sarama.TopicDetail{
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
		if err := admin.CreateTopic(topic, detail, false); err != nil && err != sarama.ErrTopicAlreadyExists {
			_ = admin.Close()
			lastErr = fmt.Errorf("create topic %s: %w", topic, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		_ = admin.Close()

		client, err := sarama.NewClient(brokers, cfg)
		if err != nil {
			lastErr = fmt.Errorf("create kafka client: %w", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		partitions, err := client.Partitions(topic)
		_ = client.Close()
		if err == nil && len(partitions) > 0 {
			return nil
		}
		if err != nil {
			lastErr = fmt.Errorf("list topic partitions: %w", err)
		} else {
			lastErr = fmt.Errorf("topic %s has no partitions yet", topic)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("topic %s was not ready before timeout", topic)
}

func actaMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "acta_db")
}

func decodeBody(t *testing.T, body io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func readBody(t *testing.T, body io.Reader) string {
	t.Helper()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return string(raw)
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)
}
