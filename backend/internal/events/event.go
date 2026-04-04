package events

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Event is a CloudEvents v1.0-compliant event envelope used across all Clario 360 services.
// See https://cloudevents.io/ for the specification.
type Event struct {
	// CloudEvents required attributes
	ID          string `json:"id"`          // UUID v4
	Source      string `json:"source"`      // e.g., "clario360/iam-service"
	SpecVersion string `json:"specversion"` // "1.0"
	Type        string `json:"type"`        // e.g., "com.clario360.iam.user.created"

	// CloudEvents optional attributes
	DataContentType string    `json:"datacontenttype"`           // "application/json"
	Subject         string    `json:"subject,omitempty"`         // Resource ID
	Time            time.Time `json:"time"`                      // Event timestamp
	Timestamp       time.Time `json:"timestamp,omitempty"`       // Deprecated: use Time

	// Clario 360 extensions
	TenantID      string `json:"tenantid"`
	UserID        string `json:"userid,omitempty"`
	CorrelationID string `json:"correlationid"`             // For request tracing
	CausationID   string `json:"causationid,omitempty"`     // ID of the event that caused this one

	// Payload
	Data     json.RawMessage   `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"` // Additional headers/metadata
}

// NewEvent creates a new CloudEvents-compliant event with a generated ID and current timestamp.
// The source is prefixed with "clario360/" and the type with "com.clario360." automatically.
func NewEvent(eventType, source, tenantID string, data any) (*Event, error) {
	var payload json.RawMessage
	if data != nil {
		p, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshaling event data: %w", err)
		}
		payload = p
	}

	now := time.Now().UTC()
	return &Event{
		ID:              GenerateUUID(),
		Source:          normalizeSource(source),
		SpecVersion:     "1.0",
		Type:            normalizeType(eventType),
		DataContentType: "application/json",
		Time:            now,
		Timestamp:       now,
		TenantID:        tenantID,
		CorrelationID:   GenerateUUID(),
		Data:            payload,
	}, nil
}

// NewEventWithCorrelation creates an event linked to a parent event via correlation and causation IDs.
func NewEventWithCorrelation(eventType, source, tenantID string, data any, correlationID, causationID string) (*Event, error) {
	evt, err := NewEvent(eventType, source, tenantID, data)
	if err != nil {
		return nil, err
	}
	evt.CorrelationID = correlationID
	evt.CausationID = causationID
	return evt, nil
}

// NewEventRaw creates an event with a pre-serialized JSON payload.
func NewEventRaw(eventType, source, tenantID string, data json.RawMessage) *Event {
	now := time.Now().UTC()
	return &Event{
		ID:              GenerateUUID(),
		Source:          normalizeSource(source),
		SpecVersion:     "1.0",
		Type:            normalizeType(eventType),
		DataContentType: "application/json",
		Time:            now,
		Timestamp:       now,
		TenantID:        tenantID,
		CorrelationID:   GenerateUUID(),
		Data:            data,
	}
}

func normalizeSource(source string) string {
	if strings.HasPrefix(source, "clario360/") {
		return source
	}
	return fmt.Sprintf("clario360/%s", source)
}

func normalizeType(eventType string) string {
	if strings.HasPrefix(eventType, "com.clario360.") {
		return eventType
	}
	return fmt.Sprintf("com.clario360.%s", eventType)
}

// Unmarshal decodes the event data payload into the given target.
func (e *Event) Unmarshal(target any) error {
	return json.Unmarshal(e.Data, target)
}

// Marshal encodes the entire event as JSON bytes.
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Validate checks that required CloudEvents fields are present.
func (e *Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if e.Source == "" {
		return fmt.Errorf("event source is required")
	}
	if e.SpecVersion == "" {
		return fmt.Errorf("event specversion is required")
	}
	if e.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if e.TenantID == "" {
		return fmt.Errorf("event tenantid is required")
	}
	return nil
}

// GenerateUUID returns a UUID v4 string.
func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
