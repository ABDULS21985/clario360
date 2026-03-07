package sections

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
)

func BuildSecurity(ctx context.Context, client *aggregator.SuiteClient, tenantID uuid.UUID) (map[string]any, string) {
	var payload map[string]any
	meta := client.Fetch(ctx, "cyber", "/dashboard", tenantID, &payload)
	if meta.Status == "unavailable" {
		return unavailable(), errorString(meta.Error)
	}
	riskScore := mustValue(payload, "$.data.kpis.risk_score")
	previous := riskScore - mustValue(payload, "$.data.risk_score.trend_delta")
	return map[string]any{
		"available":       true,
		"risk_score":      riskScore,
		"grade":           mustString(payload, "$.data.kpis.risk_grade"),
		"prev_risk_score": previous,
		"trend_word":      trendWord(riskScore-previous, false),
		"open_alerts":     mustValue(payload, "$.data.kpis.open_alerts"),
		"critical_alerts": mustValue(payload, "$.data.kpis.critical_alerts"),
		"mttr_hours":      mustValue(payload, "$.data.kpis.mttr_hours"),
		"coverage":        mitreCoverage(payload),
		"generated_at":    time.Now().UTC(),
	}, ""
}
