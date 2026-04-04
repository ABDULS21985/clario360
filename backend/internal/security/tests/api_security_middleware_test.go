package security_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

func newTestAPISecurityMiddleware(t *testing.T) func(http.Handler) http.Handler {
	t.Helper()
	cfg := security.DefaultAPISecurityConfig()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	return security.APISecurityMiddleware(cfg, secLogger, logger, metrics)
}

func TestAPISecurityMiddleware_BodySizeLimit(t *testing.T) {
	cfg := &security.APISecurityConfig{
		MaxBodySize:   100,
		RequireJSON:   false,
		SanitizeInput: false,
		MaxPerPage:    100,
	}
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	mw := security.APISecurityMiddleware(cfg, secLogger, logger, metrics)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader("x"))
	req.ContentLength = 200
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when body exceeds limit")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rr.Code)
	}
}

func TestAPISecurityMiddleware_ContentTypeValidation(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader("data"))
	req.Header.Set("Content-Type", "text/xml")
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for invalid content type")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", rr.Code)
	}
}

func TestAPISecurityMiddleware_JSONContentTypeAllowed(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called for application/json content type")
	}
}

func TestAPISecurityMiddleware_MultipartAllowed(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called for multipart/form-data")
	}
}

func TestAPISecurityMiddleware_GETSkipsContentType(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected GET to skip content type validation")
	}
}

func TestAPISecurityMiddleware_PaginationLimit(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?per_page=500", nil)
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for invalid pagination")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAPISecurityMiddleware_PaginationValid(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?per_page=25", nil)
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called for valid pagination")
	}
}

func TestAPISecurityMiddleware_LimitParam(t *testing.T) {
	mw := newTestAPISecurityMiddleware(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=999", nil)
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for excessive limit")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAPISecurityMiddleware_UUIDValidation(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	cfg := security.DefaultAPISecurityConfig()
	mw := security.APISecurityMiddleware(cfg, secLogger, logger, metrics)

	r := chi.NewRouter()
	r.With(mw).Get("/api/v1/assets/{assetID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Invalid UUID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestAPISecurityMiddleware_ValidUUID(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	cfg := security.DefaultAPISecurityConfig()
	mw := security.APISecurityMiddleware(cfg, secLogger, logger, metrics)

	r := chi.NewRouter()
	var called bool
	r.With(mw).Get("/api/v1/assets/{assetID}", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/550e8400-e29b-41d4-a716-446655440000", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called for valid UUID")
	}
}

func TestContentTypeEnforcement_MissingOnPOST(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	mw := security.ContentTypeEnforcement(secLogger, metrics)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader("body"))
	req.Header.Del("Content-Type")
	req.ContentLength = 4
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when Content-Type is missing on POST with body")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", rr.Code)
	}
}

func TestContentTypeEnforcement_GETAllowed(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	mw := security.ContentTypeEnforcement(secLogger, metrics)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected GET to pass through content type enforcement")
	}
}
