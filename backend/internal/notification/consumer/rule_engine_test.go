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
		Source:   "clario360/test-service",
		Type:     eventType,
		TenantID: tenantID,
		UserID:   "user-456",
		Data:     dataBytes,
	}
}

func TestRuleEngine_MatchAlertCreated_Critical(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.alert.created", "tenant-1", map[string]interface{}{
		"severity": "critical",
		"title":    "Ransomware detected",
		"id":       "alert-789",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.NotifType != model.NotifAlertCreated {
		t.Fatalf("expected alert created notification, got %s", matches[0].Rule.NotifType)
	}
	if priority := ResolvePriority(matches[0].Rule, matches[0].Data); priority != model.PriorityCritical {
		t.Fatalf("expected critical priority, got %s", priority)
	}
}

func TestRuleEngine_MatchAlertCreated_High(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.alert.created", "tenant-1", map[string]interface{}{
		"severity": "high",
		"title":    "Privilege escalation",
		"id":       "alert-001",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientRoleBased {
		t.Fatalf("expected role based rule, got %s", matches[0].Rule.RecipientMode)
	}
	if matches[0].Rule.Roles[0] != "security-analyst" {
		t.Fatalf("expected security-analyst recipient, got %v", matches[0].Rule.Roles)
	}
}

func TestRuleEngine_MatchAlertCreated_LowSeverity_NoMatch(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.cyber.alert.created", "tenant-1", map[string]interface{}{
		"severity": "low",
		"title":    "Minor scan",
	})

	if matches := re.Match(event); len(matches) != 0 {
		t.Fatalf("expected 0 matches for low severity alert, got %d", len(matches))
	}
}

func TestRuleEngine_ContractExpiringPriority(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.lex.contract.expiring", "tenant-1", map[string]interface{}{
		"id":                "contract-1",
		"title":             "Master Services Agreement",
		"days_until_expiry": 7,
		"owner_user_id":     "user-owner",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientMixed {
		t.Fatalf("expected mixed recipient mode, got %s", matches[0].Rule.RecipientMode)
	}
	if priority := ResolvePriority(matches[0].Rule, matches[0].Data); priority != model.PriorityCritical {
		t.Fatalf("expected critical priority, got %s", priority)
	}
}

func TestRuleEngine_WorkflowTaskCreated_MixedRecipients(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.workflow.task.created", "tenant-1", map[string]interface{}{
		"task_id":       "task-1",
		"task_name":     "Review contract",
		"assignee_id":   "user-123",
		"assignee_role": "legal-manager",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientMixed {
		t.Fatalf("expected mixed recipient mode, got %s", matches[0].Rule.RecipientMode)
	}
	roles := ResolveRoles(matches[0].Rule, matches[0].Data)
	if len(roles) != 1 || roles[0] != "legal-manager" {
		t.Fatalf("expected dynamic role resolution, got %v", roles)
	}
}

func TestRuleEngine_PipelineFailed_UsesComputedRecipients(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.data.pipeline.run.failed", "tenant-1", map[string]interface{}{
		"pipeline_id": "pipe-1",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientComputed {
		t.Fatalf("expected computed recipient mode, got %s", matches[0].Rule.RecipientMode)
	}
	if matches[0].Rule.ComputedRecipient != "pipeline_owner_from_event" {
		t.Fatalf("expected pipeline owner computation, got %s", matches[0].Rule.ComputedRecipient)
	}
}

func TestRuleEngine_MeetingScheduled_UsesCommitteeComputation(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.acta.meeting.scheduled", "tenant-1", map[string]interface{}{
		"committee_id": "committee-1",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientComputed {
		t.Fatalf("expected computed recipient mode, got %s", matches[0].Rule.RecipientMode)
	}
	if matches[0].Rule.ComputedRecipient != "committee_members_from_event" {
		t.Fatalf("expected committee members computation, got %s", matches[0].Rule.ComputedRecipient)
	}
}

func TestRuleEngine_SystemMaintenance_Broadcast(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.platform.system.maintenance", "tenant-1", map[string]interface{}{
		"title":       "Database upgrade",
		"description": "Upgrading PostgreSQL",
	})

	matches := re.Match(event)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Rule.RecipientMode != RecipientTenantBroadcast {
		t.Fatalf("expected tenant broadcast, got %s", matches[0].Rule.RecipientMode)
	}
}

func TestRuleEngine_NoMatch(t *testing.T) {
	re := NewRuleEngine()
	event := makeTestEvent("com.clario360.unknown.event", "tenant-1", nil)

	if matches := re.Match(event); len(matches) != 0 {
		t.Fatalf("expected 0 matches for unknown event, got %d", len(matches))
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
		t.Fatalf("expected critical, got %s", p)
	}
}

func TestResolveDirectUserIDs_MultiFieldDedup(t *testing.T) {
	rule := &NotificationRule{DirectFields: []string{"assigned_to", "attendee_ids"}}
	data := map[string]interface{}{
		"assigned_to":  "user-1",
		"attendee_ids": []interface{}{"user-1", "user-2", "user-3"},
	}

	ids := ResolveDirectUserIDs(rule, data)
	if len(ids) != 3 {
		t.Fatalf("expected 3 unique user IDs, got %v", ids)
	}
}

func TestExtractEventTopics(t *testing.T) {
	topics := ExtractEventTopics()
	required := map[string]bool{
		events.Topics.AlertEvents:    false,
		events.Topics.FileEvents:     false,
		events.Topics.WorkflowEvents: false,
		events.Topics.ActaEvents:     false,
		events.Topics.LexEvents:      false,
	}
	for _, topic := range topics {
		if _, ok := required[topic]; ok {
			required[topic] = true
		}
	}
	for topic, found := range required {
		if !found {
			t.Fatalf("expected topic %s to be subscribed", topic)
		}
	}
}
