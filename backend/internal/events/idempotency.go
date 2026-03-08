package events

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultProcessedTTL = 24 * time.Hour
	defaultLockTTL      = 5 * time.Minute
)

// IdempotencyGuard prevents duplicate event processing while still allowing retries
// when a handler fails before completion.
//
// A plain SETNX against the processed key would drop legitimate retries after a
// mid-handler failure. This guard therefore uses a short-lived processing lock plus
// a durable processed marker.
type IdempotencyGuard struct {
	redis         *redis.Client
	ttl           time.Duration
	processingTTL time.Duration
}

// NewIdempotencyGuard creates a Redis-backed idempotency guard.
func NewIdempotencyGuard(redis *redis.Client, ttl time.Duration) *IdempotencyGuard {
	if ttl <= 0 {
		ttl = defaultProcessedTTL
	}
	return &IdempotencyGuard{
		redis:         redis,
		ttl:           ttl,
		processingTTL: defaultLockTTL,
	}
}

// IsProcessed returns true when an event is already processed or currently locked
// by another handler instance. A false result means the caller acquired the right
// to process the event and must eventually call MarkProcessed or Release.
func (g *IdempotencyGuard) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	if g == nil || g.redis == nil {
		return false, nil
	}

	processed, err := g.redis.Exists(ctx, g.processedKey(eventID)).Result()
	if err != nil {
		return false, fmt.Errorf("check processed event: %w", err)
	}
	if processed > 0 {
		return true, nil
	}

	acquired, err := g.redis.SetNX(ctx, g.processingKey(eventID), "1", g.processingTTL).Result()
	if err != nil {
		return false, fmt.Errorf("acquire processing lock: %w", err)
	}

	return !acquired, nil
}

// MarkProcessed records an event as successfully processed and clears the
// transient processing lock.
func (g *IdempotencyGuard) MarkProcessed(ctx context.Context, eventID string) error {
	if g == nil || g.redis == nil {
		return nil
	}

	pipe := g.redis.TxPipeline()
	pipe.Set(ctx, g.processedKey(eventID), "1", g.ttl)
	pipe.Del(ctx, g.processingKey(eventID))
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("mark processed event: %w", err)
	}
	return nil
}

// Release clears the transient processing lock without marking the event as
// processed so the event can be retried.
func (g *IdempotencyGuard) Release(ctx context.Context, eventID string) error {
	if g == nil || g.redis == nil {
		return nil
	}
	if err := g.redis.Del(ctx, g.processingKey(eventID)).Err(); err != nil {
		return fmt.Errorf("release processing lock: %w", err)
	}
	return nil
}

func (g *IdempotencyGuard) processedKey(eventID string) string {
	return fmt.Sprintf("event:processed:%s", eventID)
}

func (g *IdempotencyGuard) processingKey(eventID string) string {
	return fmt.Sprintf("event:processing:%s", eventID)
}
