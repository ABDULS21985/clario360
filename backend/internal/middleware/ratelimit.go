package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
)

// RateLimitConfig holds rate limiter configuration.
type RateLimitConfig struct {
	// RequestsPerWindow is the maximum number of requests allowed in the window.
	RequestsPerWindow int
	// Window is the sliding window duration.
	Window time.Duration
	// KeyPrefix is the Redis key prefix for rate limit counters.
	KeyPrefix string
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerWindow: 100,
		Window:            time.Minute,
		KeyPrefix:         "ratelimit",
	}
}

// RateLimit implements per-tenant sliding window rate limiting using Redis.
func RateLimit(rdb *redis.Client, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine the rate limit key: use tenant ID if available, otherwise IP
			key := r.RemoteAddr
			if tenantID := auth.TenantFromContext(r.Context()); tenantID != "" {
				key = tenantID
			}

			redisKey := fmt.Sprintf("%s:%s", cfg.KeyPrefix, key)
			now := time.Now()
			windowStart := now.Add(-cfg.Window)

			ctx := r.Context()

			allowed, remaining, err := slidingWindowCheck(ctx, rdb, redisKey, now, windowStart, cfg.RequestsPerWindow, cfg.Window)
			if err != nil {
				// On Redis failure, allow the request (fail-open)
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(cfg.Window).Unix(), 10))

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(int(cfg.Window.Seconds())))
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

// slidingWindowCheck implements the Redis sorted set sliding window algorithm.
func slidingWindowCheck(ctx context.Context, rdb *redis.Client, key string, now, windowStart time.Time, limit int, window time.Duration) (bool, int, error) {
	pipe := rdb.Pipeline()

	// Remove entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixMicro(), 10))

	// Count current entries in window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	member := fmt.Sprintf("%d", now.UnixNano())
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMicro()),
		Member: member,
	})

	// Set TTL on the key
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
