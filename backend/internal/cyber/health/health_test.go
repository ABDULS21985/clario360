package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegister_HealthEndpoints(t *testing.T) {
	router := chi.NewRouter()
	Register(router)

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, resp.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal %s response: %v", path, err)
		}
		if body["status"] == "" {
			t.Fatalf("expected status in %s response", path)
		}
	}
}
