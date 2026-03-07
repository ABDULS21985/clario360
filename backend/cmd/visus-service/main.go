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
	cfg.Server.Port = 8089

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"visus-service",
	)

	ctx := context.Background()

	shutdownTracer, err := observability.InitTracer(ctx, "visus-service", cfg.Observability.OTLPEndpoint)
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

	srv.Router.Route("/api/v1/visus", func(r chi.Router) {
		auth := srv.AuthenticatedRoutes()
		auth.Get("/dashboards", notImplementedHandler("list_dashboards"))
		auth.Post("/dashboards", notImplementedHandler("create_dashboard"))
		auth.Get("/dashboards/{id}", notImplementedHandler("get_dashboard"))
		auth.Put("/dashboards/{id}", notImplementedHandler("update_dashboard"))
		auth.Get("/reports", notImplementedHandler("list_reports"))
		auth.Post("/reports", notImplementedHandler("create_report"))
		auth.Post("/reports/{id}/generate", notImplementedHandler("generate_report"))
		auth.Get("/widgets", notImplementedHandler("list_widgets"))
		r.Mount("/", auth)
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("visus-service starting")
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
				"message": "visus-service endpoint is not implemented",
				"details": map[string]string{
					"service":   "visus",
					"operation": operation,
				},
			},
		})
	}
}
