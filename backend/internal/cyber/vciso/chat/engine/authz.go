package engine

import (
	"context"
	"strings"

	"github.com/clario360/platform/internal/auth"
)

func permissionsFromContext(ctx context.Context) []string {
	values := make([]string, 0)
	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		values = append(values, claims.Permissions...)
		for _, role := range claims.Roles {
			normalized := strings.ReplaceAll(role, "-", "_")
			values = append(values, auth.RolePermissions[normalized]...)
		}
	}
	if user := auth.UserFromContext(ctx); user != nil {
		for _, role := range user.Roles {
			normalized := strings.ReplaceAll(role, "-", "_")
			values = append(values, auth.RolePermissions[normalized]...)
		}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func missingPermissions(have, required []string) []string {
	missing := make([]string, 0)
	for _, need := range required {
		if !permissionGranted(have, need) {
			missing = append(missing, need)
		}
	}
	return missing
}

func permissionGranted(have []string, required string) bool {
	for _, item := range have {
		if item == auth.PermAdminAll || item == required {
			return true
		}
		if strings.HasSuffix(item, ":*") && strings.HasPrefix(required, strings.TrimSuffix(item, "*")) {
			return true
		}
	}
	return false
}
