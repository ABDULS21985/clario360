package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/cyber/model"
)

const enrichmentCacheTTL = 24 * time.Hour

// EnrichmentCache provides a Redis-backed cache for indicator enrichment data.
// All methods are nil-safe and fail-open: if the cache is unavailable the caller
// simply falls through to the live enrichment path.
type EnrichmentCache struct {
	rdb *redis.Client
}

// NewEnrichmentCache creates a new EnrichmentCache. A nil redis.Client is safe.
func NewEnrichmentCache(rdb *redis.Client) *EnrichmentCache {
	return &EnrichmentCache{rdb: rdb}
}

// Get returns cached enrichment data. The second return value is true on a cache hit.
func (c *EnrichmentCache) Get(ctx context.Context, tenantID, indicatorID uuid.UUID) (*model.IndicatorEnrichment, bool, error) {
	if c == nil || c.rdb == nil {
		return nil, false, nil
	}
	payload, err := c.rdb.Get(ctx, enrichmentCacheKey(tenantID, indicatorID)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var enrichment model.IndicatorEnrichment
	if err := json.Unmarshal(payload, &enrichment); err != nil {
		return nil, false, err
	}
	return &enrichment, true, nil
}

// Set stores enrichment data in the cache.
func (c *EnrichmentCache) Set(ctx context.Context, tenantID, indicatorID uuid.UUID, data *model.IndicatorEnrichment) error {
	if c == nil || c.rdb == nil || data == nil {
		return nil
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, enrichmentCacheKey(tenantID, indicatorID), payload, enrichmentCacheTTL).Err()
}

// Invalidate removes cached enrichment data for an indicator.
func (c *EnrichmentCache) Invalidate(ctx context.Context, tenantID, indicatorID uuid.UUID) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, enrichmentCacheKey(tenantID, indicatorID)).Err()
}

func enrichmentCacheKey(tenantID, indicatorID uuid.UUID) string {
	return "cyber:enrichment:" + tenantID.String() + ":" + indicatorID.String()
}
