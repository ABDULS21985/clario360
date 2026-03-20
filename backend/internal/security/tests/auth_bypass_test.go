package security_test

import (
	"context"
	"errors"
	"testing"

	"github.com/clario360/platform/internal/auth"
	security "github.com/clario360/platform/internal/security"
)

// ---------- EnforceRole ----------

func TestEnforceRole_NoUserInContext(t *testing.T) {
	ctx := context.Background()
	err := security.EnforceRole(ctx, "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestEnforceRole_InsufficientRole(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "viewer@example.com",
		Roles:    []string{"viewer"},
	})

	err := security.EnforceRole(ctx, "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for viewer trying tenant_admin, got %v", err)
	}
}

func TestEnforceRole_SuperAdmin_BypassesAnyRole(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "admin-1",
		TenantID: "tenant-1",
		Email:    "admin@example.com",
		Roles:    []string{"super_admin"},
	})

	// super_admin should pass any required role check
	for _, role := range []string{"tenant_admin", "analyst", "viewer", "some_unknown_role"} {
		if err := security.EnforceRole(ctx, role); err != nil {
			t.Errorf("super_admin should pass EnforceRole(%q), got %v", role, err)
		}
	}
}

func TestEnforceRole_MatchingRole(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"analyst"},
	})

	if err := security.EnforceRole(ctx, "analyst"); err != nil {
		t.Fatalf("expected nil for matching role, got %v", err)
	}
}

func TestEnforceRole_MultipleRequiredRoles(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"analyst"},
	})

	// Should pass if any of the required roles match
	if err := security.EnforceRole(ctx, "tenant_admin", "analyst"); err != nil {
		t.Fatalf("expected nil when one of multiple required roles matches, got %v", err)
	}

	// Should fail if none of the required roles match
	err := security.EnforceRole(ctx, "tenant_admin", "super_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when no required roles match, got %v", err)
	}
}

// ---------- EnforceResourceOwner ----------

func TestEnforceResourceOwner_DifferentUser(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Roles:    []string{"analyst"},
	})

	err := security.EnforceResourceOwner(ctx, "user-2")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner, got %v", err)
	}
}

func TestEnforceResourceOwner_MatchingUser(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Roles:    []string{"analyst"},
	})

	if err := security.EnforceResourceOwner(ctx, "user-1"); err != nil {
		t.Fatalf("expected nil for resource owner, got %v", err)
	}
}

func TestEnforceResourceOwner_AdminRole(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{"super_admin bypasses ownership", "super_admin"},
		{"tenant_admin bypasses ownership", "tenant_admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := auth.WithUser(context.Background(), &auth.ContextUser{
				ID:       "admin-1",
				TenantID: "tenant-1",
				Email:    "admin@example.com",
				Roles:    []string{tt.role},
			})

			// Admin should access resources created by someone else
			if err := security.EnforceResourceOwner(ctx, "other-user-id"); err != nil {
				t.Fatalf("expected %s to bypass ownership check, got %v", tt.role, err)
			}
		})
	}
}

func TestEnforceResourceOwner_NoUserInContext(t *testing.T) {
	ctx := context.Background()
	err := security.EnforceResourceOwner(ctx, "user-1")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden with no user in context, got %v", err)
	}
}

// ---------- EnforceApprovalAuthority ----------

func TestEnforceApprovalAuthority_SameSubmitter(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Roles:    []string{"tenant_admin"},
	})

	err := security.EnforceApprovalAuthority(ctx, "user-1", "tenant_admin")
	if err == nil {
		t.Fatal("expected error for segregation of duties violation, got nil")
	}
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestEnforceApprovalAuthority_ValidApprover(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "approver-1",
		TenantID: "tenant-1",
		Email:    "approver@example.com",
		Roles:    []string{"tenant_admin"},
	})

	if err := security.EnforceApprovalAuthority(ctx, "submitter-1", "tenant_admin"); err != nil {
		t.Fatalf("expected nil for valid approver, got %v", err)
	}
}

func TestEnforceApprovalAuthority_NoUserInContext(t *testing.T) {
	ctx := context.Background()
	err := security.EnforceApprovalAuthority(ctx, "submitter-1", "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden with no user in context, got %v", err)
	}
}

func TestEnforceApprovalAuthority_WrongRole(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-2",
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Roles:    []string{"viewer"},
	})

	err := security.EnforceApprovalAuthority(ctx, "user-1", "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for wrong role, got %v", err)
	}
}

// ---------- ValidateUUID ----------

func TestValidateUUID_InvalidFormat(t *testing.T) {
	invalids := []string{
		"not-a-uuid",
		"12345",
		"",
		"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
		"123e4567-e89b-12d3-a456", // truncated
		"' OR 1=1 --",
		"<script>alert(1)</script>",
	}

	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			err := security.ValidateUUID(input)
			if !errors.Is(err, security.ErrInvalidUUID) {
				t.Errorf("expected ErrInvalidUUID for %q, got %v", input, err)
			}
		})
	}
}

func TestValidateUUID_ValidUUID(t *testing.T) {
	valids := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
	}

	for _, input := range valids {
		t.Run(input, func(t *testing.T) {
			if err := security.ValidateUUID(input); err != nil {
				t.Errorf("expected nil for valid UUID %q, got %v", input, err)
			}
		})
	}
}
