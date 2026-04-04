//go:build integration

package connector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/clario360/platform/internal/data/connector/testhelpers"
)

func TestSpark_ThriftAndRESTLifecycle(t *testing.T) {
	t.Parallel()

	thriftServer := startMockWarehouseThriftServer(t)
	restServer := testhelpers.NewSparkRESTServer(testhelpers.DefaultSparkRESTFixture())
	defer restServer.Close()

	host, port := thriftServer.HostPort()
	conn, err := NewSparkConnector(mustRawJSON(t, map[string]any{
		"thrift": map[string]any{
			"host":      host,
			"port":      port,
			"database":  "default",
			"auth_type": "noauth",
		},
		"rest": map[string]any{
			"master_url":  restServer.URL,
			"history_url": restServer.URL,
		},
		"query_timeout_seconds": 30,
	}), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewSparkConnector() error = %v", err)
	}
	sparkConn := conn.(*SparkConnector)
	sourceID, tenantID := integrationSourceContext()
	sparkConn.SetSourceContext(sourceID, tenantID)
	defer sparkConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testResult, err := sparkConn.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if !testResult.Success {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema, err := sparkConn.DiscoverSchema(ctx, DiscoveryOptions{
		MaxTables:    10,
		MaxColumns:   20,
		SampleValues: true,
		MaxSamples:   2,
	})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	customers := requireDiscoveredTable(t, schema, "customers")
	if customers.EstimatedRows != 2 {
		t.Fatalf("customers estimated rows = %d, want 2", customers.EstimatedRows)
	}

	batch, err := sparkConn.FetchData(ctx, "default.customers", FetchParams{
		OrderBy:   "id",
		BatchSize: 10,
	})
	if err != nil {
		t.Fatalf("FetchData() error = %v", err)
	}
	if batch.RowCount != 2 || fmt.Sprint(batch.Rows[0]["region"]) != "EMEA" {
		t.Fatalf("FetchData() = %+v", batch)
	}

	apps, err := sparkConn.GetActiveApplications(ctx)
	if err != nil {
		t.Fatalf("GetActiveApplications() error = %v", err)
	}
	if len(apps) != 1 || apps[0].ID != "app-1" {
		t.Fatalf("GetActiveApplications() = %+v", apps)
	}

	detail, err := sparkConn.GetApplicationDetail(ctx, "app-1")
	if err != nil {
		t.Fatalf("GetApplicationDetail() error = %v", err)
	}
	if detail.StageMetrics.InputRecords != 1024 {
		t.Fatalf("GetApplicationDetail() = %+v", detail)
	}

	events, err := sparkConn.QueryAccessLogs(ctx, time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) != 1 || events[0].Action != "spark_job" || events[0].SourceID != sourceID || events[0].TenantID != tenantID {
		t.Fatalf("QueryAccessLogs() = %+v", events)
	}
}
