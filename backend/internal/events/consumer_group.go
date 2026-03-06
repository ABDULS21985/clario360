package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/config"
)

// ConsumerGroupManager manages the lifecycle of a consumer group with multiple
// topic subscriptions, health reporting, and graceful shutdown.
type ConsumerGroupManager struct {
	consumer    *Consumer
	logger      zerolog.Logger
	serviceName string
	startedAt   time.Time

	mu       sync.RWMutex
	running  bool
	lastErr  error
	cancelFn context.CancelFunc
	done     chan struct{}
}

// NewConsumerGroupManager creates a manager that wraps a consumer with lifecycle management.
func NewConsumerGroupManager(cfg config.KafkaConfig, serviceName string, logger zerolog.Logger) (*ConsumerGroupManager, error) {
	consumerCfg := ConsumerConfig{
		Brokers:             cfg.Brokers,
		GroupID:             serviceName,
		AutoOffsetReset:     cfg.AutoOffsetReset,
		WorkersPerPartition: 1,
	}

	consumer, err := NewConsumerWithConfig(consumerCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("creating consumer: %w", err)
	}

	return &ConsumerGroupManager{
		consumer:    consumer,
		logger:      logger,
		serviceName: serviceName,
		done:        make(chan struct{}),
	}, nil
}

// Subscribe registers a handler for a topic.
func (m *ConsumerGroupManager) Subscribe(topic string, handler EventHandler) {
	m.consumer.Subscribe(topic, handler)
}

// SubscribeFunc registers a function handler for a topic.
func (m *ConsumerGroupManager) SubscribeFunc(topic string, fn func(ctx context.Context, event *Event) error) {
	m.consumer.Subscribe(topic, EventHandlerFunc(fn))
}

// Start launches the consumer loop in a goroutine.
// Returns immediately; the consumer runs in the background.
func (m *ConsumerGroupManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("consumer group manager already running")
	}
	m.running = true
	m.startedAt = time.Now()
	m.done = make(chan struct{})
	m.mu.Unlock()

	consumerCtx, cancel := context.WithCancel(ctx)
	m.cancelFn = cancel

	go func() {
		defer close(m.done)

		m.logger.Info().
			Str("service", m.serviceName).
			Msg("consumer group manager starting")

		if err := m.consumer.Start(consumerCtx); err != nil && consumerCtx.Err() == nil {
			m.mu.Lock()
			m.lastErr = err
			m.mu.Unlock()
			m.logger.Error().Err(err).Msg("consumer group manager stopped with error")
		}

		m.mu.Lock()
		m.running = false
		m.mu.Unlock()

		m.logger.Info().Msg("consumer group manager stopped")
	}()

	return nil
}

// Stop gracefully stops the consumer group manager.
// Waits for in-flight messages to complete processing.
func (m *ConsumerGroupManager) Stop() error {
	m.mu.RLock()
	if !m.running {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	m.logger.Info().Msg("stopping consumer group manager")

	if m.cancelFn != nil {
		m.cancelFn()
	}

	// Wait for the consumer goroutine to finish
	<-m.done
	return m.consumer.Close()
}

// Health returns the current health status of the consumer group.
func (m *ConsumerGroupManager) Health() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := HealthStatus{
		Service:   m.serviceName,
		Component: "consumer_group",
	}

	if m.running {
		status.Status = "healthy"
		status.Uptime = time.Since(m.startedAt).String()
	} else if m.lastErr != nil {
		status.Status = "unhealthy"
		status.Error = m.lastErr.Error()
	} else {
		status.Status = "stopped"
	}

	return status
}

// Consumer returns the underlying Consumer for direct access.
func (m *ConsumerGroupManager) Consumer() *Consumer {
	return m.consumer
}
