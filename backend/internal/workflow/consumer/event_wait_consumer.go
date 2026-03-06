package consumer

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
)

// Redis key prefix for event-wait registrations.
// Format: "workflow:eventwait:{topic}:{correlation_value}"
// Value:  "{instanceID}:{stepID}"
const eventWaitKeyPrefix = "workflow:eventwait"

// workflowAdvancer advances a workflow instance past a waiting step.
type workflowAdvancer interface {
	AdvanceWorkflow(ctx context.Context, instanceID, fromStepID string) error
}

// EventWaitConsumer listens for platform events and matches them against
// workflow instances that are parked on an event_task step waiting for a
// specific event correlation value to arrive.
type EventWaitConsumer struct {
	rdb    *redis.Client
	engine workflowAdvancer
	logger zerolog.Logger
}

// NewEventWaitConsumer creates a new EventWaitConsumer.
func NewEventWaitConsumer(
	rdb *redis.Client,
	engine workflowAdvancer,
	logger zerolog.Logger,
) *EventWaitConsumer {
	return &EventWaitConsumer{
		rdb:    rdb,
		engine: engine,
		logger: logger.With().Str("consumer", "workflow_event_wait").Logger(),
	}
}

// Handle processes an incoming event by checking Redis for any workflow instances
// that have registered a wait for the event's topic and correlation value.
//
// The registration key format is: "workflow:eventwait:{topic}:{correlation_value}"
// The stored value is: "{instanceID}:{stepID}"
//
// When a match is found, the key is deleted and the workflow engine is called
// to advance the instance past the waiting step.
func (c *EventWaitConsumer) Handle(ctx context.Context, event *events.Event) error {
	if event == nil {
		return fmt.Errorf("event wait consumer: received nil event")
	}

	if err := event.Validate(); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("invalid event received, skipping")
		return nil
	}

	// Derive the topic used for wait-key lookup.
	topic := extractTopic(event)

	// Determine the correlation value for this event.
	// The correlation value is typically the event's CorrelationID or Subject,
	// depending on how the event_task step was configured.
	correlationValues := c.extractCorrelationValues(event)

	if len(correlationValues) == 0 {
		c.logger.Debug().
			Str("event_id", event.ID).
			Str("topic", topic).
			Msg("no correlation values found in event, skipping")
		return nil
	}

	var lastErr error

	for _, corrVal := range correlationValues {
		key := fmt.Sprintf("%s:%s:%s", eventWaitKeyPrefix, topic, corrVal)

		// Atomically get and delete the wait registration.
		val, err := c.rdb.GetDel(ctx, key).Result()
		if err == redis.Nil {
			// No matching wait registration for this correlation value.
			continue
		}
		if err != nil {
			c.logger.Error().Err(err).
				Str("key", key).
				Str("event_id", event.ID).
				Msg("redis GetDel failed for event wait key")
			lastErr = fmt.Errorf("redis GetDel for key %q: %w", key, err)
			continue
		}

		// Parse the stored value: "instanceID:stepID"
		instanceID, stepID, err := parseWaitValue(val)
		if err != nil {
			c.logger.Error().Err(err).
				Str("key", key).
				Str("value", val).
				Msg("invalid wait registration value")
			lastErr = fmt.Errorf("parsing wait value %q: %w", val, err)
			continue
		}

		c.logger.Info().
			Str("event_id", event.ID).
			Str("instance_id", instanceID).
			Str("step_id", stepID).
			Str("topic", topic).
			Str("correlation_value", corrVal).
			Msg("matched event to waiting workflow step, advancing")

		if err := c.engine.AdvanceWorkflow(ctx, instanceID, stepID); err != nil {
			c.logger.Error().Err(err).
				Str("instance_id", instanceID).
				Str("step_id", stepID).
				Msg("failed to advance workflow after event match")
			lastErr = fmt.Errorf("advance workflow instance %s step %s: %w", instanceID, stepID, err)
			continue
		}

		c.logger.Info().
			Str("instance_id", instanceID).
			Str("step_id", stepID).
			Msg("workflow advanced past event wait step")
	}

	return lastErr
}

// extractCorrelationValues returns the set of correlation values that should be
// checked against wait registrations. This includes:
// 1. The event's CorrelationID (primary correlation mechanism)
// 2. The event's Subject (resource-level correlation)
// 3. The event's ID (exact event matching)
func (c *EventWaitConsumer) extractCorrelationValues(event *events.Event) []string {
	seen := make(map[string]bool)
	var values []string

	add := func(v string) {
		if v != "" && !seen[v] {
			seen[v] = true
			values = append(values, v)
		}
	}

	add(event.CorrelationID)
	add(event.Subject)
	add(event.ID)

	return values
}

// parseWaitValue splits a stored wait registration value into instanceID and stepID.
// The expected format is "{instanceID}:{stepID}".
func parseWaitValue(val string) (instanceID, stepID string, err error) {
	// Find the last colon to split on, since UUIDs contain hyphens but not colons.
	idx := strings.LastIndex(val, ":")
	if idx < 0 || idx == 0 || idx == len(val)-1 {
		return "", "", fmt.Errorf("invalid wait value format: expected 'instanceID:stepID', got %q", val)
	}

	instanceID = val[:idx]
	stepID = val[idx+1:]

	if instanceID == "" || stepID == "" {
		return "", "", fmt.Errorf("invalid wait value: instanceID or stepID is empty in %q", val)
	}

	return instanceID, stepID, nil
}
