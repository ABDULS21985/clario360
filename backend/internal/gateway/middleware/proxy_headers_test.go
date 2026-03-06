package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clario360/platform/internal/auth"
)

// TestProxyHeaders_InjectsTenantID — authenticated request gets X-Tenant-ID injected.
func TestProxyHeaders_InjectsTenantID(t *testing.T) {
	var capturedTenant string
	handler := ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get("X-Tenant-ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	// Simulate auth context set by ProxyAuth middleware.
	ctx := auth.WithUser(req.Context(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-abc",
		Email:    "test@example.com",
		Roles:    []string{"admin"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedTenant != "tenant-abc" {
		t.Errorf("expected X-Tenant-ID=tenant-abc, got %q", capturedTenant)
	}
}

// TestProxyHeaders_StripsIncomingInternalHeaders — malicious client headers are removed.
func TestProxyHeaders_StripsIncomingInternalHeaders(t *testing.T) {
	var capturedTenant, capturedUser string
	handler := ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get("X-Tenant-ID")
		capturedUser = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	// Client injects these headers to try to impersonate another tenant.
	req.Header.Set("X-Tenant-ID", "evil-tenant")
	req.Header.Set("X-User-ID", "evil-user")
	req.Header.Set("X-User-Permissions", "superadmin")

	// No auth context — headers should be stripped and not replaced.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Without auth context the headers should be empty (stripped, not replaced).
	if capturedTenant != "" {
		t.Errorf("X-Tenant-ID should be stripped when no auth context, got %q", capturedTenant)
	}
	if capturedUser != "" {
		t.Errorf("X-User-ID should be stripped when no auth context, got %q", capturedUser)
	}
}

// TestProxyHeaders_StripsResponseInternalHeaders — response from backend doesn't leak internal headers.
func TestProxyHeaders_StripsResponseInternalHeaders(t *testing.T) {
	handler := ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Backend sets internal headers on the response.
		w.Header().Set("X-User-ID", "backend-user")
		w.Header().Set("X-Tenant-ID", "backend-tenant")
		w.Header().Set("X-Powered-By", "GoBackend/1.0")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// These must not appear in the response to the client.
	if got := rr.Header().Get("X-User-ID"); got != "" {
		t.Errorf("X-User-ID must be stripped from response, got %q", got)
	}
	if got := rr.Header().Get("X-Tenant-ID"); got != "" {
		t.Errorf("X-Tenant-ID must be stripped from response, got %q", got)
	}
	if got := rr.Header().Get("X-Powered-By"); got != "" {
		t.Errorf("X-Powered-By must be stripped from response, got %q", got)
	}
}

// TestProxyHeaders_InjectsAllUserHeaders — all trusted headers are injected from JWT claims.
func TestProxyHeaders_InjectsAllUserHeaders(t *testing.T) {
	var capturedEmail, capturedRoles string
	handler := ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEmail = r.Header.Get("X-User-Email")
		capturedRoles = r.Header.Get("X-User-Roles")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	ctx := auth.WithUser(req.Context(), &auth.ContextUser{
		ID:       "user-2",
		TenantID: "tenant-x",
		Email:    "admin@company.com",
		Roles:    []string{"admin", "viewer"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedEmail != "admin@company.com" {
		t.Errorf("expected X-User-Email=admin@company.com, got %q", capturedEmail)
	}
	if capturedRoles != "admin,viewer" {
		t.Errorf("expected X-User-Roles=admin,viewer, got %q", capturedRoles)
	}
}
