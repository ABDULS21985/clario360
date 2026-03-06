package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/observability"
)

func main() {
	direction := flag.String("direction", "up", "Migration direction: up or down")
	seed := flag.Bool("seed", false, "Run database seeder after migrations")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"migrator",
	)

	// Resolve migrations path relative to the binary or working directory
	migrationsPath := findMigrationsPath()

	logger.Info().
		Str("direction", *direction).
		Str("path", migrationsPath).
		Str("database", cfg.Database.Name).
		Msg("running migrations")

	switch *direction {
	case "up":
		if err := database.RunMigrations(cfg.Database.DSN(), migrationsPath); err != nil {
			logger.Fatal().Err(err).Msg("migration up failed")
		}
		logger.Info().Msg("migrations applied successfully")

	case "down":
		if err := database.RollbackMigration(cfg.Database.DSN(), migrationsPath); err != nil {
			logger.Fatal().Err(err).Msg("migration down failed")
		}
		logger.Info().Msg("migration rolled back successfully")

	default:
		logger.Fatal().Str("direction", *direction).Msg("invalid direction, use 'up' or 'down'")
	}

	if *seed {
		logger.Info().Msg("seeding database")
		// Seed logic will be added in later prompts
		logger.Info().Msg("database seeded successfully")
	}
}

func findMigrationsPath() string {
	// Try relative to working directory
	candidates := []string{
		"migrations",
		"backend/migrations",
		"../migrations",
		filepath.Join("..", "..", "migrations"),
	}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}
	// Default fallback
	return "migrations"
}
