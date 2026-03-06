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
	cfg.Server.Port = 8087

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"acta-service",
	)

	ctx := context.Background()

	shutdownTracer, err := observability.InitTracer(ctx, "acta-service", cfg.Observability.OTLPEndpoint)
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

	srv.Router.Route("/api/v1/acta", func(r chi.Router) {
		auth := srv.AuthenticatedRoutes()
		auth.Get("/documents", stubHandler("list_documents"))
		auth.Post("/documents", stubHandler("create_document"))
		auth.Get("/documents/{id}", stubHandler("get_document"))
		auth.Put("/documents/{id}", stubHandler("update_document"))
		auth.Delete("/documents/{id}", stubHandler("delete_document"))
		auth.Post("/documents/{id}/sign", stubHandler("sign_document"))
		auth.Get("/templates", stubHandler("list_templates"))
		auth.Post("/templates", stubHandler("create_template"))
		r.Mount("/", auth)
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("acta-service starting")
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func stubHandler(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service":   "acta",
			"operation": operation,
			"status":    "not_implemented",
		})
	}
}
