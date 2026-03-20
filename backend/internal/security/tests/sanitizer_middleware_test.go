package security_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

func newTestSanitizerMiddleware(t *testing.T) func(http.Handler) http.Handler {
	t.Helper()
	sanitizer := security.NewSanitizer()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	return security.SanitizeRequestBody(sanitizer, secLogger, logger)
}

func TestSanitizeMiddleware_CleanJSON(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	body := `{"name": "Alice", "role": "analyst"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called for clean JSON")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestSanitizeMiddleware_SQLInjection(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	body := `{"name": "'; DROP TABLE users; --"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for malicious input")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "MALICIOUS_INPUT" {
		t.Errorf("expected code MALICIOUS_INPUT, got %v", errObj["code"])
	}
}

func TestSanitizeMiddleware_XSSAttempt(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	body := `{"bio": "<script>alert('xss')</script>"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for XSS input")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSanitizeMiddleware_GETPassesThrough(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected GET request to pass through without body inspection")
	}
}

func TestSanitizeMiddleware_NonJSONPassesThrough(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", strings.NewReader("raw file data"))
	req.Header.Set("Content-Type", "multipart/form-data")
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected non-JSON content to pass through")
	}
}

func TestSanitizeMiddleware_TooLargeBody(t *testing.T) {
	sanitizer := security.NewSanitizer(security.WithMaxJSONSize(50))
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	secLogger := security.NewSecurityLogger(logger, metrics, false)
	mw := security.SanitizeRequestBody(sanitizer, secLogger, logger)

	bigBody := strings.Repeat("a", 100)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader(bigBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for oversized body")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rr.Code)
	}
}

func TestSanitizeMiddleware_EmptyBody(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/action", nil)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 0
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected empty body to pass through")
	}
}

func TestSanitizeMiddleware_NestedArrayInjection(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	body := `{"items": ["safe", "1 UNION SELECT * FROM users"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for nested injection")
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSanitizeMiddleware_InvalidJSONPassesThrough(t *testing.T) {
	mw := newTestSanitizerMiddleware(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Verify body is still readable
		var buf bytes.Buffer
		buf.ReadFrom(r.Body)
		if buf.String() != "{invalid json" {
			t.Errorf("expected body to be preserved, got: %s", buf.String())
		}
	}))
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected invalid JSON to pass through to handler")
	}
}
