//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	actaapp "github.com/clario360/platform/internal/acta"
	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	lexapp "github.com/clario360/platform/internal/lex"
	lexconfig "github.com/clario360/platform/internal/lex/config"
	visusapp "github.com/clario360/platform/internal/visus"
	visusconfig "github.com/clario360/platform/internal/visus/config"
	"github.com/clario360/platform/internal/visus/model"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
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
)

type liveEnv struct {
	lexPostgres   tc.Container
	actaPostgres  tc.Container
	visusPostgres tc.Container
	redis         tc.Container
	redpanda      tc.Container

	lexDB   *pgxpool.Pool
	actaDB  *pgxpool.Pool
	visusDB *pgxpool.Pool
	rdb     *redis.Client
	jwt     *auth.JWTManager

	producer *events.Producer
	consumer *events.Consumer

	lexServer   *httptest.Server
	actaServer  *httptest.Server
	visusServer *httptest.Server
	visusApp    *visusapp.Application

	logger zerolog.Logger
}

func TestExecutiveView_LiveLexAndKafka(t *testing.T) {
	testcontainersSkipIfUnavailable(t)
	env := startLiveEnv(t)
	defer env.Close(t)

	lexTenant := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	lexUser := uuid.MustParse("22222222-2222-2222-2222-222222222201")
	env.seedVisusTenant(t, lexTenant, lexUser)

	view := env.getExecutiveView(t, lexTenant, lexUser)
	if view.Legal == nil {
		t.Fatalf("expected legal summary, got nil; suite health=%+v", view.SuiteHealth)
	}
	if !view.SuiteHealth["lex"].Available {
		t.Fatalf("expected lex suite available, got %+v", view.SuiteHealth["lex"])
	}
	if view.SuiteHealth["cyber"].Available || view.SuiteHealth["data"].Available {
		t.Fatalf("expected cyber/data unavailable in this topology, got %+v", view.SuiteHealth)
	}

	before := env.getAlerts(t, lexTenant, lexUser)
	event, err := events.NewEvent("enterprise.lex.contract.expiring", "lex-service", lexTenant.String(), map[string]any{
		"contract_id": uuid.New().String(),
		"title":       "Kafka propagated contract expiry",
	})
	if err != nil {
		t.Fatalf("create lex event: %v", err)
	}
	if err := env.producer.Publish(context.Background(), events.Topics.LexEvents, event); err != nil {
		t.Fatalf("publish lex event: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		alerts := env.getAlerts(t, lexTenant, lexUser)
		if len(alerts) > len(before) {
			for _, alert := range alerts {
				if alert.SourceSuite == "lex" && alert.SourceType == "event" {
					return
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("expected visus to ingest live Kafka lex event; alerts before=%d after=%d", len(before), len(env.getAlerts(t, lexTenant, lexUser)))
}

func TestExecutiveView_LiveActa(t *testing.T) {
	testcontainersSkipIfUnavailable(t)
	env := startLiveEnv(t)
	defer env.Close(t)

	actaTenant := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	actaUser := uuid.MustParse("11111111-1111-1111-1111-111111111201")
	env.seedVisusTenant(t, actaTenant, actaUser)

	view := env.getExecutiveView(t, actaTenant, actaUser)
	if view.Governance == nil {
		t.Fatalf("expected governance summary, got nil; suite health=%+v", view.SuiteHealth)
	}
	if !view.SuiteHealth["acta"].Available {
		t.Fatalf("expected acta suite available, got %+v", view.SuiteHealth["acta"])
	}
	if view.Governance.ComplianceScore <= 0 {
		t.Fatalf("expected populated governance score, got %+v", view.Governance)
	}

	before := env.getAlerts(t, actaTenant, actaUser)
	event, err := events.NewEvent("enterprise.acta.action_item.overdue", "acta-service", actaTenant.String(), map[string]any{
		"action_item_id": uuid.New().String(),
		"title":          "Board remediation overdue",
	})
	if err != nil {
		t.Fatalf("create acta event: %v", err)
	}
	if err := env.producer.Publish(context.Background(), events.Topics.ActaEvents, event); err != nil {
		t.Fatalf("publish acta event: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		alerts := env.getAlerts(t, actaTenant, actaUser)
		if len(alerts) > len(before) {
			for _, alert := range alerts {
				if alert.SourceSuite == "acta" && alert.SourceType == "event" {
					return
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("expected visus to ingest live Kafka acta event; alerts before=%d after=%d", len(before), len(env.getAlerts(t, actaTenant, actaUser)))
}

func startLiveEnv(t *testing.T) *liveEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger := zerolog.New(io.Discard)

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
		t.Fatalf("start redis container: %v", err)
	}

	redpandaContainer, err := redpandamod.Run(ctx, "docker.redpanda.com/redpandadata/redpanda:v24.1.8", redpandamod.WithAutoCreateTopics())
	if err != nil {
		_ = redisContainer.Terminate(context.Background())
		t.Fatalf("start redpanda container: %v", err)
	}

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("redis port: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port())})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}

	seedBroker, err := redpandaContainer.KafkaSeedBroker(ctx)
	if err != nil {
		t.Fatalf("resolve redpanda broker: %v", err)
	}
	brokers := []string{seedBroker}

	producer, err := events.NewProducer(appconfig.KafkaConfig{
		Brokers: brokers,
		GroupID: "visus-live-integration-producer",
	}, logger)
	if err != nil {
		t.Fatalf("create kafka producer: %v", err)
	}
	consumer, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         brokers,
		GroupID:         "visus-live-integration-consumer",
		AutoOffsetReset: "earliest",
	}, logger)
	if err != nil {
		t.Fatalf("create kafka consumer: %v", err)
	}

	jwtMgr, err := auth.NewJWTManager(appconfig.AuthConfig{
		JWTIssuer:       "visus-live-integration",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("create jwt manager: %v", err)
	}

	lexPostgres, lexDB := startSuiteDB(t, ctx, "lex_integration", "lex", "lex", lexMigrationsPath(), true)
	actaPostgres, actaDB := startSuiteDB(t, ctx, "acta_integration", "acta", "acta", actaMigrationsPath(), true)
	visusPostgres, visusDB := startSuiteDB(t, ctx, "visus_integration", "visus", "visus", visusMigrationsPath(), false)

	lexCfg := lexconfig.Default()
	lexCfg.KafkaBrokers = brokers
	lexApp, err := lexapp.NewApplication(lexapp.Dependencies{
		DB:                lexDB,
		Redis:             rdb,
		Publisher:         producer,
		Logger:            logger,
		Registerer:        prometheus.NewRegistry(),
		WorkflowDefRepo:   workflowrepo.NewDefinitionRepository(lexDB),
		WorkflowInstRepo:  workflowrepo.NewInstanceRepository(lexDB),
		WorkflowTaskRepo:  workflowrepo.NewTaskRepository(lexDB),
		Config:            lexCfg,
		DashboardCacheTTL: time.Minute,
		OrgJurisdiction:   "Saudi Arabia",
		KafkaTopic:        events.Topics.LexEvents,
	})
	if err != nil {
		t.Fatalf("create lex app: %v", err)
	}
	lexRouter := chi.NewRouter()
	lexApp.RegisterRoutes(lexRouter, jwtMgr, rdb, 10000)
	lexServer := httptest.NewServer(lexRouter)

	actaApp, err := actaapp.NewApplication(actaapp.Dependencies{
		DB:                actaDB,
		Redis:             rdb,
		Publisher:         producer,
		Logger:            logger,
		Registerer:        prometheus.NewRegistry(),
		DashboardCacheTTL: time.Minute,
		KafkaTopic:        events.Topics.ActaEvents,
		WorkflowDefRepo:   workflowrepo.NewDefinitionRepository(actaDB),
		WorkflowInstRepo:  workflowrepo.NewInstanceRepository(actaDB),
	})
	if err != nil {
		t.Fatalf("create acta app: %v", err)
	}
	if _, err := actaapp.SeedDemoData(ctx, actaApp.Store, logger); err != nil {
		t.Fatalf("seed acta demo data: %v", err)
	}
	actaRouter := chi.NewRouter()
	actaApp.RegisterRoutes(actaRouter, jwtMgr, rdb, 10000)
	actaServer := httptest.NewServer(actaRouter)

	visusCfg := visusconfig.Default()
	visusCfg.SuiteLexURL = lexServer.URL
	visusCfg.SuiteActaURL = actaServer.URL
	visusCfg.SuiteCyberURL = "http://127.0.0.1:1"
	visusCfg.SuiteDataURL = "http://127.0.0.1:1"
	visusCfg.SuiteTimeout = 300 * time.Millisecond
	visusCfg.SuiteMaxRetries = 1
	visusCfg.CircuitThreshold = 1
	visusCfg.CircuitReset = time.Second
	visusCfg.ServiceTokenTTL = time.Minute

	visusApp, err := visusapp.NewApplication(visusapp.Dependencies{
		DB:         visusDB,
		Redis:      rdb,
		Publisher:  producer,
		Logger:     logger,
		Registerer: prometheus.NewRegistry(),
		Config:     visusCfg,
		JWTManager: jwtMgr,
	})
	if err != nil {
		t.Fatalf("create visus app: %v", err)
	}
	visusRouter := chi.NewRouter()
	visusApp.RegisterRoutes(visusRouter, jwtMgr, rdb, 10000)
	visusServer := httptest.NewServer(visusRouter)

	visusApp.Consumer.Register(consumer)
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	go func() {
		_ = consumer.Start(consumerCtx)
	}()
	t.Cleanup(consumerCancel)

	return &liveEnv{
		lexPostgres:   lexPostgres,
		actaPostgres:  actaPostgres,
		visusPostgres: visusPostgres,
		redis:         redisContainer,
		redpanda:      redpandaContainer,
		lexDB:         lexDB,
		actaDB:        actaDB,
		visusDB:       visusDB,
		rdb:           rdb,
		jwt:           jwtMgr,
		producer:      producer,
		consumer:      consumer,
		lexServer:     lexServer,
		actaServer:    actaServer,
		visusServer:   visusServer,
		visusApp:      visusApp,
		logger:        logger,
	}
}

func startSuiteDB(t *testing.T, ctx context.Context, databaseName, username, password, migrationsPath string, workflow bool) (tc.Container, *pgxpool.Pool) {
	t.Helper()

	container, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase(databaseName),
		postgresmod.WithUsername(username),
		postgresmod.WithPassword(password),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres %s: %v", databaseName, err)
	}

	dbURL := container.MustConnectionString(ctx, "sslmode=disable")
	if err := database.RunMigrations(dbURL, migrationsPath); err != nil {
		t.Fatalf("run migrations %s: %v", migrationsPath, err)
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("open postgres pool %s: %v", databaseName, err)
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping postgres %s: %v", databaseName, err)
	}
	if workflow {
		if err := workflowrepo.RunMigration(ctx, db); err != nil {
			t.Fatalf("run workflow migration %s: %v", databaseName, err)
		}
	}
	return container, db
}

func (e *liveEnv) seedVisusTenant(t *testing.T, tenantID, userID uuid.UUID) {
	t.Helper()
	cfg := visusconfig.Default()
	cfg.DemoTenantID = tenantID.String()
	cfg.DemoUserID = userID.String()
	if _, err := visusapp.SeedDemoData(context.Background(), e.visusApp, cfg, e.logger); err != nil {
		t.Fatalf("seed visus tenant %s: %v", tenantID, err)
	}
}

func (e *liveEnv) getExecutiveView(t *testing.T, tenantID, userID uuid.UUID) model.ExecutiveView {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.visusServer.URL+"/api/v1/visus/executive", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.mustToken(t, tenantID, userID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected executive status %d: %s", resp.StatusCode, readBody(t, resp.Body))
	}
	var envelope struct {
		Data model.ExecutiveView `json:"data"`
	}
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (e *liveEnv) getAlerts(t *testing.T, tenantID, userID uuid.UUID) []model.ExecutiveAlert {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.visusServer.URL+"/api/v1/visus/alerts?per_page=50", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.mustToken(t, tenantID, userID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected alerts status %d: %s", resp.StatusCode, readBody(t, resp.Body))
	}
	var envelope struct {
		Data []model.ExecutiveAlert `json:"data"`
	}
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (e *liveEnv) mustToken(t *testing.T, tenantID, userID uuid.UUID) string {
	t.Helper()
	pair, err := e.jwt.GenerateTokenPair(userID.String(), tenantID.String(), "integration@clario.local", []string{"tenant_admin"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return pair.AccessToken
}

func (e *liveEnv) Close(t *testing.T) {
	t.Helper()
	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if e.visusServer != nil {
		e.visusServer.Close()
	}
	if e.actaServer != nil {
		e.actaServer.Close()
	}
	if e.lexServer != nil {
		e.lexServer.Close()
	}
	if e.consumer != nil {
		_ = e.consumer.Close()
	}
	if e.producer != nil {
		_ = e.producer.Close()
	}
	if e.rdb != nil {
		_ = e.rdb.Close()
	}
	if e.lexDB != nil {
		e.lexDB.Close()
	}
	if e.actaDB != nil {
		e.actaDB.Close()
	}
	if e.visusDB != nil {
		e.visusDB.Close()
	}
	if e.redpanda != nil {
		_ = e.redpanda.Terminate(closeCtx)
	}
	if e.redis != nil {
		_ = e.redis.Terminate(closeCtx)
	}
	if e.lexPostgres != nil {
		_ = e.lexPostgres.Terminate(closeCtx)
	}
	if e.actaPostgres != nil {
		_ = e.actaPostgres.Terminate(closeCtx)
	}
	if e.visusPostgres != nil {
		_ = e.visusPostgres.Terminate(closeCtx)
	}
}

func lexMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "lex_db")
}

func actaMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "acta_db")
}

func visusMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "visus_db")
}

func decodeBody(t *testing.T, body io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func readBody(t *testing.T, body io.Reader) string {
	t.Helper()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(raw)
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)
}
