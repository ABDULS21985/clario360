package consumer

import (
	"context"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
)

func (c *VisusConsumer) handleAlertCreated(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		ID                 string  `json:"id"`
		Title              string  `json:"title"`
		Severity           string  `json:"severity"`
		ConfidenceScore    float64 `json:"confidence_score"`
		AffectedAssetCount int     `json:"affected_asset_count"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed cyber alert event")
		return nil
	}
	if !strings.EqualFold(payload.Severity, "critical") {
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		payload.Title,
		fmt.Sprintf("A critical security alert was raised: %s. %d assets affected.", payload.Title, payload.AffectedAssetCount),
		model.AlertCategoryRisk,
		model.AlertSeverityCritical,
		"cyber",
		"event",
		dedupKey("cyber_alert", payload.ID),
		map[string]any{
			"alert_id":             payload.ID,
			"confidence_score":     payload.ConfidenceScore,
			"affected_asset_count": payload.AffectedAssetCount,
			"source_event_type":    event.Type,
			"source_event_id":      event.ID,
		},
	)
}

func (c *VisusConsumer) handleRiskScoreUpdated(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		OverallScore  float64        `json:"overall_score"`
		Score         float64        `json:"score"`
		PreviousScore float64        `json:"previous_score"`
		Grade         string         `json:"grade"`
		Components    map[string]any `json:"components"`
		Delta         float64        `json:"delta"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed risk score event")
		return nil
	}

	currentScore := payload.OverallScore
	if currentScore == 0 {
		currentScore = payload.Score
	}
	if _, _, err := c.updateKPIByName(ctx, tenantID, "Security Risk Score", currentScore); err != nil {
		return err
	}

	delta := payload.Delta
	if delta == 0 {
		delta = currentScore - payload.PreviousScore
	}
	if err := c.invalidateKeys(ctx, fmt.Sprintf("visus:dashboard:%s", tenantID.String())); err != nil {
		return err
	}
	if delta <= 10 {
		return nil
	}

	severity := model.AlertSeverityHigh
	if delta > 20 {
		severity = model.AlertSeverityCritical
	}
	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Security Risk Score Increased: %.2f -> %.2f (Grade %s)", payload.PreviousScore, currentScore, payload.Grade),
		"Cyber risk posture worsened significantly based on the latest score calculation.",
		model.AlertCategoryRisk,
		severity,
		"cyber",
		"kpi",
		dedupKey("security_risk_score", tenantID.String()),
		map[string]any{
			"previous_score": payload.PreviousScore,
			"overall_score":  currentScore,
			"grade":          payload.Grade,
			"components":     payload.Components,
			"delta":          delta,
		},
	)
}

func (c *VisusConsumer) handleCTEMCompleted(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		AssessmentID  string  `json:"assessment_id"`
		ID            string  `json:"id"`
		ExposureScore float64 `json:"exposure_score"`
		PreviousScore float64 `json:"previous_score"`
		FindingsCount int     `json:"findings_count"`
		FindingCount  int     `json:"finding_count"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed ctem completion event")
		return nil
	}

	assessmentID := payload.AssessmentID
	if assessmentID == "" {
		assessmentID = payload.ID
	}
	if payload.PreviousScore == 0 || payload.ExposureScore <= payload.PreviousScore+10 {
		return nil
	}

	findingsCount := payload.FindingsCount
	if findingsCount == 0 {
		findingsCount = payload.FindingCount
	}
	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Exposure Score Worsened: %.2f -> %.2f", payload.PreviousScore, payload.ExposureScore),
		"Latest CTEM assessment reported a materially worse exposure score.",
		model.AlertCategoryRisk,
		model.AlertSeverityHigh,
		"cyber",
		"ctem",
		dedupKey("ctem_assessment", assessmentID),
		map[string]any{
			"assessment_id":  assessmentID,
			"exposure_score": payload.ExposureScore,
			"previous_score": payload.PreviousScore,
			"findings_count": findingsCount,
		},
	)
}
