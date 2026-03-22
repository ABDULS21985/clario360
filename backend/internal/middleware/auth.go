package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/clario360/platform/internal/auth"
)

// Auth validates the JWT from the Authorization header, extracts claims,
// and populates the request context with user and tenant information.
func Auth(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "MISSING_TOKEN", "authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeAuthError(w, "INVALID_TOKEN_FORMAT", "authorization header must be: Bearer <token>")
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				writeAuthError(w, "INVALID_TOKEN", "token is invalid or expired")
				return
			}

			// Build context user from claims
			ctxUser := &auth.ContextUser{
				ID:        claims.UserID,
				TenantID:  claims.TenantID,
				Email:     claims.Email,
				Roles:     claims.Roles,
				SessionID: claims.SessionID,
			}

			ctx := auth.WithUser(r.Context(), ctxUser)
			ctx = auth.WithTenantID(ctx, claims.TenantID)
			ctx = auth.WithClaims(ctx, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns middleware that checks if the authenticated user
// has the required permission before allowing access.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				writeAuthError(w, "UNAUTHENTICATED", "authentication required")
				return
			}

			if !auth.HasPermission(user.Roles, permission) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status":  403,
					"code":    "FORBIDDEN",
					"message": "you do not have permission to perform this action",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth attempts to extract a JWT but does not reject unauthenticated requests.
func OptionalAuth(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctxUser := &auth.ContextUser{
				ID:       claims.UserID,
				TenantID: claims.TenantID,
				Email:    claims.Email,
				Roles:    claims.Roles,
			}

			ctx := auth.WithUser(r.Context(), ctxUser)
			ctx = auth.WithTenantID(ctx, claims.TenantID)
			ctx = auth.WithClaims(ctx, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeAuthError(w http.ResponseWriter, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  401,
		"code":    code,
		"message": message,
	})
}
