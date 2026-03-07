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
	cfg.Server.Port = 8086

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"data-service",
	)

	ctx := context.Background()

	shutdownTracer, err := observability.InitTracer(ctx, "data-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	srv, err := server.New(cfg, db, rdb, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	srv.Router.Route("/api/v1/data", func(r chi.Router) {
		auth := srv.AuthenticatedRoutes()
		auth.Get("/sources", notImplementedHandler("list_sources"))
		auth.Post("/sources", notImplementedHandler("create_source"))
		auth.Get("/sources/{id}", notImplementedHandler("get_source"))
		auth.Get("/pipelines", notImplementedHandler("list_pipelines"))
		auth.Post("/pipelines", notImplementedHandler("create_pipeline"))
		auth.Post("/pipelines/{id}/run", notImplementedHandler("run_pipeline"))
		auth.Get("/datasets", notImplementedHandler("list_datasets"))
		auth.Get("/quality", notImplementedHandler("data_quality_dashboard"))
		r.Mount("/", auth)
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("data-service starting")
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func notImplementedHandler(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "NOT_IMPLEMENTED",
				"message": "data-service endpoint is not implemented",
				"details": map[string]string{
					"service":   "data",
					"operation": operation,
				},
			},
		})
	}
}
