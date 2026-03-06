package events

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ConsumerMiddleware wraps an EventHandler with additional behavior.
type ConsumerMiddleware func(EventHandler) EventHandler

// ApplyMiddleware chains middleware around a handler.
// Middleware is applied in reverse order so the first middleware in the list
// is the outermost wrapper (executed first).
func ApplyMiddleware(handler EventHandler, middlewares ...ConsumerMiddleware) EventHandler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// WithLogging returns middleware that logs event processing with type, tenant, and duration.
func WithLogging(logger zerolog.Logger) ConsumerMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			start := time.Now()

			err := next.Handle(ctx, event)

			l := logger.With().
				Str("event_id", event.ID).
				Str("event_type", event.Type).
				Str("tenant_id", event.TenantID).
				Str("source", event.Source).
				Dur("duration", time.Since(start)).
				Logger()

			if err != nil {
				l.Error().Err(err).Msg("event processing failed")
			} else {
				l.Debug().Msg("event processed")
			}

			return err
		})
	}
}

// EventConsumerMetrics holds Prometheus metrics for event consumers.
type EventConsumerMetrics struct {
	ProcessedTotal  *prometheus.CounterVec
	ProcessDuration *prometheus.HistogramVec
	ErrorsTotal     *prometheus.CounterVec
}

// NewEventConsumerMetrics creates and registers consumer metrics.
func NewEventConsumerMetrics(namespace string) *EventConsumerMetrics {
	return &EventConsumerMetrics{
		ProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "events",
				Name:      "processed_total",
				Help:      "Total number of events processed.",
			},
			[]string{"event_type", "status"},
		),
		ProcessDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "events",
				Name:      "process_duration_seconds",
				Help:      "Event processing duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"event_type"},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "events",
				Name:      "errors_total",
				Help:      "Total number of event processing errors.",
			},
			[]string{"event_type"},
		),
	}
}

// WithMetrics returns middleware that tracks event processing time and success/failure rates.
func WithMetrics(metrics *EventConsumerMetrics) ConsumerMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			start := time.Now()

			err := next.Handle(ctx, event)

			duration := time.Since(start).Seconds()
			metrics.ProcessDuration.WithLabelValues(event.Type).Observe(duration)

			if err != nil {
				metrics.ProcessedTotal.WithLabelValues(event.Type, "error").Inc()
				metrics.ErrorsTotal.WithLabelValues(event.Type).Inc()
			} else {
				metrics.ProcessedTotal.WithLabelValues(event.Type, "success").Inc()
			}

			return err
		})
	}
}

// BackoffPolicy calculates the delay before the next retry attempt.
type BackoffPolicy func(attempt int) time.Duration

// ExponentialBackoff returns an exponential backoff policy with the given base delay.
// delay(n) = base * 2^n, capped at maxDelay.
func ExponentialBackoff(base, maxDelay time.Duration) BackoffPolicy {
	return func(attempt int) time.Duration {
		delay := base
		for i := 0; i < attempt; i++ {
			delay *= 2
			if delay > maxDelay {
				return maxDelay
			}
		}
		return delay
	}
}

// retryContextKey is used to track retry count in context.
type retryContextKey struct{}

// RetryCountFromContext returns the current retry attempt number from context.
func RetryCountFromContext(ctx context.Context) int {
	if v, ok := ctx.Value(retryContextKey{}).(int); ok {
		return v
	}
	return 0
}

// WithRetry returns middleware that retries failed event handlers with configurable backoff.
// After maxRetries exhausted, the error propagates to the next middleware (typically DLQ).
func WithRetry(maxRetries int, backoff BackoffPolicy) ConsumerMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			var lastErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				retryCtx := context.WithValue(ctx, retryContextKey{}, attempt)
				lastErr = next.Handle(retryCtx, event)
				if lastErr == nil {
					return nil
				}

				if attempt < maxRetries {
					delay := backoff(attempt)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
					}
				}
			}
			return lastErr
		})
	}
}

