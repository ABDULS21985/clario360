package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}

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

	// Connect to PostgreSQL
	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to redis")
	}

	// Create server
	srv := server.New(cfg, db, rdb, logger)

	// API routes (authenticated)
	srv.Router.Route("/api/v1", func(r chi.Router) {
		// Mount authenticated sub-router
		auth := srv.AuthenticatedRoutes()

		auth.Get("/me", meHandler())

		// Service proxy routes
		auth.Route("/cyber", func(r chi.Router) {
			r.Get("/", placeholderHandler("cyber"))
		})
		auth.Route("/data", func(r chi.Router) {
			r.Get("/", placeholderHandler("data"))
		})
		auth.Route("/acta", func(r chi.Router) {
			r.Get("/", placeholderHandler("acta"))
		})
		auth.Route("/lex", func(r chi.Router) {
			r.Get("/", placeholderHandler("lex"))
		})
		auth.Route("/visus", func(r chi.Router) {
			r.Get("/", placeholderHandler("visus"))
		})

		r.Mount("/", auth)
	})

	logger.Info().Str("addr", cfg.Server.Addr()).Msg("api-gateway starting")

	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "authenticated user endpoint",
		})
	}
}

func placeholderHandler(suite string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"suite":  suite,
			"status": "operational",
		})
	}
}
