package security_test

import (
	"context"
	"errors"
	"testing"

	"github.com/clario360/platform/internal/auth"
	security "github.com/clario360/platform/internal/security"
)

// ---------- Viewer cannot access admin functions ----------

func TestBFLA_ViewerCannotAccessAdminFunctions(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "viewer-1",
		TenantID: "tenant-1",
		Email:    "viewer@example.com",
		Roles:    []string{"viewer"},
	})

	err := security.EnforceRole(ctx, "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("viewer should not access tenant_admin functions, got %v", err)
	}
}

func TestBFLA_ViewerCannotAccessSuperAdminFunctions(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "viewer-1",
		TenantID: "tenant-1",
		Email:    "viewer@example.com",
		Roles:    []string{"viewer"},
	})

	// Viewer should not pass super_admin-only checks
	err := security.EnforceRole(ctx, "super_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("viewer should not access super_admin functions, got %v", err)
	}
}

// ---------- Analyst cannot delete resources ----------

func TestBFLA_AnalystCannotDelete(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "analyst-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"analyst"},
	})

	// Analyst should not pass tenant_admin role check (needed for delete operations)
	err := security.EnforceRole(ctx, "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("analyst should not access delete (tenant_admin) functions, got %v", err)
	}
}

// ---------- Role escalation prevention ----------

func TestBFLA_CannotGrantHigherRoleThanOwn(t *testing.T) {
	// Simulate: an analyst tries to call an admin-only function to assign roles.
	// The system should enforce that only admins can manage roles.
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "analyst-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"analyst"},
	})

	// Analyst cannot access role management (requires tenant_admin or super_admin)
	err := security.EnforceRole(ctx, "tenant_admin", "super_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("analyst should not be able to manage roles, got %v", err)
	}
}

func TestBFLA_TenantAdminCannotEscalateToSuperAdmin(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "admin-1",
		TenantID: "tenant-1",
		Email:    "admin@example.com",
		Roles:    []string{"tenant_admin"},
	})

	// tenant_admin should not pass a super_admin-only check
	// Note: EnforceRole with "super_admin" will fail because tenant_admin != super_admin
	// (super_admin bypasses all checks, but tenant_admin does not match "super_admin" directly)
	// The super_admin bypass only works when the user HAS super_admin role.
	err := security.EnforceRole(ctx, "super_admin")
	// tenant_admin should fail because the loop checks role == required || role == "super_admin"
	// tenant_admin != "super_admin" so it doesn't get the bypass
	// But wait — the function checks role == required, so "super_admin" == "super_admin" would match
	// if the user had super_admin. tenant_admin trying to match "super_admin" — should fail.
	if err == nil {
		t.Fatal("tenant_admin should not pass super_admin-only enforcement")
	}
}

// ---------- HasPermission tests ----------

func TestHasPermission_SuperAdmin_HasAllPermissions(t *testing.T) {
	roles := []string{"super_admin"}

	permissions := []string{
		auth.PermUserRead, auth.PermUserWrite, auth.PermUserDelete,
		auth.PermRoleRead, auth.PermRoleWrite,
		auth.PermTenantRead, auth.PermTenantWrite,
		auth.PermAuditRead,
		auth.PermCyberRead, auth.PermCyberWrite,
		auth.PermDataRead, auth.PermDataWrite, auth.PermDataPII,
		auth.PermDataConfidential, auth.PermDataRestricted,
		auth.PermActaRead, auth.PermActaWrite,
		auth.PermLexRead, auth.PermLexWrite,
		auth.PermVisusRead, auth.PermVisusWrite,
	}

	for _, perm := range permissions {
		if !auth.HasPermission(roles, perm) {
			t.Errorf("super_admin should have permission %q", perm)
		}
	}
}

func TestHasPermission_TenantAdmin_HasSpecificPermissions(t *testing.T) {
	roles := []string{"tenant_admin"}

	shouldHave := []string{
		auth.PermUserRead, auth.PermUserWrite, auth.PermUserDelete,
		auth.PermRoleRead, auth.PermRoleWrite,
		auth.PermTenantRead, auth.PermTenantWrite,
		auth.PermAuditRead,
		auth.PermCyberRead, auth.PermCyberWrite,
		auth.PermDataRead, auth.PermDataWrite, auth.PermDataPII,
		auth.PermDataConfidential, auth.PermDataRestricted,
		auth.PermActaRead, auth.PermActaWrite,
		auth.PermLexRead, auth.PermLexWrite,
		auth.PermVisusRead, auth.PermVisusWrite,
	}

	for _, perm := range shouldHave {
		if !auth.HasPermission(roles, perm) {
			t.Errorf("tenant_admin should have permission %q", perm)
		}
	}
}

