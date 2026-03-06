package consumer

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/events"
)

func makeTestEvent(eventType, source, tenantID string) *events.Event {
	return &events.Event{
		ID:          "evt-123",
		Source:      source,
		SpecVersion: "1.0",
		Type:        eventType,
		TenantID:    tenantID,
		UserID:      "user-456",
		Time:        time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC),
		Subject:     "user/res-789",
		Data:        json.RawMessage(`{"id": "res-789", "name": "test"}`),
		Metadata: map[string]string{
			"useremail":  "test@example.com",
			"ip_address": "192.168.1.100",
			"user_agent": "Mozilla/5.0",
		},
		CorrelationID: "corr-001",
	}
}

func TestMapCloudEvent_BasicMapping(t *testing.T) {
	mapper := NewEventMapper()
	event := makeTestEvent("com.clario360.iam.user.created", "clario360/iam-service", "tenant-1")

	entry, err := mapper.Map(event)
	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}

	if entry.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id tenant-1, got %s", entry.TenantID)
	}
	if entry.Service != "iam-service" {
		t.Errorf("expected service iam-service, got %s", entry.Service)
	}
	if entry.Action != "user.created" {
		t.Errorf("expected action user.created, got %s", entry.Action)
	}
	if entry.EventID != "evt-123" {
		t.Errorf("expected event_id evt-123, got %s", entry.EventID)
	}
	if entry.UserID == nil || *entry.UserID != "user-456" {
		t.Errorf("expected user_id user-456")
	}
	if entry.UserEmail != "test@example.com" {
		t.Errorf("expected user_email test@example.com, got %s", entry.UserEmail)
	}
	if entry.IPAddress != "192.168.1.100" {
		t.Errorf("expected ip_address 192.168.1.100, got %s", entry.IPAddress)
	}
	if entry.ResourceType != "user" {
		t.Errorf("expected resource_type user, got %s", entry.ResourceType)
	}
	if entry.ResourceID != "res-789" {
		t.Errorf("expected resource_id res-789, got %s", entry.ResourceID)
	}
}

func TestMapCloudEvent_SeverityClassification(t *testing.T) {
	mapper := NewEventMapper()

	tests := []struct {
		eventType string
		expected  string
	}{
		{"com.clario360.cyber.security.incident.detected", model.SeverityCritical},
		{"com.clario360.cyber.remediation.execute.started", model.SeverityCritical},
		{"com.clario360.iam.user.lockout.triggered", model.SeverityCritical},
		{"com.clario360.iam.mfa.disabled.success", model.SeverityCritical},
		{"com.clario360.iam.role.super_admin.assigned", model.SeverityCritical},

		{"com.clario360.iam.login.failure.password", model.SeverityHigh},
		{"com.clario360.iam.permission.changed", model.SeverityHigh},
		{"com.clario360.iam.role.assigned", model.SeverityHigh},
		{"com.clario360.iam.role.revoked", model.SeverityHigh},
		{"com.clario360.iam.api_key.created", model.SeverityHigh},
		{"com.clario360.iam.password.changed", model.SeverityHigh},
		{"com.clario360.iam.password.reset", model.SeverityHigh},
		{"com.clario360.iam.tenant.settings.updated", model.SeverityHigh},

		{"com.clario360.iam.user.deleted", model.SeverityWarning},
		{"com.clario360.data.config.changed", model.SeverityWarning},
		{"com.clario360.data.pipeline.failed", model.SeverityWarning},
		{"com.clario360.audit.export.started", model.SeverityWarning},
		{"com.clario360.iam.bulk.import", model.SeverityWarning},

		{"com.clario360.iam.user.created", model.SeverityInfo},
		{"com.clario360.iam.user.updated", model.SeverityInfo},
		{"com.clario360.iam.user.login.success", model.SeverityInfo},
	}

	for _, tt := range tests {
		event := makeTestEvent(tt.eventType, "clario360/iam-service", "tenant-1")
		entry, err := mapper.Map(event)
		if err != nil {
			t.Fatalf("Map failed for %s: %v", tt.eventType, err)
		}
		if entry.Severity != tt.expected {
			t.Errorf("event type %s: expected severity %s, got %s", tt.eventType, tt.expected, entry.Severity)
		}
	}
}

func TestMapCloudEvent_ExtractsOldNewValues(t *testing.T) {
	mapper := NewEventMapper()
	event := makeTestEvent("com.clario360.iam.user.updated", "clario360/iam-service", "tenant-1")
	event.Data = json.RawMessage(`{
		"before": {"name": "Old Name", "email": "old@test.com"},
		"after": {"name": "New Name", "email": "new@test.com"}
	}`)

	entry, err := mapper.Map(event)
	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}

	if len(entry.OldValue) == 0 {
		t.Error("expected old_value to be populated")
	}
	if len(entry.NewValue) == 0 {
		t.Error("expected new_value to be populated")
	}

	var oldVal map[string]interface{}
	if err := json.Unmarshal(entry.OldValue, &oldVal); err != nil {
		t.Fatalf("failed to unmarshal old_value: %v", err)
	}
	if oldVal["name"] != "Old Name" {
		t.Errorf("expected old name 'Old Name', got %v", oldVal["name"])
	}
}

func TestMapCloudEvent_MissingRequiredFields(t *testing.T) {
	mapper := NewEventMapper()

	// Missing tenant ID
	event := &events.Event{
		ID:          "evt-1",
		Source:      "clario360/test",
		SpecVersion: "1.0",
		Type:        "com.clario360.test.event",
		TenantID:    "", // empty
	}

	_, err := mapper.Map(event)
	if err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
}

func TestMapCloudEvent_MalformedJSON(t *testing.T) {
	mapper := NewEventMapper()
	event := makeTestEvent("com.clario360.iam.user.created", "clario360/iam-service", "tenant-1")
	event.Data = json.RawMessage(`not-valid-json`)

	// Should not panic — data extraction is best-effort
	entry, err := mapper.Map(event)
	if err != nil {
		t.Fatalf("Map should succeed even with malformed data: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
}

func TestMapCloudEvent_ExtractsServiceFromSource(t *testing.T) {
	mapper := NewEventMapper()

	tests := []struct {
		source   string
		expected string
	}{
		{"clario360/iam-service", "iam-service"},
		{"clario360/cyber-service", "cyber-service"},
		{"clario360/audit-service", "audit-service"},
		{"simple-source", "simple-source"},
	}

	for _, tt := range tests {
		event := makeTestEvent("com.clario360.iam.user.created", tt.source, "tenant-1")
		entry, err := mapper.Map(event)
		if err != nil {
			t.Fatalf("Map failed: %v", err)
		}
		if entry.Service != tt.expected {
			t.Errorf("source %s: expected service %s, got %s", tt.source, tt.expected, entry.Service)
		}
	}
}
