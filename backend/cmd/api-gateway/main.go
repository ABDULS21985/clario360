package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	gwconfig "github.com/clario360/platform/internal/gateway/config"
	"github.com/clario360/platform/internal/gateway/health"
	gwmw "github.com/clario360/platform/internal/gateway/middleware"
	"github.com/clario360/platform/internal/gateway/proxy"
	"github.com/clario360/platform/internal/gateway/ratelimit"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
)

func main() {
	ctx := context.Background()

	// Load legacy config for auth and Redis settings.
	legacyCfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("loading config")
	}

	env := envOrDefault("ENVIRONMENT", "development")

	cfg := &bootstrap.ServiceConfig{
		Name:        "api-gateway",
		Version:     "1.0.0",
		Environment: env,
		Port:        8080,
		AdminPort:   9090,
		LogLevel:    legacyCfg.Observability.LogLevel,
		Redis: &bootstrap.RedisConfig{
			Addr:     legacyCfg.Redis.Addr(),
			Password: legacyCfg.Redis.Password,
			DB:       legacyCfg.Redis.DB,
		},
		Kafka: &bootstrap.KafkaConfig{
			Brokers: legacyCfg.Kafka.Brokers,
			GroupID: "api-gateway",
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
		ReadTimeout:     legacyCfg.Server.ReadTimeout,
		WriteTimeout:    legacyCfg.Server.WriteTimeout,
	}

	svc, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to bootstrap api-gateway")
	}

	// Register gateway-specific metrics.
	svc.Metrics.Counter("gateway_proxy_requests_total", "Total proxied requests", []string{"service", "status"})
	svc.Metrics.Counter("gateway_circuit_breaker_state_changes_total", "Circuit breaker state changes", []string{"service", "state"})

	// JWT Manager.
	jwtMgr, err := auth.NewJWTManager(legacyCfg.Auth)
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// Service Registry.
	registry, err := proxy.NewServiceRegistry(gwconfig.DefaultServices())
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create service registry")
	}

	// Proxy Router with circuit breakers.
	routes := gwconfig.DefaultRoutes()
	cbCfg := proxy.DefaultCircuitBreakerConfig()
	proxyRouter, err := proxy.NewRouter(routes, registry, cbCfg, svc.Logger)
	if err != nil {
		svc.Logger.Fatal().Err(err).Msg("failed to create proxy router")
	}

	// Rate Limiter.
	limiter := ratelimit.NewLimiter(svc.Redis, ratelimit.DefaultConfig())

	// Gateway Metrics.
	gwMetrics := gwmw.NewGatewayMetrics()

	// Gateway Health Checker.
	healthChecker := health.NewChecker(registry, proxyRouter, svc.Logger)

	// Override CORS on the main router with gateway-specific origins.
	svc.Router = chi.NewRouter()
	svc.Router.Use(middleware.RequestID)
	svc.Router.Use(middleware.RecoveryWithLogger(svc.Logger))
	svc.Router.Use(tracing.ChiTracingMiddleware(cfg.Name))
	svc.Router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"https://*.clario360.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID", "X-Tenant-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))
	svc.Router.Use(middleware.Logging(svc.Logger))

	// Infrastructure endpoints (no auth).
	svc.Router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	svc.Router.Get("/readyz", healthChecker.Handler())

	// Gateway status endpoint.
	svc.Router.Get("/api/v1/gateway/status", func(w http.ResponseWriter, r *http.Request) {
		type serviceStatus struct {
			Name           string `json:"name"`
			CircuitBreaker string `json:"circuit_breaker"`
		}
		proxies := proxyRouter.Proxies()
		statuses := make([]serviceStatus, 0, len(proxies))
		for name, rp := range proxies {
			statuses = append(statuses, serviceStatus{
				Name:           name,
				CircuitBreaker: rp.CircuitState().String(),
			})
			gwMetrics.CircuitBreakerState.WithLabelValues(name).Set(float64(rp.CircuitState()))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"services": statuses})
	})

	// Register proxy routes.
	for _, route := range routes {
		route := route
		match := proxyRouter.Match(route.Prefix)
		if !match.Matched {
			svc.Logger.Warn().Str("prefix", route.Prefix).Msg("no proxy found for route, skipping")
			continue
		}
		rp := match.Proxy

		svc.Router.Route(route.Prefix, func(sub chi.Router) {
			if !route.Public {
				sub.Use(gwmw.ProxyAuth(jwtMgr, svc.Logger))
			} else {
				sub.Use(middleware.OptionalAuth(jwtMgr))
			}
			sub.Use(gwmw.ProxyHeaders)
			sub.Use(gwmw.ProxyRateLimit(limiter, route.EndpointGroup, gwMetrics, svc.Logger))
			sub.Use(gwmw.ProxyMetrics(gwMetrics, route.Service))
			sub.Use(gwmw.ProxyLogging(svc.Logger, route.Service))
			sub.Use(tracing.SpanEnricher())

			sub.HandleFunc("/*", rp.ServeHTTP)
			sub.HandleFunc("/", rp.ServeHTTP)
		})
	}

	// WebSocket proxy routes.
	svc.Router.Route("/ws/v1", func(sub chi.Router) {
		sub.Use(gwmw.ProxyAuth(jwtMgr, svc.Logger))
		sub.Use(gwmw.ProxyHeaders)

		sub.HandleFunc("/{service}/*", func(w http.ResponseWriter, r *http.Request) {
			serviceName := chi.URLParam(r, "service") + "-service"
			target, _, ok := registry.Resolve(serviceName)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status":  404,
					"code":    "SERVICE_NOT_FOUND",
					"message": "unknown service: " + serviceName,
				})
				return
			}

			wsPath := strings.TrimPrefix(r.URL.Path, "/ws/v1/"+chi.URLParam(r, "service"))
			if wsPath == "" {
				wsPath = "/"
			}
			r.URL.Path = "/ws" + wsPath

			wsProxy := proxy.NewWebSocketProxy(target, svc.Logger)
			wsProxy.ServeHTTP(w, r)
		})
	})

	svc.Logger.Info().Int("port", cfg.Port).Msg("api-gateway starting")
	if err := svc.Run(ctx); err != nil {
		svc.Logger.Fatal().Err(err).Msg("api-gateway failed")
		os.Exit(1)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
