//go:build integration

package connector

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestHive_FullLifecycle(t *testing.T) {
	t.Parallel()

	server := startMockWarehouseThriftServer(t)
	host, port := server.HostPort()

	conn, err := NewHiveConnector(mustRawJSON(t, map[string]any{
		"host":           host,
		"port":           port,
		"database":       "default",
		"auth_type":      "noauth",
		"transport_mode": "binary",
		"query_timeout_seconds": 30,
		"fetch_size":     100,
	}), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewHiveConnector() error = %v", err)
	}
	hiveConn := conn.(*HiveConnector)
	sourceID, tenantID := integrationSourceContext()
	hiveConn.SetSourceContext(sourceID, tenantID)
	defer hiveConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testResult, err := hiveConn.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if !testResult.Success || testResult.Version != "HiveServer2" {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema, err := hiveConn.DiscoverSchema(ctx, DiscoveryOptions{
		MaxTables:    10,
		MaxColumns:   20,
		SampleValues: true,
		MaxSamples:   2,
	})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	customers := requireDiscoveredTable(t, schema, "customers")
	if customers.Type != "parquet" {
		t.Fatalf("customers type = %q, want parquet", customers.Type)
	}
	if !customers.ContainsPII {
		t.Fatalf("customers missing pii classification: %+v", customers)
	}

	batch, err := hiveConn.FetchData(ctx, "customers", FetchParams{
		OrderBy:   "id",
		BatchSize: 10,
	})
	if err != nil {
		t.Fatalf("FetchData() error = %v", err)
	}
	if batch.RowCount != 2 || !strings.EqualFold(fmt.Sprint(batch.Rows[0]["user_email"]), "alice@example.com") {
		t.Fatalf("FetchData() = %+v", batch)
	}

	estimate, err := hiveConn.EstimateSize(ctx)
	if err != nil {
		t.Fatalf("EstimateSize() error = %v", err)
	}
	if estimate.TableCount == 0 || estimate.TotalRows < 2 {
		t.Fatalf("EstimateSize() = %+v", estimate)
	}

	locations, err := hiveConn.ListDataLocations(ctx)
	if err != nil {
		t.Fatalf("ListDataLocations() error = %v", err)
	}
	if len(locations) == 0 || !strings.HasPrefix(locations[0].Location, "hdfs://") {
		t.Fatalf("ListDataLocations() = %+v", locations)
	}

	events, err := hiveConn.QueryAccessLogs(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("QueryAccessLogs() = %+v, want no events without configured audit table", events)
	}
}
