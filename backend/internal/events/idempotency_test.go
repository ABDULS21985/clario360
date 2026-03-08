package events

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	server := miniredis.RunT(t)
	t.Cleanup(server.Close)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestIdempotency_FirstProcess(t *testing.T) {
	guard := NewIdempotencyGuard(newTestRedis(t), time.Hour)

	processed, err := guard.IsProcessed(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if processed {
		t.Fatal("expected first event check to be unprocessed")
	}
}

func TestIdempotency_SecondProcess(t *testing.T) {
	guard := NewIdempotencyGuard(newTestRedis(t), time.Hour)
	ctx := context.Background()

	processed, err := guard.IsProcessed(ctx, "evt-2")
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if processed {
		t.Fatal("expected first event check to be unprocessed")
	}
	if err := guard.MarkProcessed(ctx, "evt-2"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}

	processed, err = guard.IsProcessed(ctx, "evt-2")
	if err != nil {
		t.Fatalf("IsProcessed() second error = %v", err)
	}
	if !processed {
		t.Fatal("expected second event check to be processed")
	}
}

func TestIdempotency_TTLExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	guard := NewIdempotencyGuard(client, 50*time.Millisecond)
	ctx := context.Background()

	processed, err := guard.IsProcessed(ctx, "evt-3")
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if processed {
		t.Fatal("expected first event check to be unprocessed")
	}
	if err := guard.MarkProcessed(ctx, "evt-3"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}

	server.FastForward(75 * time.Millisecond)

	processed, err = guard.IsProcessed(ctx, "evt-3")
	if err != nil {
		t.Fatalf("IsProcessed() after ttl error = %v", err)
	}
	if processed {
		t.Fatal("expected event to expire from idempotency store")
	}
}
