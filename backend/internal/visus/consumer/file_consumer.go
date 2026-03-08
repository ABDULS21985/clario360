package consumer

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
)

func (c *VisusConsumer) handleMalwareDetected(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		FileID     string `json:"file_id"`
		VirusName  string `json:"virus_name"`
		UploadedBy string `json:"uploaded_by"`
		Suite      string `json:"suite"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed malware detection event")
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Malware Detected in Uploaded File %s", payload.FileID),
		"A file uploaded into the platform was flagged as infected during malware scanning.",
		model.AlertCategoryRisk,
		model.AlertSeverityCritical,
		"file",
		"malware",
		dedupKey("file_malware", payload.FileID),
		map[string]any{
			"file_id":      payload.FileID,
			"virus_name":   payload.VirusName,
			"uploaded_by":  payload.UploadedBy,
			"suite":        payload.Suite,
			"source_event": event.Type,
		},
	)
}
