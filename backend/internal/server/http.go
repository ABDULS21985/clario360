package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/middleware"
)

// Server is the shared HTTP server used by all Clario 360 services.
type Server struct {
	Router     *chi.Mux
	HTTPServer *http.Server
	DB         *pgxpool.Pool
	Redis      *redis.Client
	Logger     zerolog.Logger
	Config     *config.Config
	JWTManager *auth.JWTManager
}

// New creates a new HTTP server with the full middleware stack.
func New(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, logger zerolog.Logger) *Server {
	jwtMgr := auth.NewJWTManager(cfg.Auth)

	r := chi.NewRouter()

	// Middleware stack in specified order:
	// RequestID → Recovery → CORS → Logging → Auth → RateLimit → Tenant
	r.Use(middleware.RequestID)
	r.Use(middleware.RecoveryWithLogger(logger))
	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	r.Use(middleware.Logging(logger))

	// Health check endpoints (no auth required)
	r.Get("/healthz", healthzHandler())
	r.Get("/readyz", readyzHandler(db))

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	srv := &Server{
		Router: r,
		HTTPServer: &http.Server{
			Addr:         cfg.Server.Addr(),
			Handler:      r,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  2 * time.Minute,
		},
		DB:         db,
		Redis:      rdb,
		Logger:     logger,
		Config:     cfg,
		JWTManager: jwtMgr,
	}

	return srv
}

// AuthenticatedRoutes returns a chi.Router with auth, rate limit, and tenant middleware applied.
func (s *Server) AuthenticatedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Auth(s.JWTManager))
	r.Use(middleware.RateLimit(s.Redis, middleware.DefaultRateLimitConfig()))
	r.Use(middleware.Tenant)
	return r
}

// Start begins serving HTTP requests and blocks until shutdown signal is received.
func (s *Server) Start() error {
	// Channel for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Channel for server errors
	errCh := make(chan error, 1)

	go func() {
		s.Logger.Info().
			Str("addr", s.HTTPServer.Addr).
			Msg("starting HTTP server")
		if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for signal or error
	select {
	case sig := <-quit:
		s.Logger.Info().
			Str("signal", sig.String()).
			Msg("shutting down server")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Server.ShutdownTimeout)
	defer cancel()

	if err := s.HTTPServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.Logger.Info().Msg("server stopped gracefully")
	return nil
}

// healthzHandler returns a simple liveness probe.
func healthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// readyzHandler returns a readiness probe that checks database connectivity.
func readyzHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := database.HealthCheck(r.Context(), db); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}
