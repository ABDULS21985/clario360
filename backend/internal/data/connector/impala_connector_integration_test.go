//go:build integration

package connector

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestImpala_FullLifecycle(t *testing.T) {
	t.Parallel()

	server := startMockWarehouseThriftServer(t)
	host, port := server.HostPort()

	conn, err := NewImpalaConnector(mustRawJSON(t, map[string]any{
		"host":                  host,
		"port":                  port,
		"database":              "default",
		"auth_type":             "noauth",
		"query_timeout_seconds": 30,
		"audit_log_table":       "sys.impala_audit",
	}), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewImpalaConnector() error = %v", err)
	}
	impalaConn := conn.(*ImpalaConnector)
	sourceID, tenantID := integrationSourceContext()
	impalaConn.SetSourceContext(sourceID, tenantID)
	defer impalaConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testResult, err := impalaConn.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if !testResult.Success || !strings.Contains(testResult.Message, "Connected to Impala") {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema, err := impalaConn.DiscoverSchema(ctx, DiscoveryOptions{
		MaxTables:    10,
		MaxColumns:   20,
		SampleValues: true,
		MaxSamples:   2,
	})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	customers := requireDiscoveredTable(t, schema, "customers")
	if customers.Comment == "" || !strings.HasPrefix(customers.Comment, "hdfs://") {
		t.Fatalf("customers location metadata missing: %+v", customers)
	}

	batch, err := impalaConn.FetchData(ctx, "default.customers", FetchParams{
		OrderBy:   "id",
		BatchSize: 10,
	})
	if err != nil {
		t.Fatalf("FetchData() error = %v", err)
	}
	if batch.RowCount != 2 || fmt.Sprint(batch.Rows[1]["region"]) != "NA" {
		t.Fatalf("FetchData() = %+v", batch)
	}

	events, err := impalaConn.QueryAccessLogs(ctx, time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("QueryAccessLogs() = %+v, want 1 event", events)
	}
	if events[0].Action != "query" || events[0].Table != "customers" || events[0].SourceID != sourceID || events[0].TenantID != tenantID {
		t.Fatalf("first access event = %+v", events[0])
	}

	locations, err := impalaConn.ListDataLocations(ctx)
	if err != nil {
		t.Fatalf("ListDataLocations() error = %v", err)
	}
	if len(locations) == 0 || !strings.HasPrefix(locations[0].Location, "hdfs://") {
		t.Fatalf("ListDataLocations() = %+v", locations)
	}
}
