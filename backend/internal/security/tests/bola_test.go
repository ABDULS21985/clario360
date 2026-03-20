package security_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	security "github.com/clario360/platform/internal/security"
)

func newTestChecker(t *testing.T) *security.APISecurityChecker {
	t.Helper()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	// nil pgxpool — we only test validation paths that don't reach the DB
	return security.NewAPISecurityChecker(nil, metrics, logger)
}

func TestVerifyOwnership_NonWhitelistedTable(t *testing.T) {
	checker := newTestChecker(t)

	ctx := auth.WithTenantID(context.Background(), uuid.New().String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: uuid.New().String(),
		Roles:    []string{"analyst"},
	})

	err := checker.VerifyOwnership(ctx, "evil_table", uuid.New())
	if !errors.Is(err, security.ErrInvalidTable) {
		t.Fatalf("expected ErrInvalidTable for non-whitelisted table, got %v", err)
	}
}

func TestVerifyOwnership_SQLInjectionInTableName(t *testing.T) {
	checker := newTestChecker(t)

	ctx := auth.WithTenantID(context.Background(), uuid.New().String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: uuid.New().String(),
		Roles:    []string{"analyst"},
	})

	maliciousTables := []string{
		"users; DROP TABLE users --",
		"users UNION SELECT * FROM secrets",
		"../../../etc/passwd",
		"<script>alert(1)</script>",
	}

	for _, table := range maliciousTables {
		t.Run(table, func(t *testing.T) {
			err := checker.VerifyOwnership(ctx, table, uuid.New())
			if !errors.Is(err, security.ErrInvalidTable) {
				t.Errorf("expected ErrInvalidTable for malicious table name %q, got %v", table, err)
			}
		})
	}
}

func TestVerifyOwnership_EmptyTenantID(t *testing.T) {
	checker := newTestChecker(t)

	// Context with no tenant ID set
	ctx := context.Background()
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:    "user-1",
		Roles: []string{"analyst"},
	})

	err := checker.VerifyOwnership(ctx, "assets", uuid.New())
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for empty tenant ID, got %v", err)
	}
}

func TestVerifyOwnership_InvalidTenantUUID(t *testing.T) {
	checker := newTestChecker(t)

	// Context with non-UUID tenant ID
	ctx := auth.WithTenantID(context.Background(), "not-a-uuid")
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: "not-a-uuid",
		Roles:    []string{"analyst"},
	})

	err := checker.VerifyOwnership(ctx, "assets", uuid.New())
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for invalid tenant UUID, got %v", err)
	}
}

func TestAllowedTables_ContainsExpectedTables(t *testing.T) {
	// Verify the whitelist indirectly: whitelisted tables should NOT return ErrInvalidTable.
	// Use a context with NO tenant ID so VerifyOwnership returns ErrForbidden before
	// reaching the database pool (avoids nil pool panic).
	checker := newTestChecker(t)

	// No tenant ID in context — will get ErrForbidden, but NOT ErrInvalidTable
	ctx := context.Background()

	expectedTables := []string{
		"assets", "vulnerabilities", "alerts", "threats",
		"threat_indicators", "detection_rules",
		"ctem_assessments", "ctem_findings", "ctem_remediation_groups",
		"remediation_actions", "dspm_data_assets", "dspm_scans",
		"vciso_briefings", "asset_relationships", "security_events",
		"users", "roles", "audit_logs",
		"workflow_definitions", "workflow_instances", "human_tasks",
		"notifications",
	}

	for _, table := range expectedTables {
		t.Run(table, func(t *testing.T) {
			err := checker.VerifyOwnership(ctx, table, uuid.New())
			// Should NOT be ErrInvalidTable — table is whitelisted
			if errors.Is(err, security.ErrInvalidTable) {
				t.Errorf("table %q should be whitelisted but got ErrInvalidTable", table)
			}
			// We expect ErrForbidden (empty tenant ID), which confirms the table
			// passed the whitelist check and proceeded to the tenant check
			if !errors.Is(err, security.ErrForbidden) {
				t.Errorf("expected ErrForbidden for whitelisted table %q with no tenant, got %v", table, err)
			}
		})
	}
}

func TestPreventMassAssignment_ForbiddenFields(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)

	forbidden := []string{
		"id", "tenant_id", "created_at", "updated_at", "deleted_at",
		"created_by", "updated_by", "password_hash", "mfa_secret",
		"mfa_recovery_codes", "refresh_token_hash", "api_key_hash",
		"is_superadmin", "is_system",
	}

	for _, field := range forbidden {
		t.Run(field, func(t *testing.T) {
			body := map[string]interface{}{
				field: "malicious_value",
			}
			err := security.PreventMassAssignment([]string{"name", "email"}, body, metrics)
			if !errors.Is(err, security.ErrForbiddenField) {
				t.Errorf("expected ErrForbiddenField for %q, got %v", field, err)
			}
		})
	}
}

func TestPreventMassAssignment_UnknownFields(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)

	body := map[string]interface{}{
		"role": "super_admin", // not in allowed list
	}
	err := security.PreventMassAssignment([]string{"name", "email"}, body, metrics)
	if !errors.Is(err, security.ErrUnknownField) {
		t.Fatalf("expected ErrUnknownField for unknown field, got %v", err)
	}
}

func TestPreventMassAssignment_AllowedFields(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)

	body := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	}
	err := security.PreventMassAssignment([]string{"name", "email"}, body, metrics)
	if err != nil {
		t.Fatalf("expected nil for allowed fields, got %v", err)
	}
}
