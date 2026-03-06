package health

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// Handler provides HTTP handler functions for health check endpoints.
type Handler struct {
	checker     *CompositeHealthChecker
	serviceName string
	version     string
	startTime   time.Time
}

// NewHandler creates health check HTTP handlers.
func NewHandler(checker *CompositeHealthChecker, serviceName, version string) *Handler {
	return &Handler{
		checker:     checker,
		serviceName: serviceName,
		version:     version,
		startTime:   time.Now(),
	}
}

// Healthz returns a liveness probe handler.
//
// Always returns 200 {"status": "alive"}.
// No dependency checks. Used by Kubernetes to detect process hangs/deadlocks.
// CRITICAL: This must NEVER block on any external dependency.
func (h *Handler) Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	}
}

// Readyz returns a readiness probe handler.
//
// Runs CompositeHealthChecker.CheckAll().
// Returns 200 if overall status is "healthy" or "degraded".
// Returns 503 if overall status is "unhealthy".
// Used by Kubernetes to determine if the pod should receive traffic.
func (h *Handler) Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := h.checker.CheckAll(r.Context())

		w.Header().Set("Content-Type", "application/json")
		if result.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}

// Health returns a detailed health handler for dashboards.
//
// Same checks as /readyz but always returns 200 (so dashboards can parse JSON regardless of status).
// Includes additional detail: service uptime, version, Go runtime version.
func (h *Handler) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := h.checker.CheckAll(r.Context())

		response := map[string]interface{}{
			"status":     result.Status,
			"checks":     result.Checks,
			"service":    h.serviceName,
			"version":    h.version,
			"go_version": runtime.Version(),
			"uptime":     time.Since(h.startTime).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}
