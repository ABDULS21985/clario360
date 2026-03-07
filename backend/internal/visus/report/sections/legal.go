package sections

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
)

func BuildLegal(ctx context.Context, client *aggregator.SuiteClient, tenantID uuid.UUID) (map[string]any, string) {
	var payload map[string]any
	meta := client.Fetch(ctx, "lex", "/dashboard", tenantID, &payload)
	if meta.Status == "unavailable" {
		return unavailable(), errorString(meta.Error)
	}
	return map[string]any{
		"available":        true,
		"active_contracts": mustValue(payload, "$.data.kpis.active_contracts"),
		"value":            mustValue(payload, "$.data.kpis.total_active_value"),
		"expiring_count":   mustValue(payload, "$.data.kpis.expiring_in_30_days"),
		"high_risk_count":  mustValue(payload, "$.data.kpis.high_risk_contracts"),
		"pending_review":   mustValue(payload, "$.data.kpis.pending_review"),
		"generated_at":     time.Now().UTC(),
	}, ""
}
