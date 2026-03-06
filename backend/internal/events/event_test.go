package events

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewEvent(t *testing.T) {
	data := map[string]string{"key": "value"}
	event, err := NewEvent("user.created", "iam-service", "tenant-123", data)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	if event.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if !strings.HasPrefix(event.Source, "clario360/") {
		t.Errorf("expected source prefix clario360/, got %s", event.Source)
	}
	if event.Source != "clario360/iam-service" {
		t.Errorf("expected source clario360/iam-service, got %s", event.Source)
	}
	if event.SpecVersion != "1.0" {
		t.Errorf("expected specversion 1.0, got %s", event.SpecVersion)
	}
	if !strings.HasPrefix(event.Type, "com.clario360.") {
		t.Errorf("expected type prefix com.clario360., got %s", event.Type)
	}
	if event.Type != "com.clario360.user.created" {
		t.Errorf("expected type com.clario360.user.created, got %s", event.Type)
	}
	if event.DataContentType != "application/json" {
		t.Errorf("expected datacontenttype application/json, got %s", event.DataContentType)
	}
	if event.TenantID != "tenant-123" {
		t.Errorf("expected tenant_id tenant-123, got %s", event.TenantID)
	}
	if event.Time.IsZero() {
		t.Error("expected non-zero time")
	}
	if event.CorrelationID == "" {
		t.Error("expected non-empty correlation_id")
	}
	if event.Data == nil {
		t.Error("expected non-nil data")
	}

	var payload map[string]string
	if err := event.Unmarshal(&payload); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if payload["key"] != "value" {
		t.Errorf("expected data key=value, got %s", payload["key"])
	}
}

func TestNewEvent_NilData(t *testing.T) {
	event, err := NewEvent("user.deleted", "iam-service", "tenant-123", nil)
	if err != nil {
		t.Fatalf("NewEvent with nil data failed: %v", err)
	}
	if event.Data != nil {
		t.Errorf("expected nil data, got %s", string(event.Data))
	}
}

func TestNewEventWithCorrelation(t *testing.T) {
	event, err := NewEventWithCorrelation("alert.created", "cyber-service", "tenant-456", nil, "corr-123", "cause-789")
	if err != nil {
		t.Fatalf("NewEventWithCorrelation failed: %v", err)
	}
	if event.CorrelationID != "corr-123" {
		t.Errorf("expected correlation_id corr-123, got %s", event.CorrelationID)
	}
	if event.CausationID != "cause-789" {
		t.Errorf("expected causation_id cause-789, got %s", event.CausationID)
	}
}

func TestNewEventRaw(t *testing.T) {
	raw := json.RawMessage(`{"alert_id": "a1"}`)
	event := NewEventRaw("alert.created", "cyber-service", "tenant-789", raw)

	if event.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if string(event.Data) != `{"alert_id": "a1"}` {
		t.Errorf("unexpected data: %s", string(event.Data))
	}
}

func TestEvent_Marshal_Unmarshal(t *testing.T) {
	original, err := NewEvent("test.event", "test-service", "tenant-1", map[string]int{"count": 42})
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}
	original.UserID = "user-1"
	original.Subject = "resource-1"

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Event
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %s != %s", restored.ID, original.ID)
	}
	if restored.Source != original.Source {
		t.Errorf("Source mismatch: %s != %s", restored.Source, original.Source)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: %s != %s", restored.Type, original.Type)
	}
	if restored.TenantID != original.TenantID {
		t.Errorf("TenantID mismatch: %s != %s", restored.TenantID, original.TenantID)
	}
	if restored.UserID != original.UserID {
		t.Errorf("UserID mismatch: %s != %s", restored.UserID, original.UserID)
	}
	if restored.Subject != original.Subject {
		t.Errorf("Subject mismatch: %s != %s", restored.Subject, original.Subject)
	}
	if restored.CorrelationID != original.CorrelationID {
		t.Errorf("CorrelationID mismatch: %s != %s", restored.CorrelationID, original.CorrelationID)
	}
}

func TestEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid event",
			event: Event{
				ID: "id-1", Source: "src", SpecVersion: "1.0",
				Type: "test", TenantID: "t-1",
			},
			wantErr: false,
		},
		{
			name:    "missing ID",
			event:   Event{Source: "src", SpecVersion: "1.0", Type: "test", TenantID: "t-1"},
			wantErr: true,
		},
		{
			name:    "missing Source",
			event:   Event{ID: "id-1", SpecVersion: "1.0", Type: "test", TenantID: "t-1"},
			wantErr: true,
		},
		{
			name:    "missing SpecVersion",
			event:   Event{ID: "id-1", Source: "src", Type: "test", TenantID: "t-1"},
			wantErr: true,
		},
		{
			name:    "missing Type",
			event:   Event{ID: "id-1", Source: "src", SpecVersion: "1.0", TenantID: "t-1"},
			wantErr: true,
		},
		{
			name:    "missing TenantID",
			event:   Event{ID: "id-1", Source: "src", SpecVersion: "1.0", Type: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	ids := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := GenerateUUID()
		if len(id) != 36 {
			t.Fatalf("expected UUID length 36, got %d: %s", len(id), id)
		}
		// Verify UUID v4 format: xxxxxxxx-xxxx-4xxx-{8,9,a,b}xxx-xxxxxxxxxxxx
		if id[14] != '4' {
			t.Errorf("expected version 4 at position 14, got %c in %s", id[14], id)
		}
		if _, ok := ids[id]; ok {
			t.Fatalf("duplicate UUID generated: %s", id)
		}
		ids[id] = struct{}{}
	}
}
