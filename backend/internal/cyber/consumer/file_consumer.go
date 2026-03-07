package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/events"
)

const fileConsumerName = "cyber_file_consumer"

type FileEventConsumer struct {
	alertService alertEventService
	guard        *events.IdempotencyGuard
	producer     *events.Producer
	logger       zerolog.Logger
	metrics      *events.CrossSuiteMetrics
	now          func() time.Time
}

func NewFileEventConsumer(alertService *service.AlertService, guard *events.IdempotencyGuard, producer *events.Producer, logger zerolog.Logger, metrics *events.CrossSuiteMetrics) *FileEventConsumer {
	return &FileEventConsumer{
		alertService: alertService,
		guard:        guard,
		producer:     producer,
		logger:       logger.With().Str("component", fileConsumerName).Logger(),
		metrics:      metrics,
		now:          time.Now,
	}
}

func (c *FileEventConsumer) EventTypes() []string {
	return []string{"com.clario360.file.scan.infected"}
}

func (c *FileEventConsumer) Handle(ctx context.Context, event *events.Event) error {
	if event.Type != "com.clario360.file.scan.infected" {
		return nil
	}

	var payload struct {
		FileID      string `json:"file_id"`
		VirusName   string `json:"virus_name"`
		UploadedBy  string `json:"uploaded_by"`
		Suite       string `json:"suite"`
		ContentType string `json:"content_type"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed event data")
		return nil
	}
	if strings.TrimSpace(payload.FileID) == "" {
		c.logger.Warn().Str("event_id", event.ID).Msg("missing required field: file_id")
		return nil
	}

	tenantID, err := uuid.Parse(strings.TrimSpace(event.TenantID))
	if err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("invalid tenant id")
		return nil
	}

	processed, err := c.guard.IsProcessed(ctx, event.ID)
	if err != nil {
		return err
	}
	if processed {
		if c.metrics != nil {
			c.metrics.SkippedIdempotentTotal.WithLabelValues(fileConsumerName, event.Type).Inc()
		}
		return nil
	}

	explanation := model.AlertExplanation{
		Summary: "A file uploaded to the platform was flagged as infected during malware scanning.",
		ConfidenceFactors: []model.ConfidenceFactor{
			{Factor: "Direct anti-malware detection", Impact: 0.6},
		},
		RecommendedActions: []string{
			"Review the uploaded file and quarantine disposition",
			"Confirm the uploader account has not been compromised",
			"Inspect other recent uploads from the same source",
		},
		Details: map[string]interface{}{
			"virus_name":   payload.VirusName,
			"file_id":      payload.FileID,
			"uploaded_by":  payload.UploadedBy,
			"suite":        payload.Suite,
			"content_type": payload.ContentType,
		},
	}

	metadata, err := json.Marshal(map[string]interface{}{
		"file_id":      payload.FileID,
		"virus_name":   payload.VirusName,
		"uploaded_by":  payload.UploadedBy,
		"suite":        payload.Suite,
		"content_type": payload.ContentType,
		"event_id":     event.ID,
	})
	if err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return fmt.Errorf("marshal malware alert metadata: %w", err)
	}

	techniqueID := "T1204"
	techniqueName := "User Execution"
	alert := &model.Alert{
		TenantID:           tenantID,
		Title:              fmt.Sprintf("Malware Detected in Uploaded File %s", payload.FileID),
		Description:        fmt.Sprintf("The malware scanner detected %s in uploaded file %s.", fallbackString(payload.VirusName, "malicious content"), payload.FileID),
		Severity:           model.SeverityCritical,
		Status:             model.AlertStatusNew,
		Source:             "file_malware_detected",
		Explanation:        explanation,
		ConfidenceScore:    1.0,
		MITRETechniqueID:   &techniqueID,
		MITRETechniqueName: &techniqueName,
		EventCount:         1,
		FirstEventAt:       c.now().UTC(),
		LastEventAt:        c.now().UTC(),
		Metadata:           metadata,
	}

	if _, err := c.alertService.CreateFromEvent(ctx, alert); err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return err
	}
	if c.metrics != nil {
		c.metrics.AlertsCreatedTotal.WithLabelValues(fileConsumerName, string(model.SeverityCritical)).Inc()
	}
	return c.guard.MarkProcessed(ctx, event.ID)
}
