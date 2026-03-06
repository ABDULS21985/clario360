package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/workflow/expression"
	"github.com/clario360/platform/internal/workflow/model"
)

// EventPublisher is a narrow interface wrapping events.Producer.Publish, used to
// avoid a direct dependency on the concrete producer implementation.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

// EventTaskExecutor handles two modes of event interaction within a workflow:
//   - PUBLISH: builds and publishes a CloudEvents event to a Kafka topic
//   - WAIT: registers a wait in Redis for a correlated event and parks the workflow
type EventTaskExecutor struct {
	producer EventPublisher
	rdb      *redis.Client
	resolver *expression.VariableResolver
	logger   zerolog.Logger
}

// NewEventTaskExecutor creates an EventTaskExecutor.
func NewEventTaskExecutor(producer EventPublisher, rdb *redis.Client, logger zerolog.Logger) *EventTaskExecutor {
	return &EventTaskExecutor{
		producer: producer,
		rdb:      rdb,
		resolver: expression.NewVariableResolver(),
		logger:   logger.With().Str("executor", "event_task").Logger(),
	}
}

// Execute publishes an event or registers an event wait based on the step configuration.
//
// Expected step.Config keys:
//   - mode (string, required): "PUBLISH" or "WAIT"
//
// For PUBLISH mode:
//   - topic (string, required): Kafka topic to publish to
//   - event_type (string, required): CloudEvents event type
//   - data (map, optional): event payload with possible ${...} references
//   - source (string, optional): event source, defaults to "workflow-engine"
//
// For WAIT mode:
//   - topic (string, required): topic to wait for events on
//   - correlation_field (string, required): field path in the incoming event to match
//   - correlation_value (string, required): expected value (may be a ${...} reference)
//   - timeout_seconds (float64, optional): how long to wait before timing out
func (e *EventTaskExecutor) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	mode, err := configString(step.Config, "mode")
	if err != nil {
		return nil, fmt.Errorf("event_task %s: %w", step.ID, err)
	}

	switch mode {
	case "PUBLISH":
		return e.executePublish(ctx, instance, step, exec)
	case "WAIT":
		return e.executeWait(ctx, instance, step, exec)
	default:
		return nil, fmt.Errorf("event_task %s: unsupported mode %q (expected PUBLISH or WAIT)", step.ID, mode)
	}
}

// executePublish builds an event from configuration and publishes it to Kafka.
func (e *EventTaskExecutor) executePublish(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	topic, err := configString(step.Config, "topic")
	if err != nil {
		return nil, fmt.Errorf("event_task %s: %w", step.ID, err)
	}

	eventType, err := configString(step.Config, "event_type")
	if err != nil {
		return nil, fmt.Errorf("event_task %s: %w", step.ID, err)
	}

	source := configStringOptional(step.Config, "source")
	if source == "" {
		source = "workflow-engine"
	}

	dataCtx := buildDataContext(instance)

	// Resolve event data payload.
	var eventData interface{}
	if dataRaw, ok := step.Config["data"]; ok && dataRaw != nil {
		resolved, err := e.resolver.Resolve(dataRaw, dataCtx)
		if err != nil {
			return nil, fmt.Errorf("event_task %s: resolving event data: %w", step.ID, err)
		}
		eventData = resolved
	} else {
		eventData = map[string]interface{}{
			"instance_id": instance.ID,
			"step_id":     step.ID,
		}
	}

	// Marshal event data to JSON.
	dataBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("event_task %s: marshaling event data: %w", step.ID, err)
	}

	evt := events.NewEventRaw(eventType, source, instance.TenantID, dataBytes)
	evt.Subject = instance.ID

	if err := e.producer.Publish(ctx, topic, evt); err != nil {
		return nil, fmt.Errorf("event_task %s: publishing event: %w", step.ID, err)
	}

	e.logger.Info().
		Str("step_id", step.ID).
		Str("instance_id", instance.ID).
		Str("topic", topic).
		Str("event_type", eventType).
		Str("event_id", evt.ID).
		Msg("event published")

	return &ExecutionResult{
		Output: map[string]interface{}{
			"event_id":   evt.ID,
			"topic":      topic,
			"event_type": eventType,
			"published":  true,
		},
	}, nil
}

// executeWait registers a correlation wait in Redis and parks the workflow.
// When the matching event arrives, a separate consumer will resume the workflow.
func (e *EventTaskExecutor) executeWait(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	topic, err := configString(step.Config, "topic")
	if err != nil {
		return nil, fmt.Errorf("event_task %s: %w", step.ID, err)
	}

	correlationValue, err := configString(step.Config, "correlation_value")
	if err != nil {
		return nil, fmt.Errorf("event_task %s: %w", step.ID, err)
	}

	// Resolve the correlation value if it is a variable reference.
	dataCtx := buildDataContext(instance)
	resolved, err := e.resolver.Resolve(correlationValue, dataCtx)
	if err != nil {
		return nil, fmt.Errorf("event_task %s: resolving correlation_value: %w", step.ID, err)
	}
	resolvedCorrelation := fmt.Sprintf("%v", resolved)

	// Build Redis key: workflow:eventwait:{topic}:{correlation_value}
	redisKey := fmt.Sprintf("workflow:eventwait:%s:%s", topic, resolvedCorrelation)
	redisValue := fmt.Sprintf("%s:%s", instance.ID, step.ID)

	// Determine TTL from timeout_seconds config (default 7 days).
	ttl := 7 * 24 * time.Hour
	if v, ok := step.Config["timeout_seconds"]; ok {
		if seconds := toFloat(v); seconds > 0 {
			ttl = time.Duration(seconds * float64(time.Second))
		}
	}

	// Register the wait in Redis.
	if err := e.rdb.Set(ctx, redisKey, redisValue, ttl).Err(); err != nil {
		return nil, fmt.Errorf("event_task %s: registering event wait in Redis: %w", step.ID, err)
	}

	e.logger.Info().
		Str("step_id", step.ID).
		Str("instance_id", instance.ID).
		Str("topic", topic).
		Str("correlation_value", resolvedCorrelation).
		Str("redis_key", redisKey).
		Dur("ttl", ttl).
		Msg("event wait registered, parking workflow")

	return &ExecutionResult{
		Output: map[string]interface{}{
			"waiting_for":       topic,
			"correlation_value": resolvedCorrelation,
			"redis_key":         redisKey,
		},
		Parked: true,
	}, nil
}
