package events

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestDLQTracker_IncrementAndCount(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	tracker := NewDLQTracker(client)
	ctx := context.Background()

	if err := tracker.Increment(ctx, "cyber-service", "cyber.alert.events"); err != nil {
		t.Fatalf("increment dlq tracker: %v", err)
	}
	if err := tracker.Increment(ctx, "cyber-service", "cyber.alert.events"); err != nil {
		t.Fatalf("increment dlq tracker: %v", err)
	}

	count, err := tracker.Count(ctx, "cyber-service")
	if err != nil {
		t.Fatalf("count dlq tracker: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}
