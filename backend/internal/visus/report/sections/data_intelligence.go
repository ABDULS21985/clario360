package sections

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
)

func BuildDataIntelligence(ctx context.Context, client *aggregator.SuiteClient, tenantID uuid.UUID) (map[string]any, string) {
	var payload map[string]any
	meta := client.Fetch(ctx, "data", "/dashboard", tenantID, &payload)
	if meta.Status == "unavailable" {
		return unavailable(), errorString(meta.Error)
	}
	return map[string]any{
		"available":           true,
		"quality_score":       mustValue(payload, "$.data.kpis.quality_score"),
		"quality_grade":       mustString(payload, "$.data.kpis.quality_grade"),
		"success_rate":        mustValue(payload, "$.data.pipeline_success_rate_30d"),
		"failed_count":        mustValue(payload, "$.data.kpis.failed_pipelines_24h"),
		"contradiction_count": mustValue(payload, "$.data.kpis.open_contradictions"),
		"dark_data_assets":    mustValue(payload, "$.data.kpis.dark_data_assets"),
		"generated_at":        time.Now().UTC(),
	}, ""
}
