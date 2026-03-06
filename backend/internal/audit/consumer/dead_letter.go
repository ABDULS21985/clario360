package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/events"
)

const dlqTopic = "platform.audit.dlq"

// DeadLetterProducer publishes unmappable or poison events to the DLQ topic.
type DeadLetterProducer struct {
	producer *events.Producer
	logger   zerolog.Logger
}

// NewDeadLetterProducer creates a new DeadLetterProducer.
func NewDeadLetterProducer(producer *events.Producer, logger zerolog.Logger) *DeadLetterProducer {
	return &DeadLetterProducer{
		producer: producer,
		logger:   logger,
	}
}

// DLQEntry is the envelope for a dead letter message.
type DLQEntry struct {
	OriginalTopic string          `json:"original_topic"`
	OriginalEvent json.RawMessage `json:"original_event"`
	Error         string          `json:"error"`
	Reason        string          `json:"reason"`
	Timestamp     time.Time       `json:"timestamp"`
	Partition     int32           `json:"partition"`
	Offset        int64           `json:"offset"`
}

// Publish sends a failed event to the dead letter queue.
func (d *DeadLetterProducer) Publish(ctx context.Context, topic string, rawEvent []byte, reason, errMsg string, partition int32, offset int64) {
	entry := DLQEntry{
		OriginalTopic: topic,
		OriginalEvent: rawEvent,
		Error:         errMsg,
		Reason:        reason,
		Timestamp:     time.Now().UTC(),
		Partition:     partition,
		Offset:        offset,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to marshal DLQ entry")
		return
	}

	evt := events.NewEventRaw("audit.dlq.event", "audit-service", "", data)
	if publishErr := d.producer.Publish(ctx, dlqTopic, evt); publishErr != nil {
		d.logger.Error().
			Err(publishErr).
			Str("original_topic", topic).
			Str("reason", reason).
			Msg("failed to publish to DLQ")
		return
	}

	metrics.DLQPublished.WithLabelValues(topic, reason).Inc()

	d.logger.Warn().
		Str("original_topic", topic).
		Str("reason", reason).
		Str("error", errMsg).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("event published to DLQ")
}

// PublishWithEvent is a convenience method for when the event was partially parsed.
func (d *DeadLetterProducer) PublishWithEvent(ctx context.Context, topic string, event *events.Event, reason string, processErr error) {
	raw, _ := json.Marshal(event)
	errMsg := ""
	if processErr != nil {
		errMsg = processErr.Error()
	}
	d.Publish(ctx, topic, raw, reason, errMsg, 0, 0)
}

// NilSafePublish handles the case where the producer may be nil (Kafka unavailable).
func NilSafePublish(dlq *DeadLetterProducer, ctx context.Context, topic string, rawEvent []byte, reason, errMsg string) {
	if dlq == nil || dlq.producer == nil {
		return
	}
	dlq.Publish(ctx, topic, rawEvent, reason, errMsg, 0, 0)
}

// Topic returns the DLQ topic name.
func Topic() string {
	return dlqTopic
}

// DLQTopicName returns the dead letter queue topic for a given service.
func DLQTopicName(service string) string {
	return fmt.Sprintf("platform.%s.dlq", service)
}
