package consumer

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
)

func (c *VisusConsumer) handleConsecutiveFailures(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		PipelineID       string `json:"pipeline_id"`
		PipelineName     string `json:"pipeline_name"`
		ConsecutiveCount int    `json:"consecutive_count"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed consecutive failures event")
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Pipeline Reliability Issue - %d consecutive failures: %s", payload.ConsecutiveCount, payload.PipelineName),
		"A data pipeline has crossed the reliability warning threshold.",
		model.AlertCategoryDataQuality,
		model.AlertSeverityHigh,
		"data",
		"pipeline_reliability",
		dedupKey("pipeline_failures", payload.PipelineID, fmt.Sprintf("%d", payload.ConsecutiveCount)),
		map[string]any{
			"pipeline_id":       payload.PipelineID,
			"pipeline_name":     payload.PipelineName,
			"consecutive_count": payload.ConsecutiveCount,
			"source_event_type": event.Type,
			"source_event_id":   event.ID,
		},
	)
}

func (c *VisusConsumer) handleCriticalReliability(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		PipelineID       string `json:"pipeline_id"`
		PipelineName     string `json:"pipeline_name"`
		ConsecutiveCount int    `json:"consecutive_count"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed critical reliability event")
		return nil
	}
	if payload.ConsecutiveCount == 0 {
		payload.ConsecutiveCount = 5
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("CRITICAL: Pipeline %s has failed %d consecutive times", payload.PipelineName, payload.ConsecutiveCount),
		"A pipeline is critically unreliable and needs intervention.",
		model.AlertCategoryDataQuality,
		model.AlertSeverityCritical,
		"data",
		"pipeline_reliability",
		dedupKey("pipeline_critical", payload.PipelineID),
		map[string]any{
			"pipeline_id":       payload.PipelineID,
			"pipeline_name":     payload.PipelineName,
			"consecutive_count": payload.ConsecutiveCount,
			"source_event_type": event.Type,
			"source_event_id":   event.ID,
		},
	)
}

func (c *VisusConsumer) handleQualityScoreChanged(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		OldScore float64 `json:"old_score"`
		NewScore float64 `json:"new_score"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed quality score event")
		return nil
	}
	if _, _, err := c.updateKPIByName(ctx, tenantID, "Data Quality Score", payload.NewScore); err != nil {
		return err
	}
	if payload.NewScore >= payload.OldScore-5 {
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Data Quality Score Dropped: %.2f%% -> %.2f%%", payload.OldScore, payload.NewScore),
		"Data quality deteriorated materially from the previous score.",
		model.AlertCategoryDataQuality,
		model.AlertSeverityHigh,
		"data",
		"kpi",
		dedupKey("data_quality_drop", tenantID.String()),
		map[string]any{
			"old_score": payload.OldScore,
			"new_score": payload.NewScore,
		},
	)
}

func (c *VisusConsumer) handleContradictionDetected(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	definition, status, value, err := c.incrementKPIByName(ctx, tenantID, "Open Contradictions", 1)
	if err != nil || definition == nil {
		return err
	}
	if status == model.KPIStatusNormal {
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Open Contradictions Elevated: %.0f", value),
		"Open contradiction volume crossed a configured executive threshold.",
		model.AlertCategoryDataQuality,
		kpiBreachSeverity(status),
		"data",
		"kpi",
		dedupKey("open_contradictions", tenantID.String()),
		map[string]any{
			"kpi_name": definition.Name,
			"value":    value,
			"status":   status,
		},
	)
}

func (c *VisusConsumer) handleLineageUpdated(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}
	return c.invalidateKeys(ctx, fmt.Sprintf("visus:lineage_graph:%s", tenantID.String()))
}
