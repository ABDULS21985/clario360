package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/auth"
)

// RateLimiter creates a per-tenant rate limiting middleware backed by Redis.
// Uses a sliding window counter algorithm.
// If Redis is unavailable, fails open (allows the request) but logs a warning.
func RateLimiter(rdb *redis.Client, requestsPerMinute int, logger zerolog.Logger) func(http.Handler) http.Handler {
	window := time.Minute

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				next.ServeHTTP(w, r)
				return
			}

			tenantID := auth.TenantFromContext(r.Context())
			if tenantID == "" {
				tenantID = user.TenantID
			}

			key := fmt.Sprintf("audit:ratelimit:%s", tenantID)
			now := time.Now()
			windowStart := now.Add(-window)

			allowed, remaining, err := checkRateLimit(r.Context(), rdb, key, now, windowStart, requestsPerMinute, window)
			if err != nil {
				// Fail open on Redis errors
				logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("rate limiter Redis failure — allowing request")
				metrics.RateLimitRedisFailures.Inc()
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			resetTime := now.Add(window)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			if !allowed {
				metrics.RateLimitRejected.WithLabelValues(tenantID).Inc()
				retryAfter := int(window.Seconds())
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "RATE_LIMITED",
						"message": "too many requests — please retry later",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit implements a sliding window counter using Redis sorted sets.
func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, now, windowStart time.Time, limit int, window time.Duration) (bool, int, error) {
	pipe := rdb.Pipeline()

	// Remove entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current entries
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set TTL to window duration
	pipe.Expire(ctx, key, window+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := int(countCmd.Val())
	remaining := limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	return count < limit, remaining, nil
}
