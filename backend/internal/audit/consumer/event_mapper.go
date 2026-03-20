package consumer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/events"
)

// EventMapper converts CloudEvents into AuditEntry records.
type EventMapper struct{}

// NewEventMapper creates a new EventMapper.
func NewEventMapper() *EventMapper {
	return &EventMapper{}
}

// Map converts a CloudEvents event into an AuditEntry.
func (m *EventMapper) Map(event *events.Event) (*model.AuditEntry, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("invalid event: %w", err)
	}

	entry := &model.AuditEntry{
		ID:            events.GenerateUUID(),
		TenantID:      event.TenantID,
		UserEmail:     m.extractUserEmail(event),
		Service:       m.extractService(event.Source),
		Action:        m.extractAction(event.Type),
		Severity:      m.classifySeverity(event.Type),
		ResourceType:  m.extractResourceType(event),
		ResourceID:    m.extractResourceID(event),
		EventID:       event.ID,
		CorrelationID: event.CorrelationID,
		CreatedAt:     event.Time,
	}

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	// Extract user_id from event
	if event.UserID != "" {
		uid := event.UserID
		entry.UserID = &uid
	}

	// Extract old_value/new_value from data payload
	m.extractChangeData(event.Data, entry)

	// Extract metadata
	entry.Metadata = m.extractMetadata(event)

	// Extract IP address and user agent from metadata
	if event.Metadata != nil {
		if ip, ok := event.Metadata["ip_address"]; ok {
			entry.IPAddress = ip
		}
		if ua, ok := event.Metadata["user_agent"]; ok {
			entry.UserAgent = ua
		}
	}

	return entry, nil
}

// extractService extracts the service name from the CloudEvents source.
// "clario360/iam-service" → "iam-service"
func (m *EventMapper) extractService(source string) string {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return source
}

// extractAction extracts the action from the CloudEvents type.
// "com.clario360.iam.user.login.success" → "user.login.success"
func (m *EventMapper) extractAction(eventType string) string {
	// Remove "com.clario360." prefix and first domain segment
	prefix := "com.clario360."
	if strings.HasPrefix(eventType, prefix) {
		rest := eventType[len(prefix):]
		// Remove the domain part (e.g., "iam." or "cyber.")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return rest
	}
	return eventType
}

// extractResourceType extracts the resource type from the event subject.
// "user/uuid" → "user"
func (m *EventMapper) extractResourceType(event *events.Event) string {
	if event.Subject != "" {
		parts := strings.SplitN(event.Subject, "/", 2)
		return parts[0]
	}
	// Fall back to extracting from event type
	action := m.extractAction(event.Type)
	parts := strings.SplitN(action, ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// extractResourceID extracts the resource ID from the event subject.
// "user/uuid" → "uuid"
func (m *EventMapper) extractResourceID(event *events.Event) string {
	if event.Subject != "" {
		parts := strings.SplitN(event.Subject, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	// Try extracting from data payload
	if len(event.Data) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(event.Data, &data); err == nil {
			if id, ok := data["id"]; ok {
				return fmt.Sprintf("%v", id)
			}
		}
	}
	return ""
}

// extractUserEmail gets user email from event extensions.
func (m *EventMapper) extractUserEmail(event *events.Event) string {
	if event.Metadata != nil {
		if email, ok := event.Metadata["useremail"]; ok {
			return email
		}
		if email, ok := event.Metadata["user_email"]; ok {
			return email
		}
	}
	return ""
}

// extractChangeData extracts old_value/new_value from the event data payload.
// Looks for standard "before"/"after" change capture pattern.
func (m *EventMapper) extractChangeData(data json.RawMessage, entry *model.AuditEntry) {
	if len(data) == 0 {
		return
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	if before, ok := payload["before"]; ok {
		entry.OldValue = before
	}
	if after, ok := payload["after"]; ok {
		entry.NewValue = after
	}
}

// extractMetadata builds the metadata JSON from event metadata fields.
func (m *EventMapper) extractMetadata(event *events.Event) json.RawMessage {
	meta := make(map[string]interface{})
	if event.CausationID != "" {
		meta["causation_id"] = event.CausationID
	}
	if event.DataContentType != "" {
		meta["content_type"] = event.DataContentType
	}
	for k, v := range event.Metadata {
		if k == "ip_address" || k == "user_agent" || k == "useremail" || k == "user_email" {
			continue // Already extracted to top-level fields
		}
		meta[k] = v
	}

	// Preserve the structured payload for data access telemetry so downstream
	// behavioral analytics can reconstruct the underlying access event from the
	// immutable audit log without needing cross-service joins.
	if event.Type == "data.access.event.collected" && len(event.Data) > 0 {
		var payload map[string]interface{}
		if err := json.Unmarshal(event.Data, &payload); err == nil {
			for _, key := range []string{
				"timestamp",
				"user",
				"source_ip",
				"action",
				"database",
				"table",
				"query_hash",
				"query_preview",
				"rows_read",
				"rows_written",
				"bytes_read",
				"bytes_written",
				"duration_ms",
				"success",
				"error_message",
				"source_type",
				"source_id",
				"source_name",
			} {
				if value, ok := payload[key]; ok {
					meta[key] = value
				}
			}
		}
	}

	if len(meta) == 0 {
		return json.RawMessage(`{}`)
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

// classifySeverity determines the severity level based on the event type.
// Uses pattern matching against predefined severity rules.
func (m *EventMapper) classifySeverity(eventType string) string {
	lower := strings.ToLower(eventType)

	// CRITICAL patterns
	for _, pattern := range criticalPatterns {
		if strings.Contains(lower, pattern) {
			return model.SeverityCritical
		}
	}

	// HIGH patterns
	for _, pattern := range highPatterns {
		if strings.Contains(lower, pattern) {
			return model.SeverityHigh
		}
	}

	// WARNING patterns
	for _, pattern := range warningPatterns {
		if strings.Contains(lower, pattern) {
			return model.SeverityWarning
		}
	}

	return model.SeverityInfo
}

var criticalPatterns = []string{
	"security.incident",
	"remediation.execute",
	"user.lockout",
	"mfa.disabled",
	"role.super_admin",
}

var highPatterns = []string{
	"login.failure",
	"permission.changed",
	"role.assigned",
	"role.revoked",
	"api_key.created",
	"api_key.revoked",
	"password.changed",
	"password.reset",
	"tenant.settings",
}

var warningPatterns = []string{
	"deleted",
	"config.changed",
	"pipeline.failed",
	"export",
	"bulk",
}
