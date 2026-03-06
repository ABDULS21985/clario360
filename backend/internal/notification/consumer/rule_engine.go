package consumer

import (
	"encoding/json"
	"strings"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/notification/model"
)

// RecipientMode describes how to resolve recipients for a rule.
type RecipientMode string

const (
	RecipientRoleBased       RecipientMode = "role_based"
	RecipientDirect          RecipientMode = "direct"
	RecipientTenantBroadcast RecipientMode = "tenant_broadcast"
)

// NotificationRule defines the mapping from a domain event to a notification.
type NotificationRule struct {
	EventType     string
	Condition     func(data map[string]interface{}) bool
	NotifType     model.NotificationType
	Category      string
	Priority      string
	PriorityFunc  func(data map[string]interface{}) string
	RecipientMode RecipientMode
	Roles         []string
	DirectField   string
	TitleTemplate string
	BodyTemplate  string
	ActionURLTmpl string
}

// MatchedRule is a rule that matched an event.
type MatchedRule struct {
	Rule *NotificationRule
	Data map[string]interface{}
}

// RuleEngine matches domain events to notification rules.
type RuleEngine struct {
	rules map[string][]*NotificationRule // eventType → rules
}

// NewRuleEngine creates a new RuleEngine with all pre-configured rules.
func NewRuleEngine() *RuleEngine {
	re := &RuleEngine{rules: make(map[string][]*NotificationRule)}

	re.addRule(&NotificationRule{
		EventType: "com.clario360.cyber.alert.created",
		Condition: func(data map[string]interface{}) bool {
			sev, _ := data["severity"].(string)
			return sev == "critical" || sev == "high"
		},
		NotifType: model.NotifAlertCreated,
		Category:  model.CategorySecurity,
		PriorityFunc: func(data map[string]interface{}) string {
			if sev, _ := data["severity"].(string); sev == "critical" {
				return model.PriorityCritical
			}
			return model.PriorityHigh
		},
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"security-analyst", "security-manager"},
		TitleTemplate: "{{.severity}} Security Alert: {{.title}}",
		BodyTemplate:  "A {{.severity}} severity alert has been detected in {{.source}}. {{.description}}",
		ActionURLTmpl: "/cyber/alerts/{{.id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.cyber.alert.escalated",
		NotifType:     model.NotifAlertEscalated,
		Category:      model.CategorySecurity,
		Priority:      model.PriorityCritical,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"security-manager", "ciso"},
		TitleTemplate: "Alert Escalated: {{.title}}",
		BodyTemplate:  "Alert {{.id}} has been escalated. Previous assignee: {{.previous_assignee}}. Reason: {{.reason}}",
		ActionURLTmpl: "/cyber/alerts/{{.id}}",
	})

	re.addRule(&NotificationRule{
		EventType: "com.clario360.workflow.task.created",
		Condition: func(data map[string]interface{}) bool {
			stepID, _ := data["step_id"].(string)
			return stepID == "approve_remediation"
		},
		NotifType:     model.NotifRemediationApproval,
		Category:      model.CategorySecurity,
		Priority:      model.PriorityHigh,
		RecipientMode: RecipientDirect,
		DirectField:   "assignee_id",
		TitleTemplate: "Remediation Approval Required",
		BodyTemplate:  "A remediation plan requires your approval. Alert: {{.alert_id}}. Plan: {{.remediation_plan}}",
		ActionURLTmpl: "/workflows/tasks/{{.task_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.cyber.remediation.completed",
		NotifType:     model.NotifRemediationCompleted,
		Category:      model.CategorySecurity,
		Priority:      model.PriorityMedium,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"security-analyst", "security-manager"},
		TitleTemplate: "Remediation Completed: {{.alert_title}}",
		BodyTemplate:  "Remediation for alert {{.alert_id}} has been successfully completed.",
		ActionURLTmpl: "/cyber/alerts/{{.alert_id}}",
	})

	re.addRule(&NotificationRule{
		EventType: "com.clario360.workflow.task.created",
		Condition: func(data map[string]interface{}) bool {
			stepID, _ := data["step_id"].(string)
			return stepID != "approve_remediation"
		},
		NotifType:     model.NotifTaskAssigned,
		Category:      model.CategoryWorkflow,
		Priority:      model.PriorityMedium,
		RecipientMode: RecipientDirect,
		DirectField:   "assignee_id",
		TitleTemplate: "New Task: {{.name}}",
		BodyTemplate:  "You have been assigned: {{.name}}.",
		ActionURLTmpl: "/workflows/tasks/{{.task_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.workflow.task.sla_breached",
		NotifType:     model.NotifTaskOverdue,
		Category:      model.CategoryWorkflow,
		Priority:      model.PriorityHigh,
		RecipientMode: RecipientDirect,
		DirectField:   "claimed_by",
		TitleTemplate: "Task Overdue: {{.name}}",
		BodyTemplate:  "Task {{.name}} has exceeded its SLA deadline by {{.hours_overdue}} hours.",
		ActionURLTmpl: "/workflows/tasks/{{.task_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.workflow.task.escalated",
		NotifType:     model.NotifTaskEscalated,
		Category:      model.CategoryWorkflow,
		Priority:      model.PriorityHigh,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{}, // resolved from data["escalation_role"]
		DirectField:   "escalation_role",
		TitleTemplate: "Task Escalated: {{.name}}",
		BodyTemplate:  "Task {{.name}} has been escalated to your attention. Original assignee did not complete within SLA.",
		ActionURLTmpl: "/workflows/tasks/{{.task_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.data.pipeline.failed",
		NotifType:     model.NotifPipelineFailed,
		Category:      model.CategoryData,
		Priority:      model.PriorityHigh,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"data-engineer", "data-steward"},
		TitleTemplate: "Pipeline Failed: {{.pipeline_name}}",
		BodyTemplate:  "Data pipeline {{.pipeline_name}} failed at stage {{.failed_stage}}. Error: {{.error_message}}",
		ActionURLTmpl: "/data/pipelines/{{.pipeline_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.data.quality.issue",
		NotifType:     model.NotifQualityIssue,
		Category:      model.CategoryData,
		Priority:      model.PriorityMedium,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"data-steward", "data-analyst"},
		TitleTemplate: "Data Quality Issue: {{.dataset_name}}",
		BodyTemplate:  "{{.issue_count}} quality issues detected in dataset {{.dataset_name}}. Severity: {{.severity}}",
		ActionURLTmpl: "/data/quality/{{.dataset_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.data.contradiction.detected",
		NotifType:     model.NotifContradictionFound,
		Category:      model.CategoryData,
		Priority:      model.PriorityHigh,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"data-steward", "compliance-officer"},
		TitleTemplate: "Data Contradiction Detected",
		BodyTemplate:  "Conflicting data found between {{.source_a}} and {{.source_b}}. Field: {{.field_name}}",
		ActionURLTmpl: "/data/contradictions/{{.contradiction_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.enterprise.lex.contract.expiring",
		NotifType:     model.NotifContractExpiring,
		Category:      model.CategoryLegal,
		Priority:      model.PriorityMedium,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"legal-counsel", "contract-manager"},
		TitleTemplate: "Contract Expiring: {{.contract_name}}",
		BodyTemplate:  "Contract {{.contract_name}} with {{.counterparty}} expires on {{.expiry_date}}.",
		ActionURLTmpl: "/legal/contracts/{{.contract_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.enterprise.acta.meeting.scheduled",
		NotifType:     model.NotifMeetingScheduled,
		Category:      model.CategoryGovernance,
		Priority:      model.PriorityLow,
		RecipientMode: RecipientDirect,
		DirectField:   "attendees",
		TitleTemplate: "Meeting Scheduled: {{.title}}",
		BodyTemplate:  "{{.title}} scheduled for {{.scheduled_at}}. Location: {{.location}}",
		ActionURLTmpl: "/governance/meetings/{{.meeting_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.cyber.security.incident",
		NotifType:     model.NotifSecurityIncident,
		Category:      model.CategorySecurity,
		Priority:      model.PriorityCritical,
		RecipientMode: RecipientRoleBased,
		Roles:         []string{"security-manager", "ciso", "tenant-admin"},
		TitleTemplate: "SECURITY INCIDENT: {{.title}}",
		BodyTemplate:  "A security incident has been declared. Severity: {{.severity}}. Immediate action required.",
		ActionURLTmpl: "/cyber/incidents/{{.incident_id}}",
	})

	re.addRule(&NotificationRule{
		EventType:     "com.clario360.platform.system.maintenance",
		NotifType:     model.NotifSystemMaintenance,
		Category:      model.CategorySystem,
		Priority:      model.PriorityLow,
		RecipientMode: RecipientTenantBroadcast,
		TitleTemplate: "Scheduled Maintenance: {{.title}}",
		BodyTemplate:  "{{.description}}. Scheduled: {{.start_time}} to {{.end_time}}.",
		ActionURLTmpl: "",
	})

	return re
}

