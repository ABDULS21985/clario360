package consumer

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
)

func (c *VisusConsumer) handleContractExpiring(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		ID              string `json:"id"`
		Title           string `json:"title"`
		PartyName       string `json:"party_name"`
		DaysUntilExpiry int    `json:"days_until_expiry"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed contract expiry event")
		return nil
	}
	if payload.DaysUntilExpiry <= 30 {
		if _, _, _, err := c.incrementKPIByName(ctx, tenantID, "Contracts Expiring 30d", 1); err != nil {
			return err
		}
	}
	if payload.DaysUntilExpiry > 7 {
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		fmt.Sprintf("Contract Expiring in %d Days: %s (%s)", payload.DaysUntilExpiry, payload.Title, payload.PartyName),
		"A contract is approaching expiry within the critical review window.",
		model.AlertCategoryLegal,
		model.AlertSeverityCritical,
		"lex",
		"contract",
		dedupKey("contract_expiring", payload.ID),
		map[string]any{
			"contract_id":       payload.ID,
			"title":             payload.Title,
			"party_name":        payload.PartyName,
			"days_until_expiry": payload.DaysUntilExpiry,
			"source_event_type": event.Type,
			"source_event_id":   event.ID,
		},
	)
}

func (c *VisusConsumer) handleComplianceAlert(ctx context.Context, event *events.Event) error {
	tenantID, err := c.tenantID(event)
	if err != nil {
		return err
	}

	var payload struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Severity   string `json:"severity"`
		ContractID string `json:"contract_id"`
	}
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed lex compliance alert event")
		return nil
	}

	return c.createExecutiveAlert(
		ctx,
		tenantID,
		payload.Title,
		"A legal compliance alert was raised for a contract and has been promoted to the executive console.",
		model.AlertCategoryLegal,
		severityFromString(payload.Severity),
		"lex",
		"compliance_alert",
		dedupKey("lex_compliance_alert", payload.ID),
		map[string]any{
			"alert_id":     payload.ID,
			"contract_id":  payload.ContractID,
			"severity":     payload.Severity,
			"source_event": event.Type,
		},
	)
}
