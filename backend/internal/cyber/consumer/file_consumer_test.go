package consumer

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/events"
)

func TestMalwareDetection(t *testing.T) {
	alerts := newFakeAlertEventService()
	consumer := NewFileEventConsumer(alerts, events.NewIdempotencyGuard(nil, 0), nil, zerolog.New(nil), nil)

	event, err := events.NewEvent("file.scan.infected", "file-service", "00000000-0000-0000-0000-000000000001", map[string]any{
		"file_id":      "file-1",
		"virus_name":   "EICAR-Test-File",
		"uploaded_by":  "user-1",
		"suite":        "lex",
		"content_type": "application/pdf",
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = "evt-file-1"

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.order); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}
	if alerts.order[0].Severity != model.SeverityCritical {
		t.Fatalf("expected critical severity, got %s", alerts.order[0].Severity)
	}
	if alerts.order[0].MITRETechniqueID == nil || *alerts.order[0].MITRETechniqueID != "T1204" {
		t.Fatalf("expected MITRE technique T1204, got %v", alerts.order[0].MITRETechniqueID)
	}
}
