package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/data/dto"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewCache(rdb *redis.Client, ttl time.Duration) *Cache {
	return &Cache{rdb: rdb, ttl: ttl}
}

func (c *Cache) Get(ctx context.Context, tenantID uuid.UUID) (*dto.DataSuiteDashboard, error) {
	if c == nil || c.rdb == nil {
		return nil, redis.Nil
	}
	payload, err := c.rdb.Get(ctx, c.key(tenantID)).Bytes()
	if err != nil {
		return nil, err
	}
	var dashboard dto.DataSuiteDashboard
	if err := json.Unmarshal(payload, &dashboard); err != nil {
		return nil, fmt.Errorf("decode data dashboard cache: %w", err)
	}
	return &dashboard, nil
}

func (c *Cache) Set(ctx context.Context, tenantID uuid.UUID, dashboard *dto.DataSuiteDashboard) error {
	if c == nil || c.rdb == nil || dashboard == nil {
		return nil
	}
	payload, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("encode data dashboard cache: %w", err)
	}
	return c.rdb.Set(ctx, c.key(tenantID), payload, c.ttl).Err()
}

func (c *Cache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, c.key(tenantID)).Err()
}

func (c *Cache) key(tenantID uuid.UUID) string {
	return "data:dashboard:" + tenantID.String()
}

