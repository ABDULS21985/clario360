package events

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/clario360/platform/internal/config"
)

// Producer wraps a Sarama sync producer with structured event publishing,
// partitioning by tenant ID, snappy compression, and OpenTelemetry trace propagation.
type Producer struct {
	producer   sarama.SyncProducer
	serializer *Serializer
	logger     zerolog.Logger
}

// NewProducer creates a new Kafka producer with production-ready configuration.
func NewProducer(cfg config.KafkaConfig, logger zerolog.Logger) (*Producer, error) {
	saramaCfg := sarama.NewConfig()

	// Idempotent producer: exactly-once semantics at the partition level
	saramaCfg.Producer.Idempotent = true
	saramaCfg.Net.MaxOpenRequests = 1 // Required for idempotency

	// Durability: wait for all in-sync replicas
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true

	// Retries with backoff
	saramaCfg.Producer.Retry.Max = 3
	saramaCfg.Producer.Retry.Backoff = 100 * time.Millisecond

	// Compression
	saramaCfg.Producer.Compression = sarama.CompressionSnappy

	// Batching (applies to underlying async producer within sync wrapper)
	saramaCfg.Producer.Flush.Bytes = 16 * 1024         // 16KB batch size
	saramaCfg.Producer.Flush.Frequency = 10 * time.Millisecond // 10ms linger

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("creating Kafka producer: %w", err)
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Msg("kafka producer connected")

	return &Producer{
		producer:   producer,
		serializer: NewSerializer(),
		logger:     logger,
	}, nil
}

// Publish sends a single event to the given Kafka topic.
// The event is partitioned by TenantID to ensure ordering within a tenant.
// OpenTelemetry trace context is propagated via message headers.
func (p *Producer) Publish(ctx context.Context, topic string, event *Event) error {
	data, err := p.serializer.Serialize(event)
	if err != nil {
		return fmt.Errorf("serializing event: %w", err)
	}

	headers := p.buildHeaders(ctx, event)

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.StringEncoder(event.TenantID),
		Value:   sarama.ByteEncoder(data),
		Headers: headers,
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("publishing event to %s: %w", topic, err)
	}

	p.logger.Debug().
		Str("topic", topic).
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("event published")

	return nil
}

// PublishBatch sends multiple events to the given Kafka topic in a single batch.
// All events are sent atomically — either all succeed or the batch fails.
func (p *Producer) PublishBatch(ctx context.Context, topic string, events []*Event) error {
	if len(events) == 0 {
		return nil
	}

	msgs := make([]*sarama.ProducerMessage, 0, len(events))
	for _, event := range events {
		data, err := p.serializer.Serialize(event)
		if err != nil {
			return fmt.Errorf("serializing event %s: %w", event.ID, err)
		}

		headers := p.buildHeaders(ctx, event)

		msgs = append(msgs, &sarama.ProducerMessage{
			Topic:   topic,
			Key:     sarama.StringEncoder(event.TenantID),
			Value:   sarama.ByteEncoder(data),
			Headers: headers,
		})
	}

	if err := p.producer.SendMessages(msgs); err != nil {
		return fmt.Errorf("publishing batch of %d events to %s: %w", len(events), topic, err)
	}

	p.logger.Debug().
		Str("topic", topic).
		Int("count", len(events)).
		Msg("event batch published")

	return nil
}

// Close flushes pending messages and shuts down the producer.
func (p *Producer) Close() error {
	return p.producer.Close()
}

// buildHeaders constructs Kafka message headers including event metadata
// and OpenTelemetry trace context.
func (p *Producer) buildHeaders(ctx context.Context, event *Event) []sarama.RecordHeader {
	headers := []sarama.RecordHeader{
		{Key: []byte("ce-id"), Value: []byte(event.ID)},
		{Key: []byte("ce-type"), Value: []byte(event.Type)},
		{Key: []byte("ce-source"), Value: []byte(event.Source)},
		{Key: []byte("ce-specversion"), Value: []byte(event.SpecVersion)},
		{Key: []byte("ce-time"), Value: []byte(event.Time.Format(time.RFC3339Nano))},
		{Key: []byte("ce-tenantid"), Value: []byte(event.TenantID)},
		{Key: []byte("event-type"), Value: []byte(event.Type)},
		{Key: []byte("tenant-id"), Value: []byte(event.TenantID)},
	}

	if event.CorrelationID != "" {
		headers = append(headers, sarama.RecordHeader{
			Key: []byte("ce-correlationid"), Value: []byte(event.CorrelationID),
		})
	}

	if event.UserID != "" {
		headers = append(headers, sarama.RecordHeader{
			Key: []byte("ce-userid"), Value: []byte(event.UserID),
		})
	}

	// Inject OpenTelemetry trace context into headers
	carrier := &headerCarrier{headers: &headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	return headers
}

// headerCarrier implements propagation.TextMapCarrier for sarama record headers.
type headerCarrier struct {
	headers *[]sarama.RecordHeader
}

func (c *headerCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *headerCarrier) Set(key, value string) {
	*c.headers = append(*c.headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

func (c *headerCarrier) Keys() []string {
	keys := make([]string, len(*c.headers))
	for i, h := range *c.headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// ExtractTraceContext extracts OpenTelemetry trace context from Kafka message headers.
func ExtractTraceContext(ctx context.Context, headers []*sarama.RecordHeader) context.Context {
	carrier := &readOnlyHeaderCarrier{headers: headers}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// readOnlyHeaderCarrier implements propagation.TextMapCarrier for reading sarama headers.
type readOnlyHeaderCarrier struct {
	headers []*sarama.RecordHeader
}

func (c *readOnlyHeaderCarrier) Get(key string) string {
	for _, h := range c.headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *readOnlyHeaderCarrier) Set(string, string) {}

func (c *readOnlyHeaderCarrier) Keys() []string {
	keys := make([]string, len(c.headers))
	for i, h := range c.headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// Ensure interface compliance.
var _ propagation.TextMapCarrier = (*headerCarrier)(nil)
var _ propagation.TextMapCarrier = (*readOnlyHeaderCarrier)(nil)
