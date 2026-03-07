package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/events"
)

// Actor carries user and request metadata for audit emission.
type Actor struct {
	UserID    uuid.UUID
	UserName  string
	UserEmail string
	IPAddress string
	UserAgent string
}

func publishEvent(ctx context.Context, producer *events.Producer, topic string, eventType string, tenantID uuid.UUID, actor *Actor, data interface{}) error {
	if producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), data)
	if err != nil {
		return err
	}
	if actor != nil {
		event.UserID = actor.UserID.String()
		if event.Metadata == nil {
			event.Metadata = make(map[string]string)
		}
		if actor.UserEmail != "" {
			event.Metadata["user_email"] = actor.UserEmail
		}
		if actor.IPAddress != "" {
			event.Metadata["ip_address"] = actor.IPAddress
		}
		if actor.UserAgent != "" {
			event.Metadata["user_agent"] = actor.UserAgent
		}
	}
	return producer.Publish(ctx, topic, event)
}

func publishAuditEvent(ctx context.Context, producer *events.Producer, eventType string, tenantID uuid.UUID, actor *Actor, data interface{}) error {
	return publishEvent(ctx, producer, events.Topics.AuditEvents, eventType, tenantID, actor, data)
}

func mustJSON(payload interface{}) json.RawMessage {
	if payload == nil {
		return json.RawMessage("{}")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage("{}")
	}
	return encoded
}
