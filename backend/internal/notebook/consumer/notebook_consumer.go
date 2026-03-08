package consumer

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
)

const notebookTopic = "platform.notebook.events"

type NotebookConsumer struct {
	producer *events.Producer
	logger   zerolog.Logger
}

func NewNotebookConsumer(producer *events.Producer, logger zerolog.Logger) *NotebookConsumer {
	return &NotebookConsumer{producer: producer, logger: logger}
}

func (c *NotebookConsumer) Handle(ctx context.Context, event *events.Event) error {
	if c.producer == nil {
		return nil
	}

	var payload map[string]any
	if len(event.Data) > 0 {
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("failed to decode notebook event payload")
			payload = map[string]any{}
		}
	} else {
		payload = map[string]any{}
	}

	payload["source_event_id"] = event.ID
	payload["source_event_type"] = event.Type

	auditType := "audit.notebook.activity"
	switch event.Type {
	case "com.clario360.notebook.server.started":
		auditType = "audit.notebook.server.started"
	case "com.clario360.notebook.server.stopped":
		auditType = "audit.notebook.server.stopped"
	case "com.clario360.notebook.template.copied":
		auditType = "audit.notebook.template.copied"
	}

	auditEvent, err := events.NewEventWithCorrelation(auditType, "iam-service", event.TenantID, payload, event.CorrelationID, event.ID)
	if err != nil {
		c.logger.Error().Err(err).Str("event_id", event.ID).Msg("failed to build notebook audit event")
		return nil
	}
	auditEvent.UserID = event.UserID
	if event.Metadata != nil {
		auditEvent.Metadata = event.Metadata
	}

	if err := c.producer.Publish(ctx, events.Topics.AuditEvents, auditEvent); err != nil {
		c.logger.Error().Err(err).Str("event_id", event.ID).Msg("failed to publish notebook audit event")
	}
	return nil
}

func (c *NotebookConsumer) Topic() string {
	return notebookTopic
}
