// +build integration

package bootstrap_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/health"
	"github.com/clario360/platform/internal/observability/tracing"
)

func TestBootstrap_NoDB(t *testing.T) {
	ctx := context.Background()

	cfg := &bootstrap.ServiceConfig{
		Name:        "test-service",
		Version:     "0.1.0",
		Environment: "development",
		Port:        0,
		AdminPort:   0,
		LogLevel:    "debug",
		Tracing: tracing.TracerConfig{
			Enabled: false,
		},
		ShutdownTimeout: 5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
	}

	svc, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// DB should be nil when not configured.
	if svc.DB != nil {
		t.Error("expected DB to be nil when no DB config provided")
	}
	if svc.DBPool != nil {
		t.Error("expected DBPool to be nil when no DB config provided")
	}
	if svc.Redis != nil {
		t.Error("expected Redis to be nil when no Redis config provided")
	}

	// Logger, Metrics, Router, AdminRouter should all be non-nil.
	if svc.Metrics == nil {
		t.Error("expected Metrics to be non-nil")
	}
	if svc.Router == nil {
		t.Error("expected Router to be non-nil")
	}
	if svc.AdminRouter == nil {
		t.Error("expected AdminRouter to be non-nil")
	}
	if svc.Health == nil {
		t.Error("expected Health to be non-nil")
	}
}

func TestBootstrap_HealthEndpoints_NoDB(t *testing.T) {
	ctx := context.Background()

	cfg := &bootstrap.ServiceConfig{
		Name:        "test-health",
		Version:     "0.1.0",
		Environment: "development",
		Port:        0,
		AdminPort:   0,
		LogLevel:    "info",
		Tracing: tracing.TracerConfig{
			Enabled: false,
		},
		ShutdownTimeout: 5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
	}

	svc, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Test /healthz on admin router.
	t.Run("healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		svc.AdminRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET /healthz status = %d, want %d", w.Code, http.StatusOK)
		}

		var body map[string]string
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decoding healthz response: %v", err)
		}
		if body["status"] != "alive" {
			t.Errorf("healthz status = %q, want %q", body["status"], "alive")
		}
	})

	// Test /readyz on admin router (should be healthy with no checks).
	t.Run("readyz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()
		svc.AdminRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET /readyz status = %d, want %d", w.Code, http.StatusOK)
		}

		var body health.CompositeResult
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decoding readyz response: %v", err)
		}
		if body.Status != "healthy" {
			t.Errorf("readyz status = %q, want %q", body.Status, "healthy")
		}
	})

	// Test /health on admin router.
	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		svc.AdminRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET /health status = %d, want %d", w.Code, http.StatusOK)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decoding health response: %v", err)
		}
		if body["service"] != "test-health" {
			t.Errorf("health service = %v, want %q", body["service"], "test-health")
		}
		if body["version"] != "0.1.0" {
			t.Errorf("health version = %v, want %q", body["version"], "0.1.0")
		}
	})

	// Test /metrics on admin router.
	t.Run("metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		svc.AdminRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET /metrics status = %d, want %d", w.Code, http.StatusOK)
		}

		body, _ := io.ReadAll(w.Body)
		bodyStr := string(body)

		// Verify Go runtime metrics are present.
		if !containsStr(bodyStr, "go_goroutines") {
			t.Error("expected /metrics to contain go_goroutines")
		}
	})
}

func TestBootstrap_DBDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &bootstrap.ServiceConfig{
		Name:        "test-db-down",
		Version:     "0.1.0",
		Environment: "development",
		Port:        0,
		AdminPort:   0,
		LogLevel:    "error",
		DB: &bootstrap.DBConfig{
			URL:               "postgres://nonexistent:5432/testdb?sslmode=disable&connect_timeout=1",
			MinConns:          1,
			MaxConns:          2,
			MaxConnLife:       1 * time.Hour,
			MaxConnIdle:       30 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		},
		Tracing: tracing.TracerConfig{
			Enabled: false,
		},
		ShutdownTimeout: 5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
	}

	_, err := bootstrap.Bootstrap(ctx, cfg)
	if err == nil {
		t.Fatal("Bootstrap() with unreachable DB should return error")
	}
}

func TestBootstrap_HTTPMetrics_Recorded(t *testing.T) {
	ctx := context.Background()

	cfg := &bootstrap.ServiceConfig{
		Name:        "test-metrics",
		Version:     "0.1.0",
		Environment: "development",
		Port:        0,
		AdminPort:   0,
		LogLevel:    "error",
		Tracing: tracing.TracerConfig{
			Enabled: false,
		},
		ShutdownTimeout: 5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
	}

	svc, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Register a test endpoint.
	svc.Router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Send 5 requests.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		svc.Router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}

	// Check metrics endpoint contains http_requests_total.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	svc.AdminRouter.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Body)
	bodyStr := string(body)

	if !containsStr(bodyStr, "http_requests_total") {
		t.Error("expected /metrics to contain http_requests_total")
	}
	if !containsStr(bodyStr, "http_request_duration_seconds") {
		t.Error("expected /metrics to contain http_request_duration_seconds")
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
