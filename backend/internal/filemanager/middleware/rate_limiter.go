package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
)

// RateLimiter enforces per-tenant upload rate limits using Redis.
func RateLimiter(rdb *redis.Client, maxUploadsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only rate-limit upload endpoints
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			tenantID := auth.TenantFromContext(r.Context())
			if tenantID == "" {
				next.ServeHTTP(w, r)
				return
			}

			key := fmt.Sprintf("file:ratelimit:%s", tenantID)
			allowed, err := checkRateLimit(r.Context(), rdb, key, maxUploadsPerMinute, time.Minute)
			if err != nil {
				// Fail open: if Redis is down, allow the request
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				writeUploadError(w, http.StatusTooManyRequests, "RATE_LIMITED",
					"upload rate limit exceeded, try again later", r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit implements a sliding window counter using Redis INCR + EXPIRE.
func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	pipe := rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := incrCmd.Val()
	return count <= int64(limit), nil
}
