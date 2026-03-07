package middleware

import (
	"net/http"

	"github.com/clario360/platform/internal/auth"
	sharedmw "github.com/clario360/platform/internal/middleware"
)

// TenantGuard applies the shared authentication and tenant extraction middleware.
func TenantGuard(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return sharedmw.Auth(jwtMgr)(sharedmw.Tenant(next))
	}
}
