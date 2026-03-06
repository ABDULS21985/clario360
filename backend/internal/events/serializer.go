package events

import (
	"encoding/json"
	"fmt"
)

// Serializer handles JSON serialization and deserialization of events
// with schema validation.
type Serializer struct{}

// NewSerializer creates a new event serializer.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serialize encodes an event into JSON bytes for Kafka message value.
// Validates the event before serialization.
func (s *Serializer) Serialize(event *Event) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("serializing event: %w", err)
	}
	return data, nil
}

// Deserialize decodes JSON bytes into an Event.
// Validates the event after deserialization.
func (s *Serializer) Deserialize(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("deserializing event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}
	return &event, nil
}

// DeserializeData extracts and unmarshals the event's data payload into the given target.
func (s *Serializer) DeserializeData(event *Event, target any) error {
	if event.Data == nil {
		return fmt.Errorf("event data is nil")
	}
	return json.Unmarshal(event.Data, target)
}

// SerializeData marshals a payload and sets it as the event's data field.
func (s *Serializer) SerializeData(event *Event, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializing event data: %w", err)
	}
	event.Data = data
	event.DataContentType = "application/json"
	return nil
}
