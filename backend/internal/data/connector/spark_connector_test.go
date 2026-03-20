package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

func TestSparkRESTApplicationParsing(t *testing.T) {
	server := newSparkMockServer()
	defer server.Close()

	connector := mustNewSparkConnector(t, model.SparkConnectionConfig{
		REST: model.SparkRESTConfig{MasterURL: server.URL},
	})

	apps, err := connector.GetActiveApplications(context.Background())
	if err != nil {
		t.Fatalf("GetActiveApplications() error = %v", err)
	}
	if len(apps) != 1 || apps[0].ID != "app-1" || apps[0].User != "analyst" {
		t.Fatalf("GetActiveApplications() = %+v", apps)
	}
}

func TestSparkAccessLogMapping(t *testing.T) {
	server := newSparkMockServer()
	defer server.Close()

	connector := mustNewSparkConnector(t, model.SparkConnectionConfig{
		REST: model.SparkRESTConfig{
			MasterURL:  server.URL,
			HistoryURL: server.URL,
		},
	})

	events, err := connector.QueryAccessLogs(context.Background(), time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Action != "spark_job" || events[0].User != "analyst" || events[0].RowsRead != 1024 {
		t.Fatalf("event = %+v", events[0])
	}
}

func mustNewSparkConnector(t *testing.T, cfg model.SparkConnectionConfig) *SparkConnector {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	conn, err := NewSparkConnector(raw, FactoryOptions{
		Limits: ConnectorLimits{
			MaxPoolSize:      4,
			StatementTimeout: 30 * time.Second,
			ConnectTimeout:   5 * time.Second,
		},
		Logger: zerolog.Nop(),
	})
	if err != nil {
		t.Fatalf("NewSparkConnector() error = %v", err)
	}
	value, ok := conn.(*SparkConnector)
	if !ok {
		t.Fatalf("connector type = %T, want *SparkConnector", conn)
	}
	return value
}

func newSparkMockServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":        "app-1",
				"name":      "daily-sales-job",
				"sparkUser": "analyst",
				"attempts": []map[string]any{
					{
						"startTime": "2026-03-08T10:00:00.000GMT",
						"endTime":   "2026-03-08T10:05:00.000GMT",
						"completed": true,
					},
				},
			},
		})
	})
	handler.HandleFunc("/api/v1/applications/app-1/jobs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"jobId": 7, "name": "daily-sales-job", "status": "SUCCEEDED", "numTasks": 4, "numCompletedTasks": 4},
		})
	})
	handler.HandleFunc("/api/v1/applications/app-1/stages", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"inputBytes": 4096, "inputRecords": 1024},
		})
	})
	handler.HandleFunc("/api/v1/applications/app-1/executors", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "1", "hostPort": "executor:1234", "totalTasks": 4, "totalDuration": 3000},
		})
	})
	return httptest.NewServer(handler)
}
