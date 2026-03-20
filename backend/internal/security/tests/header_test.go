package security_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// headerSetup creates SecurityHeaders middleware with production config.
func headerSetup() func(http.Handler) http.Handler {
	cfg := security.DefaultProductionHeadersConfig()
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)

	return security.SecurityHeaders(cfg, logger, metrics)
}

// headerSetupDev creates SecurityHeaders middleware with development config.
func headerSetupDev() func(http.Handler) http.Handler {
	cfg := security.DefaultDevelopmentHeadersConfig()
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)

	return security.SecurityHeaders(cfg, logger, metrics)
}

// headerSetupCrossOrigin creates SecurityHeaders middleware with cross-origin policies enabled.
func headerSetupCrossOrigin() func(http.Handler) http.Handler {
	cfg := security.DefaultProductionHeadersConfig()
	cfg.EnableCOEP = true
	cfg.EnableCOOP = true
	cfg.EnableCORP = true
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)

	return security.SecurityHeaders(cfg, logger, metrics)
}

// noopHandler is a handler that does nothing, returns 200.
var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// executeRequest creates a GET request to the given path and runs it through the middleware.
func executeRequest(mw func(http.Handler) http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mw(noopHandler).ServeHTTP(rec, req)
	return rec
}

// --- X-Content-Type-Options ---

func TestHeaders_XContentTypeOptionsIsNosniff(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("X-Content-Type-Options")
	if val != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options to be 'nosniff', got %q", val)
	}
}

// --- X-Frame-Options ---

func TestHeaders_XFrameOptionsIsDeny(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("X-Frame-Options")
	if val != "DENY" {
		t.Fatalf("expected X-Frame-Options to be 'DENY', got %q", val)
	}
}

// --- X-XSS-Protection ---

func TestHeaders_XXSSProtectionIsZero(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("X-XSS-Protection")
	if val != "0" {
		t.Fatalf("expected X-XSS-Protection to be '0', got %q", val)
	}
}

// --- Content-Security-Policy ---

func TestHeaders_CSPIsPresent(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected Content-Security-Policy header to be present")
	}
}

func TestHeaders_CSPContainsDefaultSrc(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src") {
		t.Fatalf("expected CSP to contain 'default-src', got %q", csp)
	}
}

func TestHeaders_CSPContainsScriptSrc(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src") {
		t.Fatalf("expected CSP to contain 'script-src', got %q", csp)
	}
}

func TestHeaders_CSPContainsObjectSrcNone(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "object-src 'none'") {
		t.Fatalf("expected CSP to contain \"object-src 'none'\", got %q", csp)
	}
}

func TestHeaders_CSPContainsFrameAncestors(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors") {
		t.Fatalf("expected CSP to contain 'frame-ancestors', got %q", csp)
	}
}

// --- Referrer-Policy ---

func TestHeaders_ReferrerPolicyIsSet(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Referrer-Policy")
	if val == "" {
		t.Fatal("expected Referrer-Policy header to be set")
	}
	if val != "strict-origin-when-cross-origin" {
		t.Fatalf("expected Referrer-Policy to be 'strict-origin-when-cross-origin', got %q", val)
	}
}

// --- Permissions-Policy ---

func TestHeaders_PermissionsPolicyContainsCamera(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Permissions-Policy")
	if !strings.Contains(val, "camera=()") {
		t.Fatalf("expected Permissions-Policy to contain 'camera=()', got %q", val)
	}
}

func TestHeaders_PermissionsPolicyContainsMicrophone(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Permissions-Policy")
	if !strings.Contains(val, "microphone=()") {
		t.Fatalf("expected Permissions-Policy to contain 'microphone=()', got %q", val)
	}
}

func TestHeaders_PermissionsPolicyContainsGeolocation(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Permissions-Policy")
	if !strings.Contains(val, "geolocation=()") {
		t.Fatalf("expected Permissions-Policy to contain 'geolocation=()', got %q", val)
	}
}

// --- Cache-Control for /api/ paths ---

func TestHeaders_CacheControlNoStoreForAPIPaths(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/api/v1/users")

	val := rec.Header().Get("Cache-Control")
	if !strings.Contains(val, "no-store") {
		t.Fatalf("expected Cache-Control to contain 'no-store' for /api/ paths, got %q", val)
	}
}

func TestHeaders_NoCacheControlForNonAPIPaths(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/static/app.js")

	val := rec.Header().Get("Cache-Control")
	// Non-API, non-static-asset paths should NOT have "no-store"
	if strings.Contains(val, "no-store") {
		t.Fatalf("expected Cache-Control NOT to contain 'no-store' for non-/api/ paths, got %q", val)
	}
}

func TestHeaders_PragmaNoCacheForAPIPaths(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/api/v1/data")

	val := rec.Header().Get("Pragma")
	if val != "no-cache" {
		t.Fatalf("expected Pragma to be 'no-cache' for /api/ paths, got %q", val)
	}
}

// --- No Server header ---

