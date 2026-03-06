package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	gwconfig "github.com/clario360/platform/internal/gateway/config"
	"github.com/clario360/platform/internal/gateway/health"
	gwmw "github.com/clario360/platform/internal/gateway/middleware"
	gwmetrics "github.com/clario360/platform/internal/gateway/metrics"
	"github.com/clario360/platform/internal/gateway/proxy"
	"github.com/clario360/platform/internal/gateway/ratelimit"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
)

func main() {
	ctx := context.Background()

	// ── 1. Load config ────────────────────────────────────────────────────────
	legacyCfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("loading config")
	}

	gCfg, err := gwconfig.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("loading gateway config")
	}

	env := gCfg.Environment

	// ── 2. Bootstrap infrastructure ──────────────────────────────────────────
	svcCfg := &bootstrap.ServiceConfig{
		Name:        "api-gateway",
		Version:     "1.0.0",
		Environment: env,
		Port:        gCfg.HTTPPort,
		AdminPort:   gCfg.AdminPort,
		LogLevel:    legacyCfg.Observability.LogLevel,
		Redis: &bootstrap.RedisConfig{
			Addr:     legacyCfg.Redis.Addr(),
			Password: legacyCfg.Redis.Password,
			DB:       legacyCfg.Redis.DB,
		},
		Tracing: tracing.TracerConfig{
			Enabled:     legacyCfg.Observability.OTLPEndpoint != "",
			Endpoint:    legacyCfg.Observability.OTLPEndpoint,
			ServiceName: "api-gateway",
			Version:     "1.0.0",
			Environment: env,
			SampleRate:  0.1,
			Insecure:    true,
		},
		ShutdownTimeout: legacyCfg.Server.ShutdownTimeout,
		ReadTimeout:     gCfg.ReadTimeout,
		WriteTimeout:    gCfg.WriteTimeout,
	}

	svc, err := bootstrap.Bootstrap(ctx, svcCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to bootstrap api-gateway")
	}

	// ── 3. JWT Manager ────────────────────────────────────────────────────────
	jwtMgr, err := auth.NewJWTManager(legacyCfg.Auth)
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// ── 4. Service Registry ───────────────────────────────────────────────────
	registry, err := proxy.NewServiceRegistry(gwconfig.DefaultServices())
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create service registry")
	}

	// ── 5. Circuit Breakers + Proxy Router ────────────────────────────────────
	routes := gwconfig.DefaultRoutes()
	cbCfg := proxy.CircuitBreakerConfig{
		FailureThreshold:   gCfg.CBFailureThreshold,
		FailureRateWindow:  time.Duration(gCfg.CBIntervalSec) * time.Second,
		FailureRatePercent: 50,
		OpenTimeout:        time.Duration(gCfg.CBTimeoutSec) * time.Second,
		HalfOpenSuccesses:  gCfg.CBMaxRequests,
	}
	proxyRouter, err := proxy.NewRouter(routes, registry, cbCfg, svc.Logger)
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create proxy router")
	}

	// ── 6. Rate Limiter ───────────────────────────────────────────────────────
	rlCfg := ratelimit.ConfigFromGateway(
		gCfg.RateLimitAuthPerMin,
		gCfg.RateLimitReadPerMin,
		gCfg.RateLimitWritePerMin,
		gCfg.RateLimitAdminPerMin,
		gCfg.RateLimitUploadPerMin,
		gCfg.RateLimitWSPerMin,
	)
	limiter := ratelimit.NewLimiter(svc.Redis, rlCfg)

	// ── 7. Gateway Metrics ────────────────────────────────────────────────────
	gwMetrics := gwmetrics.NewGatewayMetrics()

	// ── 8. Health Checker ─────────────────────────────────────────────────────
	healthChecker := health.NewChecker(registry, proxyRouter, svc.Logger)

	// ── 9. Build main router ──────────────────────────────────────────────────
	// Build per-route body size and timeout override maps.
	bodyOverrides := make(map[string]int)
	timeoutOverrides := make(map[string]time.Duration)
	for _, r := range routes {
		if r.MaxBodyMB > 0 {
			bodyOverrides[r.Prefix] = r.MaxBodyMB
		}
		if r.TimeoutSec > 0 {
			timeoutOverrides[r.Prefix] = time.Duration(r.TimeoutSec) * time.Second
		}
	}

	svc.Router = chi.NewRouter()

	// Middleware chain in security-critical order:
	// Recovery → RequestID → SecurityHeaders → CORS → BodyLimit → Logging → Metrics → Timeout → (Auth per-route) → RateLimit → ProxyHeaders
	svc.Router.Use(middleware.RecoveryWithLogger(svc.Logger))
	svc.Router.Use(middleware.RequestID)
	svc.Router.Use(middleware.SecurityHeaders())
	svc.Router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   gCfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID", "X-API-Key"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))
	svc.Router.Use(gwmw.BodyLimit(gCfg.MaxRequestBodyMB, bodyOverrides))
	svc.Router.Use(middleware.Logging(svc.Logger))
	svc.Router.Use(tracing.ChiTracingMiddleware(svcCfg.Name))
	svc.Router.Use(gwmw.Timeout(gCfg.ProxyTimeout, timeoutOverrides))

	// ── 10. Health endpoints (no auth) ────────────────────────────────────────
	svc.Router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	svc.Router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check Redis connectivity (rate limiter depends on it).
		ctx2, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := svc.Redis.Ping(ctx2).Err(); err != nil {
			svc.Logger.Warn().Err(err).Msg("readyz: redis unavailable")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "degraded", "reason": "redis unavailable"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// ── 11. Gateway status (proxied to admin port via Bootstrap) ─────────────
	svc.Router.Get("/api/v1/gateway/status", func(w http.ResponseWriter, r *http.Request) {
		type serviceStatus struct {
			Name           string `json:"name"`
			CircuitBreaker string `json:"circuit_breaker"`
		}
		proxies := proxyRouter.Proxies()
		statuses := make([]serviceStatus, 0, len(proxies))
		for name, rp := range proxies {
			cbState := rp.CircuitState()
			statuses = append(statuses, serviceStatus{
				Name:           name,
				CircuitBreaker: cbState.String(),
			})
			gwMetrics.CircuitBreakerState.WithLabelValues(name).Set(float64(cbState))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"services": statuses})
	})

	// ── 12. Register HTTP proxy routes ────────────────────────────────────────
	for _, route := range routes {
		route := route // capture
		match := proxyRouter.Match(route.Prefix)
		if !match.Matched {
			svc.Logger.Warn().Str("prefix", route.Prefix).Msg("no proxy found for route, skipping")
			continue
		}
		rp := match.Proxy

		svc.Router.Route(route.Prefix, func(sub chi.Router) {
			// Auth middleware: validate JWT for protected routes; optional for public.
			if !route.Public {
				sub.Use(gwmw.ProxyAuth(jwtMgr, gwMetrics, svc.Logger))
			} else {
				sub.Use(middleware.OptionalAuth(jwtMgr))
			}

			// Inject/strip gateway headers.
			sub.Use(gwmw.ProxyHeaders)

			// Rate limiting (uses tenant_id from context set by ProxyAuth).
			sub.Use(gwmw.ProxyRateLimit(limiter, route.EndpointGroup, gwMetrics, svc.Logger))

			// Metrics (no tenant_id label).
			sub.Use(gwmw.ProxyMetrics(gwMetrics, route.Service))

			// Structured logging per proxied request.
			sub.Use(gwmw.ProxyLogging(svc.Logger, route.Service))

			sub.Use(tracing.SpanEnricher())

			sub.HandleFunc("/*", rp.ServeHTTP)
			sub.HandleFunc("/", rp.ServeHTTP)
		})
	}

	// ── 13. WebSocket proxy routes ────────────────────────────────────────────
	for _, wsRoute := range gwconfig.DefaultWSRoutes() {
		wsRoute := wsRoute // capture
		target, _, ok := registry.Resolve(wsRoute.Service)
		if !ok {
			svc.Logger.Warn().Str("service", wsRoute.Service).Msg("WS service not found in registry")
			continue
		}

		wsProxy := proxy.NewWebSocketProxy(
			target,
			jwtMgr,
			gCfg.CORSAllowedOrigins,
			nil, // limiter — implement WSLimiter adapter if needed
			gwMetrics,
			svc.Logger,
		)

		svc.Router.HandleFunc(wsRoute.Prefix+"/*", wsProxy.ServeHTTP)
		svc.Router.HandleFunc(wsRoute.Prefix, wsProxy.ServeHTTP)
	}

	// Aggregated backend health (on admin router to avoid public exposure of topology).
	svc.AdminRouter.Get("/health", healthChecker.Handler())

	// 404 handler with structured error.
	svc.Router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "no route matches " + r.URL.Path,
			},
		})
	})

	svc.Logger.Info().Int("port", gCfg.HTTPPort).Int("admin_port", gCfg.AdminPort).Msg("api-gateway starting")
	if err := svc.Run(ctx); err != nil {
		svc.Logger.Fatal().Err(err).Msg("api-gateway failed")
		os.Exit(1)
	}
}
