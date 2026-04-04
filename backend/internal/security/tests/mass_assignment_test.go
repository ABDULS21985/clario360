package security_test

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	security "github.com/clario360/platform/internal/security"
)

func newTestMetrics() *security.Metrics {
	reg := prometheus.NewRegistry()
	return security.NewMetrics(reg)
}

func TestMassAssignment_ForbiddenField_ID(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email", "role"}
	body := map[string]interface{}{
		"name": "Alice",
		"id":   "injected-uuid",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'id', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_ForbiddenField_TenantID(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":      "Alice",
		"tenant_id": "stolen-tenant",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'tenant_id', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_ForbiddenField_PasswordHash(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":          "Alice",
		"password_hash": "$2a$10$attackhash",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'password_hash', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_ForbiddenField_CreatedAt(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":       "Alice",
		"created_at": "2020-01-01T00:00:00Z",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'created_at', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_ForbiddenField_IsSuperadmin(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":          "Alice",
		"is_superadmin": true,
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'is_superadmin', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_UnknownField(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":           "Alice",
		"evil_privilege": "admin",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for unknown field 'evil_privilege', got nil")
	}
	if !errors.Is(err, security.ErrUnknownField) {
		t.Fatalf("expected ErrUnknownField, got: %v", err)
	}
}

func TestMassAssignment_AllAllowedFields(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email", "role", "description"}
	body := map[string]interface{}{
		"name":        "Alice",
		"email":       "alice@example.com",
		"role":        "analyst",
		"description": "Security analyst",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err != nil {
		t.Fatalf("expected all allowed fields to pass, got: %v", err)
	}
}

func TestMassAssignment_SubsetOfAllowedFields(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email", "role", "description"}
	body := map[string]interface{}{
		"name": "Alice",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err != nil {
		t.Fatalf("expected subset of allowed fields to pass, got: %v", err)
	}
}

func TestMassAssignment_CaseInsensitive_TenantID(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}

	// The forbidden fields check normalizes to lowercase, so these
	// case variants should all be rejected.
	caseVariants := []string{"Tenant_ID", "TENANT_ID", "Tenant_Id", "tenant_ID"}

	for _, variant := range caseVariants {
		body := map[string]interface{}{
			"name":  "Alice",
			variant: "injected",
		}
		err := security.PreventMassAssignment(allowed, body, metrics)
		if err == nil {
			t.Fatalf("expected error for case variant %q of forbidden field, got nil", variant)
		}
		if !errors.Is(err, security.ErrForbiddenField) {
			t.Fatalf("expected ErrForbiddenField for %q, got: %v", variant, err)
		}
	}
}

func TestMassAssignment_CaseInsensitive_IsSuperadmin(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}

	body := map[string]interface{}{
		"name":          "Alice",
		"IS_SUPERADMIN": true,
	}
	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for 'IS_SUPERADMIN' case variant, got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_EmptyBody(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err != nil {
		t.Fatalf("expected empty body to pass, got: %v", err)
	}
}

func TestMassAssignment_NilMetrics(t *testing.T) {
	// PreventMassAssignment should work even with nil metrics
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":      "Alice",
		"tenant_id": "injected",
	}

	err := security.PreventMassAssignment(allowed, body, nil)
	if err == nil {
		t.Fatal("expected error for forbidden field even with nil metrics, got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_ForbiddenField_MFASecret(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":       "Alice",
		"mfa_secret": "JBSWY3DPEHPK3PXP",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'mfa_secret', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}

func TestMassAssignment_ForbiddenField_RefreshTokenHash(t *testing.T) {
	metrics := newTestMetrics()
	allowed := []string{"name", "email"}
	body := map[string]interface{}{
		"name":               "Alice",
		"refresh_token_hash": "stolen_hash",
	}

	err := security.PreventMassAssignment(allowed, body, metrics)
	if err == nil {
		t.Fatal("expected error for forbidden field 'refresh_token_hash', got nil")
	}
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Fatalf("expected ErrForbiddenField, got: %v", err)
	}
}
