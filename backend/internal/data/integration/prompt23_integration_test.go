//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	mysqlmod "github.com/testcontainers/testcontainers-go/modules/mysql"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	dataconfig "github.com/clario360/platform/internal/data/config"
	"github.com/clario360/platform/internal/data/connector"
	datadto "github.com/clario360/platform/internal/data/dto"
	datahandler "github.com/clario360/platform/internal/data/handler"
	datahealth "github.com/clario360/platform/internal/data/health"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	datamodel "github.com/clario360/platform/internal/data/model"
	datarepo "github.com/clario360/platform/internal/data/repository"
	datasvc "github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/database"
	obshealth "github.com/clario360/platform/internal/observability/health"
)

type integrationHarness struct {
	ctx        context.Context
	serviceDB  *pgxpool.Pool
	httpServer *httptest.Server
	client     *http.Client
	token      string
	tenantID   uuid.UUID

	sourcePostgresHost string
	sourcePostgresPort int
	sourceMySQLHost    string
	sourceMySQLPort    int
	minioEndpoint      string
	apiBaseURL         string
}

type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

func TestPrompt23_DataSuite_ConnectorE2E_HTTP(t *testing.T) {
	t.Parallel()
	h := newIntegrationHarness(t)

	postgresSourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "pg-customers",
		Description: "postgres integration source",
		Type:        string(datamodel.DataSourceTypePostgreSQL),
		ConnectionConfig: mustRawJSON(t, map[string]any{
			"host":        h.sourcePostgresHost,
			"port":        h.sourcePostgresPort,
			"database":    "source",
			"schema":      "app",
			"username":    "sourceuser",
			"password":    "sourcepass",
			"ssl_mode":    "disable",
		}),
	})
	mysqlSourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "mysql-hr",
		Description: "mysql integration source",
		Type:        string(datamodel.DataSourceTypeMySQL),
		ConnectionConfig: mustRawJSON(t, map[string]any{
			"host":     h.sourceMySQLHost,
			"port":     h.sourceMySQLPort,
			"database": "source",
			"username": "mysqluser",
			"password": "mysqlpass",
			"tls_mode": "false",
		}),
	})
	csvSourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "csv-directory",
		Description: "csv integration source",
		Type:        string(datamodel.DataSourceTypeCSV),
		ConnectionConfig: mustRawJSON(t, map[string]any{
			"minio_endpoint": h.minioEndpoint,
			"bucket":         "integration-data",
			"file_path":      "employee_directory.csv",
			"delimiter":      ",",
			"has_header":     true,
			"encoding":       "utf-8",
			"quote_char":     "\"",
			"access_key":     "minioadmin",
			"secret_key":     "minioadmin",
			"use_ssl":        false,
		}),
	})
	apiSourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "crm-api",
		Description: "api integration source",
		Type:        string(datamodel.DataSourceTypeAPI),
		ConnectionConfig: mustRawJSON(t, map[string]any{
			"base_url":                 h.apiBaseURL + "/data",
			"data_path":                "$.data",
			"auth_type":                "bearer",
			"auth_config":              map[string]any{"token": "integration-api-token"},
			"pagination_type":          "offset",
			"allow_http":               true,
			"allow_private_addresses":  true,
			"rate_limit":               5,
		}),
	})

	for _, id := range []uuid.UUID{postgresSourceID, mysqlSourceID, csvSourceID, apiSourceID} {
		testResult := h.testSource(t, id)
		if !testResult.Success {
			t.Fatalf("source %s connection test failed: %+v", id, testResult)
		}
	}

	postgresSchema := h.discoverSource(t, postgresSourceID)
	if !schemaHasTable(postgresSchema, "customers") {
		t.Fatalf("postgres schema missing customers table: %+v", postgresSchema.Tables)
	}
	if !schemaContainsPII(postgresSchema, "email") || !schemaContainsPII(postgresSchema, "ssn") {
		t.Fatalf("postgres schema missing expected PII inference")
	}

	mysqlSchema := h.discoverSource(t, mysqlSourceID)
	if !schemaHasTable(mysqlSchema, "employees") || !schemaContainsPII(mysqlSchema, "email_address") {
		t.Fatalf("mysql discovery missing expected tables or pii")
	}

	csvSchema := h.discoverSource(t, csvSourceID)
	if !schemaHasTable(csvSchema, "employee_directory.csv") || !schemaContainsPII(csvSchema, "email_address") {
		t.Fatalf("csv discovery missing expected table or pii")
	}

	apiSchema := h.discoverSource(t, apiSourceID)
	if !schemaHasTable(apiSchema, "api_response") || !schemaContainsPII(apiSchema, "contact_email") {
		t.Fatalf("api discovery missing expected table or pii")
	}

	storedSchema := h.getSchema(t, postgresSourceID)
	if !schemaHasTable(storedSchema, "customers") {
		t.Fatalf("stored schema missing discovered table")
	}

	h.assertEncryptedAtRest(t, postgresSourceID)
	h.assertSanitizedSource(t, postgresSourceID)

	modelID := h.deriveModel(t, postgresSourceID, "customers", "customer_master")
	validation := h.validateModel(t, modelID)
	if !validation.Success {
		t.Fatalf("derived model validation failed: %+v", validation.Errors)
	}
}

