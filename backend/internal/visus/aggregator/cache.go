package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/repository"
)

type SuiteCache struct {
	redis *redis.Client
	db    *repository.SuiteCacheRepository
	ttl   time.Duration
	log   zerolog.Logger
}

func NewSuiteCache(redisClient *redis.Client, repo *repository.SuiteCacheRepository, ttl time.Duration, logger zerolog.Logger) *SuiteCache {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &SuiteCache{
		redis: redisClient,
		db:    repo,
		ttl:   ttl,
		log:   logger.With().Str("component", "visus_suite_cache").Logger(),
	}
}

func (c *SuiteCache) Get(ctx context.Context, tenantID uuid.UUID, suite, endpoint string) (map[string]any, bool, error) {
	key := c.redisKey(tenantID, suite, endpoint)
	if c.redis != nil {
		if payload, err := c.redis.Get(ctx, key).Bytes(); err == nil {
			out := map[string]any{}
			if unmarshalErr := json.Unmarshal(payload, &out); unmarshalErr == nil {
				return out, true, nil
			}
		}
	}
	if c.db == nil {
		return nil, false, nil
	}
	record, err := c.db.Get(ctx, tenantID, suite, endpoint)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if time.Since(record.FetchedAt) > time.Duration(record.TTLSeconds)*time.Second {
		return nil, false, nil
	}
	return record.ResponseData, true, nil
}

func (c *SuiteCache) Set(ctx context.Context, tenantID uuid.UUID, suite, endpoint string, payload map[string]any, latency time.Duration) error {
	key := c.redisKey(tenantID, suite, endpoint)
	if c.redis != nil {
		raw, err := json.Marshal(payload)
		if err == nil {
			_ = c.redis.Set(ctx, key, raw, c.ttl).Err()
		}
	}
	if c.db == nil {
		return nil
	}
	latencyMS := int(latency / time.Millisecond)
	return c.db.Upsert(ctx, &repository.SuiteCacheRecord{
		TenantID:       tenantID,
		Suite:          suite,
		Endpoint:       endpoint,
		ResponseData:   payload,
		FetchedAt:      time.Now().UTC(),
		TTLSeconds:     int(c.ttl / time.Second),
		FetchLatencyMS: &latencyMS,
	})
}

func (c *SuiteCache) redisKey(tenantID uuid.UUID, suite, endpoint string) string {
	return fmt.Sprintf("visus:suite-cache:%s:%s:%s", tenantID.String(), suite, endpoint)
}
