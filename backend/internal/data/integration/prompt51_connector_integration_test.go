//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/connector/testhelpers"
	datadto "github.com/clario360/platform/internal/data/dto"
	datamodel "github.com/clario360/platform/internal/data/model"
)

func TestPrompt51_ClickHouseEndToEnd(t *testing.T) {
	h := newIntegrationHarness(t)

	ctx, cancel := context.WithTimeout(h.ctx, 3*time.Minute)
	defer cancel()

	container := testhelpers.StartClickHouseContainer(ctx, t, "analytics")
	db, err := container.OpenDB(ctx, "")
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	mustExecClickHouseSQL(t, ctx, db, `CREATE TABLE analytics.customers (
		id UInt64,
		user_email String,
		region String
	) ENGINE = MergeTree ORDER BY id`)
	mustExecClickHouseSQL(t, ctx, db, `INSERT INTO analytics.customers (id, user_email, region) VALUES
		(1, 'alice@example.com', 'EMEA'),
		(2, 'bob@example.com', 'NA')`)

	sourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "clickhouse-analytics",
		Description: "clickhouse integration source",
		Type:        string(datamodel.DataSourceTypeClickHouse),
		ConnectionConfig: mustRawJSON(t, container.NativeConfig("")),
	})

	testResult := h.testSource(t, sourceID)
	if !testResult.Success {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema := h.discoverSource(t, sourceID)
	customers := schemaTableByName(t, schema, "customers")
	if !customers.ContainsPII {
		t.Fatalf("discovered clickhouse customers missing pii metadata: %+v", customers)
	}

	pipelineID := createPipeline(t, h, datadto.CreatePipelineRequest{
		Name:        "clickhouse-customer-sync",
		Description: "extract customers from clickhouse",
		Type:        string(datamodel.PipelineTypeETL),
		SourceID:    sourceID,
		Config: mustRawJSON(t, map[string]any{
			"source_table": "customers",
			"batch_size":   1000,
			"quality_gates": []map[string]any{
				{
					"name":      "minimum rows",
					"metric":    "min_row_count",
					"operator":  "gte",
					"threshold": 1,
					"severity":  "warning",
				},
			},
		}),
	})

	run := runPipeline(t, h, pipelineID)
	if run.Status != datamodel.PipelineRunStatusCompleted {
		t.Fatalf("pipeline run status = %s, want completed: %+v", run.Status, run)
	}
	if run.RecordsExtracted != 2 || run.RecordsLoaded != 2 {
		t.Fatalf("pipeline run counts = %+v, want 2 extracted/loaded", run)
	}
	if run.QualityGatesPassed != 1 {
		t.Fatalf("pipeline run quality gates = %+v, want 1 passed gate", run)
	}
}

