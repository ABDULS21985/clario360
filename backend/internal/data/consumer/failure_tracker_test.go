package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
)

type fakePublisher struct {
	published []*events.Event
	topics    []string
}

func (f *fakePublisher) Publish(_ context.Context, topic string, event *events.Event) error {
	f.topics = append(f.topics, topic)
	f.published = append(f.published, event)
	return nil
}

func newFailureTracker(t *testing.T) (*FailureTracker, *fakePublisher, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})
	publisher := &fakePublisher{}
	return NewFailureTracker(client, events.NewIdempotencyGuard(client, time.Hour), publisher, zerolog.New(nil), nil), publisher, server, client
}

func pipelineEvent(t *testing.T, eventType string, payload map[string]any, eventID string) *events.Event {
	t.Helper()
	event, err := events.NewEvent(eventType, "data-service", "tenant-1", payload)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = eventID
	return event
}

func TestConsecutiveFailures_Under3(t *testing.T) {
	tracker, publisher, _, _ := newFailureTracker(t)
	ctx := context.Background()

	for idx := 0; idx < 2; idx++ {
		err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.failed", map[string]any{
			"pipeline_id":   "pipe-1",
			"pipeline_name": "Daily ETL",
			"status":        "failed",
			"error_message": "boom",
		}, "evt-fail-"+string(rune('a'+idx))))
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(publisher.published); got != 0 {
		t.Fatalf("expected no escalation events, got %d", got)
	}
}

func TestConsecutiveFailures_At3(t *testing.T) {
	tracker, publisher, _, _ := newFailureTracker(t)
	ctx := context.Background()

	for idx := 0; idx < 3; idx++ {
		err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.failed", map[string]any{
			"pipeline_id":   "pipe-1",
			"pipeline_name": "Daily ETL",
			"status":        "failed",
			"error_message": "boom",
		}, "evt-fail-"+string(rune('a'+idx))))
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(publisher.published); got != 1 {
		t.Fatalf("expected 1 escalation event, got %d", got)
	}
	if publisher.published[0].Type != "com.clario360.data.pipeline.consecutive_failures" {
		t.Fatalf("unexpected event type %s", publisher.published[0].Type)
	}
}

func TestConsecutiveFailures_At5(t *testing.T) {
	tracker, publisher, _, _ := newFailureTracker(t)
	ctx := context.Background()

	for idx := 0; idx < 5; idx++ {
		err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.failed", map[string]any{
			"pipeline_id":   "pipe-1",
			"pipeline_name": "Daily ETL",
			"status":        "failed",
			"error_message": "boom",
		}, "evt-fail-"+string(rune('a'+idx))))
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(publisher.published); got != 2 {
		t.Fatalf("expected 2 escalation events, got %d", got)
	}
	if publisher.published[1].Type != "com.clario360.data.pipeline.critical_reliability" {
		t.Fatalf("unexpected event type %s", publisher.published[1].Type)
	}
}

func TestConsecutiveFailures_Reset(t *testing.T) {
	tracker, publisher, _, client := newFailureTracker(t)
	ctx := context.Background()

	for idx := 0; idx < 2; idx++ {
		if err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.failed", map[string]any{
			"pipeline_id":   "pipe-1",
			"pipeline_name": "Daily ETL",
			"status":        "failed",
		}, "evt-fail-"+string(rune('a'+idx)))); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}
	if err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.completed", map[string]any{
		"pipeline_id":   "pipe-1",
		"pipeline_name": "Daily ETL",
		"status":        "completed",
	}, "evt-complete")); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if _, err := client.Get(ctx, "pipeline_failures:tenant-1:pipe-1").Result(); err == nil {
		t.Fatal("expected pipeline failure counter to be reset")
	}
	if got := len(publisher.published); got != 0 {
		t.Fatalf("expected no escalation events, got %d", got)
	}
}

func TestConsecutiveFailures_Recovery(t *testing.T) {
	tracker, publisher, _, client := newFailureTracker(t)
	ctx := context.Background()

	for idx := 0; idx < 3; idx++ {
		if err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.failed", map[string]any{
			"pipeline_id":   "pipe-1",
			"pipeline_name": "Daily ETL",
			"status":        "failed",
			"error_message": "boom",
		}, "evt-fail-"+string(rune('a'+idx)))); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}
	if err := tracker.Handle(ctx, pipelineEvent(t, "data.pipeline.run.completed", map[string]any{
		"pipeline_id":   "pipe-1",
		"pipeline_name": "Daily ETL",
		"status":        "completed",
	}, "evt-complete")); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got := len(publisher.published); got != 1 {
		t.Fatalf("expected 1 escalation event before recovery, got %d", got)
	}
	if _, err := client.Get(ctx, "pipeline_failures:tenant-1:pipe-1").Result(); err == nil {
		t.Fatal("expected pipeline failure counter to be cleared after recovery")
	}
}
