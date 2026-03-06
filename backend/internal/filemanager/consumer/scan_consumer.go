package consumer

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/filemanager/service"
)

// ScanConsumer handles virus scan events from Kafka.
type ScanConsumer struct {
	scanSvc *service.ScanService
	logger  zerolog.Logger
}

// NewScanConsumer creates a scan consumer.
func NewScanConsumer(scanSvc *service.ScanService, logger zerolog.Logger) *ScanConsumer {
	return &ScanConsumer{
		scanSvc: scanSvc,
		logger:  logger,
	}
}

// Handle processes a file.uploaded event by triggering virus scan.
func (c *ScanConsumer) Handle(ctx context.Context, event *events.Event) error {
	if event.Type != "com.clario360.file.uploaded" {
		return nil
	}

	var payload struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		c.logger.Error().Err(err).Str("event_id", event.ID).Msg("failed to unmarshal file.uploaded event")
		return nil // don't retry malformed events
	}

	if payload.FileID == "" {
		c.logger.Warn().Str("event_id", event.ID).Msg("file.uploaded event missing file_id")
		return nil
	}

	c.logger.Info().Str("file_id", payload.FileID).Msg("processing virus scan")

	if err := c.scanSvc.ScanFile(ctx, payload.FileID); err != nil {
		c.logger.Error().Err(err).Str("file_id", payload.FileID).Msg("virus scan failed")
		return err // return error to trigger retry
	}

	return nil
}

// EventTypes returns the event types this consumer handles.
func (c *ScanConsumer) EventTypes() []string {
	return []string{"com.clario360.file.uploaded"}
}
