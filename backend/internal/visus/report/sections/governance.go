package sections

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
)

func BuildGovernance(ctx context.Context, client *aggregator.SuiteClient, tenantID uuid.UUID) (map[string]any, string) {
	var payload map[string]any
	meta := client.Fetch(ctx, "acta", "/dashboard", tenantID, &payload)
	if meta.Status == "unavailable" {
		return unavailable(), errorString(meta.Error)
	}
	return map[string]any{
		"available":         true,
		"compliance_score":  mustValue(payload, "$.data.kpis.compliance_score"),
		"meeting_count":     mustValue(payload, "$.data.kpis.upcoming_meetings_30d"),
		"overdue_count":     mustValue(payload, "$.data.kpis.overdue_action_items"),
		"minutes_pending":   mustValue(payload, "$.data.kpis.minutes_pending_approval"),
		"open_action_items": mustValue(payload, "$.data.kpis.open_action_items"),
		"generated_at":      time.Now().UTC(),
	}, ""
}
