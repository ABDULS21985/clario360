package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/clario360/platform/internal/auth"
)

const TenantIDHeader = "X-Tenant-ID"

// Tenant extracts the tenant ID from the authenticated user context or from
// the X-Tenant-ID header and stores it in the request context. If neither
// source provides a tenant ID, the request is rejected.
func Tenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First, try to get tenant from auth context (set by Auth middleware)
		tenantID := auth.TenantFromContext(r.Context())

		// Fall back to X-Tenant-ID header (for service-to-service calls)
		if tenantID == "" {
			tenantID = r.Header.Get(TenantIDHeader)
		}

		if tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  400,
				"code":    "MISSING_TENANT",
				"message": "tenant context is required",
			})
			return
		}

		// Ensure the tenant ID is in context even if it came from header
		ctx := auth.WithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TenantOptional extracts tenant from context/header but does not reject
// requests without a tenant (useful for super-admin endpoints).
func TenantOptional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := auth.TenantFromContext(r.Context())
		if tenantID == "" {
			tenantID = r.Header.Get(TenantIDHeader)
		}
		if tenantID != "" {
			ctx := auth.WithTenantID(r.Context(), tenantID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
