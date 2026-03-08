package consumer

import (
	"context"
	"encoding/json"
	"strings"

	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/events"
	"github.com/rs/zerolog"
)

type CacheInvalidationConsumer struct {
	predictionLogger *aigovmiddleware.PredictionLogger
	logger           zerolog.Logger
}

func NewCacheInvalidationConsumer(predictionLogger *aigovmiddleware.PredictionLogger, logger zerolog.Logger) *CacheInvalidationConsumer {
	return &CacheInvalidationConsumer{
		predictionLogger: predictionLogger,
		logger:           logger.With().Str("component", "ai_cache_invalidation_consumer").Logger(),
	}
}

func (c *CacheInvalidationConsumer) EventTypes() []string {
	return []string{
		"com.clario360.ai.model.version.promoted",
		"com.clario360.ai.model.version.retired",
		"com.clario360.ai.model.version.rolled_back",
		"com.clario360.ai.shadow.started",
		"com.clario360.ai.shadow.stopped",
	}
}

func (c *CacheInvalidationConsumer) Handle(_ context.Context, event *events.Event) error {
	if c.predictionLogger == nil || event == nil {
		return nil
	}
	var payload struct {
		ModelSlug string `json:"model_slug"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		c.logger.Warn().Err(err).Str("event_type", event.Type).Msg("failed to decode ai cache invalidation event")
		return nil
	}
	if strings.TrimSpace(payload.ModelSlug) == "" {
		c.logger.Warn().Str("event_type", event.Type).Msg("ai cache invalidation event missing model_slug")
		return nil
	}
	c.predictionLogger.InvalidateModel(payload.ModelSlug)
	return nil
}
