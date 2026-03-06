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
	cfg.Server.Port = 8085

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"cyber-service",
	)

	ctx := context.Background()

	shutdownTracer, err := observability.InitTracer(ctx, "cyber-service", cfg.Observability.OTLPEndpoint)
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

	srv := server.New(cfg, db, rdb, logger)

	srv.Router.Route("/api/v1/cyber", func(r chi.Router) {
		auth := srv.AuthenticatedRoutes()
		auth.Get("/alerts", stubHandler("list_alerts"))
		auth.Get("/alerts/{id}", stubHandler("get_alert"))
		auth.Post("/alerts/{id}/acknowledge", stubHandler("ack_alert"))
		auth.Get("/threats", stubHandler("list_threats"))
		auth.Get("/vulnerabilities", stubHandler("list_vulnerabilities"))
		auth.Get("/dashboard", stubHandler("cyber_dashboard"))
		auth.Get("/reports", stubHandler("list_reports"))
		auth.Post("/scans", stubHandler("trigger_scan"))
		r.Mount("/", auth)
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("cyber-service starting")
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func stubHandler(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service":   "cyber",
			"operation": operation,
			"status":    "not_implemented",
		})
	}
}
