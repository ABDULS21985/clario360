package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	gwconfig "github.com/clario360/platform/internal/gateway/config"
	"github.com/clario360/platform/internal/gateway/ratelimit"
)

// ProxyRateLimit applies per-tenant (or per-IP for auth) sliding window rate limiting.
func ProxyRateLimit(limiter *ratelimit.Limiter, group gwconfig.EndpointGroup, metrics *GatewayMetrics, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine key: IP for auth endpoints, tenant ID for everything else
			key := getClientIP(r)
			if group != gwconfig.EndpointGroupAuth {
				if user := auth.UserFromContext(r.Context()); user != nil {
					key = user.TenantID
				}
			}

			result := limiter.Check(r.Context(), key, group)

			// Always set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(result.ResetAt.Unix(), 10))

				if metrics != nil {
					tenantID := key
					metrics.RateLimitExceeded.WithLabelValues(tenantID, string(group)).Inc()
				}

				logger.Warn().
					Str("key", key).
					Str("group", string(group)).
					Msg("rate limit exceeded")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status":  429,
					"code":    "RATE_LIMITED",
					"message": "too many requests, please try again later",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
