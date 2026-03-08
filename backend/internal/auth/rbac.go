package auth

import "strings"

// Permission constants follow the pattern "resource:action".
const (
	PermUserRead         = "user:read"
	PermUserWrite        = "user:write"
	PermUserDelete       = "user:delete"
	PermRoleRead         = "role:read"
	PermRoleWrite        = "role:write"
	PermTenantRead       = "tenant:read"
	PermTenantWrite      = "tenant:write"
	PermAuditRead        = "audit:read"
	PermCyberRead        = "cyber:read"
	PermCyberWrite       = "cyber:write"
	PermDataRead         = "data:read"
	PermDataWrite        = "data:write"
	PermDataPII          = "data:pii"
	PermDataConfidential = "data:confidential"
	PermDataRestricted   = "data:restricted"
	PermActaRead         = "acta:read"
	PermActaWrite        = "acta:write"
	PermLexRead          = "lex:read"
	PermLexWrite         = "lex:write"
	PermVisusRead        = "visus:read"
	PermVisusWrite       = "visus:write"
	PermAdminAll         = "admin:*"
)

// RolePermissions maps built-in roles to their permissions.
var RolePermissions = map[string][]string{
	"super_admin": {PermAdminAll},
	"tenant_admin": {
		PermUserRead, PermUserWrite, PermUserDelete,
		PermRoleRead, PermRoleWrite,
		PermTenantRead, PermTenantWrite,
		PermAuditRead,
		PermCyberRead, PermCyberWrite,
		PermDataRead, PermDataWrite, PermDataPII, PermDataConfidential, PermDataRestricted,
		PermActaRead, PermActaWrite,
		PermLexRead, PermLexWrite,
		PermVisusRead, PermVisusWrite,
	},
	"analyst": {
		PermCyberRead, PermDataRead, PermActaRead, PermLexRead, PermVisusRead,
		PermAuditRead,
	},
	"viewer": {
		PermCyberRead, PermDataRead, PermActaRead, PermLexRead, PermVisusRead,
	},
}

// HasPermission checks if any of the user's roles grant the required permission.
func HasPermission(roles []string, required string) bool {
	for _, role := range roles {
		normalizedRole := strings.ReplaceAll(role, "-", "_")
		perms, ok := RolePermissions[normalizedRole]
		if !ok {
			continue
		}
		for _, perm := range perms {
			if perm == PermAdminAll {
				return true
			}
			if perm == required {
				return true
			}
			// Check wildcard: "resource:*" matches "resource:read"
			if strings.HasSuffix(perm, ":*") {
				prefix := strings.TrimSuffix(perm, "*")
				if strings.HasPrefix(required, prefix) {
					return true
				}
			}
		}
	}
	return false
}

// HasAnyPermission checks if any of the user's roles grant at least one of the required permissions.
func HasAnyPermission(roles []string, required ...string) bool {
	for _, perm := range required {
		if HasPermission(roles, perm) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the user's roles grant all of the required permissions.
func HasAllPermissions(roles []string, required ...string) bool {
	for _, perm := range required {
		if !HasPermission(roles, perm) {
			return false
		}
	}
	return true
}