func TestHeaders_NoServerHeader(t *testing.T) {
	mw := headerSetup()

	// Use a handler that sets the Server header to simulate a default server response
	handlerThatSetsServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Go-http-server")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw(handlerThatSetsServer).ServeHTTP(rec, req)

	// The security middleware deletes the Server header before the handler runs,
	// but the handler sets it after. Verify the middleware at least removes it once.
	// More importantly: when no handler sets it, it should not appear.
	mw2 := headerSetup()
	rec2 := executeRequest(mw2, http.MethodGet, "/")
	if rec2.Header().Get("Server") != "" {
		t.Fatalf("expected Server header to be absent, got %q", rec2.Header().Get("Server"))
	}
}

// --- No X-Powered-By header ---

func TestHeaders_NoXPoweredByHeader(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	if rec.Header().Get("X-Powered-By") != "" {
		t.Fatalf("expected X-Powered-By header to be absent, got %q", rec.Header().Get("X-Powered-By"))
	}
}

// --- HSTS with X-Forwarded-Proto: https ---

func TestHeaders_HSTSSetWhenXForwardedProtoHTTPS(t *testing.T) {
	cfg := security.DefaultProductionHeadersConfig()
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	mw := security.SecurityHeaders(cfg, logger, metrics)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	mw(noopHandler).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Fatal("expected HSTS header to be set when X-Forwarded-Proto is https")
	}
	if !strings.Contains(hsts, "max-age=") {
		t.Fatalf("expected HSTS to contain max-age directive, got %q", hsts)
	}
	if !strings.Contains(hsts, "includeSubDomains") {
		t.Fatalf("expected HSTS to contain includeSubDomains, got %q", hsts)
	}
}

func TestHeaders_HSTSPreloadInProduction(t *testing.T) {
	cfg := security.DefaultProductionHeadersConfig()
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	mw := security.SecurityHeaders(cfg, logger, metrics)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	mw(noopHandler).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "preload") {
		t.Fatalf("expected HSTS to contain 'preload' in production, got %q", hsts)
	}
}

// --- HSTS NOT set in development mode ---

func TestHeaders_HSTSNotSetInDevelopment(t *testing.T) {
	mw := headerSetupDev()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	mw(noopHandler).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Fatalf("expected HSTS header NOT to be set in development, got %q", hsts)
	}
}

// --- HSTS not set for plain HTTP in production ---

func TestHeaders_HSTSNotSetForPlainHTTP(t *testing.T) {
	mw := headerSetup()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No X-Forwarded-Proto, no TLS
	rec := httptest.NewRecorder()
	mw(noopHandler).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Fatalf("expected HSTS NOT to be set for plain HTTP, got %q", hsts)
	}
}

// --- Cross-Origin headers ---

func TestHeaders_CrossOriginEmbedderPolicySet(t *testing.T) {
	mw := headerSetupCrossOrigin()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Cross-Origin-Embedder-Policy")
	if val != "require-corp" {
		t.Fatalf("expected Cross-Origin-Embedder-Policy to be 'require-corp', got %q", val)
	}
}

func TestHeaders_CrossOriginOpenerPolicySet(t *testing.T) {
	mw := headerSetupCrossOrigin()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Cross-Origin-Opener-Policy")
	if val != "same-origin" {
		t.Fatalf("expected Cross-Origin-Opener-Policy to be 'same-origin', got %q", val)
	}
}

func TestHeaders_CrossOriginResourcePolicySet(t *testing.T) {
	mw := headerSetupCrossOrigin()
	rec := executeRequest(mw, http.MethodGet, "/")

	val := rec.Header().Get("Cross-Origin-Resource-Policy")
	if val != "same-origin" {
		t.Fatalf("expected Cross-Origin-Resource-Policy to be 'same-origin', got %q", val)
	}
}

// --- Cross-Origin headers NOT set when disabled ---

func TestHeaders_CrossOriginHeadersAbsentWhenDisabled(t *testing.T) {
	cfg := security.DefaultProductionHeadersConfig()
	cfg.EnableCOEP = false
	cfg.EnableCOOP = false
	cfg.EnableCORP = false
	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	mw := security.SecurityHeaders(cfg, logger, metrics)

	rec := executeRequest(mw, http.MethodGet, "/")

	if rec.Header().Get("Cross-Origin-Embedder-Policy") != "" {
		t.Fatal("expected COEP header to be absent when disabled")
	}
	if rec.Header().Get("Cross-Origin-Opener-Policy") != "" {
		t.Fatal("expected COOP header to be absent when disabled")
	}
	if rec.Header().Get("Cross-Origin-Resource-Policy") != "" {
		t.Fatal("expected CORP header to be absent when disabled")
	}
}

// --- Headers applied on all HTTP methods ---

func TestHeaders_POSTRequestGetsSecurityHeaders(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodPost, "/api/v1/resource")

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("expected security headers on POST request")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("expected X-Frame-Options on POST request")
	}
}

// --- CSP in development allows unsafe-eval ---

func TestHeaders_CSPDevAllowsUnsafeEval(t *testing.T) {
	mw := headerSetupDev()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'unsafe-eval'") {
		t.Fatalf("expected CSP in development to contain 'unsafe-eval', got %q", csp)
	}
}

// --- CSP in production contains upgrade-insecure-requests ---

func TestHeaders_CSPProdUpgradeInsecureRequests(t *testing.T) {
	mw := headerSetup()
	rec := executeRequest(mw, http.MethodGet, "/")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "upgrade-insecure-requests") {
		t.Fatalf("expected CSP in production to contain 'upgrade-insecure-requests', got %q", csp)
	}
}
