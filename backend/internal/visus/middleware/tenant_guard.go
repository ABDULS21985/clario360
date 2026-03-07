package middleware

import (
	"net/http"

	sharedmw "github.com/clario360/platform/internal/middleware"
)

func TenantGuard(next http.Handler) http.Handler {
	return sharedmw.Tenant(next)
}
