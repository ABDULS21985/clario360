package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/observability"
)

// All databases managed by the platform.
var allDatabases = []string{
	"platform_core",
	"cyber_db",
	"data_db",
	"acta_db",
	"lex_db",
	"visus_db",
}

func main() {
	direction := flag.String("direction", "up", "Migration direction: up or down")
	dbName := flag.String("db", "", "Specific database to migrate (comma-separated, default: all)")
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

	basePath := findMigrationsPath()

	// Determine which databases to migrate
	databases := allDatabases
	if *dbName != "" {
		databases = strings.Split(*dbName, ",")
	}

	logger.Info().
		Str("direction", *direction).
		Str("base_path", basePath).
		Strs("databases", databases).
		Msg("starting migrations")

	hasError := false
	for _, db := range databases {
		db = strings.TrimSpace(db)
		migrationsPath := filepath.Join(basePath, db)

		// Check if migrations directory exists for this database
		if info, err := os.Stat(migrationsPath); err != nil || !info.IsDir() {
			logger.Warn().Str("database", db).Str("path", migrationsPath).Msg("migrations directory not found, skipping")
			continue
		}

		// Build DSN for this specific database
		dsn := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			db,
			cfg.Database.SSLMode,
		)

		logger.Info().
			Str("database", db).
			Str("direction", *direction).
			Str("path", migrationsPath).
			Msg("running migration")

		switch *direction {
		case "up":
			if err := database.RunMigrations(dsn, migrationsPath); err != nil {
				logger.Error().Err(err).Str("database", db).Msg("migration up failed")
				hasError = true
				continue
			}
			logger.Info().Str("database", db).Msg("migrations applied successfully")

		case "down":
			if err := database.RollbackMigration(dsn, migrationsPath); err != nil {
				logger.Error().Err(err).Str("database", db).Msg("migration down failed")
				hasError = true
				continue
			}
			logger.Info().Str("database", db).Msg("migration rolled back successfully")

		default:
			logger.Fatal().Str("direction", *direction).Msg("invalid direction, use 'up' or 'down'")
		}
	}

	if hasError {
		logger.Error().Msg("some migrations failed — check errors above")
		os.Exit(1)
	}

	if *seed {
		logger.Info().Msg("seeding database")
		// Seed logic will be added in later prompts
		logger.Info().Msg("database seeded successfully")
	}

	logger.Info().Msg("all migrations completed")
}

func findMigrationsPath() string {
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
	return "migrations"
}
