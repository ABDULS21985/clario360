package middleware

import (
	"net/http"

	"github.com/clario360/platform/internal/auth"
	sharedmw "github.com/clario360/platform/internal/middleware"
)

func TenantGuard(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return sharedmw.Auth(jwtMgr)(sharedmw.Tenant(next))
	}
}
