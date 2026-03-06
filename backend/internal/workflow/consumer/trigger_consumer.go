package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/model"
)

// deduplication TTL for trigger events: prevents the same event from starting
// duplicate workflow instances within a 24-hour window.
const triggerDedupTTL = 24 * time.Hour

// definitionRepoReader provides read access to workflow definitions by trigger topic.
type definitionRepoReader interface {
	GetActiveByTriggerTopic(ctx context.Context, topic string) ([]*model.WorkflowDefinition, error)
}

// workflowStarter starts new workflow instances.
type workflowStarter interface {
	StartInstance(ctx context.Context, tenantID, userID string, req dto.StartInstanceRequest) (*model.WorkflowInstance, error)
}

// TriggerConsumer listens for platform events and starts workflow instances
// whose trigger configuration matches the incoming event's source topic.
type TriggerConsumer struct {
	defRepo definitionRepoReader
	engine  workflowStarter
	rdb     *redis.Client
	logger  zerolog.Logger
}

// NewTriggerConsumer creates a new TriggerConsumer.
func NewTriggerConsumer(
	defRepo definitionRepoReader,
	engine workflowStarter,
	rdb *redis.Client,
	logger zerolog.Logger,
) *TriggerConsumer {
	return &TriggerConsumer{
		defRepo: defRepo,
		engine:  engine,
		rdb:     rdb,
		logger:  logger.With().Str("consumer", "workflow_trigger").Logger(),
	}
}

// Handle processes an incoming event by matching it against active workflow
// definitions that are configured to trigger on the event's source topic.
// For each matching definition whose filter evaluates to true, it deduplicates
// via Redis SET NX and starts a new workflow instance.
func (c *TriggerConsumer) Handle(ctx context.Context, event *events.Event) error {
	if event == nil {
		return fmt.Errorf("trigger consumer: received nil event")
	}

	if err := event.Validate(); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("invalid event received, skipping")
		return nil
	}

	// Extract the source topic from the event type.
	// Event types follow the pattern "com.clario360.<domain>.<entity>.<action>".
	// The trigger topic in definitions maps to the Kafka topic the event came from.
	// We derive the topic from the event source field (e.g., "clario360/workflow-service")
	// or use the event type itself for matching.
	topic := extractTopic(event)

	definitions, err := c.defRepo.GetActiveByTriggerTopic(ctx, topic)
	if err != nil {
		return fmt.Errorf("trigger consumer: fetching definitions for topic %q: %w", topic, err)
	}

	if len(definitions) == 0 {
		c.logger.Debug().
			Str("topic", topic).
			Str("event_id", event.ID).
			Msg("no active definitions for topic")
		return nil
	}

	// Parse event data once for filter evaluation.
	var eventData map[string]interface{}
	if len(event.Data) > 0 {
		if err := json.Unmarshal(event.Data, &eventData); err != nil {
			c.logger.Warn().Err(err).
				Str("event_id", event.ID).
				Msg("failed to unmarshal event data for filter evaluation")
			eventData = make(map[string]interface{})
		}
	} else {
		eventData = make(map[string]interface{})
	}

	var startErrors []error

	for _, def := range definitions {
		logger := c.logger.With().
			Str("definition_id", def.ID).
			Str("event_id", event.ID).
			Str("tenant_id", def.TenantID).
			Logger()

		// Evaluate the trigger filter against the event data.
		if !evaluateFilter(def.TriggerConfig.Filter, eventData) {
			logger.Debug().Msg("event did not match trigger filter, skipping")
			continue
		}

		// Deduplicate: ensure the same event does not trigger the same definition twice.
		dedupKey := fmt.Sprintf("workflow:trigger:%s:%s", def.ID, event.ID)
		set, err := c.rdb.SetNX(ctx, dedupKey, "1", triggerDedupTTL).Result()
		if err != nil {
			logger.Error().Err(err).Str("dedup_key", dedupKey).Msg("redis dedup check failed")
			startErrors = append(startErrors, fmt.Errorf("dedup check for def %s: %w", def.ID, err))
			continue
		}
		if !set {
			logger.Debug().Str("dedup_key", dedupKey).Msg("duplicate trigger detected, skipping")
			continue
		}

		// Start the workflow instance.
		req := dto.StartInstanceRequest{
			DefinitionID:   def.ID,
			InputVariables: buildInputVariables(def, eventData),
			TriggerData:    event.Data,
		}

		instance, err := c.engine.StartInstance(ctx, def.TenantID, "system", req)
		if err != nil {
			logger.Error().Err(err).Msg("failed to start workflow instance from trigger")
			startErrors = append(startErrors, fmt.Errorf("start instance for def %s: %w", def.ID, err))
			// Clean up the dedup key so the event can be retried.
			_ = c.rdb.Del(ctx, dedupKey).Err()
			continue
		}

		logger.Info().
			Str("instance_id", instance.ID).
			Msg("workflow instance started from trigger event")
	}

	if len(startErrors) > 0 {
		return fmt.Errorf("trigger consumer: %d error(s) processing event %s: %v", len(startErrors), event.ID, startErrors[0])
	}

	return nil
}