func (re *RuleEngine) addRule(rule *NotificationRule) {
	re.rules[rule.EventType] = append(re.rules[rule.EventType], rule)
}

// Match returns all rules that match the given event.
func (re *RuleEngine) Match(event *events.Event) []MatchedRule {
	rules, ok := re.rules[event.Type]
	if !ok {
		return nil
	}

	var data map[string]interface{}
	if len(event.Data) > 0 {
		_ = json.Unmarshal(event.Data, &data)
	}
	if data == nil {
		data = make(map[string]interface{})
	}

	// Inject event metadata.
	data["_event_id"] = event.ID
	data["_tenant_id"] = event.TenantID
	data["_user_id"] = event.UserID
	data["_source"] = event.Source
	data["_correlation_id"] = event.CorrelationID

	var matched []MatchedRule
	for _, rule := range rules {
		if rule.Condition != nil && !rule.Condition(data) {
			continue
		}
		matched = append(matched, MatchedRule{Rule: rule, Data: data})
	}
	return matched
}

// ResolvePriority returns the priority for a matched rule.
func ResolvePriority(rule *NotificationRule, data map[string]interface{}) string {
	if rule.PriorityFunc != nil {
		return rule.PriorityFunc(data)
	}
	return rule.Priority
}

// ResolveRoles returns the IAM roles for recipient resolution.
// If the rule specifies static roles, those are returned.
// If the rule references a data field for dynamic roles, that field is checked.
func ResolveRoles(rule *NotificationRule, data map[string]interface{}) []string {
	if len(rule.Roles) > 0 {
		return rule.Roles
	}
	if rule.DirectField != "" {
		if roleStr, ok := data[rule.DirectField].(string); ok {
			return []string{roleStr}
		}
	}
	return nil
}

// ResolveDirectUserIDs extracts user IDs from event data for direct recipient mode.
func ResolveDirectUserIDs(rule *NotificationRule, data map[string]interface{}) []string {
	if rule.DirectField == "" {
		return nil
	}
	val, ok := data[rule.DirectField]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case string:
		return []string{v}
	case []interface{}:
		ids := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				ids = append(ids, s)
			}
		}
		return ids
	}
	return nil
}

// ExtractEventTopics returns all Kafka topics the notification consumer should subscribe to.
func ExtractEventTopics() []string {
	return []string{
		"platform.iam.events",
		"platform.workflow.events",
		"platform.audit.events",
		"cyber.alert.events",
		"cyber.remediation.events",
		"data.pipeline.events",
		"data.quality.events",
		"data.contradiction.events",
		"enterprise.acta.events",
		"enterprise.lex.events",
	}
}

// extractService extracts the service name from a CloudEvents source.
func extractService(source string) string {
	if idx := strings.LastIndex(source, "/"); idx >= 0 && idx < len(source)-1 {
		return source[idx+1:]
	}
	return source
}
