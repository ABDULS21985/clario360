package consumer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
)

const failureTrackerConsumer = "data_failure_tracker"

type FailureTracker struct {
	redis    *redis.Client
	guard    *events.IdempotencyGuard
	producer *events.Producer
	logger   zerolog.Logger
	metrics  *events.CrossSuiteMetrics
}

func NewFailureTracker(redisClient *redis.Client, guard *events.IdempotencyGuard, producer *events.Producer, logger zerolog.Logger, metrics *events.CrossSuiteMetrics) *FailureTracker {
	return &FailureTracker{
		redis:    redisClient,
		guard:    guard,
		producer: producer,
		logger:   logger.With().Str("component", failureTrackerConsumer).Logger(),
		metrics:  metrics,
	}
}

func (t *FailureTracker) EventTypes() []string {
	return []string{
		"com.clario360.data.pipeline.run.completed",
		"com.clario360.data.pipeline.run.failed",
	}
}

func (t *FailureTracker) Handle(ctx context.Context, event *events.Event) error {
	var payload struct {
		PipelineID    string `json:"pipeline_id"`
		PipelineName  string `json:"pipeline_name"`
		TenantID      string `json:"tenant_id"`
		Status        string `json:"status"`
		ErrorMessage  string `json:"error_message"`
		Error         string `json:"error"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		t.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed event data")
		return nil
	}
	if strings.TrimSpace(payload.PipelineID) == "" {
		t.logger.Warn().Str("event_id", event.ID).Msg("missing required field: pipeline_id")
		return nil
	}

	processed, err := t.guard.IsProcessed(ctx, event.ID)
	if err != nil {
		return err
	}
	if processed {
		if t.metrics != nil {
			t.metrics.SkippedIdempotentTotal.WithLabelValues(failureTrackerConsumer, event.Type).Inc()
		}
		return nil
	}

	status := strings.TrimSpace(payload.Status)
	if status == "" {
		switch event.Type {
		case "com.clario360.data.pipeline.run.failed":
			status = "failed"
		case "com.clario360.data.pipeline.run.completed":
			status = "completed"
		}
	}

	key := fmt.Sprintf("pipeline_failures:%s:%s", event.TenantID, payload.PipelineID)

	switch status {
	case "failed":
		count, err := t.redis.Incr(ctx, key).Result()
		if err != nil {
			_ = t.guard.Release(ctx, event.ID)
			return err
		}
		if count == 1 {
			if err := t.redis.Expire(ctx, key, 72*time.Hour).Err(); err != nil {
				_ = t.guard.Release(ctx, event.ID)
				return err
			}
		}

		t.logger.Info().
			Str("pipeline_id", payload.PipelineID).
			Str("pipeline_name", payload.PipelineName).
			Int64("failure_count", count).
			Msg("pipeline failure tracked")

		if count == 3 {
			if err := t.publishEscalation(ctx, event.TenantID, "data.pipeline.consecutive_failures", map[string]any{
				"pipeline_id":        payload.PipelineID,
				"pipeline_name":      payload.PipelineName,
				"tenant_id":          event.TenantID,
				"consecutive_count":  3,
				"last_error":         fallbackString(payload.ErrorMessage, payload.Error),
			}); err != nil {
				_ = t.guard.Release(ctx, event.ID)
				return err
			}
			t.logger.Info().Str("pipeline_name", payload.PipelineName).Msg("pipeline has failed 3 consecutive times - escalating")
		}
		if count == 5 {
			if err := t.publishEscalation(ctx, event.TenantID, "data.pipeline.critical_reliability", map[string]any{
				"pipeline_id":       payload.PipelineID,
				"pipeline_name":     payload.PipelineName,
				"tenant_id":         event.TenantID,
				"consecutive_count": 5,
			}); err != nil {
				_ = t.guard.Release(ctx, event.ID)
				return err
			}
			t.logger.Info().Str("pipeline_name", payload.PipelineName).Msg("pipeline critically unreliable - 5 consecutive failures")
		}
	case "completed":
		previousCount := int64(0)
		if value, err := t.redis.Get(ctx, key).Result(); err == nil {
			parsed, parseErr := strconv.ParseInt(value, 10, 64)
			if parseErr == nil {
				previousCount = parsed
			}
		}
		if err := t.redis.Del(ctx, key).Err(); err != nil {
			_ = t.guard.Release(ctx, event.ID)
			return err
		}
		if previousCount >= 3 {
			t.logger.Info().
				Str("pipeline_name", payload.PipelineName).
				Int64("previous_failures", previousCount).
				Msg("pipeline recovered after consecutive failures")
		}
	default:
		t.logger.Warn().Str("status", status).Str("event_id", event.ID).Msg("unsupported pipeline run status")
	}

	return t.guard.MarkProcessed(ctx, event.ID)
}

func (t *FailureTracker) publishEscalation(ctx context.Context, tenantID, eventType string, payload map[string]any) error {
	if t.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID, payload)
	if err != nil {
		return fmt.Errorf("create escalation event: %w", err)
	}
	return t.producer.Publish(ctx, events.Topics.PipelineEvents, event)
}
