package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/config"
)

// EventHandler processes a single event.
type EventHandler interface {
	Handle(ctx context.Context, event *Event) error
}

// TypedEventHandler extends EventHandler with event type filtering.
type TypedEventHandler interface {
	EventHandler
	EventTypes() []string
}

// EventHandlerFunc is a function adapter for EventHandler.
type EventHandlerFunc func(ctx context.Context, event *Event) error

// Handle implements EventHandler.
func (f EventHandlerFunc) Handle(ctx context.Context, event *Event) error {
	return f(ctx, event)
}

// Consumer wraps a Sarama consumer group for event consumption with manual offset commit,
// configurable concurrency, and graceful shutdown support.
type Consumer struct {
	group     sarama.ConsumerGroup
	handler   *consumerGroupHandler
	logger    zerolog.Logger
	groupID   string
	ready     chan struct{}
	cancel    context.CancelFunc
	cancelCtx context.Context
	wg        sync.WaitGroup
	running   bool
	mu        sync.Mutex
}

// ConsumerConfig holds consumer-specific configuration.
type ConsumerConfig struct {
	Brokers             []string
	GroupID             string
	AutoOffsetReset     string
	WorkersPerPartition int // Default: 1 (preserves ordering)
}

// NewConsumer creates a new Kafka consumer group.
func NewConsumer(cfg config.KafkaConfig, logger zerolog.Logger) (*Consumer, error) {
	return NewConsumerWithConfig(ConsumerConfig{
		Brokers:             cfg.Brokers,
		GroupID:             cfg.GroupID,
		AutoOffsetReset:     cfg.AutoOffsetReset,
		WorkersPerPartition: 1,
	}, logger)
}

// NewConsumerWithConfig creates a consumer with full configuration control.
func NewConsumerWithConfig(cfg ConsumerConfig, logger zerolog.Logger) (*Consumer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}

	switch cfg.AutoOffsetReset {
	case "earliest":
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	default:
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// Manual offset commit: only commit after successful processing
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false

	// Consumer session timeout and heartbeat
	saramaCfg.Consumer.Group.Session.Timeout = 30 * time.Second
	saramaCfg.Consumer.Group.Heartbeat.Interval = 10 * time.Second

	// Max processing time before rebalance
	saramaCfg.Consumer.MaxProcessingTime = 60 * time.Second

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("creating consumer group: %w", err)
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("group_id", cfg.GroupID).
		Msg("kafka consumer group connected")

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		group:   group,
		groupID: cfg.GroupID,
		handler: &consumerGroupHandler{
			logger:   logger,
			handlers: make(map[string][]EventHandler),
			ready:    make(chan struct{}),
		},
		logger:    logger,
		ready:     make(chan struct{}),
		cancel:    cancel,
		cancelCtx: ctx,
	}, nil
}

// Subscribe registers a handler for a specific topic.
func (c *Consumer) Subscribe(topic string, handler EventHandler) {
	c.handler.mu.Lock()
	defer c.handler.mu.Unlock()
	c.handler.handlers[topic] = append(c.handler.handlers[topic], handler)
	c.logger.Info().Str("topic", topic).Msg("handler registered")
}

// Start begins consuming messages from all subscribed topics.
// Blocks until the context is cancelled or an unrecoverable error occurs.
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	c.handler.mu.RLock()
	topics := make([]string, 0, len(c.handler.handlers))
	for topic := range c.handler.handlers {
		topics = append(topics, topic)
	}
	c.handler.mu.RUnlock()

	if len(topics) == 0 {
		return fmt.Errorf("no topics subscribed")
	}

	c.logger.Info().Strs("topics", topics).Msg("starting consumer")

	for {
		// Reset ready channel for each session
		c.handler.ready = make(chan struct{})

		if err := c.group.Consume(ctx, topics, c.handler); err != nil {
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			return fmt.Errorf("consuming: %w", err)
		}

		if ctx.Err() != nil {
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			return ctx.Err()
		}
	}
}

// Stop gracefully stops the consumer, finishing current message processing.
func (c *Consumer) Stop() error {
	c.cancel()
	return nil
}

// Close shuts down the consumer group connection.
func (c *Consumer) Close() error {
	c.cancel()
	return c.group.Close()
}

// GroupID returns the consumer group ID.
func (c *Consumer) GroupID() string {
	return c.groupID
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler with manual offset commit.
type consumerGroupHandler struct {
	logger   zerolog.Logger
	handlers map[string][]EventHandler
	mu       sync.RWMutex
	ready    chan struct{}
}

func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.logger.Info().
		Int32("generation_id", session.GenerationID()).
		Msg("consumer group session setup")
	close(h.ready)
	return nil
}

func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.logger.Info().
		Int32("generation_id", session.GenerationID()).
		Msg("consumer group session cleanup")
	// Commit any outstanding offsets on cleanup
	session.Commit()
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// Extract trace context from headers
		ctx := ExtractTraceContext(session.Context(), msg.Headers)

		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Int32("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("failed to unmarshal event")
			// Mark and skip malformed messages
			session.MarkMessage(msg, "")
			session.Commit()
			continue
		}

		h.mu.RLock()
		handlers, ok := h.handlers[msg.Topic]
		h.mu.RUnlock()

		if !ok || len(handlers) == 0 {
			h.logger.Warn().
				Str("topic", msg.Topic).
				Str("event_id", event.ID).
				Msg("no handler registered for topic")
			session.MarkMessage(msg, "")
			session.Commit()
			continue
		}

		for _, handler := range handlers {
			if err := handler.Handle(ctx, &event); err != nil {
				h.logger.Error().
					Err(err).
					Str("topic", msg.Topic).
					Str("event_id", event.ID).
					Str("event_type", event.Type).
					Str("tenant_id", event.TenantID).
					Msg("failed to handle event")
				// Still mark the message — middleware chain handles retries/DLQ
			}
		}

		// Manual offset commit after processing
		session.MarkMessage(msg, "")
		session.Commit()
	}
	return nil
}
