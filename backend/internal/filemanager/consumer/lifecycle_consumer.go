package consumer

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/filemanager/service"
)

// LifecycleConsumer handles file lifecycle events.
type LifecycleConsumer struct {
	fileSvc *service.FileService
	logger  zerolog.Logger
}

// NewLifecycleConsumer creates a lifecycle consumer.
func NewLifecycleConsumer(fileSvc *service.FileService, logger zerolog.Logger) *LifecycleConsumer {
	return &LifecycleConsumer{
		fileSvc: fileSvc,
		logger:  logger,
	}
}

// Handle processes file lifecycle events.
func (c *LifecycleConsumer) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "com.clario360.file.scan.infected":
		return c.handleInfected(ctx, event)
	case "com.clario360.file.expired":
		return c.handleExpired(ctx, event)
	default:
		return nil
	}
}

func (c *LifecycleConsumer) handleInfected(ctx context.Context, event *events.Event) error {
	var payload struct {
		FileID    string `json:"file_id"`
		VirusName string `json:"virus_name"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		c.logger.Error().Err(err).Msg("failed to unmarshal infected event")
		return nil
	}

	c.logger.Error().
		Str("file_id", payload.FileID).
		Str("virus", payload.VirusName).
		Msg("CRITICAL: infected file detected")

	return nil
}

func (c *LifecycleConsumer) handleExpired(ctx context.Context, event *events.Event) error {
	var payload struct {
		FileID          string `json:"file_id"`
		LifecyclePolicy string `json:"lifecycle_policy"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		c.logger.Error().Err(err).Msg("failed to unmarshal expired event")
		return nil
	}

	c.logger.Info().
		Str("file_id", payload.FileID).
		Str("policy", payload.LifecyclePolicy).
		Msg("file expired by lifecycle policy")

	return nil
}

// EventTypes returns the event types this consumer handles.
func (c *LifecycleConsumer) EventTypes() []string {
	return []string{
		"com.clario360.file.scan.infected",
		"com.clario360.file.expired",
	}
}
