package consumer

import (
	"encoding/json"
	"testing"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/notification/model"
)

func makeTestEvent(eventType, tenantID string, data map[string]interface{}) *events.Event {
	dataBytes, _ := json.Marshal(data)
	return &events.Event{
		ID:       "evt-123",
		Source:   "clario360/cyber-service",
		Type:     eventType,
		TenantID: tenantID,
		UserID:   "user-456",
		Data:     dataBytes,
	}
}

func TestRuleEngine_MatchAlertCreated_Critical(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.alert.created", "tenant-1", map[string]interface{}{
		"severity":    "critical",
		"title":       "Ransomware detected",
		"source":      "endpoint-protection",
		"description": "Ransomware activity detected on server-01",
		"id":          "alert-789",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	m := matches[0]
	if m.Rule.NotifType != model.NotifAlertCreated {
		t.Errorf("expected NotifAlertCreated, got %s", m.Rule.NotifType)
	}
	if m.Rule.Category != model.CategorySecurity {
		t.Errorf("expected security category, got %s", m.Rule.Category)
	}

	priority := ResolvePriority(m.Rule, m.Data)
	if priority != model.PriorityCritical {
		t.Errorf("expected critical priority, got %s", priority)
	}
}

func TestRuleEngine_MatchAlertCreated_LowSeverity_NoMatch(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.alert.created", "tenant-1", map[string]interface{}{
		"severity": "low",
		"title":    "Minor scan",
	})

	matches := re.Match(event)
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for low severity alert, got %d", len(matches))
	}
}

func TestRuleEngine_MatchAlertEscalated(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.alert.escalated", "tenant-1", map[string]interface{}{
		"title":             "Critical Alert",
		"id":                "alert-001",
		"previous_assignee": "analyst-1",
		"reason":            "SLA breach",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.Priority != model.PriorityCritical {
		t.Errorf("expected critical priority for escalation")
	}
}

func TestRuleEngine_TaskCreated_RemediationApproval(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.workflow.task.created", "tenant-1", map[string]interface{}{
		"step_id":     "approve_remediation",
		"assignee_id": "user-approver",
		"task_id":     "task-001",
	})

	matches := re.Match(event)
	found := false
	for _, m := range matches {
		if m.Rule.NotifType == model.NotifRemediationApproval {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected remediation approval match")
	}
}

func TestRuleEngine_TaskCreated_RegularTask(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.workflow.task.created", "tenant-1", map[string]interface{}{
		"step_id":     "review_document",
		"assignee_id": "user-reviewer",
		"name":        "Review Q4 Report",
		"task_id":     "task-002",
	})

	matches := re.Match(event)
	found := false
	for _, m := range matches {
		if m.Rule.NotifType == model.NotifTaskAssigned {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected task assigned match")
	}
}

func TestRuleEngine_SecurityIncident(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.security.incident", "tenant-1", map[string]interface{}{
		"title":       "Data Breach",
		"severity":    "critical",
		"incident_id": "inc-001",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.NotifType != model.NotifSecurityIncident {
		t.Errorf("expected security incident type, got %s", matches[0].Rule.NotifType)
	}
}

func TestRuleEngine_SystemMaintenance_Broadcast(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.platform.system.maintenance", "tenant-1", map[string]interface{}{
		"title":       "Database upgrade",
		"description": "Upgrading PostgreSQL",
		"start_time":  "2026-03-07T02:00:00Z",
		"end_time":    "2026-03-07T04:00:00Z",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientTenantBroadcast {
		t.Errorf("expected tenant broadcast, got %s", matches[0].Rule.RecipientMode)
	}
}

func TestRuleEngine_NoMatch(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.unknown.event", "tenant-1", nil)

	matches := re.Match(event)
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for unknown event, got %d", len(matches))
	}
}

func TestResolvePriority_Static(t *testing.T) {
	rule := &NotificationRule{Priority: model.PriorityHigh}
	if p := ResolvePriority(rule, nil); p != model.PriorityHigh {
		t.Errorf("expected high, got %s", p)
	}
}

func TestResolvePriority_Dynamic(t *testing.T) {
	rule := &NotificationRule{
		PriorityFunc: func(data map[string]interface{}) string {
			if data["severity"] == "critical" {
				return model.PriorityCritical
			}
			return model.PriorityMedium
		},
	}
	data := map[string]interface{}{"severity": "critical"}
	if p := ResolvePriority(rule, data); p != model.PriorityCritical {
		t.Errorf("expected critical, got %s", p)
	}
}

func TestResolveDirectUserIDs_String(t *testing.T) {
	rule := &NotificationRule{DirectField: "assignee_id"}
	data := map[string]interface{}{"assignee_id": "user-123"}

	ids := ResolveDirectUserIDs(rule, data)
	if len(ids) != 1 || ids[0] != "user-123" {
		t.Errorf("expected [user-123], got %v", ids)
	}
}

func TestResolveDirectUserIDs_Array(t *testing.T) {
	rule := &NotificationRule{DirectField: "attendees"}
	data := map[string]interface{}{
		"attendees": []interface{}{"user-1", "user-2", "user-3"},
	}

	ids := ResolveDirectUserIDs(rule, data)
	if len(ids) != 3 {
		t.Errorf("expected 3 user IDs, got %d", len(ids))
	}
}

func TestResolveDirectUserIDs_Missing(t *testing.T) {
	rule := &NotificationRule{DirectField: "assignee_id"}
	data := map[string]interface{}{}

	ids := ResolveDirectUserIDs(rule, data)
	if ids != nil {
		t.Errorf("expected nil for missing field, got %v", ids)
	}
}

func TestExtractEventTopics(t *testing.T) {
	topics := ExtractEventTopics()
	if len(topics) < 5 {
		t.Errorf("expected at least 5 topics, got %d", len(topics))
	}
}
