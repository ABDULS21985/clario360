package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	gwconfig "github.com/clario360/platform/internal/gateway/config"
	"github.com/clario360/platform/internal/gateway/health"
	gwmw "github.com/clario360/platform/internal/gateway/middleware"
	"github.com/clario360/platform/internal/gateway/proxy"
	"github.com/clario360/platform/internal/gateway/ratelimit"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}
	cfg.Server.Port = 8080

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"api-gateway",
	)

	ctx := context.Background()

	// Initialize tracing
	shutdownTracer, err := observability.InitTracer(ctx, "api-gateway", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer, continuing without tracing")
	} else {
		defer shutdownTracer(ctx)
	}

	// Connect to Redis (required for rate limiting)
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to redis")
	}

	// ---- JWT Manager (local token validation) ----
	jwtMgr, err := auth.NewJWTManager(cfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// ---- Service Registry ----
	registry, err := proxy.NewServiceRegistry(gwconfig.DefaultServices())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create service registry")
	}

	// ---- Proxy Router with circuit breakers ----
	routes := gwconfig.DefaultRoutes()
	cbCfg := proxy.DefaultCircuitBreakerConfig()
	proxyRouter, err := proxy.NewRouter(routes, registry, cbCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create proxy router")
	}

	// ---- Rate Limiter ----
	limiter := ratelimit.NewLimiter(rdb, ratelimit.DefaultConfig())

	// ---- Gateway Metrics ----
	gwMetrics := gwmw.NewGatewayMetrics()

	// ---- Health Checker ----
	healthChecker := health.NewChecker(registry, proxyRouter, logger)

	// ---- Build Chi Router ----
	r := chi.NewRouter()

	// Global middleware: RequestID → Recovery → CORS → Logging
	r.Use(middleware.RequestID)
	r.Use(middleware.RecoveryWithLogger(logger))
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"https://*.clario360.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID", "X-Tenant-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))
	r.Use(middleware.Logging(logger))

	// ---- Infrastructure endpoints (no auth) ----
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	r.Get("/readyz", healthChecker.Handler())
	r.Handle("/metrics", promhttp.Handler())

	// ---- Gateway status endpoint ----
	r.Get("/api/v1/gateway/status", func(w http.ResponseWriter, r *http.Request) {
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

	// ---- Register proxy routes ----
	for _, route := range routes {
		route := route

		match := proxyRouter.Match(route.Prefix)
		if !match.Matched {
			logger.Warn().Str("prefix", route.Prefix).Msg("no proxy found for route, skipping")
			continue
		}

		rp := match.Proxy

		r.Route(route.Prefix, func(sub chi.Router) {
			if !route.Public {
				sub.Use(gwmw.ProxyAuth(jwtMgr, logger))
			} else {
				sub.Use(middleware.OptionalAuth(jwtMgr))
			}

			sub.Use(gwmw.ProxyHeaders)
			sub.Use(gwmw.ProxyRateLimit(limiter, route.EndpointGroup, gwMetrics, logger))
			sub.Use(gwmw.ProxyMetrics(gwMetrics, route.Service))
			sub.Use(gwmw.ProxyLogging(logger, route.Service))

			sub.HandleFunc("/*", rp.ServeHTTP)
			sub.HandleFunc("/", rp.ServeHTTP)
		})
	}

	// ---- WebSocket proxy routes ----
	r.Route("/ws/v1", func(sub chi.Router) {
		sub.Use(gwmw.ProxyAuth(jwtMgr, logger))
		sub.Use(gwmw.ProxyHeaders)

		sub.HandleFunc("/{service}/*", func(w http.ResponseWriter, r *http.Request) {
			serviceName := chi.URLParam(r, "service") + "-service"
			target, ok := registry.Resolve(serviceName)
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

			wsProxy := proxy.NewWebSocketProxy(target, logger)
			wsProxy.ServeHTTP(w, r)
		})
	})

	// ---- Start server with graceful shutdown ----
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  2 * time.Minute,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", srv.Addr).Msg("api-gateway starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case sig := <-quit:
		logger.Info().Str("signal", sig.String()).Msg("shutting down api-gateway")
	case err := <-errCh:
		logger.Fatal().Err(err).Msg("api-gateway failed")
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().Err(err).Msg("api-gateway shutdown failed")
		os.Exit(1)
	}

	logger.Info().Msg("api-gateway stopped gracefully")
}
