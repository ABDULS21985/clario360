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
	cfg.Server.Port = 8081

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"iam-service",
	)

	ctx := context.Background()

	shutdownTracer, err := observability.InitTracer(ctx, "iam-service", cfg.Observability.OTLPEndpoint)
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

	srv.Router.Route("/api/v1/iam", func(r chi.Router) {
		// Public auth routes
		r.Post("/auth/login", stubHandler("login"))
		r.Post("/auth/refresh", stubHandler("refresh"))

		// Protected routes
		auth := srv.AuthenticatedRoutes()
		auth.Get("/users", stubHandler("list_users"))
		auth.Post("/users", stubHandler("create_user"))
		auth.Get("/users/{id}", stubHandler("get_user"))
		auth.Put("/users/{id}", stubHandler("update_user"))
		auth.Delete("/users/{id}", stubHandler("delete_user"))
		auth.Get("/roles", stubHandler("list_roles"))
		auth.Post("/roles", stubHandler("create_role"))
		auth.Get("/tenants", stubHandler("list_tenants"))
		auth.Post("/tenants", stubHandler("create_tenant"))

		r.Mount("/", auth)
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("iam-service starting")
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func stubHandler(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service":   "iam",
			"operation": operation,
			"status":    "not_implemented",
		})
	}
}