func TestPrompt23_PostgresSchemaDiscovery_LargeDB_Under10Seconds(t *testing.T) {
	t.Parallel()
	testcontainersSkipIfUnavailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pgContainer, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("perf"),
		postgresmod.WithUsername("perfuser"),
		postgresmod.WithPassword("perfpass"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	tc.CleanupContainer(t, pgContainer)

	dsn := pgContainer.MustConnectionString(ctx, "sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open perf postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := seedWidePostgresSchema(ctx, pool); err != nil {
		t.Fatalf("seed wide postgres schema: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("postgres host: %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("postgres port: %v", err)
	}

	registry := connector.NewConnectorRegistry(connector.ConnectorLimits{
		MaxPoolSize:      3,
		StatementTimeout: 30 * time.Second,
		ConnectTimeout:   10 * time.Second,
		MaxSampleRows:    0,
		MaxTables:        200,
		APIRateLimit:     5,
	}, zerolog.Nop())

	configJSON := mustRawJSON(t, map[string]any{
		"host":        host,
		"port":        port.Int(),
		"database":    "perf",
		"schema":      "perf",
		"username":    "perfuser",
		"password":    "perfpass",
		"ssl_mode":    "disable",
	})
	conn, err := registry.Create(datamodel.DataSourceTypePostgreSQL, configJSON)
	if err != nil {
		t.Fatalf("create postgres connector: %v", err)
	}
	defer conn.Close()

	start := time.Now()
	schema, err := conn.DiscoverSchema(ctx, connector.DiscoveryOptions{
		MaxTables:    150,
		MaxColumns:   50,
		SampleValues: false,
		MaxSamples:   0,
		IncludeViews: false,
		SchemaFilter: "perf",
	})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	duration := time.Since(start)
	if schema.TableCount != 100 {
		t.Fatalf("schema.TableCount = %d, want 100", schema.TableCount)
	}
	if duration >= 10*time.Second {
		t.Fatalf("schema discovery duration = %s, want < 10s", duration)
	}
}

func newIntegrationHarness(t *testing.T) *integrationHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	servicePG, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("data_service"),
		postgresmod.WithUsername("svcuser"),
		postgresmod.WithPassword("svcpass"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start service postgres: %v", err)
	}
	tc.CleanupContainer(t, servicePG)

	sourcePG, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("source"),
		postgresmod.WithUsername("sourceuser"),
		postgresmod.WithPassword("sourcepass"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start source postgres: %v", err)
	}
	tc.CleanupContainer(t, sourcePG)

	mysqlContainer, err := mysqlmod.Run(ctx, "mysql:8.0.36",
		mysqlmod.WithDatabase("source"),
		mysqlmod.WithUsername("mysqluser"),
		mysqlmod.WithPassword("mysqlpass"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	tc.CleanupContainer(t, mysqlContainer)

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
		t.Fatalf("start minio container: %v", err)
	}
	tc.CleanupContainer(t, minioContainer)

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer integration-api-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/data" {
			http.NotFound(w, r)
			return
		}
		payload := map[string]any{
			"data": []map[string]any{
				{"account_id": "acct-1001", "contact_email": "owner@example.com", "contact_phone": "+15550100", "region": "EMEA"},
				{"account_id": "acct-1002", "contact_email": "ops@example.com", "contact_phone": "+15550101", "region": "NA"},
			},
			"meta": map[string]any{"page": 1, "total_pages": 1},
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(apiServer.Close)

	serviceDBURL := servicePG.MustConnectionString(ctx, "sslmode=disable")
	if err := database.RunMigrations(serviceDBURL, integrationMigrationsPath(t)); err != nil {
		t.Fatalf("run data migrations: %v", err)
	}
	servicePool, err := pgxpool.New(ctx, serviceDBURL)
	if err != nil {
		t.Fatalf("open service postgres pool: %v", err)
	}
	t.Cleanup(servicePool.Close)

	sourceDBURL := sourcePG.MustConnectionString(ctx, "sslmode=disable")
	sourcePool, err := pgxpool.New(ctx, sourceDBURL)
	if err != nil {
		t.Fatalf("open source postgres pool: %v", err)
	}
	t.Cleanup(sourcePool.Close)
	if err := seedSourcePostgres(ctx, sourcePool); err != nil {
		t.Fatalf("seed source postgres: %v", err)
	}

	mysqlDSN := mysqlContainer.MustConnectionString(ctx)
	mysqlDB, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		t.Fatalf("open source mysql: %v", err)
	}
	t.Cleanup(func() { _ = mysqlDB.Close() })
	if err := seedSourceMySQL(ctx, mysqlDB); err != nil {
		t.Fatalf("seed source mysql: %v", err)
	}

	minioHost, err := minioContainer.Host(ctx)
	if err != nil {
		t.Fatalf("minio host: %v", err)
	}
	minioPort, err := minioContainer.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatalf("minio port: %v", err)
	}
	minioEndpoint := fmt.Sprintf("%s:%s", minioHost, minioPort.Port())
	if err := seedMinIO(ctx, minioEndpoint); err != nil {
		t.Fatalf("seed minio: %v", err)
	}

	sourcePGHost, err := sourcePG.Host(ctx)
	if err != nil {
		t.Fatalf("source postgres host: %v", err)
	}
	sourcePGPort, err := sourcePG.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("source postgres port: %v", err)
	}
	mysqlHost, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatalf("mysql host: %v", err)
	}
	mysqlPort, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	if err != nil {
		t.Fatalf("mysql port: %v", err)
	}

	tenantID := uuid.New()
	userID := uuid.New()
	logger := zerolog.Nop()
	encryptionKey := make([]byte, 32)
	if _, err := rand.Read(encryptionKey); err != nil {
		t.Fatalf("rand.Read(encryptionKey): %v", err)
	}

	cfg := &dataconfig.Config{
		ConnectorMaxPoolSize:      3,
		ConnectorStatementTimeout: 30 * time.Second,
		ConnectorConnectTimeout:   10 * time.Second,
		ConnectorMaxSampleRows:    20,
		ConnectorMaxTables:        500,
		ConnectorAPIRateLimit:     5,
		DiscoveryMaxColumns:       1000,
		DiscoverySampleValues:     true,
		DiscoveryPIIEnabled:       true,
		EncryptionKey:             encryptionKey,
		EncryptionKeyID:           base64.StdEncoding.EncodeToString(encryptionKey)[:8],
	}
	encryptor, err := datasvc.NewConfigEncryptorFromBytes(encryptionKey)
	if err != nil {
		t.Fatalf("NewConfigEncryptorFromBytes() error = %v", err)
	}

	dataSuiteMetrics := datametrics.New(prometheus.NewRegistry())
	registry := connector.NewConnectorRegistry(connector.ConnectorLimits{
		MaxPoolSize:      cfg.ConnectorMaxPoolSize,
		StatementTimeout: cfg.ConnectorStatementTimeout,
		ConnectTimeout:   cfg.ConnectorConnectTimeout,
		MaxSampleRows:    cfg.ConnectorMaxSampleRows,
		MaxTables:        cfg.ConnectorMaxTables,
		APIRateLimit:     cfg.ConnectorAPIRateLimit,
	}, logger)
	discoveryOpts := connector.DiscoveryOptions{
		MaxTables:    cfg.ConnectorMaxTables,
		MaxColumns:   cfg.DiscoveryMaxColumns,
		SampleValues: cfg.DiscoverySampleValues,
		MaxSamples:   cfg.ConnectorMaxSampleRows,
		IncludeViews: true,
	}

	sourceRepo := datarepo.NewSourceRepository(servicePool, logger)
	modelRepo := datarepo.NewModelRepository(servicePool, logger)
	syncRepo := datarepo.NewSyncRepository(servicePool, logger)
	tester := datasvc.NewConnectionTester(registry, dataSuiteMetrics)
	discoverySvc := datasvc.NewSchemaDiscoveryService(registry, discoveryOpts, dataSuiteMetrics)
	ingestionSvc := datasvc.NewIngestionService(registry, sourceRepo, syncRepo, discoveryOpts, dataSuiteMetrics, logger)
	sourceSvc := datasvc.NewSourceService(cfg, sourceRepo, syncRepo, tester, discoverySvc, ingestionSvc, encryptor, nil, dataSuiteMetrics, logger)
	modelSvc := datasvc.NewModelService(modelRepo, sourceRepo, nil, dataSuiteMetrics, logger)

	jwtMgr, err := auth.NewJWTManager(appconfig.AuthConfig{
		JWTIssuer:       "integration-test",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}
	tokenPair, err := jwtMgr.GenerateTokenPair(userID.String(), tenantID.String(), "integration@clario360.test", []string{"tenant_admin"})
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	router := chi.NewRouter()
	checker := obshealth.NewCompositeHealthChecker(time.Second)
	datahealth.Register(router, checker, "data-service-test", "integration")
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  25 * time.Millisecond,
		ReadTimeout:  25 * time.Millisecond,
		WriteTimeout: 25 * time.Millisecond,
	})
	t.Cleanup(func() { _ = redisClient.Close() })
	datahandler.RegisterRoutes(
		router,
		datahandler.NewSourceHandler(sourceSvc, logger),
		datahandler.NewModelHandler(modelSvc, logger),
		jwtMgr,
		redisClient,
	)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &integrationHarness{
		ctx:                ctx,
		serviceDB:          servicePool,
		httpServer:         server,
		client:             server.Client(),
		token:              tokenPair.AccessToken,
		tenantID:           tenantID,
		sourcePostgresHost: sourcePGHost,
		sourcePostgresPort: sourcePGPort.Int(),
		sourceMySQLHost:    mysqlHost,
		sourceMySQLPort:    mysqlPort.Int(),
		minioEndpoint:      minioEndpoint,
		apiBaseURL:         apiServer.URL,
	}
}

