package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// ProxyAuth validates JWT tokens at the gateway level for non-public routes.
// It extracts user and tenant information and stores them in the request context.
func ProxyAuth(jwtMgr *auth.JWTManager, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeGatewayAuthError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeGatewayAuthError(w, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "authorization header must be: Bearer <token>")
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				logger.Debug().Err(err).Msg("invalid access token at gateway")
				writeGatewayAuthError(w, http.StatusUnauthorized, "INVALID_TOKEN", "token is invalid or expired")
				return
			}

			// Populate context with user info for downstream middleware
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

func writeGatewayAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  status,
		"code":    code,
		"message": message,
	})
}
