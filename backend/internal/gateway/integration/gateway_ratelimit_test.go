package integration

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"context"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
	gwmetrics "github.com/clario360/platform/internal/gateway/metrics"
	gwmw "github.com/clario360/platform/internal/gateway/middleware"
	"github.com/clario360/platform/internal/gateway/ratelimit"
	"github.com/clario360/platform/internal/middleware"
)

func newTestRedisForRL(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 14})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available, skipping rate limit integration test")
	}
	rdb.FlushDB(ctx)
	t.Cleanup(func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	})
	return rdb
}

// TestRateLimit_BlocksExcess — after N requests, the (N+1)th is rejected with 429.
func TestRateLimit_BlocksExcess(t *testing.T) {
	rdb := newTestRedisForRL(t)

	const limit = 5
	rlCfg := ratelimit.DefaultConfig()
	rlCfg.Groups[gwconfig.EndpointGroupAuth] = ratelimit.GroupLimit{
		RequestsPerWindow: limit,
		Window:            1 * time.Minute,
	}
	limiter := ratelimit.NewLimiter(rdb, rlCfg)
	gwMetrics := gwmetrics.NewGatewayMetrics()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Route("/api/v1/auth", func(sub chi.Router) {
		sub.Use(gwmw.ProxyRateLimit(limiter, gwconfig.EndpointGroupAuth, gwMetrics, zerolog.Nop()))
		sub.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// First N requests should succeed.
	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	// (N+1)th should be rate limited.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}

	// Retry-After header must be present.
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

// TestRateLimit_DifferentTenants — each tenant has an independent rate limit.
func TestRateLimit_DifferentTenants(t *testing.T) {
	rdb := newTestRedisForRL(t)

	const limit = 3
	rlCfg := ratelimit.DefaultConfig()
	rlCfg.Groups[gwconfig.EndpointGroupWrite] = ratelimit.GroupLimit{
		RequestsPerWindow: limit,
		Window:            1 * time.Minute,
	}
	limiter := ratelimit.NewLimiter(rdb, rlCfg)
	gwMetrics := gwmetrics.NewGatewayMetrics()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// Build two separate handler chains for tenant-A and tenant-B using their X-Tenant-ID header.
	makeHandler := func(tenantID string) http.Handler {
		r := chi.NewRouter()
		r.Route("/api/v1/data", func(sub chi.Router) {
			sub.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate auth context by adding tenant key in header.
					r.Header.Set("X-Tenant-ID", tenantID)
					next.ServeHTTP(w, r)
				})
			})
			// Use tenant-keyed rate limiter: pass tenant header as key directly.
			sub.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					key := r.Header.Get("X-Tenant-ID")
					result := limiter.Check(r.Context(), key, gwconfig.EndpointGroupWrite)
					w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
					if !result.Allowed {
						gwMetrics.RateLimitExceeded.WithLabelValues(string(gwconfig.EndpointGroupWrite)).Inc()
						w.WriteHeader(http.StatusTooManyRequests)
						return
					}
					next.ServeHTTP(w, r)
				})
			})
			sub.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		})
		return r
	}

	handlerA := makeHandler("tenant-A")
	handlerB := makeHandler("tenant-B")

	// Exhaust tenant-A's limit.
	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/data/items", nil)
		rr := httptest.NewRecorder()
		handlerA.ServeHTTP(rr, req)
	}
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/data/items", nil)
	rrA := httptest.NewRecorder()
	handlerA.ServeHTTP(rrA, reqA)
	if rrA.Code != http.StatusTooManyRequests {
		t.Errorf("tenant-A should be rate limited, got %d", rrA.Code)
	}

	// tenant-B should still be allowed.
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/data/items", nil)
	rrB := httptest.NewRecorder()
	handlerB.ServeHTTP(rrB, reqB)
	if rrB.Code != http.StatusOK {
		t.Errorf("tenant-B must not be affected by tenant-A's rate limit, got %d", rrB.Code)
	}
}

// TestRateLimit_Headers — response includes correct X-RateLimit-* headers.
func TestRateLimit_Headers(t *testing.T) {
	rdb := newTestRedisForRL(t)

	const limit = 10
	rlCfg := ratelimit.DefaultConfig()
	rlCfg.Groups[gwconfig.EndpointGroupRead] = ratelimit.GroupLimit{
		RequestsPerWindow: limit,
		Window:            1 * time.Minute,
	}
	limiter := ratelimit.NewLimiter(rdb, rlCfg)
	gwMetrics := gwmetrics.NewGatewayMetrics()

	r := chi.NewRouter()
	r.Route("/api/v1/audit", func(sub chi.Router) {
		sub.Use(gwmw.ProxyRateLimit(limiter, gwconfig.EndpointGroupRead, gwMetrics, zerolog.Nop()))
		sub.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// Send 3 requests (limit is 10).
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}

	// On the 4th request, check headers.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	limitHdr := rr.Header().Get("X-RateLimit-Limit")
	remainHdr := rr.Header().Get("X-RateLimit-Remaining")
	resetHdr := rr.Header().Get("X-RateLimit-Reset")

	if limitHdr != strconv.Itoa(limit) {
		t.Errorf("expected X-RateLimit-Limit=%d, got %q", limit, limitHdr)
	}
	if remainHdr == "" {
		t.Error("expected X-RateLimit-Remaining to be set")
	}
	if resetHdr == "" {
		t.Error("expected X-RateLimit-Reset to be set")
	}

	remaining, err := strconv.Atoi(remainHdr)
	if err != nil || remaining > limit-4 {
		t.Errorf("expected X-RateLimit-Remaining <= %d, got %q", limit-4, remainHdr)
	}
}
