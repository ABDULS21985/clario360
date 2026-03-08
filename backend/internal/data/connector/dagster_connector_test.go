package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestDagsterVersionQuery(t *testing.T) {
	server := newDagsterMockServer()
	defer server.Close()
	connector := mustNewDagsterConnector(t, server.URL+"/graphql")

	result, err := connector.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if result.Version != "1.6.0" {
		t.Fatalf("Version = %q, want 1.6.0", result.Version)
	}
}

func TestDagsterAssetDiscovery(t *testing.T) {
	server := newDagsterMockServer()
	defer server.Close()
	connector := mustNewDagsterConnector(t, server.URL+"/graphql")

	schema, err := connector.DiscoverSchema(context.Background(), DiscoveryOptions{})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	if len(schema.Tables) != 1 || schema.Tables[0].Name != "warehouse.customers" {
		t.Fatalf("DiscoverSchema() = %+v", schema.Tables)
	}
}

func TestDagsterRunHistory(t *testing.T) {
	server := newDagsterMockServer()
	defer server.Close()
	connector := mustNewDagsterConnector(t, server.URL+"/graphql")

	events, err := connector.QueryAccessLogs(context.Background(), time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) != 1 || events[0].Action != "pipeline_run" || events[0].Database != "nightly_ingest" {
		t.Fatalf("QueryAccessLogs() = %+v", events)
	}
}

func TestDagsterLineageExtraction(t *testing.T) {
	server := newDagsterMockServer()
	defer server.Close()
	connector := mustNewDagsterConnector(t, server.URL+"/graphql")

	edges, err := connector.GetAssetLineage(context.Background())
	if err != nil {
		t.Fatalf("GetAssetLineage() error = %v", err)
	}
	if len(edges) != 1 || edges[0].SourceAsset != "raw.customers" || edges[0].TargetAsset != "warehouse.customers" {
		t.Fatalf("GetAssetLineage() = %+v", edges)
	}
}

func TestDagsterFetchDataReturnsUnsupported(t *testing.T) {
	server := newDagsterMockServer()
	defer server.Close()
	connector := mustNewDagsterConnector(t, server.URL+"/graphql")

	_, err := connector.FetchData(context.Background(), "warehouse.customers", FetchParams{})
	if err == nil {
		t.Fatal("FetchData() expected error")
	}
	var connErr *ConnectorError
	if !AsConnectorError(err, &connErr) || connErr.Code != ErrorCodeUnsupportedOperation {
		t.Fatalf("FetchData() error = %v, want unsupported connector error", err)
	}
}

func mustNewDagsterConnector(t *testing.T, graphqlURL string) *DagsterConnector {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"graphql_url": graphqlURL})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	conn, err := NewDagsterConnector(raw, FactoryOptions{
		Limits: ConnectorLimits{StatementTimeout: 30 * time.Second},
		Logger: zerolog.Nop(),
	})
	if err != nil {
		t.Fatalf("NewDagsterConnector() error = %v", err)
	}
	value, ok := conn.(*DagsterConnector)
	if !ok {
		t.Fatalf("connector type = %T, want *DagsterConnector", conn)
	}
	return value
}

func newDagsterMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(payload.Query, "query { version }"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"version": "1.6.0"}})
		case strings.Contains(payload.Query, "repositoriesOrError"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"repositoriesOrError": map[string]any{
					"nodes": []map[string]any{{"name": "analytics"}},
				},
			}})
		case strings.Contains(payload.Query, "assetsOrError"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"assetsOrError": map[string]any{
					"nodes": []map[string]any{
						{
							"key":           map[string]any{"path": []string{"warehouse", "customers"}},
							"description":   "Curated customer asset",
							"graphName":     "nightly_ingest",
							"computeKind":   "sql",
							"isPartitioned": false,
							"metadataEntries": []map[string]any{
								{"label": "owner", "__typename": "TextMetadataEntry", "text": "data-platform"},
							},
							"dependencyKeys": []map[string]any{{"path": []string{"raw", "customers"}}},
						},
					},
				},
			}})
		case strings.Contains(payload.Query, "runsOrError"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"runsOrError": map[string]any{
					"results": []map[string]any{
						{
							"runId":        "run-1",
							"pipelineName": "nightly_ingest",
							"status":       "SUCCESS",
							"startTime":    1741428000,
							"endTime":      1741428300,
							"tags": []map[string]any{
								{"key": "dagster/user", "value": "data-engineer"},
							},
							"runConfigYaml": "asset: warehouse.customers",
						},
					},
				},
			}})
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
		}
	}))
}
