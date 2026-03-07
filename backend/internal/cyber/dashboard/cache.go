package dashboard

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/cyber/model"
)

const dashboardCacheTTL = 60 * time.Second

type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

func (c *Cache) Get(ctx context.Context, tenantID uuid.UUID) (*model.SOCDashboard, bool, error) {
	if c == nil || c.rdb == nil {
		return nil, false, nil
	}
	payload, err := c.rdb.Get(ctx, dashboardCacheKey(tenantID)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var dashboard model.SOCDashboard
	if err := json.Unmarshal(payload, &dashboard); err != nil {
		return nil, false, err
	}
	now := time.Now().UTC()
	dashboard.CachedAt = &now
	return &dashboard, true, nil
}

func (c *Cache) Set(ctx context.Context, tenantID uuid.UUID, dashboard *model.SOCDashboard) error {
	if c == nil || c.rdb == nil || dashboard == nil {
		return nil
	}
	payload, err := json.Marshal(dashboard)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, dashboardCacheKey(tenantID), payload, dashboardCacheTTL).Err()
}

func (c *Cache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, dashboardCacheKey(tenantID)).Err()
}

func dashboardCacheKey(tenantID uuid.UUID) string {
	return "cyber:dashboard:" + tenantID.String()
}
