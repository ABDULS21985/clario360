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
	group       sarama.ConsumerGroup
	handler     *consumerGroupHandler
	logger      zerolog.Logger
	groupID     string
	ready       chan struct{}
	cancel      context.CancelFunc
	cancelCtx   context.Context
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
	dlqProducer *Producer
	dlqTracker  *DLQTracker
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
			logger:           logger,
			handlers:         make(map[string][]EventHandler),
			ready:            make(chan struct{}),
			maxHandlerErrors: 3,
			consumerName:     cfg.GroupID,
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

// SetDeadLetterProducer configures per-topic DLQ publishing for this consumer.
func (c *Consumer) SetDeadLetterProducer(producer *Producer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dlqProducer = producer
	c.handler.dlqProducer = producer
}

// SetCrossSuiteMetrics configures consumer-level metrics used by cross-suite handlers.
func (c *Consumer) SetCrossSuiteMetrics(metrics *CrossSuiteMetrics) {
	c.handler.mu.Lock()
	defer c.handler.mu.Unlock()
	c.handler.metrics = metrics
}

// SetDLQTracker configures Redis-backed DLQ counting for this consumer.
func (c *Consumer) SetDLQTracker(tracker *DLQTracker, serviceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dlqTracker = tracker

	c.handler.mu.Lock()
	defer c.handler.mu.Unlock()
	c.handler.dlqTracker = tracker
	c.handler.dlqServiceName = serviceName
}

// SetMaxHandlerErrors defines how many failed attempts are allowed before
// an event is moved to the DLQ. The default is 3.
func (c *Consumer) SetMaxHandlerErrors(max int) {
	if max < 1 {
		max = 1
	}
	c.handler.mu.Lock()
	defer c.handler.mu.Unlock()
	c.handler.maxHandlerErrors = max
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
	logger           zerolog.Logger
	handlers         map[string][]EventHandler
	mu               sync.RWMutex
	ready            chan struct{}
	dlqProducer      *Producer
	dlqTracker       *DLQTracker
	dlqServiceName   string
	maxHandlerErrors int
	metrics          *CrossSuiteMetrics
	consumerName     string
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
		start := time.Now()

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
		if event.Metadata == nil {
			event.Metadata = map[string]string{}
		}
		event.Metadata["kafka.topic"] = msg.Topic

		h.mu.RLock()
		handlers, ok := h.handlers[msg.Topic]
		metrics := h.metrics
		consumerName := h.consumerName
		maxHandlerErrors := h.maxHandlerErrors
		h.mu.RUnlock()

		if metrics != nil {
			metrics.ReceivedTotal.WithLabelValues(consumerName, sourceSuiteFromEventType(event.Type), event.Type).Inc()
		}

		if !ok || len(handlers) == 0 {
			h.logger.Warn().
				Str("topic", msg.Topic).
				Str("event_id", event.ID).
				Msg("no handler registered for topic")
			if metrics != nil {
				metrics.ProcessedTotal.WithLabelValues(consumerName, sourceSuiteFromEventType(event.Type), event.Type, "skipped").Inc()
				metrics.ProcessingDurationSeconds.WithLabelValues(consumerName, event.Type).Observe(time.Since(start).Seconds())
			}
			session.MarkMessage(msg, "")
			session.Commit()
			continue
		}

		handlerFailed := false
		for _, handler := range handlers {
			if typed, ok := handler.(TypedEventHandler); ok && !eventTypeAllowed(typed.EventTypes(), event.Type) {
				continue
			}
			if err := h.processWithRetry(ctx, handler, &event, msg.Topic, maxHandlerErrors); err != nil {
				handlerFailed = true
				h.logger.Error().
					Err(err).
					Str("topic", msg.Topic).
					Str("event_id", event.ID).
					Str("event_type", event.Type).
					Str("tenant_id", event.TenantID).
					Msg("failed to handle event")
			}
		}

		if metrics != nil {
			result := "success"
			if handlerFailed {
				result = "error"
			}
			metrics.ProcessedTotal.WithLabelValues(consumerName, sourceSuiteFromEventType(event.Type), event.Type, result).Inc()
			metrics.ProcessingDurationSeconds.WithLabelValues(consumerName, event.Type).Observe(time.Since(start).Seconds())
		}

		// Manual offset commit after processing
		session.MarkMessage(msg, "")
		session.Commit()
	}
	return nil
}

