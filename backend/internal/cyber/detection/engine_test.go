package detection

import (
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestNormalizeEvents_InitializesMatchedRules(t *testing.T) {
	tenantID := uuid.New()
	events := normalizeEvents(tenantID, []model.SecurityEvent{
		{},
		{MatchedRules: []uuid.UUID{uuid.New()}},
	})

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].MatchedRules == nil {
		t.Fatal("expected first event matched rules to be initialized")
	}
	if len(events[0].MatchedRules) != 0 {
		t.Fatalf("expected first event matched rules to be empty, got %d", len(events[0].MatchedRules))
	}
	if len(events[1].MatchedRules) != 1 {
		t.Fatalf("expected second event matched rules to be preserved, got %d", len(events[1].MatchedRules))
	}
}
