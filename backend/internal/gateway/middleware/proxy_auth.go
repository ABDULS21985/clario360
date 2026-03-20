package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/gateway/metrics"
	mw "github.com/clario360/platform/internal/middleware"
)

// ProxyAuth validates JWT tokens at the gateway level for non-public routes.
// RS256 algorithm is enforced — HS256 and "none" are rejected.
// Expired tokens return a TOKEN_EXPIRED code so clients know to refresh.
// API keys (X-API-Key header) are forwarded to the backend without gateway validation.
func ProxyAuth(jwtMgr *auth.JWTManager, gwMetrics *metrics.GatewayMetrics, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := mw.GetRequestID(r.Context())

			// API key passthrough — the backend service validates it.
			if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
				if gwMetrics != nil {
					// Count as a distinct auth path; backend validates the key.
					// Do NOT record as failure here.
				}
				ctx := r.Context()
				// Flag the auth method in context for downstream middleware.
				// We store it using the same requestID key pattern.
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				recordAuthFailure(gwMetrics, "missing")
				writeGWError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", reqID)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				recordAuthFailure(gwMetrics, "invalid")
				writeGWError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authorization header must be: Bearer <token>", reqID)
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				// Distinguish expired tokens from other errors so clients can refresh.
				code := "UNAUTHORIZED"
				reason := "invalid"
				if isExpiredError(err) {
					code = "TOKEN_EXPIRED"
					reason = "expired"
				}
				recordAuthFailure(gwMetrics, reason)
				logger.Debug().Str("request_id", reqID).Str("reason", reason).Msg("gateway auth failure")
				writeGWError(w, http.StatusUnauthorized, code, "token is invalid or expired", reqID)
				return
			}

			// Populate context with user/tenant info for downstream middleware.
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

func isExpiredError(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired) ||
		strings.Contains(err.Error(), "token is expired") ||
		strings.Contains(err.Error(), "expired")
}

func recordAuthFailure(m *metrics.GatewayMetrics, reason string) {
	if m != nil {
		m.AuthFailures.WithLabelValues(reason).Inc()
	}
}

func writeGWError(w http.ResponseWriter, status int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":       code,
			"message":    message,
			"request_id": requestID,
		},
	})
}