func (h *integrationHarness) createSource(t *testing.T, req datadto.CreateSourceRequest) uuid.UUID {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, "/api/v1/data/sources", req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create source status = %d, want %d, body=%s", resp.StatusCode, http.StatusCreated, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataSource]
	decodeBody(t, resp.Body, &envelope)
	if len(envelope.Data.ConnectionConfig) == 0 {
		t.Fatalf("create source response missing sanitized connection config")
	}
	if bytes.Contains(envelope.Data.ConnectionConfig, []byte("password")) || bytes.Contains(envelope.Data.ConnectionConfig, []byte("secret_key")) {
		t.Fatalf("create source response leaked secrets: %s", string(envelope.Data.ConnectionConfig))
	}
	return envelope.Data.ID
}

func (h *integrationHarness) testSource(t *testing.T, sourceID uuid.UUID) connector.ConnectionTestResult {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/data/sources/%s/test", sourceID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test source status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[connector.ConnectionTestResult]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) discoverSource(t *testing.T, sourceID uuid.UUID) datamodel.DiscoveredSchema {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/data/sources/%s/discover", sourceID), map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("discover source status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DiscoveredSchema]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) getSchema(t *testing.T, sourceID uuid.UUID) datamodel.DiscoveredSchema {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/sources/%s/schema", sourceID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get schema status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DiscoveredSchema]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) deriveModel(t *testing.T, sourceID uuid.UUID, tableName, name string) uuid.UUID {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, "/api/v1/data/models/derive", datadto.DeriveModelRequest{
		SourceID:  sourceID,
		TableName: tableName,
		Name:      name,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("derive model status = %d, want %d, body=%s", resp.StatusCode, http.StatusCreated, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataModel]
	decodeBody(t, resp.Body, &envelope)
	if !envelope.Data.ContainsPII || len(envelope.Data.QualityRules) == 0 {
		t.Fatalf("derived model missing pii/rules: %+v", envelope.Data)
	}
	return envelope.Data.ID
}

func (h *integrationHarness) validateModel(t *testing.T, modelID uuid.UUID) datamodel.ModelValidationResult {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/data/models/%s/validate", modelID), map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("validate model status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.ModelValidationResult]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func (h *integrationHarness) assertEncryptedAtRest(t *testing.T, sourceID uuid.UUID) {
	t.Helper()
	var raw []byte
	err := h.serviceDB.QueryRow(h.ctx, `SELECT connection_config FROM data_sources WHERE id = $1 AND tenant_id = $2`, sourceID, h.tenantID).Scan(&raw)
	if err != nil {
		t.Fatalf("query encrypted config: %v", err)
	}
	if json.Valid(raw) {
		t.Fatalf("connection_config stored as plaintext JSON: %s", string(raw))
	}
	if bytes.Contains(raw, []byte("sourcepass")) {
		t.Fatal("encrypted connection_config still contains plaintext password")
	}
}

func (h *integrationHarness) assertSanitizedSource(t *testing.T, sourceID uuid.UUID) {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/sources/%s", sourceID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get source status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataSource]
	decodeBody(t, resp.Body, &envelope)
	if bytes.Contains(envelope.Data.ConnectionConfig, []byte("password")) || bytes.Contains(envelope.Data.ConnectionConfig, []byte("sourcepass")) {
		t.Fatalf("sanitized source leaked credentials: %s", string(envelope.Data.ConnectionConfig))
	}
}

func (h *integrationHarness) doJSON(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal(%T): %v", body, err)
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(h.ctx, method, h.httpServer.URL+path, payload)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("http.Do() error = %v", err)
	}
	return resp
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	tc.SkipIfProviderIsNotHealthy(t)
}

