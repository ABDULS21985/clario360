package middleware

import (
	"net/http"

	"github.com/clario360/platform/internal/auth"
	mw "github.com/clario360/platform/internal/middleware"
)

// ProxyHeaders adds tracing headers (X-Request-ID, X-Tenant-ID, X-User-ID, X-Forwarded-For)
// to the request before proxying to backend services.
func ProxyHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// X-Request-ID: propagate from outer middleware
		if reqID := mw.GetRequestID(r.Context()); reqID != "" {
			r.Header.Set("X-Request-ID", reqID)
		}

		// X-Tenant-ID and X-User-ID from auth context
		if user := auth.UserFromContext(r.Context()); user != nil {
			r.Header.Set("X-Tenant-ID", user.TenantID)
			r.Header.Set("X-User-ID", user.ID)
		}

		// X-Forwarded-For: append client IP
		clientIP := getClientIP(r)
		if existing := r.Header.Get("X-Forwarded-For"); existing != "" {
			r.Header.Set("X-Forwarded-For", existing+", "+clientIP)
		} else {
			r.Header.Set("X-Forwarded-For", clientIP)
		}

		next.ServeHTTP(w, r)
	})
}
