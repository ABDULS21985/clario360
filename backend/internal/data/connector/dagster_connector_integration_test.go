//go:build integration

package connector

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/clario360/platform/internal/data/connector/testhelpers"
)

func TestDagster_FullLifecycle(t *testing.T) {
	t.Parallel()

	server := testhelpers.NewDagsterMockServer(testhelpers.DefaultDagsterMockConfig())
	defer server.Close()

	conn, err := NewDagsterConnector(mustRawJSON(t, map[string]any{
		"graphql_url": server.URL + "/graphql",
	}), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewDagsterConnector() error = %v", err)
	}
	dagsterConn := conn.(*DagsterConnector)
	sourceID, tenantID := integrationSourceContext()
	dagsterConn.SetSourceContext(sourceID, tenantID)
	defer dagsterConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testResult, err := dagsterConn.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if !testResult.Success || testResult.Version != "1.6.0" {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema, err := dagsterConn.DiscoverSchema(ctx, DiscoveryOptions{MaxTables: 20})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	asset := requireDiscoveredTable(t, schema, "warehouse.customers")
	if asset.Type != "asset" {
		t.Fatalf("dagster asset type = %q, want asset", asset.Type)
	}
	if !strings.Contains(asset.Comment, "compute kind: sql") {
		t.Fatalf("dagster asset comment = %q, want compute kind metadata", asset.Comment)
	}

	events, err := dagsterConn.QueryAccessLogs(ctx, time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) != 1 || events[0].Action != "pipeline_run" || events[0].SourceID != sourceID {
		t.Fatalf("QueryAccessLogs() = %+v", events)
	}

	edges, err := dagsterConn.GetAssetLineage(ctx)
	if err != nil {
		t.Fatalf("GetAssetLineage() error = %v", err)
	}
	if len(edges) != 1 || edges[0].SourceAsset != "raw.customers" || edges[0].TargetAsset != "warehouse.customers" {
		t.Fatalf("GetAssetLineage() = %+v", edges)
	}

	if err := dagsterConn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
