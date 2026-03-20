package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestDeadLetterStore_StorAndGet(t *testing.T) {
	store := NewDeadLetterStore()

	entry := &DeadLetterEntry{
		ID:              "dlq-1",
		OriginalEventID: "evt-1",
		OriginalType:    "com.clario360.test",
		TenantID:        "t1",
		Error:           "handler failed",
		EventData:       json.RawMessage(`{"key":"value"}`),
		FailedAt:        time.Now().UTC(),
		Status:          "pending",
	}

	store.Store(entry)

	got, ok := store.Get("dlq-1")
	if !ok {
		t.Fatal("expected to find entry")
	}
	if got.OriginalEventID != "evt-1" {
		t.Errorf("expected original event ID evt-1, got %s", got.OriginalEventID)
	}
	if got.Status != "pending" {
		t.Errorf("expected status pending, got %s", got.Status)
	}
}

func TestDeadLetterStore_Get_NotFound(t *testing.T) {
	store := NewDeadLetterStore()
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestDeadLetterStore_List(t *testing.T) {
	store := NewDeadLetterStore()

	for i := 0; i < 5; i++ {
		store.Store(&DeadLetterEntry{
			ID:       GenerateUUID(),
			TenantID: "t1",
			Status:   "pending",
		})
	}
	store.Store(&DeadLetterEntry{
		ID:       GenerateUUID(),
		TenantID: "t2",
		Status:   "pending",
	})

	// List all
	all := store.List("", "", 100, 0)
	if len(all) != 6 {
		t.Errorf("expected 6 entries, got %d", len(all))
	}

	// Filter by tenant
	t1Entries := store.List("t1", "", 100, 0)
	if len(t1Entries) != 5 {
		t.Errorf("expected 5 entries for t1, got %d", len(t1Entries))
	}

	// Pagination
	page := store.List("", "", 3, 0)
	if len(page) != 3 {
		t.Errorf("expected 3 entries in page, got %d", len(page))
	}
}

func TestDeadLetterStore_Delete(t *testing.T) {
	store := NewDeadLetterStore()
	store.Store(&DeadLetterEntry{ID: "to-delete", TenantID: "t1", Status: "pending"})

	if !store.Delete("to-delete") {
		t.Error("expected delete to succeed")
	}
	if store.Delete("to-delete") {
		t.Error("expected second delete to fail (already deleted)")
	}
	if store.Count() != 0 {
		t.Errorf("expected 0 entries after delete, got %d", store.Count())
	}
}

func TestDeadLetterStore_MarkReplayed(t *testing.T) {
	store := NewDeadLetterStore()
	store.Store(&DeadLetterEntry{ID: "replay-me", TenantID: "t1", Status: "pending"})

	if !store.MarkReplayed("replay-me") {
		t.Error("expected mark replayed to succeed")
	}

	entry, _ := store.Get("replay-me")
	if entry.Status != "replayed" {
		t.Errorf("expected status replayed, got %s", entry.Status)
	}
}

func TestDeadLetterConsumer_Handle(t *testing.T) {
	store := NewDeadLetterStore()
	// producer is nil — we won't call Replay in this test
	consumer := NewDeadLetterConsumer(store, nil, testLogger())

	event := &Event{
		ID:       "dlq-evt-1",
		Type:     "com.clario360.test.failed",
		TenantID: "t1",
		Time:     time.Now().UTC(),
		Data:     json.RawMessage(`{"test": true}`),
		Metadata: map[string]string{
			"dlq.original_event_id": "orig-1",
			"dlq.original_type":     "com.clario360.test",
			"dlq.error":             "handler failed",
			"dlq.failed_at":         time.Now().UTC().Format(time.RFC3339),
			"dlq.retry_count":       "3",
		},
	}

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	entry, ok := store.Get("dlq-evt-1")
	if !ok {
		t.Fatal("expected entry to be stored")
	}
	if entry.OriginalEventID != "orig-1" {
		t.Errorf("expected original event ID orig-1, got %s", entry.OriginalEventID)
	}
	if entry.Error != "handler failed" {
		t.Errorf("expected error 'handler failed', got %s", entry.Error)
	}
	if entry.Status != "pending" {
		t.Errorf("expected status pending, got %s", entry.Status)
	}
}

func TestDeadLetterConsumer_HandleWrappedEvent(t *testing.T) {
	store := NewDeadLetterStore()
	consumer := NewDeadLetterConsumer(store, nil, testLogger())

	original := &Event{
		ID:       "orig-evt-1",
		Type:     "com.clario360.test",
		TenantID: "t1",
		Data:     json.RawMessage(`{"key":"value"}`),
	}
	payload, err := json.Marshal(map[string]any{
		"original_event": original,
		"error":          "handler failed",
		"retry_count":    3,
		"timestamp":      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal wrapped payload: %v", err)
	}

	event := &Event{
		ID:       "dlq-evt-2",
		Type:     "com.clario360.test",
		TenantID: "t1",
		Time:     time.Now().UTC(),
		Data:     payload,
		Metadata: map[string]string{
			"dlq.original_topic": "cyber.alert.events",
		},
	}

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	entry, ok := store.Get("dlq-evt-2")
	if !ok {
		t.Fatal("expected entry to be stored")
	}
	if entry.OriginalEventID != "orig-evt-1" {
		t.Fatalf("expected original event id orig-evt-1, got %s", entry.OriginalEventID)
	}
	if string(entry.EventData) != `{"key":"value"}` {
		t.Fatalf("expected original event data to be stored, got %s", string(entry.EventData))
	}
	if entry.RetryCount != 3 {
		t.Fatalf("expected retry count 3, got %d", entry.RetryCount)
	}
}
