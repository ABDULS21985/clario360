package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

func TestRateLimiter_AllowsRequestWithoutRedis(t *testing.T) {
	hit := false
	handler := RateLimiter(nil, 10, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/assets", nil)
	ctx := auth.WithUser(req.Context(), &auth.ContextUser{ID: "user-1", TenantID: "tenant-1"})
	ctx = auth.WithTenantID(ctx, "tenant-1")
	req = req.WithContext(ctx)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if !hit {
		t.Fatal("expected next handler to be invoked")
	}
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if resp.Header().Get("X-RateLimit-Limit") != "10" {
		t.Fatalf("expected rate limit header to be set, got %q", resp.Header().Get("X-RateLimit-Limit"))
	}
}