// WithDeadLetter returns middleware that sends failed events to the dead letter queue.
// Should be placed after WithRetry in the middleware chain so it only catches
// events that exhausted all retries.
func WithDeadLetter(dlqProducer *Producer, logger zerolog.Logger) ConsumerMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			err := next.Handle(ctx, event)
			if err != nil {
				dlqErr := sendToDeadLetter(ctx, dlqProducer, event, err, logger)
				if dlqErr != nil {
					logger.Error().
						Err(dlqErr).
						Str("event_id", event.ID).
						Str("event_type", event.Type).
						Msg("failed to send event to dead letter queue")
				}
				// Return nil to prevent the consumer from retrying
				// The event is now in the DLQ for manual inspection
				return nil
			}
			return nil
		})
	}
}

// sendToDeadLetter publishes a failed event to the dead letter topic with error metadata.
func sendToDeadLetter(ctx context.Context, producer *Producer, event *Event, originalErr error, logger zerolog.Logger) error {
	dlqEvent := &Event{
		ID:              GenerateUUID(),
		Source:          event.Source,
		SpecVersion:     "1.0",
		Type:            event.Type,
		DataContentType: event.DataContentType,
		Subject:         event.Subject,
		Time:            time.Now().UTC(),
		TenantID:        event.TenantID,
		UserID:          event.UserID,
		CorrelationID:   event.CorrelationID,
		CausationID:     event.ID,
		Data:            event.Data,
		Metadata: map[string]string{
			"dlq.original_event_id": event.ID,
			"dlq.original_type":     event.Type,
			"dlq.error":             originalErr.Error(),
			"dlq.failed_at":         time.Now().UTC().Format(time.RFC3339),
			"dlq.retry_count":       fmt.Sprintf("%d", RetryCountFromContext(ctx)),
		},
	}

	logger.Warn().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Str("error", originalErr.Error()).
		Msg("sending event to dead letter queue")

	return producer.Publish(ctx, Topics.DeadLetter, dlqEvent)
}

// IdempotencyStore checks and records processed event IDs to prevent duplicate processing.
type IdempotencyStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewIdempotencyStore creates a Redis-backed idempotency store.
// TTL controls how long processed event IDs are retained (default: 7 days).
func NewIdempotencyStore(rdb *redis.Client, ttl time.Duration) *IdempotencyStore {
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &IdempotencyStore{
		rdb: rdb,
		ttl: ttl,
	}
}

// IsProcessed checks if an event has already been processed.
func (s *IdempotencyStore) IsProcessed(ctx context.Context, consumerGroup, eventID string) (bool, error) {
	key := fmt.Sprintf("idempotent:%s:%s", consumerGroup, eventID)
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("checking idempotency: %w", err)
	}
	return exists > 0, nil
}

// MarkProcessed records that an event has been processed.
func (s *IdempotencyStore) MarkProcessed(ctx context.Context, consumerGroup, eventID string) error {
	key := fmt.Sprintf("idempotent:%s:%s", consumerGroup, eventID)
	return s.rdb.Set(ctx, key, time.Now().UTC().Format(time.RFC3339), s.ttl).Err()
}

// WithIdempotency returns middleware that skips already-processed events.
// Uses Redis to track processed event IDs per consumer group.
func WithIdempotency(store *IdempotencyStore, consumerGroup string) ConsumerMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			processed, err := store.IsProcessed(ctx, consumerGroup, event.ID)
			if err != nil {
				// Fail open: if Redis is unavailable, process the event
				// The handler should be idempotent regardless
				return next.Handle(ctx, event)
			}

			if processed {
				return nil
			}

			if err := next.Handle(ctx, event); err != nil {
				return err
			}

			// Mark as processed only after successful handling
			if markErr := store.MarkProcessed(ctx, consumerGroup, event.ID); markErr != nil {
				// Log but don't fail — the event was processed successfully
				// Worst case: it gets processed again (at-least-once)
			}

			return nil
		})
	}
}

// WithTracing returns middleware that creates an OpenTelemetry span per event.
func WithTracing(tracer trace.Tracer) ConsumerMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event *Event) error {
			ctx, span := tracer.Start(ctx, fmt.Sprintf("event.process.%s", event.Type),
				trace.WithAttributes(
					attribute.String("event.id", event.ID),
					attribute.String("event.type", event.Type),
					attribute.String("event.source", event.Source),
					attribute.String("event.tenant_id", event.TenantID),
				),
			)
			defer span.End()

			err := next.Handle(ctx, event)
			if err != nil {
				span.RecordError(err)
			}
			return err
		})
	}
}