func TestHasPermission_Analyst_ReadOnlyAccess(t *testing.T) {
	roles := []string{"analyst"}

	shouldHave := []string{
		auth.PermCyberRead, auth.PermDataRead,
		auth.PermActaRead, auth.PermLexRead, auth.PermVisusRead,
		auth.PermAuditRead,
	}

	shouldNotHave := []string{
		auth.PermUserWrite, auth.PermUserDelete,
		auth.PermRoleWrite,
		auth.PermTenantWrite,
		auth.PermCyberWrite, auth.PermDataWrite,
		auth.PermActaWrite, auth.PermLexWrite, auth.PermVisusWrite,
		auth.PermDataPII, auth.PermDataConfidential, auth.PermDataRestricted,
	}

	for _, perm := range shouldHave {
		if !auth.HasPermission(roles, perm) {
			t.Errorf("analyst should have read permission %q", perm)
		}
	}
	for _, perm := range shouldNotHave {
		if auth.HasPermission(roles, perm) {
			t.Errorf("analyst should NOT have write permission %q", perm)
		}
	}
}

func TestHasPermission_Viewer_ReadOnlyWithoutAudit(t *testing.T) {
	roles := []string{"viewer"}

	shouldHave := []string{
		auth.PermCyberRead, auth.PermDataRead,
		auth.PermActaRead, auth.PermLexRead, auth.PermVisusRead,
	}

	shouldNotHave := []string{
		auth.PermAuditRead, // viewer lacks audit read
		auth.PermUserWrite, auth.PermUserDelete,
		auth.PermRoleWrite, auth.PermTenantWrite,
		auth.PermCyberWrite, auth.PermDataWrite,
		auth.PermActaWrite, auth.PermLexWrite, auth.PermVisusWrite,
		auth.PermDataPII, auth.PermDataConfidential, auth.PermDataRestricted,
	}

	for _, perm := range shouldHave {
		if !auth.HasPermission(roles, perm) {
			t.Errorf("viewer should have read permission %q", perm)
		}
	}
	for _, perm := range shouldNotHave {
		if auth.HasPermission(roles, perm) {
			t.Errorf("viewer should NOT have permission %q", perm)
		}
	}
}

// ---------- HasAnyPermission / HasAllPermissions ----------

func TestHasAnyPermission(t *testing.T) {
	// Analyst has cyber:read but NOT cyber:write
	roles := []string{"analyst"}

	if !auth.HasAnyPermission(roles, auth.PermCyberWrite, auth.PermCyberRead) {
		t.Error("analyst should match at least one of cyber:write, cyber:read")
	}

	if auth.HasAnyPermission(roles, auth.PermUserWrite, auth.PermRoleWrite) {
		t.Error("analyst should NOT match any write permissions for user/role")
	}
}

func TestHasAllPermissions(t *testing.T) {
	roles := []string{"analyst"}

	if !auth.HasAllPermissions(roles, auth.PermCyberRead, auth.PermDataRead) {
		t.Error("analyst should have both cyber:read and data:read")
	}

	if auth.HasAllPermissions(roles, auth.PermCyberRead, auth.PermCyberWrite) {
		t.Error("analyst should NOT have both cyber:read and cyber:write")
	}
}

func TestHasPermission_UnknownRole(t *testing.T) {
	roles := []string{"unknown_role"}

	if auth.HasPermission(roles, auth.PermUserRead) {
		t.Error("unknown role should have no permissions")
	}
}

func TestHasPermission_EmptyRoles(t *testing.T) {
	var roles []string

	if auth.HasPermission(roles, auth.PermUserRead) {
		t.Error("empty roles should have no permissions")
	}
}

// ---------- BFLA escalation via approval authority ----------

func TestBFLA_AnalystCannotApproveActions(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "analyst-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"analyst"},
	})

	err := security.EnforceApprovalAuthority(ctx, "other-user", "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("analyst should not be able to approve actions requiring tenant_admin, got %v", err)
	}
}

func TestBFLA_ViewerCannotApproveActions(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "viewer-1",
		TenantID: "tenant-1",
		Email:    "viewer@example.com",
		Roles:    []string{"viewer"},
	})

	err := security.EnforceApprovalAuthority(ctx, "other-user", "tenant_admin")
	if !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("viewer should not be able to approve actions, got %v", err)
	}
}
