package health

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisHealthChecker pings Redis and reports pool statistics.
type RedisHealthChecker struct {
	client *redis.Client
}

// NewRedisHealthChecker creates a Redis health checker.
func NewRedisHealthChecker(client *redis.Client) *RedisHealthChecker {
	return &RedisHealthChecker{client: client}
}

// Name returns "redis".
func (h *RedisHealthChecker) Name() string { return "redis" }

// Check pings Redis and returns pool stats.
func (h *RedisHealthChecker) Check(ctx context.Context) HealthResult {
	if err := h.client.Ping(ctx).Err(); err != nil {
		return HealthResult{
			Status: "unhealthy",
			Error:  fmt.Sprintf("ping failed: %s", err.Error()),
		}
	}

	stats := h.client.PoolStats()
	details := map[string]interface{}{
		"active_connections": stats.TotalConns - stats.IdleConns,
		"idle_connections":   stats.IdleConns,
		"stale_connections":  stats.StaleConns,
	}

	return HealthResult{
		Status:  "healthy",
		Details: details,
	}
}