// extractTopic derives the topic string used for matching trigger configurations.
// It first checks if the event type contains a recognizable topic pattern, then
// falls back to mapping from the event source.
func extractTopic(event *events.Event) string {
	// The event type follows "com.clario360.{domain}.{entity}.{action}".
	// The trigger_config.topic in definitions stores the Kafka topic name like
	// "platform.iam.events" or "cyber.alert.events".
	// We need to resolve from the event metadata or source to the actual Kafka topic.

	// Check if the event has an explicit topic in metadata.
	if topic, ok := event.Metadata["topic"]; ok && topic != "" {
		return topic
	}

	// Map from event type prefix to topic.
	// "com.clario360.iam.user.created" -> domain = "iam"
	eventType := event.Type
	eventType = strings.TrimPrefix(eventType, "com.clario360.")

	parts := strings.SplitN(eventType, ".", 3)
	if len(parts) < 2 {
		return event.Type
	}

	domain := parts[0]

	// Map known domains to their Kafka topic names.
	domainTopicMap := map[string]string{
		"iam":           events.Topics.IAMEvents,
		"audit":         events.Topics.AuditEvents,
		"notification":  events.Topics.NotificationEvents,
		"workflow":      events.Topics.WorkflowEvents,
		"asset":         events.Topics.AssetEvents,
		"threat":        events.Topics.ThreatEvents,
		"alert":         events.Topics.AlertEvents,
		"remediation":   events.Topics.RemediationEvents,
		"datasource":    events.Topics.DataSourceEvents,
		"pipeline":      events.Topics.PipelineEvents,
		"quality":       events.Topics.QualityEvents,
		"contradiction": events.Topics.ContradictionEvents,
		"lineage":       events.Topics.LineageEvents,
		"acta":          events.Topics.ActaEvents,
		"lex":           events.Topics.LexEvents,
		"visus":         events.Topics.VisusEvents,
	}

	if topic, ok := domainTopicMap[domain]; ok {
		return topic
	}

	return event.Type
}

// evaluateFilter checks whether the event data satisfies all key-value conditions
// defined in the trigger filter. An empty or nil filter matches all events.
// Filter values support simple equality matching.
func evaluateFilter(filter map[string]interface{}, eventData map[string]interface{}) bool {
	if len(filter) == 0 {
		return true
	}

	for key, expected := range filter {
		actual, exists := resolveNestedField(eventData, key)
		if !exists {
			return false
		}

		if !matchValue(expected, actual) {
			return false
		}
	}

	return true
}

// resolveNestedField resolves a potentially dot-separated key path in a nested map.
// For example, "user.role" resolves eventData["user"]["role"].
func resolveNestedField(data map[string]interface{}, key string) (interface{}, bool) {
	parts := strings.Split(key, ".")

	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}

// matchValue compares an expected filter value against an actual event data value.
// It supports string, float64 (JSON numbers), and bool comparison.
// For slice expected values, it checks whether the actual value is contained in the slice.
func matchValue(expected, actual interface{}) bool {
	// Handle slice-based "in" matching: filter value is a list, actual must be in it.
	if expSlice, ok := expected.([]interface{}); ok {
		for _, v := range expSlice {
			if matchValue(v, actual) {
				return true
			}
		}
		return false
	}

	// Compare as strings for simplicity and JSON interop.
	return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", actual)
}

// buildInputVariables constructs the initial variable map for a new workflow instance
// by extracting values from the event data according to the definition's variable
// source configuration.
func buildInputVariables(def *model.WorkflowDefinition, eventData map[string]interface{}) map[string]interface{} {
	vars := make(map[string]interface{})

	for name, varDef := range def.Variables {
		if varDef.Source != "" {
			// Resolve the variable value from event data using the source path.
			if val, ok := resolveNestedField(eventData, varDef.Source); ok {
				vars[name] = val
				continue
			}
		}
		// Fall back to the default value.
		if varDef.Default != nil {
			vars[name] = varDef.Default
		}
	}

	return vars
}
