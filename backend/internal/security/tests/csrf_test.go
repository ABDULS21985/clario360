package security_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// csrfSetup creates CSRF middleware and dependencies for each test.
func csrfSetup() (func(http.Handler) http.Handler, *security.CSRFConfig) {
	cfg := security.DefaultCSRFConfig()
	cfg.CookieSecure = false // not relevant in tests

	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	secLogger := security.NewSecurityLogger(logger, metrics, false)

	mw := security.CSRFProtection(cfg, secLogger, logger, metrics)
	return mw, cfg
}

// okHandler is a simple handler that returns 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
})

// --- Missing CSRF cookie returns 403 ---

func TestCSRF_MissingCookieReturns403(t *testing.T) {
	mw, cfg := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.Header.Set(cfg.HeaderName, "some-token-value")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when CSRF cookie is missing, got %d", rec.Code)
	}
}

// --- Missing CSRF header returns 403 ---

func TestCSRF_MissingHeaderReturns403(t *testing.T) {
	mw, cfg := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.CookieName,
		Value: "valid-token",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when CSRF header is missing, got %d", rec.Code)
	}
}

// --- Mismatched cookie and header returns 403 ---

func TestCSRF_MismatchedTokensReturns403(t *testing.T) {
	mw, cfg := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.CookieName,
		Value: "cookie-token-value",
	})
	req.Header.Set(cfg.HeaderName, "different-header-token-value")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when cookie and header tokens mismatch, got %d", rec.Code)
	}
}

// --- Valid matching cookie and header passes through ---

func TestCSRF_ValidMatchingTokensPassThrough(t *testing.T) {
	mw, cfg := csrfSetup()
	handler := mw(okHandler)

	token := "my-secure-csrf-token-12345"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.CookieName,
		Value: token,
	})
	req.Header.Set(cfg.HeaderName, token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when CSRF tokens match, got %d", rec.Code)
	}
}

// --- GET requests are exempt ---

func TestCSRF_GETRequestExempt(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected GET to pass through without CSRF, got %d", rec.Code)
	}
}

// --- HEAD requests are exempt ---

func TestCSRF_HEADRequestExempt(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodHead, "/api/v1/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HEAD to pass through without CSRF, got %d", rec.Code)
	}
}

// --- OPTIONS requests are exempt ---

func TestCSRF_OPTIONSRequestExempt(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected OPTIONS to pass through without CSRF, got %d", rec.Code)
	}
}

// --- Exempt paths: webhook ---

func TestCSRF_ExemptPathWebhook(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected webhook path to be exempt from CSRF, got %d", rec.Code)
	}
}

// --- Exempt paths: health ---

func TestCSRF_ExemptPathHealth(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected health path to be exempt from CSRF, got %d", rec.Code)
	}
}

// --- Exempt paths: healthz ---

func TestCSRF_ExemptPathHealthz(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /healthz path to be exempt from CSRF, got %d", rec.Code)
	}
}

// --- Exempt paths: readyz ---

func TestCSRF_ExemptPathReadyz(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /readyz path to be exempt from CSRF, got %d", rec.Code)
	}
}

// --- Bearer token without CSRF cookie passes through (API key auth) ---

func TestCSRF_BearerTokenWithoutCSRFCookiePassesThrough(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected Bearer token without CSRF cookie to pass through, got %d", rec.Code)
	}
}

// --- Bearer token WITH CSRF cookie requires matching token ---

func TestCSRF_BearerTokenWithCSRFCookieRequiresToken(t *testing.T) {
	mw, cfg := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	req.Header.Set("Authorization", "Bearer some-jwt-token")
	req.AddCookie(&http.Cookie{
		Name:  cfg.CookieName,
		Value: "csrf-cookie-value",
	})
	// No X-CSRF-Token header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when Bearer token is present WITH CSRF cookie but no header, got %d", rec.Code)
	}
}

// --- PUT request requires CSRF ---

func TestCSRF_PUTRequestRequiresCSRF(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected PUT without CSRF to return 403, got %d", rec.Code)
	}
}

// --- DELETE request requires CSRF ---

func TestCSRF_DELETERequestRequiresCSRF(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected DELETE without CSRF to return 403, got %d", rec.Code)
	}
}

// --- PATCH request requires CSRF ---

func TestCSRF_PATCHRequestRequiresCSRF(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected PATCH without CSRF to return 403, got %d", rec.Code)
	}
}

// --- Valid CSRF on PUT passes through ---

func TestCSRF_ValidCSRFOnPUT(t *testing.T) {
	mw, cfg := csrfSetup()
	handler := mw(okHandler)

	token := "valid-put-csrf-token"
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/123", nil)
	req.AddCookie(&http.Cookie{
		Name:  cfg.CookieName,
		Value: token,
	})
	req.Header.Set(cfg.HeaderName, token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected valid CSRF on PUT to pass through, got %d", rec.Code)
	}
}

// --- Response body contains JSON error on failure ---

func TestCSRF_ResponseBodyContainsErrorJSON(t *testing.T) {
	mw, _ := csrfSetup()
	handler := mw(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json on CSRF failure, got %q", contentType)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected non-empty response body on CSRF failure")
	}
}
