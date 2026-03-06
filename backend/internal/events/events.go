package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Topic constants for all Clario 360 event streams.
const (
	TopicAuditLog      = "clario360.audit.log"
	TopicUserCreated   = "clario360.iam.user.created"
	TopicUserUpdated   = "clario360.iam.user.updated"
	TopicUserDeleted   = "clario360.iam.user.deleted"
	TopicTenantCreated = "clario360.iam.tenant.created"
	TopicWorkflowStart = "clario360.workflow.started"
	TopicWorkflowEnd   = "clario360.workflow.completed"
	TopicCyberAlert    = "clario360.cyber.alert"
	TopicDataPipeline  = "clario360.data.pipeline"
	TopicActaDocument  = "clario360.acta.document"
	TopicLexCase       = "clario360.lex.case"
	TopicVisusReport   = "clario360.visus.report"
)

// Event is the canonical event envelope used across all services.
type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	TenantID  string            `json:"tenant_id"`
	UserID    string            `json:"user_id,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Data      json.RawMessage   `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewEvent creates a new event with a generated ID and current timestamp.
func NewEvent(eventType, source, tenantID string, data any) (*Event, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		TenantID:  tenantID,
		Timestamp: time.Now().UTC(),
		Data:      payload,
	}, nil
}

// Unmarshal decodes the event data into the given target.
func (e *Event) Unmarshal(target any) error {
	return json.Unmarshal(e.Data, target)
}

// Marshal encodes the event as JSON bytes.
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func generateEventID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
