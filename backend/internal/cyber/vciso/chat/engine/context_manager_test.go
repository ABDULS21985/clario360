package engine

import (
	"testing"
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

func TestContextManager_ResolveEntities(t *testing.T) {
	t.Parallel()

	manager := NewContextManager(
		WithClock(func() time.Time { return time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC) }),
		WithIdleTimeout(30*time.Minute),
	)
	conversation := manager.NewContext(uuid.New(), uuid.New(), uuid.New())
	conversation.LastEntities = []chatmodel.EntityReference{
		{Type: "alert", ID: "alert-1", Name: "Alert One", Index: 0},
		{Type: "alert", ID: "alert-2", Name: "Alert Two", Index: 1},
		{Type: "alert", ID: "alert-3", Name: "Alert Three", Index: 2},
	}

	tests := []struct {
		message string
		want    string
	}{
		{"Tell me about the first one", "alert-1"},
		{"Investigate the second one", "alert-2"},
		{"Explain it", "alert-1"},
		{"Show the last one", "alert-3"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.message, func(t *testing.T) {
			t.Parallel()
			entities, clarification := manager.ResolveEntities(tc.message, "alert_detail", map[string]string{}, &conversation, "alert_id")
			if clarification != nil {
				t.Fatalf("unexpected clarification: %+v", clarification)
			}
			if entities["alert_id"] != tc.want {
				t.Fatalf("alert_id = %q, want %q", entities["alert_id"], tc.want)
			}
		})
	}
}

func TestContextManager_FilterCarryover(t *testing.T) {
	t.Parallel()

	manager := NewContextManager(
		WithClock(func() time.Time { return time.Now().UTC() }),
		WithIdleTimeout(30*time.Minute),
	)
	conversation := manager.NewContext(uuid.New(), uuid.New(), uuid.New())

	entities := manager.ApplyFilterCarryover("Show critical alerts", map[string]string{"severity": "critical"}, &conversation)
	if entities["severity"] != "critical" {
		t.Fatalf("severity = %q, want critical", entities["severity"])
	}

	entities = manager.ApplyFilterCarryover("and high ones too", map[string]string{"severity": "high"}, &conversation)
	if entities["severity"] != "critical,high" {
		t.Fatalf("severity = %q, want critical,high", entities["severity"])
	}

	entities = manager.ApplyFilterCarryover("show all alerts", map[string]string{}, &conversation)
	if _, ok := entities["severity"]; ok {
		t.Fatalf("severity filter should be cleared, got %q", entities["severity"])
	}
}

func TestContextManager_MaxTurnsAndExpiry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	current := now
	manager := NewContextManager(
		WithClock(func() time.Time { return current }),
		WithIdleTimeout(30*time.Minute),
	)
	conversation := manager.NewContext(uuid.New(), uuid.New(), uuid.New())

	for i := 0; i < 12; i++ {
		manager.AddTurn(&conversation, chatmodel.Turn{Role: "user", Content: "turn", At: current})
	}
	if len(conversation.Turns) != 10 {
		t.Fatalf("turn count = %d, want 10", len(conversation.Turns))
	}

	current = now.Add(31 * time.Minute)
	if !manager.IsExpired(conversation) {
		t.Fatal("conversation should be expired")
	}
}
