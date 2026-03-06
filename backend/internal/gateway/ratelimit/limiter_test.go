package ratelimit

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	bgCtx := context.Background()
	if err := rdb.Ping(bgCtx).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}
	rdb.FlushDB(bgCtx)
	t.Cleanup(func() {
		rdb.FlushDB(bgCtx)
		rdb.Close()
	})
	return rdb
}

func TestLimiter_AllowsWithinLimit(t *testing.T) {
	rdb := newTestRedis(t)
	cfg := DefaultConfig()
	limiter := NewLimiter(rdb, cfg)

	ctx := context.Background()
	result := limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupRead)

	if !result.Allowed {
		t.Error("expected request to be allowed within limit")
	}
	if result.Limit != 2000 {
		t.Errorf("expected limit 2000, got %d", result.Limit)
	}
}

func TestLimiter_RejectsOverLimit(t *testing.T) {
	rdb := newTestRedis(t)
	cfg := DefaultConfig()
	// Override auth limits to a very low value for testing
	cfg.Groups[gwconfig.EndpointGroupAuth] = GroupLimit{
		RequestsPerWindow: 3,
		Window:            cfg.Groups[gwconfig.EndpointGroupAuth].Window,
		BurstPerSecond:    1,
	}

	limiter := NewLimiter(rdb, cfg)
	ctx := context.Background()

	// First 3 should be allowed
	for i := 0; i < 3; i++ {
		result := limiter.Check(ctx, "192.168.1.1", gwconfig.EndpointGroupAuth)
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th should be rejected
	result := limiter.Check(ctx, "192.168.1.1", gwconfig.EndpointGroupAuth)
	if result.Allowed {
		t.Error("expected request to be rejected over limit")
	}
	if result.Remaining != 0 {
		t.Errorf("expected remaining 0, got %d", result.Remaining)
	}
}

func TestLimiter_DifferentKeysIndependent(t *testing.T) {
	rdb := newTestRedis(t)
	cfg := DefaultConfig()
	cfg.Groups[gwconfig.EndpointGroupAuth] = GroupLimit{
		RequestsPerWindow: 2,
		Window:            cfg.Groups[gwconfig.EndpointGroupAuth].Window,
		BurstPerSecond:    1,
	}

	limiter := NewLimiter(rdb, cfg)
	ctx := context.Background()

	// Use up tenant-1's limit
	limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupAuth)
	limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupAuth)
	result := limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupAuth)
	if result.Allowed {
		t.Error("expected tenant-1 to be rate limited")
	}

	// tenant-2 should still be allowed
	result2 := limiter.Check(ctx, "tenant-2", gwconfig.EndpointGroupAuth)
	if !result2.Allowed {
		t.Error("expected tenant-2 to be allowed (independent key)")
	}
}

func TestLimiter_DifferentGroups(t *testing.T) {
	rdb := newTestRedis(t)
	cfg := DefaultConfig()
	limiter := NewLimiter(rdb, cfg)
	ctx := context.Background()

	// Read limit should be 2000
	result := limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupRead)
	if result.Limit != 2000 {
		t.Errorf("expected read limit 2000, got %d", result.Limit)
	}

	// Write limit should be 500
	result = limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupWrite)
	if result.Limit != 500 {
		t.Errorf("expected write limit 500, got %d", result.Limit)
	}

	// Admin limit should be 100
	result = limiter.Check(ctx, "tenant-1", gwconfig.EndpointGroupAdmin)
	if result.Limit != 100 {
		t.Errorf("expected admin limit 100, got %d", result.Limit)
	}
}
