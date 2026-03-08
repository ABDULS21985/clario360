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

	"github.com/clario360/platform/internal/auth"
)

// RateLimiter applies a per-tenant rate limit for cyber endpoints.
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

			now := time.Now().UTC()
			allowed, remaining, err := checkRateLimit(r.Context(), rdb, fmt.Sprintf("cyber:ratelimit:%s", tenantID), now, now.Add(-window), requestsPerMinute, window)
			if err != nil {
				logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("cyber rate limiter Redis failure — allowing request")
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(window).Unix(), 10))

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    "RATE_LIMITED",
					"message": "too many requests — please retry later",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, now, windowStart time.Time, limit int, window time.Duration) (bool, int, error) {
	if rdb == nil {
		return true, limit, nil
	}

	pipe := rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart.UnixNano()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: fmt.Sprintf("%d", now.UnixNano())})
	pipe.Expire(ctx, key, window+time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, err
	}

	count := int(countCmd.Val())
	remaining := limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	return count < limit, remaining, nil
}
