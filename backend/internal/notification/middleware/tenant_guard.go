package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/clario360/platform/internal/auth"
)

// TenantGuard extracts the tenant_id from the JWT and enforces tenant isolation.
// Super-admin callers may override tenant_id via query parameter.
func TenantGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
			return
		}

		tenantID := user.TenantID
		if tenantID == "" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "tenant context missing")
			return
		}

		if isSuperAdmin(user.Roles) {
			if overrideTenant := r.URL.Query().Get("tenant_id"); overrideTenant != "" {
				tenantID = overrideTenant
			}
		}

		ctx := auth.WithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isSuperAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "super_admin" {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}
