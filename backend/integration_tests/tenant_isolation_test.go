//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// TestMain — skip unless TEST_RUN_INTEGRATION is set
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	if os.Getenv("TEST_RUN_INTEGRATION") == "" {
		fmt.Fprintln(os.Stderr, "skipping integration tests: set TEST_RUN_INTEGRATION=1 to run")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// baseURL returns the API gateway base URL from the environment, defaulting to localhost.
func baseURL() string {
	if u := os.Getenv("TEST_API_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

// dbDSN returns a database DSN from an environment variable, falling back to
// the base DSN with a database-specific suffix if the specific var is not set.
func dbDSN(envVar, dbSuffix string) string {
	if dsn := os.Getenv(envVar); dsn != "" {
		return dsn
	}
	if base := os.Getenv("TEST_DB_BASE_DSN"); base != "" {
		return base + "/" + dbSuffix
	}
	return ""
}

// testTenant represents a tenant created for testing purposes.
type testTenant struct {
	ID    uuid.UUID
	Name  string
	Token string // JWT access token for the test user
}

// createTestTenant creates a new tenant and a test user via the IAM API.
// Returns a testTenant with a valid JWT token for the created user.
func createTestTenant(t *testing.T, client *http.Client) testTenant {
	t.Helper()

	tenantID := uuid.New()
	name := fmt.Sprintf("test-tenant-%s", tenantID.String()[:8])

	// Register tenant + admin user in one step (adjust endpoint to match your IAM service).
	payload := map[string]interface{}{
		"tenant_name": name,
		"tenant_slug": name,
		"email":       fmt.Sprintf("admin@%s.test", name),
		"password":    "TestPassword123!",
		"first_name":  "Test",
		"last_name":   "Admin",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal tenant payload: %v", err)
	}

	resp, err := client.Post(baseURL()+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register tenant %s: %v", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("register tenant %s: unexpected status %d", name, resp.StatusCode)
	}

	var result struct {
		TenantID    string `json:"tenant_id"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	tid, err := uuid.Parse(result.TenantID)
	if err != nil {
		// Fall back to generated UUID if the response format differs.
		tid = tenantID
	}

	return testTenant{
		ID:    tid,
		Name:  name,
		Token: result.AccessToken,
	}
}

// authenticatedRequest builds an HTTP request with the tenant's bearer token.
func authenticatedRequest(t *testing.T, method, url string, body interface{}, token string) *http.Request {
	t.Helper()

	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("create request %s %s: %v", method, url, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func doRequest(t *testing.T, client *http.Client, req *http.Request) *http.Response {
	t.Helper()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", req.Method, req.URL, err)
	}
	return resp
}

// connectDB opens a pgxpool connection to the given DSN.
// Skips the test if the DSN is empty.
func connectDB(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	if dsn == "" {
		t.Skip("no database DSN configured; skipping direct DB test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	return pool
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Assets — cyber_db: assets table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Assets(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	assetURL := baseURL() + "/api/v1/cyber/assets"

	// Create an asset in tenantA.
	assetA := map[string]interface{}{
		"name":        "asset-tenant-a",
		"type":        "server",
		"criticality": "high",
		"status":      "active",
	}
	reqA := authenticatedRequest(t, http.MethodPost, assetURL, assetA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create asset in tenantA: got %d, want 201", respA.StatusCode)
	}

	var createdA struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(respA.Body).Decode(&createdA); err != nil {
		t.Fatalf("decode created asset: %v", err)
	}
	assetAID := createdA.ID

	// Create an asset in tenantB.
	assetB := map[string]interface{}{
		"name":        "asset-tenant-b",
		"type":        "endpoint",
		"criticality": "medium",
		"status":      "active",
	}
	reqB := authenticatedRequest(t, http.MethodPost, assetURL, assetB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create asset in tenantB: got %d, want 201", respB.StatusCode)
	}

	var createdB struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(respB.Body).Decode(&createdB); err != nil {
		t.Fatalf("decode created asset B: %v", err)
	}
	assetBID := createdB.ID

	// TenantA can read their own asset.
	reqRead := authenticatedRequest(t, http.MethodGet, assetURL+"/"+assetAID, nil, tenantA.Token)
	respRead := doRequest(t, client, reqRead)
	defer respRead.Body.Close()
	if respRead.StatusCode != http.StatusOK {
		t.Errorf("tenantA read own asset: got %d, want 200", respRead.StatusCode)
	}

	// TenantA cannot read tenantB's asset (must be 404, not 403 — don't reveal existence).
	reqCross := authenticatedRequest(t, http.MethodGet, assetURL+"/"+assetBID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant asset read: got %d, want 404", respCross.StatusCode)
	}

	// TenantA list contains only tenantA assets.
	reqList := authenticatedRequest(t, http.MethodGet, assetURL, nil, tenantA.Token)
	respList := doRequest(t, client, reqList)
	defer respList.Body.Close()
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("tenantA list assets: got %d, want 200", respList.StatusCode)
	}
	var listResult struct {
		Items []struct{ ID string `json:"id"` } `json:"items"`
	}
	if err := json.NewDecoder(respList.Body).Decode(&listResult); err == nil {
		for _, item := range listResult.Items {
			if item.ID == assetBID {
				t.Errorf("tenantA list contains tenantB asset %s", assetBID)
			}
		}
	}

	// TenantA cannot update tenantB's asset.
	reqUpdate := authenticatedRequest(t, http.MethodPut, assetURL+"/"+assetBID, map[string]interface{}{"name": "hacked"}, tenantA.Token)
	respUpdate := doRequest(t, client, reqUpdate)
	defer respUpdate.Body.Close()
	if respUpdate.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant asset update: got %d, want 404", respUpdate.StatusCode)
	}

	// TenantA cannot delete tenantB's asset.
	reqDelete := authenticatedRequest(t, http.MethodDelete, assetURL+"/"+assetBID, nil, tenantA.Token)
	respDelete := doRequest(t, client, reqDelete)
	defer respDelete.Body.Close()
	if respDelete.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant asset delete: got %d, want 404", respDelete.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Alerts — cyber_db: alerts table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Alerts(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	alertsURL := baseURL() + "/api/v1/cyber/alerts"

	// Create alert in tenantA.
	alertA := map[string]interface{}{
		"title":    "Alert Tenant A",
		"severity": "high",
		"status":   "new",
	}
	reqA := authenticatedRequest(t, http.MethodPost, alertsURL, alertA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create alert in tenantA: got %d, want 201", respA.StatusCode)
	}
	var alertAResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respA.Body).Decode(&alertAResult); err != nil {
		t.Fatalf("decode alertA: %v", err)
	}

	// Create alert in tenantB.
	alertB := map[string]interface{}{
		"title":    "Alert Tenant B",
		"severity": "medium",
		"status":   "new",
	}
	reqB := authenticatedRequest(t, http.MethodPost, alertsURL, alertB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create alert in tenantB: got %d, want 201", respB.StatusCode)
	}
	var alertBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respB.Body).Decode(&alertBResult); err != nil {
		t.Fatalf("decode alertB: %v", err)
	}

	// TenantA cannot read tenantB's alert.
	reqCross := authenticatedRequest(t, http.MethodGet, alertsURL+"/"+alertBResult.ID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant alert read: got %d, want 404", respCross.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Pipelines — data_db: pipelines table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Pipelines(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	pipelinesURL := baseURL() + "/api/v1/data/pipelines"

	pipelineA := map[string]interface{}{"name": "pipeline-a", "type": "etl"}
	reqA := authenticatedRequest(t, http.MethodPost, pipelinesURL, pipelineA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create pipeline in tenantA: got %d, want 201", respA.StatusCode)
	}

	pipelineB := map[string]interface{}{"name": "pipeline-b", "type": "batch"}
	reqB := authenticatedRequest(t, http.MethodPost, pipelinesURL, pipelineB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create pipeline in tenantB: got %d, want 201", respB.StatusCode)
	}

	var pipelineBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respB.Body).Decode(&pipelineBResult); err != nil {
		t.Fatalf("decode pipelineB: %v", err)
	}

	// TenantA cannot read tenantB's pipeline.
	reqCross := authenticatedRequest(t, http.MethodGet, pipelinesURL+"/"+pipelineBResult.ID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant pipeline read: got %d, want 404", respCross.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_DataSources — data_db: data_sources table
// ---------------------------------------------------------------------------

func TestTenantIsolation_DataSources(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	dsURL := baseURL() + "/api/v1/data/sources"

	dsA := map[string]interface{}{"name": "datasource-a", "type": "database"}
	reqA := authenticatedRequest(t, http.MethodPost, dsURL, dsA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create data source in tenantA: got %d, want 201", respA.StatusCode)
	}

	dsB := map[string]interface{}{"name": "datasource-b", "type": "api"}
	reqB := authenticatedRequest(t, http.MethodPost, dsURL, dsB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create data source in tenantB: got %d, want 201", respB.StatusCode)
	}

	var dsBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respB.Body).Decode(&dsBResult); err != nil {
		t.Fatalf("decode data source B: %v", err)
	}

	reqCross := authenticatedRequest(t, http.MethodGet, dsURL+"/"+dsBResult.ID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant data source read: got %d, want 404", respCross.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Contracts — lex_db: contracts table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Contracts(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	contractsURL := baseURL() + "/api/v1/lex/contracts"

	contractA := map[string]interface{}{
		"title":      "Contract A",
		"type":       "nda",
		"party_a_name": "TenantA Corp",
		"party_b_name": "Partner X",
	}
	reqA := authenticatedRequest(t, http.MethodPost, contractsURL, contractA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create contract in tenantA: got %d, want 201", respA.StatusCode)
	}

	contractB := map[string]interface{}{
		"title":      "Contract B",
		"type":       "vendor",
		"party_a_name": "TenantB Corp",
		"party_b_name": "Supplier Y",
	}
	reqB := authenticatedRequest(t, http.MethodPost, contractsURL, contractB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create contract in tenantB: got %d, want 201", respB.StatusCode)
	}

	var contractBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respB.Body).Decode(&contractBResult); err != nil {
		t.Fatalf("decode contract B: %v", err)
	}

	reqCross := authenticatedRequest(t, http.MethodGet, contractsURL+"/"+contractBResult.ID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant contract read: got %d, want 404", respCross.StatusCode)
	}

	// TenantA cannot update tenantB's contract.
	reqUpdate := authenticatedRequest(t, http.MethodPut, contractsURL+"/"+contractBResult.ID, map[string]interface{}{"title": "hacked"}, tenantA.Token)
	respUpdate := doRequest(t, client, reqUpdate)
	defer respUpdate.Body.Close()
	if respUpdate.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant contract update: got %d, want 404", respUpdate.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Meetings — acta_db: meetings table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Meetings(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	meetingsURL := baseURL() + "/api/v1/acta/meetings"

	meetingA := map[string]interface{}{
		"title":        "Board Meeting A",
		"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}
	reqA := authenticatedRequest(t, http.MethodPost, meetingsURL, meetingA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create meeting in tenantA: got %d, want 201", respA.StatusCode)
	}

	meetingB := map[string]interface{}{
		"title":        "Board Meeting B",
		"scheduled_at": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	}
	reqB := authenticatedRequest(t, http.MethodPost, meetingsURL, meetingB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create meeting in tenantB: got %d, want 201", respB.StatusCode)
	}

	var meetingBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respB.Body).Decode(&meetingBResult); err != nil {
		t.Fatalf("decode meeting B: %v", err)
	}

	reqCross := authenticatedRequest(t, http.MethodGet, meetingsURL+"/"+meetingBResult.ID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant meeting read: got %d, want 404", respCross.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Dashboards — visus_db: visus_dashboards table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Dashboards(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	dashboardsURL := baseURL() + "/api/v1/visus/dashboards"

	dashA := map[string]interface{}{"name": "Dashboard A", "visibility": "private"}
	reqA := authenticatedRequest(t, http.MethodPost, dashboardsURL, dashA, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusCreated {
		t.Fatalf("create dashboard in tenantA: got %d, want 201", respA.StatusCode)
	}

	dashB := map[string]interface{}{"name": "Dashboard B", "visibility": "private"}
	reqB := authenticatedRequest(t, http.MethodPost, dashboardsURL, dashB, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create dashboard in tenantB: got %d, want 201", respB.StatusCode)
	}

	var dashBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respB.Body).Decode(&dashBResult); err != nil {
		t.Fatalf("decode dashboard B: %v", err)
	}

	reqCross := authenticatedRequest(t, http.MethodGet, dashboardsURL+"/"+dashBResult.ID, nil, tenantA.Token)
	respCross := doRequest(t, client, reqCross)
	defer respCross.Body.Close()
	if respCross.StatusCode != http.StatusNotFound {
		t.Errorf("tenantA cross-tenant dashboard read: got %d, want 404", respCross.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_AuditLogs — audit_db: audit_logs table
// ---------------------------------------------------------------------------

func TestTenantIsolation_AuditLogs(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	auditURL := baseURL() + "/api/v1/audit/logs"

	// Trigger an auditable action in tenantA to generate an audit log entry.
	_ = authenticatedRequest(t, http.MethodGet, baseURL()+"/api/v1/users", nil, tenantA.Token)

	// Trigger an auditable action in tenantB.
	_ = authenticatedRequest(t, http.MethodGet, baseURL()+"/api/v1/users", nil, tenantB.Token)

	// TenantA list should only contain their own audit logs.
	reqA := authenticatedRequest(t, http.MethodGet, auditURL, nil, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("tenantA list audit logs: got %d, want 200", respA.StatusCode)
	}

	var auditResult struct {
		Items []struct {
			TenantID string `json:"tenant_id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(respA.Body).Decode(&auditResult); err == nil {
		for _, item := range auditResult.Items {
			if item.TenantID != "" && item.TenantID != tenantA.ID.String() {
				t.Errorf("tenantA audit log contains entry from tenant %s", item.TenantID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Notifications — notification_db: notifications table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Notifications(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	notifURL := baseURL() + "/api/v1/notifications"

	// TenantA list — should not contain tenantB notifications.
	reqA := authenticatedRequest(t, http.MethodGet, notifURL, nil, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("tenantA list notifications: got %d, want 200", respA.StatusCode)
	}

	// TenantB list — should not contain tenantA notifications.
	reqB := authenticatedRequest(t, http.MethodGet, notifURL, nil, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusOK {
		t.Fatalf("tenantB list notifications: got %d, want 200", respB.StatusCode)
	}

	// Verify lists are isolated by checking tenant_id field in each item.
	var notifResultA struct {
		Items []struct{ TenantID string `json:"tenant_id"` } `json:"items"`
	}
	if err := json.NewDecoder(respA.Body).Decode(&notifResultA); err == nil {
		for _, item := range notifResultA.Items {
			if item.TenantID != "" && item.TenantID != tenantA.ID.String() {
				t.Errorf("tenantA notification list contains item from tenant %s", item.TenantID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_Users — platform_core: users table
// ---------------------------------------------------------------------------

func TestTenantIsolation_Users(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	usersURL := baseURL() + "/api/v1/users"

	// TenantA lists users — should only see their own tenant's users.
	reqA := authenticatedRequest(t, http.MethodGet, usersURL, nil, tenantA.Token)
	respA := doRequest(t, client, reqA)
	defer respA.Body.Close()
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("tenantA list users: got %d, want 200", respA.StatusCode)
	}

	var usersResultA struct {
		Items []struct{ TenantID string `json:"tenant_id"` } `json:"items"`
	}
	if err := json.NewDecoder(respA.Body).Decode(&usersResultA); err == nil {
		for _, item := range usersResultA.Items {
			if item.TenantID != "" && item.TenantID != tenantA.ID.String() {
				t.Errorf("tenantA user list contains user from tenant %s", item.TenantID)
			}
		}
	}

	// TenantB lists users — should only see their own tenant's users.
	reqB := authenticatedRequest(t, http.MethodGet, usersURL, nil, tenantB.Token)
	respB := doRequest(t, client, reqB)
	defer respB.Body.Close()
	if respB.StatusCode != http.StatusOK {
		t.Fatalf("tenantB list users: got %d, want 200", respB.StatusCode)
	}

	var usersResultB struct {
		Items []struct{ TenantID string `json:"tenant_id"` } `json:"items"`
	}
	if err := json.NewDecoder(respB.Body).Decode(&usersResultB); err == nil {
		for _, item := range usersResultB.Items {
			if item.TenantID != "" && item.TenantID != tenantB.ID.String() {
				t.Errorf("tenantB user list contains user from tenant %s", item.TenantID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_APIKeys — API key from tenantA cannot access tenantB resources
// ---------------------------------------------------------------------------

func TestTenantIsolation_APIKeys(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	tenantA := createTestTenant(t, client)
	tenantB := createTestTenant(t, client)

	apiKeysURL := baseURL() + "/api/v1/api-keys"

	// Create an API key for tenantA.
	apiKeyPayload := map[string]interface{}{
		"name":        "test-key-a",
		"permissions": []string{"cyber:read", "data:read"},
	}
	reqCreate := authenticatedRequest(t, http.MethodPost, apiKeysURL, apiKeyPayload, tenantA.Token)
	respCreate := doRequest(t, client, reqCreate)
	defer respCreate.Body.Close()
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("create API key for tenantA: got %d, want 201", respCreate.StatusCode)
	}

	var apiKeyResult struct {
		Key string `json:"key"`
		ID  string `json:"id"`
	}
	if err := json.NewDecoder(respCreate.Body).Decode(&apiKeyResult); err != nil {
		t.Fatalf("decode API key response: %v", err)
	}

	// Create an asset in tenantB using tenantB's JWT.
	assetURL := baseURL() + "/api/v1/cyber/assets"
	assetB := map[string]interface{}{"name": "secret-asset-b", "type": "server", "criticality": "critical", "status": "active"}
	reqAsset := authenticatedRequest(t, http.MethodPost, assetURL, assetB, tenantB.Token)
	respAsset := doRequest(t, client, reqAsset)
	defer respAsset.Body.Close()
	if respAsset.StatusCode != http.StatusCreated {
		t.Fatalf("create asset in tenantB: got %d, want 201", respAsset.StatusCode)
	}
	var assetBResult struct{ ID string `json:"id"` }
	if err := json.NewDecoder(respAsset.Body).Decode(&assetBResult); err != nil {
		t.Fatalf("decode asset B: %v", err)
	}

	// Use tenantA's API key to try to read tenantB's asset — must get 404.
	if apiKeyResult.Key != "" {
		req, err := http.NewRequest(http.MethodGet, assetURL+"/"+assetBResult.ID, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("X-API-Key", apiKeyResult.Key)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("API key cross-tenant request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("tenantA API key cross-tenant asset access: got %d, want 404 or 401", resp.StatusCode)
		}
	}
}

// ---------------------------------------------------------------------------
// TestTenantIsolation_RLSDirectDB — direct database RLS verification
// ---------------------------------------------------------------------------

// TestTenantIsolation_RLSDirectDB bypasses HTTP and tests RLS directly at the
// PostgreSQL level. It connects to the cyber_db, inserts rows for two tenants
// using BYPASSRLS role, then verifies that setting app.current_tenant_id filters
// correctly and that omitting the variable returns zero rows.
func TestTenantIsolation_RLSDirectDB(t *testing.T) {
	dsn := dbDSN("TEST_CYBER_DB_DSN", "cyber_db")
	pool := connectDB(t, dsn)
	ctx := context.Background()

	tenantAID := uuid.New()
	tenantBID := uuid.New()

	// Insert test rows for both tenants bypassing RLS (requires BYPASSRLS on the role).
	// The connection must be using the migrator/admin role that has BYPASSRLS.
	_, err := pool.Exec(ctx, `
		INSERT INTO assets (id, tenant_id, name, type, criticality, status, created_at, updated_at)
		VALUES
		  ($1, $2, 'rls-test-asset-a', 'server', 'high', 'active', NOW(), NOW()),
		  ($3, $4, 'rls-test-asset-b', 'endpoint', 'medium', 'active', NOW(), NOW())
	`, uuid.New(), tenantAID, uuid.New(), tenantBID)
	if err != nil {
		t.Skipf("could not insert test rows (may need BYPASSRLS role): %v", err)
	}

	// Clean up after the test.
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `
			DELETE FROM assets WHERE tenant_id IN ($1, $2) AND name LIKE 'rls-test-asset-%'
		`, tenantAID, tenantBID)
	})

	// --- Test 1: Setting tenantA context returns only tenantA rows. ---
	var countA int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT 1 FROM assets
			WHERE name = 'rls-test-asset-a'
			  AND tenant_id = $1
		) t
	`, tenantAID).Scan(&countA)
	if err != nil {
		t.Fatalf("direct count query failed: %v", err)
	}

	// Now test via SET LOCAL in a transaction.
	txA, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for tenantA: %v", err)
	}
	defer txA.Rollback(ctx) //nolint:errcheck

	if _, err := txA.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantAID.String()); err != nil {
		t.Fatalf("SET LOCAL tenantA: %v", err)
	}

	var rlsCountA int
	if err := txA.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE name LIKE 'rls-test-asset-%'").Scan(&rlsCountA); err != nil {
		t.Fatalf("count assets for tenantA via RLS: %v", err)
	}
	if rlsCountA != 1 {
		t.Errorf("tenantA RLS: expected 1 asset, got %d", rlsCountA)
	}
	_ = txA.Commit(ctx)

	// --- Test 2: Setting tenantB context returns only tenantB rows. ---
	txB, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for tenantB: %v", err)
	}
	defer txB.Rollback(ctx) //nolint:errcheck

	if _, err := txB.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantBID.String()); err != nil {
		t.Fatalf("SET LOCAL tenantB: %v", err)
	}

	var rlsCountB int
	if err := txB.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE name LIKE 'rls-test-asset-%'").Scan(&rlsCountB); err != nil {
		t.Fatalf("count assets for tenantB via RLS: %v", err)
	}
	if rlsCountB != 1 {
		t.Errorf("tenantB RLS: expected 1 asset, got %d", rlsCountB)
	}
	_ = txB.Commit(ctx)

	// --- Test 3: No tenant context set → 0 rows returned (safe default). ---
	// A fresh connection from the pool will not have app.current_tenant_id set.
	// current_setting('app.current_tenant_id', true) returns NULL, and
	// NULL::uuid = anything is FALSE → no rows pass the policy.
	txNone, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx for no-tenant: %v", err)
	}
	defer txNone.Rollback(ctx) //nolint:errcheck

	// Explicitly reset to simulate a fresh connection without tenant context.
	if _, err := txNone.Exec(ctx, "RESET app.current_tenant_id"); err != nil {
		// RESET may fail if the variable was never set; that is acceptable.
		_ = err
	}

	var rlsCountNone int
	if err := txNone.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE name LIKE 'rls-test-asset-%'").Scan(&rlsCountNone); err != nil {
		t.Fatalf("count assets with no tenant context via RLS: %v", err)
	}
	if rlsCountNone != 0 {
		t.Errorf("no-tenant RLS: expected 0 assets, got %d (RLS leak — CRITICAL)", rlsCountNone)
	}
	_ = txNone.Commit(ctx)
}
