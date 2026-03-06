package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// DeadLetterEntry represents a failed event stored for later inspection and replay.
type DeadLetterEntry struct {
	ID              string          `json:"id"`
	OriginalEventID string          `json:"original_event_id"`
	OriginalType    string          `json:"original_type"`
	OriginalTopic   string          `json:"original_topic"`
	TenantID        string          `json:"tenant_id"`
	Error           string          `json:"error"`
	RetryCount      int             `json:"retry_count"`
	EventData       json.RawMessage `json:"event_data"`
	FailedAt        time.Time       `json:"failed_at"`
	Status          string          `json:"status"` // "pending", "replayed", "acknowledged"
}

// DeadLetterStore stores failed events for inspection and replay.
// In production, this would be backed by PostgreSQL. This implementation
// provides the interface and an in-memory store for the DLQ consumer.
type DeadLetterStore struct {
	mu      sync.RWMutex
	entries map[string]*DeadLetterEntry
}

// NewDeadLetterStore creates a new DLQ store.
func NewDeadLetterStore() *DeadLetterStore {
	return &DeadLetterStore{
		entries: make(map[string]*DeadLetterEntry),
	}
}

// Store saves a dead letter entry.
func (s *DeadLetterStore) Store(entry *DeadLetterEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[entry.ID] = entry
}

// Get retrieves a dead letter entry by ID.
func (s *DeadLetterStore) Get(id string) (*DeadLetterEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[id]
	return entry, ok
}

// List returns all dead letter entries with optional filtering.
func (s *DeadLetterStore) List(tenantID, status string, limit, offset int) []*DeadLetterEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*DeadLetterEntry
	for _, entry := range s.entries {
		if tenantID != "" && entry.TenantID != tenantID {
			continue
		}
		if status != "" && entry.Status != status {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Apply pagination
	if offset >= len(filtered) {
		return nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end]
}

// Delete removes a dead letter entry (acknowledge).
func (s *DeadLetterStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.entries[id]
	if ok {
		delete(s.entries, id)
	}
	return ok
}

// MarkReplayed updates the status of a DLQ entry to "replayed".
func (s *DeadLetterStore) MarkReplayed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[id]
	if ok {
		entry.Status = "replayed"
	}
	return ok
}

// Count returns the total number of entries.
func (s *DeadLetterStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// DeadLetterConsumer processes events from the dead letter topic and stores them.
type DeadLetterConsumer struct {
	store    *DeadLetterStore
	producer *Producer
	logger   zerolog.Logger
}

// NewDeadLetterConsumer creates a DLQ consumer that stores failed events.
func NewDeadLetterConsumer(store *DeadLetterStore, producer *Producer, logger zerolog.Logger) *DeadLetterConsumer {
	return &DeadLetterConsumer{
		store:    store,
		producer: producer,
		logger:   logger,
	}
}

// Handle processes a dead letter event by extracting metadata and storing it.
func (c *DeadLetterConsumer) Handle(ctx context.Context, event *Event) error {
	entry := &DeadLetterEntry{
		ID:              event.ID,
		OriginalEventID: event.Metadata["dlq.original_event_id"],
		OriginalType:    event.Metadata["dlq.original_type"],
		TenantID:        event.TenantID,
		Error:           event.Metadata["dlq.error"],
		EventData:       event.Data,
		FailedAt:        event.Time,
		Status:          "pending",
	}

	if event.Metadata["dlq.original_topic"] != "" {
		entry.OriginalTopic = event.Metadata["dlq.original_topic"]
	}

	c.store.Store(entry)

	c.logger.Warn().
		Str("dlq_id", entry.ID).
		Str("original_event_id", entry.OriginalEventID).
		Str("original_type", entry.OriginalType).
		Str("tenant_id", entry.TenantID).
		Str("error", entry.Error).
		Msg("dead letter event stored")

	return nil
}

// Replay re-publishes a dead letter event back to its original topic.
func (c *DeadLetterConsumer) Replay(ctx context.Context, entryID string) error {
	entry, ok := c.store.Get(entryID)
	if !ok {
		return fmt.Errorf("dead letter entry %s not found", entryID)
	}

	if entry.OriginalTopic == "" {
		return fmt.Errorf("original topic not recorded for entry %s", entryID)
	}

	// Reconstruct the original event
	replayEvent := NewEventRaw(
		entry.OriginalType,
		"dead-letter-replay",
		entry.TenantID,
		entry.EventData,
	)
	replayEvent.CausationID = entry.OriginalEventID
	replayEvent.Metadata = map[string]string{
		"dlq.replayed_from": entryID,
		"dlq.replayed_at":   time.Now().UTC().Format(time.RFC3339),
	}

	if err := c.producer.Publish(ctx, entry.OriginalTopic, replayEvent); err != nil {
		return fmt.Errorf("replaying event to %s: %w", entry.OriginalTopic, err)
	}

	c.store.MarkReplayed(entryID)

	c.logger.Info().
		Str("dlq_id", entryID).
		Str("topic", entry.OriginalTopic).
		Str("replay_event_id", replayEvent.ID).
		Msg("dead letter event replayed")

	return nil
}

// Store returns the underlying DeadLetterStore.
func (c *DeadLetterConsumer) Store() *DeadLetterStore {
	return c.store
}
