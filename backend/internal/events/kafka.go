package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/config"
)

// Producer wraps a Sarama sync producer with structured event publishing.
type Producer struct {
	producer sarama.SyncProducer
	logger   zerolog.Logger
}

// NewProducer creates a new Kafka producer.
func NewProducer(cfg config.KafkaConfig, logger zerolog.Logger) (*Producer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Retry.Max = 3
	saramaCfg.Producer.Idempotent = true
	saramaCfg.Net.MaxOpenRequests = 1

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("creating Kafka producer: %w", err)
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Msg("kafka producer connected")

	return &Producer{
		producer: producer,
		logger:   logger,
	}, nil
}

// Publish sends an event to the given Kafka topic.
func (p *Producer) Publish(ctx context.Context, topic string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.TenantID),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event-type"), Value: []byte(event.Type)},
			{Key: []byte("tenant-id"), Value: []byte(event.TenantID)},
		},
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("sending Kafka message: %w", err)
	}

	p.logger.Debug().
		Str("topic", topic).
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("event published")

	return nil
}

// Close shuts down the producer.
func (p *Producer) Close() error {
	return p.producer.Close()
}

// Consumer wraps a Sarama consumer group for event consumption.
type Consumer struct {
	group   sarama.ConsumerGroup
	logger  zerolog.Logger
	handler *consumerGroupHandler
}

// NewConsumer creates a new Kafka consumer group.
func NewConsumer(cfg config.KafkaConfig, logger zerolog.Logger) (*Consumer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}

	switch cfg.AutoOffsetReset {
	case "earliest":
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	default:
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("creating consumer group: %w", err)
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("group_id", cfg.GroupID).
		Msg("kafka consumer group connected")

	return &Consumer{
		group:  group,
		logger: logger,
		handler: &consumerGroupHandler{
			logger:   logger,
			handlers: make(map[string]EventHandler),
		},
	}, nil
}

// Subscribe registers a handler for a specific topic.
func (c *Consumer) Subscribe(topic string, handler EventHandler) {
	c.handler.mu.Lock()
	defer c.handler.mu.Unlock()
	c.handler.handlers[topic] = handler
}

// Start begins consuming messages from all subscribed topics.
func (c *Consumer) Start(ctx context.Context) error {
	c.handler.mu.RLock()
	topics := make([]string, 0, len(c.handler.handlers))
	for topic := range c.handler.handlers {
		topics = append(topics, topic)
	}
	c.handler.mu.RUnlock()

	c.logger.Info().
		Strs("topics", topics).
		Msg("starting consumer")

	for {
		if err := c.group.Consume(ctx, topics, c.handler); err != nil {
			return fmt.Errorf("consuming: %w", err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// Close shuts down the consumer group.
func (c *Consumer) Close() error {
	return c.group.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	logger   zerolog.Logger
	handlers map[string]EventHandler
	mu       sync.RWMutex
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Msg("failed to unmarshal event")
			session.MarkMessage(msg, "")
			continue
		}

		h.mu.RLock()
		handler, ok := h.handlers[msg.Topic]
		h.mu.RUnlock()

		if !ok {
			h.logger.Warn().
				Str("topic", msg.Topic).
				Msg("no handler registered for topic")
			session.MarkMessage(msg, "")
			continue
		}

		if err := handler.Handle(session.Context(), &event); err != nil {
			h.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Str("event_id", event.ID).
				Str("event_type", event.Type).
				Msg("failed to handle event")
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