func (h *consumerGroupHandler) processWithRetry(ctx context.Context, handler EventHandler, event *Event, topic string, maxAttempts int) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		retryCtx := context.WithValue(ctx, retryContextKey{}, attempt-1)
		lastErr = handler.Handle(retryCtx, event)
		if lastErr == nil {
			return nil
		}
		if attempt == maxAttempts {
			break
		}

		delay := time.Duration(attempt) * 250 * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	if h.dlqProducer != nil {
		if err := h.publishToDLQ(ctx, topic, event, lastErr, maxAttempts); err != nil {
			h.logger.Error().
				Err(err).
				Str("topic", topic).
				Str("event_id", event.ID).
				Msg("failed to publish dead letter event")
		}
	}
	if h.metrics != nil {
		h.metrics.DeadLetteredTotal.WithLabelValues(h.consumerName, event.Type).Inc()
	}

	h.logger.Error().
		Err(lastErr).
		Str("topic", topic).
		Str("event_type", event.Type).
		Str("event_id", event.ID).
		Int("retries", maxAttempts).
		Msg("Event moved to DLQ")

	return lastErr
}

func (h *consumerGroupHandler) publishToDLQ(ctx context.Context, topic string, event *Event, handlerErr error, retryCount int) error {
	if h.dlqProducer == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"original_event": event,
		"error":          handlerErr.Error(),
		"retry_count":    retryCount,
		"timestamp":      time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal dlq payload: %w", err)
	}

	dlqEvent := NewEventRaw(event.Type, event.Source, event.TenantID, payload)
	dlqEvent.CorrelationID = event.CorrelationID
	dlqEvent.CausationID = event.ID
	dlqEvent.UserID = event.UserID
	dlqEvent.Metadata = map[string]string{
		"dlq.original_event_id": event.ID,
		"dlq.original_type":     event.Type,
		"dlq.original_topic":    topic,
		"dlq.service_name":      h.serviceName(),
		"dlq.error":             handlerErr.Error(),
		"dlq.retry_count":       fmt.Sprintf("%d", retryCount),
		"dlq.failed_at":         time.Now().UTC().Format(time.RFC3339Nano),
	}

	if err := h.dlqProducer.Publish(ctx, topic+".dlq", dlqEvent); err != nil {
		return err
	}
	if err := h.dlqProducer.Publish(ctx, Topics.DeadLetter, dlqEvent); err != nil {
		return err
	}
	if h.dlqTracker != nil {
		if err := h.dlqTracker.Increment(ctx, h.serviceName(), topic); err != nil {
			h.logger.Warn().
				Err(err).
				Str("service", h.serviceName()).
				Str("topic", topic).
				Msg("failed to increment dlq tracker")
		}
	}
	return nil
}

func (h *consumerGroupHandler) serviceName() string {
	if h.dlqServiceName != "" {
		return h.dlqServiceName
	}
	return h.consumerName
}

func sourceSuiteFromEventType(eventType string) string {
	switch {
	case eventType == "":
		return "unknown"
	case eventType == "com.clario360.file.uploaded",
		eventType == "com.clario360.file.scan.infected",
		eventType == "com.clario360.file.quarantined",
		eventType == "com.clario360.file.scan.error":
		return "file"
	case len(eventType) > len("com.clario360.") && eventType[:len("com.clario360.")] == "com.clario360.":
		trimmed := eventType[len("com.clario360."):]
		for idx := 0; idx < len(trimmed); idx++ {
			if trimmed[idx] == '.' {
				return trimmed[:idx]
			}
		}
		return trimmed
	default:
		return "unknown"
	}
}

func eventTypeAllowed(allowed []string, eventType string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if candidate == eventType {
			return true
		}
	}
	return false
}