func integrationMigrationsPath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "data_db")
}

func seedSourcePostgres(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE SCHEMA IF NOT EXISTS app`,
		`CREATE TABLE IF NOT EXISTS app.customers (
			customer_id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			phone VARCHAR(40),
			ssn VARCHAR(20),
			first_name VARCHAR(120) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS app.orders (
			order_id UUID PRIMARY KEY,
			customer_id UUID NOT NULL REFERENCES app.customers(customer_id),
			total_amount NUMERIC(12,2) NOT NULL,
			ordered_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`TRUNCATE app.orders, app.customers`,
		`INSERT INTO app.customers (customer_id, email, phone, ssn, first_name) VALUES
			('11111111-1111-1111-1111-111111111111', 'alice@example.com', '+15550100', '123-45-6789', 'Alice'),
			('22222222-2222-2222-2222-222222222222', 'bob@example.com', '+15550101', '987-65-4321', 'Bob')`,
		`INSERT INTO app.orders (order_id, customer_id, total_amount) VALUES
			('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 149.95),
			('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 89.50)`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func seedWidePostgresSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS perf`); err != nil {
		return err
	}
	for tableIdx := 1; tableIdx <= 100; tableIdx++ {
		columns := make([]string, 0, 31)
		columns = append(columns, "id BIGSERIAL PRIMARY KEY")
		for columnIdx := 1; columnIdx <= 30; columnIdx++ {
			columns = append(columns, fmt.Sprintf("col_%02d VARCHAR(128)", columnIdx))
		}
		stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS perf.table_%03d (%s)", tableIdx, strings.Join(columns, ", "))
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func seedSourceMySQL(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS employees (
			employee_id CHAR(36) PRIMARY KEY,
			email_address VARCHAR(255) NOT NULL,
			phone_number VARCHAR(40),
			base_salary DECIMAL(12,2) NOT NULL,
			date_of_birth DATE
		)`,
		`TRUNCATE TABLE employees`,
		`INSERT INTO employees (employee_id, email_address, phone_number, base_salary, date_of_birth) VALUES
			('11111111-1111-1111-1111-111111111111', 'hr1@example.com', '+15550200', 85000.00, '1990-02-01'),
			('22222222-2222-2222-2222-222222222222', 'hr2@example.com', '+15550201', 92000.00, '1988-08-11')`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func seedMinIO(ctx context.Context, endpoint string) error {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		return err
	}
	exists, err := client.BucketExists(ctx, "integration-data")
	if err != nil {
		return err
	}
	if !exists {
		if err := client.MakeBucket(ctx, "integration-data", minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	content := "employee_id,full_name,email_address,phone_number\nemp-001,Alice Doe,alice@example.com,+15550111\nemp-002,Bob Doe,bob@example.com,+15550112\n"
	_, err = client.PutObject(ctx, "integration-data", "employee_directory.csv", strings.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: "text/csv"})
	return err
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func decodeBody(t *testing.T, reader io.Reader, dest any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(dest); err != nil {
		t.Fatalf("json decode: %v", err)
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("<read error: %v>", err)
	}
	return string(body)
}

func schemaHasTable(schema datamodel.DiscoveredSchema, name string) bool {
	for _, table := range schema.Tables {
		if strings.EqualFold(table.Name, name) {
			return true
		}
	}
	return false
}

func schemaContainsPII(schema datamodel.DiscoveredSchema, columnName string) bool {
	for _, table := range schema.Tables {
		for _, column := range table.Columns {
			if strings.EqualFold(column.Name, columnName) && column.InferredPII {
				return true
			}
		}
	}
	return false
}
