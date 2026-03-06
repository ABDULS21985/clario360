package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	gwconfig "github.com/clario360/platform/internal/gateway/config"
	"github.com/clario360/platform/internal/gateway/metrics"
	"github.com/clario360/platform/internal/gateway/ratelimit"
)

// ProxyRateLimit applies per-tenant (or per-IP for auth, per-user for ws) sliding window rate limiting.
func ProxyRateLimit(limiter *ratelimit.Limiter, group gwconfig.EndpointGroup, gwMetrics *metrics.GatewayMetrics, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := getReqID(r)

			// Determine the rate-limit key based on the endpoint group.
			var key string
			switch group {
			case gwconfig.EndpointGroupAuth:
				// Pre-auth: rate limit by IP to prevent brute force.
				key = getClientIP(r)
			case gwconfig.EndpointGroupWS:
				// WebSocket: rate limit by user_id (per-connection limit).
				if user := auth.UserFromContext(r.Context()); user != nil {
					key = "user:" + user.ID
				} else {
					key = getClientIP(r)
				}
			default:
				// All other authenticated endpoints: rate limit by tenant_id.
				if user := auth.UserFromContext(r.Context()); user != nil {
					key = user.TenantID
				} else {
					key = getClientIP(r)
				}
			}

			result := limiter.Check(r.Context(), key, group)

			// Always set rate limit response headers.
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				retryAfter := int64(time.Until(result.ResetAt).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))

				if gwMetrics != nil {
					gwMetrics.RateLimitExceeded.WithLabelValues(string(group)).Inc()
				}

				logger.Warn().
					Str("key", key).
					Str("group", string(group)).
					Str("request_id", reqID).
					Msg("rate limit exceeded")

				writeGWError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, please try again later", reqID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getReqID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return ""
}