func TestPrompt51_DoltEndToEnd(t *testing.T) {
	h := newIntegrationHarness(t)

	ctx, cancel := context.WithTimeout(h.ctx, 3*time.Minute)
	defer cancel()

	container := testhelpers.StartDoltContainer(ctx, t, "app")
	db, err := container.OpenUserDB(ctx)
	if err != nil {
		t.Fatalf("OpenUserDB() error = %v", err)
	}
	defer db.Close()

	mustExecSQL(t, ctx, db, `CREATE TABLE customers (id INT PRIMARY KEY, user_email VARCHAR(255), region VARCHAR(50))`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (1, 'alice@example.com', 'EMEA')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'main baseline')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_BRANCH('staging')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_CHECKOUT('staging')`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (2, 'bob@example.com', 'NA')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'staging change')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_CHECKOUT('main')`)

	sourceID := h.createSource(t, datadto.CreateSourceRequest{
		Name:        "dolt-versioned-source",
		Description: "dolt integration source",
		Type:        string(datamodel.DataSourceTypeDolt),
		ConnectionConfig: mustRawJSON(t, container.ConnectionConfig("staging")),
	})

	testResult := h.testSource(t, sourceID)
	if !testResult.Success {
		t.Fatalf("TestConnection() = %+v", testResult)
	}
	if !containsString(testResult.Permissions, "main") || !containsString(testResult.Permissions, "staging") {
		t.Fatalf("branch permissions = %+v, want main and staging", testResult.Permissions)
	}

	schema := h.discoverSource(t, sourceID)
	customers := schemaTableByName(t, schema, "customers")
	if customers.EstimatedRows < 2 {
		t.Fatalf("staging discovery should expose 2 rows, got %+v", customers)
	}

	source := getSource(t, h, sourceID)
	if !strings.Contains(string(source.ConnectionConfig), "\"branch\":\"staging\"") {
		t.Fatalf("sanitized source config missing branch metadata: %s", string(source.ConnectionConfig))
	}

	pipelineID := createPipeline(t, h, datadto.CreatePipelineRequest{
		Name:        "dolt-staging-sync",
		Description: "extract customers from dolt staging",
		Type:        string(datamodel.PipelineTypeETL),
		SourceID:    sourceID,
		Config: mustRawJSON(t, map[string]any{
			"source_table": "customers",
			"batch_size":   1000,
			"quality_gates": []map[string]any{
				{
					"name":      "minimum rows",
					"metric":    "min_row_count",
					"operator":  "gte",
					"threshold": 2,
					"severity":  "warning",
				},
			},
		}),
	})

	run := runPipeline(t, h, pipelineID)
	if run.Status != datamodel.PipelineRunStatusCompleted {
		t.Fatalf("pipeline run status = %s, want completed: %+v", run.Status, run)
	}
	if run.RecordsExtracted != 2 || run.RecordsLoaded != 2 {
		t.Fatalf("pipeline run counts = %+v, want 2 extracted/loaded", run)
	}
}

func createPipeline(t *testing.T, h *integrationHarness, req datadto.CreatePipelineRequest) uuid.UUID {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, "/api/v1/data/pipelines", req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create pipeline status = %d, want %d, body=%s", resp.StatusCode, http.StatusCreated, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.Pipeline]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data.ID
}

func runPipeline(t *testing.T, h *integrationHarness, pipelineID uuid.UUID) datamodel.PipelineRun {
	t.Helper()
	resp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/data/pipelines/%s/run", pipelineID), map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("run pipeline status = %d, want %d, body=%s", resp.StatusCode, http.StatusAccepted, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.PipelineRun]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func getSource(t *testing.T, h *integrationHarness, sourceID uuid.UUID) datamodel.DataSource {
	t.Helper()
	resp := h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/data/sources/%s", sourceID), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get source status = %d, want %d, body=%s", resp.StatusCode, http.StatusOK, readBody(t, resp))
	}
	var envelope dataEnvelope[datamodel.DataSource]
	decodeBody(t, resp.Body, &envelope)
	return envelope.Data
}

func schemaTableByName(t *testing.T, schema datamodel.DiscoveredSchema, tableName string) datamodel.DiscoveredTable {
	t.Helper()
	for _, table := range schema.Tables {
		if strings.EqualFold(table.Name, tableName) {
			return table
		}
	}
	payload, _ := json.Marshal(schema.Tables)
	t.Fatalf("table %q not found in schema: %s", tableName, string(payload))
	return datamodel.DiscoveredTable{}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func mustExecClickHouseSQL(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("clickhouse ExecContext(%q) error = %v", query, err)
	}
}

func mustExecSQL(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("ExecContext(%q) error = %v", query, err)
	}
}

func mustCallDoltProcedure(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		t.Fatalf("QueryContext(%q) error = %v", query, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatalf("rows.Columns(%q) error = %v", query, err)
	}
	values := make([]any, len(columns))
	targets := make([]any, len(columns))
	for i := range values {
		targets[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			t.Fatalf("rows.Scan(%q) error = %v", query, err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(%q) error = %v", query, err)
	}
}
