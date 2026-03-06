package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

// Result holds the outcome of a rate limit check.
type Result struct {
	Allowed   bool
	Limit     int
	Remaining int
	ResetAt   time.Time
}

// Limiter implements Redis-based sliding window rate limiting.
type Limiter struct {
	rdb    *redis.Client
	config Config
}

// NewLimiter creates a new rate limiter.
func NewLimiter(rdb *redis.Client, cfg Config) *Limiter {
	return &Limiter{rdb: rdb, config: cfg}
}

// Check determines whether a request should be allowed.
// For auth endpoints, key is the IP address. For other endpoints, key is the tenant ID.
func (l *Limiter) Check(ctx context.Context, key string, group gwconfig.EndpointGroup) Result {
	limit := l.config.GetLimit(group)
	redisKey := fmt.Sprintf("gw_rl:%s:%s", group, key)

	now := time.Now()
	windowStart := now.Add(-limit.Window)
	resetAt := now.Add(limit.Window)

	// Fail open when Redis is unavailable (nil client or error).
	if l.rdb == nil {
		return Result{
			Allowed:   true,
			Limit:     limit.RequestsPerWindow,
			Remaining: limit.RequestsPerWindow,
			ResetAt:   resetAt,
		}
	}

	allowed, remaining, err := l.slidingWindowCheck(ctx, redisKey, now, windowStart, limit.RequestsPerWindow, limit.Window)
	if err != nil {
		// Fail open on Redis error
		return Result{
			Allowed:   true,
			Limit:     limit.RequestsPerWindow,
			Remaining: limit.RequestsPerWindow,
			ResetAt:   resetAt,
		}
	}

	return Result{
		Allowed:   allowed,
		Limit:     limit.RequestsPerWindow,
		Remaining: remaining,
		ResetAt:   resetAt,
	}
}

// slidingWindowCheck uses Redis sorted sets for precise sliding window rate limiting.
func (l *Limiter) slidingWindowCheck(ctx context.Context, key string, now, windowStart time.Time, limit int, window time.Duration) (bool, int, error) {
	pipe := l.rdb.Pipeline()

	// Remove entries outside the current window
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixMicro(), 10))

	// Count entries in the current window
	countCmd := pipe.ZCard(ctx, key)

	// Add the current request
	member := fmt.Sprintf("%d", now.UnixNano())
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMicro()),
		Member: member,
	})

	// Set TTL to auto-cleanup
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
