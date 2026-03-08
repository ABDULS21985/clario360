package consumer

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
)

func (c *VisusConsumer) handleComplianceChecked(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		Score             float64 `json:"score"`
		NonCompliantCount int     `json:"non_compliant_count"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed governance compliance event")
		return nil
	}
	if _, _, err := c.updateKPIByName(ctx, tenantID, "Governance Compliance", payload.Score); err != nil {
		return err
	}
	if payload.NonCompliantCount <= 0 {
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Governance Compliance Issues: %d checks failed", payload.NonCompliantCount),
		"Governance compliance checks detected non-compliant controls that require committee attention.",
		model.AlertCategoryCompliance,
		model.AlertSeverityHigh,
		"acta",
		"compliance",
		dedupKey("governance_compliance", tenantID.String()),
		map[string]any{
			"score":               payload.Score,
			"non_compliant_count": payload.NonCompliantCount,
		},
	)
}

func (c *VisusConsumer) handleEnterpriseActaOverdue(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		ActionItemID string `json:"action_item_id"`
		Title        string `json:"title"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed enterprise acta overdue event")
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Action Item Overdue: %s", payload.Title),
		"A governance action item has passed its deadline and requires executive attention.",
		model.AlertCategoryCompliance,
		model.AlertSeverityHigh,
		"acta",
		"event",
		dedupKey("enterprise_acta_overdue", payload.ActionItemID),
		map[string]any{
			"action_item_id":    payload.ActionItemID,
			"title":             payload.Title,
			"source_event_type": event.Type,
			"source_event_id":   event.ID,
		},
	)
}

func (c *VisusConsumer) handleActionItemOverdue(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	definition, status, value, err := c.incrementKPIByName(ctx, tenantID, "Overdue Action Items", 1)
	if err != nil || definition == nil {
		return err
	}
	if status == model.KPIStatusNormal {
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Overdue Action Items Increased: %.0f", value),
		"Overdue governance action items crossed an executive warning threshold.",
		model.AlertCategoryCompliance,
		kpiBreachSeverity(status),
		"acta",
		"kpi",
		dedupKey("overdue_action_items", tenantID.String()),
		map[string]any{
			"kpi_name": definition.Name,
			"value":    value,
			"status":   status,
		},
	)
}
