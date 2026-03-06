package consumer

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/audit/service"
	"github.com/clario360/platform/internal/events"
)

// SubscribedTopics lists all Kafka topics the audit consumer subscribes to.
var SubscribedTopics = []string{
	"platform.iam.events",
	"platform.workflow.events",
	"platform.notification.events",
	"cyber.asset.events",
	"cyber.threat.events",
	"cyber.alert.events",
	"cyber.remediation.events",
	"data.source.events",
	"data.pipeline.events",
	"data.quality.events",
	"data.contradiction.events",
	"data.lineage.events",
	"enterprise.acta.events",
	"enterprise.lex.events",
	"enterprise.visus.events",
}

// AuditConsumer subscribes to all platform event topics and ingests them as audit entries.
type AuditConsumer struct {
	consumer  *events.Consumer
	mapper    *EventMapper
	auditSvc  *service.AuditService
	dlq       *DeadLetterProducer
	logger    zerolog.Logger
}

// NewAuditConsumer creates a new audit consumer that subscribes to all platform topics.
func NewAuditConsumer(
	consumer *events.Consumer,
	auditSvc *service.AuditService,
	dlq *DeadLetterProducer,
	logger zerolog.Logger,
) *AuditConsumer {
	ac := &AuditConsumer{
		consumer: consumer,
		mapper:   NewEventMapper(),
		auditSvc: auditSvc,
		dlq:      dlq,
		logger:   logger,
	}

	// Register handler for all topics
	handler := events.EventHandlerFunc(ac.handleEvent)
	for _, topic := range SubscribedTopics {
		consumer.Subscribe(topic, handler)
	}

	return ac
}

// Start begins consuming events. Blocks until context is cancelled.
func (ac *AuditConsumer) Start(ctx context.Context) error {
	ac.logger.Info().
		Int("topic_count", len(SubscribedTopics)).
		Strs("topics", SubscribedTopics).
		Msg("audit consumer starting")

	return ac.consumer.Start(ctx)
}

// Stop gracefully shuts down the consumer.
func (ac *AuditConsumer) Stop() error {
	return ac.consumer.Close()
}

// handleEvent processes a single event from any subscribed topic.
func (ac *AuditConsumer) handleEvent(ctx context.Context, event *events.Event) error {
	metrics.EventsConsumed.WithLabelValues(ac.topicFromEvent(event), "received").Inc()

	// Map the event to an audit entry
	entry, err := ac.mapper.Map(event)
	if err != nil {
		ac.logger.Warn().
			Err(err).
			Str("event_id", event.ID).
			Str("event_type", event.Type).
			Msg("failed to map event to audit entry — sending to DLQ")

		raw, _ := json.Marshal(event)
		NilSafePublish(ac.dlq, ctx, ac.topicFromEvent(event), raw, "mapping_error", err.Error())
		metrics.EventsConsumed.WithLabelValues(ac.topicFromEvent(event), "error").Inc()
		return nil // Don't return error — we handled it via DLQ
	}

	// Ingest the entry (buffered for batch insert)
	ac.auditSvc.IngestFromEvent(*entry)
	metrics.EventsConsumed.WithLabelValues(ac.topicFromEvent(event), "ok").Inc()

	return nil
}

// topicFromEvent extracts the source topic from the event.
func (ac *AuditConsumer) topicFromEvent(event *events.Event) string {
	if event.Source != "" {
		return event.Source
	}
	return "unknown"
}
