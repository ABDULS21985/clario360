//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/clario360/platform/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestCompatibilityMigration_CreatesSchemaWithoutBackfill(t *testing.T) {
	testcontainersSkipIfUnavailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("visus_compat"),
		postgresmod.WithUsername("visus"),
		postgresmod.WithPassword("visus"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	dbURL := container.MustConnectionString(ctx, "sslmode=disable")
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	if _, err := db.Exec(ctx, legacyVisusSchemaSQL); err != nil {
		t.Fatalf("create legacy visus schema: %v", err)
	}

	tenantID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333301")
	if _, err := db.Exec(ctx, `
		INSERT INTO dashboards (tenant_id, name, description, layout, is_default, owner_user_id, shared_with, created_by, updated_by)
		VALUES ($1, $2, $3, '{}'::jsonb, true, $4, '[]'::jsonb, $4, $4)`,
		tenantID, "Legacy Executive Dashboard", "Legacy dashboard row", userID,
	); err != nil {
		t.Fatalf("insert legacy dashboard: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO kpi_definitions (tenant_id, name, description, suite, query_config, created_by, updated_by)
		VALUES ($1, $2, $3, $4, '{}'::jsonb, $5, $5)`,
		tenantID, "Legacy KPI", "Legacy KPI row", "lex", userID,
	); err != nil {
		t.Fatalf("insert legacy kpi: %v", err)
	}

	tempDir := t.TempDir()
	compatBytes, err := os.ReadFile(filepath.Join(visusMigrationsPath(), "000002_modular_schema_compat.up.sql"))
	if err != nil {
		t.Fatalf("read compatibility migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "000002_modular_schema_compat.up.sql"), compatBytes, 0o644); err != nil {
		t.Fatalf("write compatibility migration: %v", err)
	}
	if err := database.RunMigrations(dbURL, tempDir); err != nil {
		t.Fatalf("run compatibility migration: %v", err)
	}

	assertTableExists(t, ctx, db, "visus_dashboards")
	assertTableExists(t, ctx, db, "visus_kpi_definitions")

	if got := countRows(t, ctx, db, "dashboards"); got != 1 {
		t.Fatalf("expected legacy dashboards to retain 1 row, got %d", got)
	}
	if got := countRows(t, ctx, db, "kpi_definitions"); got != 1 {
		t.Fatalf("expected legacy kpi_definitions to retain 1 row, got %d", got)
	}
	if got := countRows(t, ctx, db, "visus_dashboards"); got != 0 {
		t.Fatalf("expected no backfilled rows in visus_dashboards, got %d", got)
	}
	if got := countRows(t, ctx, db, "visus_kpi_definitions"); got != 0 {
		t.Fatalf("expected no backfilled rows in visus_kpi_definitions, got %d", got)
	}
}

func assertTableExists(t *testing.T, ctx context.Context, db *pgxpool.Pool, table string) {
	t.Helper()
	var exists bool
	if err := db.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists); err != nil {
		t.Fatalf("check table %s: %v", table, err)
	}
	if !exists {
		t.Fatalf("expected table %s to exist", table)
	}
}

func countRows(t *testing.T, ctx context.Context, db *pgxpool.Pool, table string) int {
	t.Helper()
	var count int
	query := `SELECT COUNT(*) FROM ` + table
	if err := db.QueryRow(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	return count
}

const legacyVisusSchemaSQL = `
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE dashboards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    layout        JSONB NOT NULL DEFAULT '{}',
    is_default    BOOLEAN NOT NULL DEFAULT false,
    owner_user_id UUID,
    shared_with   JSONB DEFAULT '[]',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by    UUID,
    updated_by    UUID
);

CREATE TABLE kpi_definitions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    suite        TEXT NOT NULL,
    query_config JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by   UUID,
    updated_by   UUID
);
`
