package security_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	security "github.com/clario360/platform/internal/security"
)

func setupMockPool(t *testing.T) (pgxmock.PgxPoolIface, *security.APISecurityChecker) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock pool: %v", err)
	}

	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	checker := security.NewAPISecurityCheckerWithPool(mock, metrics, logger)
	return mock, checker
}

func TestVerifyOwnership_ResourceExists(t *testing.T) {
	mock, checker := setupMockPool(t)
	defer mock.Close()

	tenantID := uuid.New()
	resourceID := uuid.New()

	ctx := auth.WithTenantID(context.Background(), tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: tenantID.String(),
		Roles:    []string{"analyst"},
	})

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(resourceID, tenantID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	err := checker.VerifyOwnership(ctx, "assets", resourceID)
	if err != nil {
		t.Fatalf("expected nil for existing resource, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestVerifyOwnership_ResourceNotFound(t *testing.T) {
	mock, checker := setupMockPool(t)
	defer mock.Close()

	tenantID := uuid.New()
	resourceID := uuid.New()

	ctx := auth.WithTenantID(context.Background(), tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: tenantID.String(),
		Roles:    []string{"analyst"},
	})

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(resourceID, tenantID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	err := checker.VerifyOwnership(ctx, "assets", resourceID)
	if !errors.Is(err, security.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-existent resource, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestVerifyOwnership_CrossTenantAccess(t *testing.T) {
	mock, checker := setupMockPool(t)
	defer mock.Close()

	tenantA := uuid.New()
	tenantB := uuid.New()
	resourceID := uuid.New()

	// User is in tenant A
	ctx := auth.WithTenantID(context.Background(), tenantA.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: tenantA.String(),
		Roles:    []string{"analyst"},
	})

	// Resource belongs to tenant B — the query uses tenant A's ID
	// so it should return false (no match)
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(resourceID, tenantA).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	err := checker.VerifyOwnership(ctx, "assets", resourceID)
	// Should return 404, NOT 403 — to prevent info disclosure
	if !errors.Is(err, security.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-tenant access, got %v", err)
	}

	_ = tenantB // tenantB only for conceptual clarity

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestVerifyOwnership_DatabaseError(t *testing.T) {
	mock, checker := setupMockPool(t)
	defer mock.Close()

	tenantID := uuid.New()
	resourceID := uuid.New()

	ctx := auth.WithTenantID(context.Background(), tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: tenantID.String(),
		Roles:    []string{"analyst"},
	})

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(resourceID, tenantID).
		WillReturnError(pgx.ErrNoRows)

	err := checker.VerifyOwnership(ctx, "assets", resourceID)
	// On DB error, we return ErrNotFound to prevent info disclosure
	if !errors.Is(err, security.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on DB error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestVerifyOwnership_MultipleTables(t *testing.T) {
	tables := []string{"assets", "vulnerabilities", "alerts", "threats", "users", "audit_logs", "notifications"}

	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			mock, checker := setupMockPool(t)
			defer mock.Close()

			tenantID := uuid.New()
			resourceID := uuid.New()

			ctx := auth.WithTenantID(context.Background(), tenantID.String())
			ctx = auth.WithUser(ctx, &auth.ContextUser{
				ID:       "user-1",
				TenantID: tenantID.String(),
				Roles:    []string{"analyst"},
			})

			mock.ExpectQuery(`SELECT EXISTS`).
				WithArgs(resourceID, tenantID).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

			err := checker.VerifyOwnership(ctx, table, resourceID)
			if err != nil {
				t.Fatalf("expected nil for whitelisted table %q, got %v", table, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

func TestVerifyOwnership_SoftDeletedResource(t *testing.T) {
	mock, checker := setupMockPool(t)
	defer mock.Close()

	tenantID := uuid.New()
	resourceID := uuid.New()

	ctx := auth.WithTenantID(context.Background(), tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: tenantID.String(),
		Roles:    []string{"analyst"},
	})

	// Query includes "AND deleted_at IS NULL", so soft-deleted resources return false
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(resourceID, tenantID).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	err := checker.VerifyOwnership(ctx, "assets", resourceID)
	if !errors.Is(err, security.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for soft-deleted resource, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
