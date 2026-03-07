package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	actasuite "github.com/clario360/platform/internal/acta"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}
	cfg.Server.Port = 8086
	cfg.Database.Name = "acta_db"

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

	migrationsPath := "migrations/acta_db"
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "acta_db")
	}
	if err := database.RunMigrations(cfg.Database.DSN(), migrationsPath); err != nil {
		logger.Fatal().Err(err).Str("path", migrationsPath).Msg("failed to run acta-service migrations")
	}

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
		actasuite.MountRoutes(auth, db, logger)
		r.Mount("/", auth)
	})

	logger.Info().Int("port", cfg.Server.Port).Msg("acta-service starting")
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("server failed")
		os.Exit(1)
	}
}
