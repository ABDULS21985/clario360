package visus

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type CTICache struct {
	redis  *redis.Client
	logger zerolog.Logger
	ttl    time.Duration
}

func NewCTICache(redisClient *redis.Client, logger zerolog.Logger) *CTICache {
	return &CTICache{
		redis:  redisClient,
		logger: logger.With().Str("component", "visus_cti_cache").Logger(),
		ttl:    60 * time.Second,
	}
}

func (c *CTICache) GetOrFetch(ctx context.Context, key string, dest interface{}, fetchFn func() (interface{}, error)) error {
	if dest == nil {
		return fmt.Errorf("cti_cache: destination is required")
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("cti_cache: key is required")
	}
	if fetchFn == nil {
		return fmt.Errorf("cti_cache: fetch function is required")
	}

	if c.redis != nil {
		payload, err := c.redis.Get(ctx, key).Bytes()
		switch {
		case err == nil:
			unmarshalErr := json.Unmarshal(payload, dest)
			if unmarshalErr == nil {
				c.logger.Debug().Str("key", key).Msg("cti_cache: cache hit")
				return nil
			}
			c.logger.Warn().Err(unmarshalErr).Str("key", key).Msg("cti_cache: cached payload decode failed")
		case err == redis.Nil:
			c.logger.Debug().Str("key", key).Msg("cti_cache: cache miss")
		default:
			c.logger.Warn().Err(err).Str("key", key).Msg("cti_cache: redis get failed, falling back to fetch")
		}
	}

	result, err := fetchFn()
	if err != nil {
		return err
	}

	raw, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		c.logger.Error().Err(marshalErr).Str("key", key).Msg("cti_cache: marshal fetched payload failed")
		return copyIntoDestination(dest, result)
	}

	if c.redis != nil {
		if setErr := c.redis.Set(ctx, key, raw, c.ttl).Err(); setErr != nil {
			c.logger.Warn().Err(setErr).Str("key", key).Dur("ttl", c.ttl).Msg("cti_cache: redis set failed")
		}
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("cti_cache: decode fetched payload failed")
		return copyIntoDestination(dest, result)
	}
	return nil
}

func (c *CTICache) Invalidate(ctx context.Context, key string) error {
	if c.redis == nil {
		return nil
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("cti_cache: key is required")
	}
	if err := c.redis.Del(ctx, key).Err(); err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("cti_cache: invalidate failed")
		return fmt.Errorf("cti_cache: invalidate %q: %w", key, err)
	}
	return nil
}

func (c *CTICache) InvalidateTenant(ctx context.Context, tenantID string) error {
	if c.redis == nil {
		return nil
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return fmt.Errorf("cti_cache: tenant id is required")
	}

	pattern := fmt.Sprintf("visus:cti:%s:*", tenantID)
	var cursor uint64

	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("cti_cache: scan failed")
			return fmt.Errorf("cti_cache: invalidate tenant scan: %w", err)
		}

		if len(keys) > 0 {
			if err := c.redis.Del(ctx, keys...).Err(); err != nil {
				c.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("cti_cache: delete during tenant invalidation failed")
				return fmt.Errorf("cti_cache: invalidate tenant delete: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}

func copyIntoDestination(dest interface{}, src interface{}) error {
	if dest == nil {
		return fmt.Errorf("cti_cache: destination is required")
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Pointer || destValue.IsNil() {
		return fmt.Errorf("cti_cache: destination must be a non-nil pointer")
	}

	srcValue := reflect.ValueOf(src)
	if !srcValue.IsValid() {
		destValue.Elem().Set(reflect.Zero(destValue.Elem().Type()))
		return nil
	}

	if srcValue.Kind() == reflect.Pointer {
		if srcValue.IsNil() {
			destValue.Elem().Set(reflect.Zero(destValue.Elem().Type()))
			return nil
		}
		srcValue = srcValue.Elem()
	}

	if srcValue.Type().AssignableTo(destValue.Elem().Type()) {
		destValue.Elem().Set(srcValue)
		return nil
	}

	raw, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("cti_cache: marshal for copy: %w", err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("cti_cache: copy into destination: %w", err)
	}
	return nil
}
