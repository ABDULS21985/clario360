package proxy

import (
	"testing"
	"time"

	"github.com/rs/zerolog"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

func testRegistry(t *testing.T) *ServiceRegistry {
	t.Helper()
	configs := []gwconfig.ServiceConfig{
		{Name: "iam-service", URL: "http://localhost:8081", Timeout: 30 * time.Second},
		{Name: "cyber-service", URL: "http://localhost:8084", Timeout: 30 * time.Second},
	}
	reg, err := NewServiceRegistry(configs)
	if err != nil {
		t.Fatalf("NewServiceRegistry failed: %v", err)
	}
	return reg
}

func TestRouter_MatchLongestPrefix(t *testing.T) {
	registry := testRegistry(t)
	logger := zerolog.Nop()

	routes := []gwconfig.RouteConfig{
		{Prefix: "/api/v1/auth", Service: "iam-service", Public: true, EndpointGroup: gwconfig.EndpointGroupAuth},
		{Prefix: "/api/v1/users", Service: "iam-service", Public: false, EndpointGroup: gwconfig.EndpointGroupWrite},
		{Prefix: "/api/v1/cyber", Service: "cyber-service", Public: false, EndpointGroup: gwconfig.EndpointGroupWrite},
	}

	router, err := NewRouter(routes, registry, DefaultCircuitBreakerConfig(), logger)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	tests := []struct {
		path        string
		wantMatched bool
		wantService string
	}{
		{"/api/v1/auth/login", true, "iam-service"},
		{"/api/v1/users/123", true, "iam-service"},
		{"/api/v1/cyber/alerts", true, "cyber-service"},
		{"/api/v1/unknown", false, ""},
		{"/other", false, ""},
	}

	for _, tt := range tests {
		match := router.Match(tt.path)
		if match.Matched != tt.wantMatched {
			t.Errorf("Match(%s).Matched = %v, want %v", tt.path, match.Matched, tt.wantMatched)
			continue
		}
		if match.Matched && match.Proxy.ServiceName() != tt.wantService {
			t.Errorf("Match(%s).ServiceName = %s, want %s", tt.path, match.Proxy.ServiceName(), tt.wantService)
		}
	}
}

func TestRouter_GetProxy(t *testing.T) {
	registry := testRegistry(t)
	logger := zerolog.Nop()

	routes := []gwconfig.RouteConfig{
		{Prefix: "/api/v1/auth", Service: "iam-service", Public: true},
	}

	router, err := NewRouter(routes, registry, DefaultCircuitBreakerConfig(), logger)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	rp, ok := router.GetProxy("iam-service")
	if !ok {
		t.Fatal("expected iam-service proxy to be found")
	}
	if rp.ServiceName() != "iam-service" {
		t.Errorf("expected iam-service, got %s", rp.ServiceName())
	}

	_, ok = router.GetProxy("nonexistent")
	if ok {
		t.Error("expected nonexistent proxy to not be found")
	}
}

func TestRouter_SharesProxiesForSameService(t *testing.T) {
	registry := testRegistry(t)
	logger := zerolog.Nop()

	routes := []gwconfig.RouteConfig{
		{Prefix: "/api/v1/auth", Service: "iam-service", Public: true},
		{Prefix: "/api/v1/users", Service: "iam-service", Public: false},
	}

	router, err := NewRouter(routes, registry, DefaultCircuitBreakerConfig(), logger)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	m1 := router.Match("/api/v1/auth/login")
	m2 := router.Match("/api/v1/users/123")

	if m1.Proxy != m2.Proxy {
		t.Error("expected same proxy instance for same service")
	}
}

func TestRouter_SkipsUnknownService(t *testing.T) {
	registry := testRegistry(t)
	logger := zerolog.Nop()

	routes := []gwconfig.RouteConfig{
		{Prefix: "/api/v1/auth", Service: "iam-service", Public: true},
		{Prefix: "/api/v1/unknown", Service: "nonexistent-service", Public: false},
	}

	router, err := NewRouter(routes, registry, DefaultCircuitBreakerConfig(), logger)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	match := router.Match("/api/v1/unknown/something")
	if match.Matched {
		t.Error("expected unregistered service route to not match")
	}
}
