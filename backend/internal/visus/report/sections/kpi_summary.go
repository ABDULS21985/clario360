package sections

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

func BuildKPISummary(ctx context.Context, kpis *repository.KPIRepository, snapshots *repository.KPISnapshotRepository, tenantID uuid.UUID, start, end time.Time) (map[string]any, string) {
	definitions, _, err := kpis.List(ctx, tenantID, 1, 500, "category", "asc", "", "", nil)
	if err != nil {
		return unavailable(), err.Error()
	}
	history, err := snapshots.ListForPeriod(ctx, tenantID, start, end)
	if err != nil {
		return unavailable(), err.Error()
	}
	index := make(map[uuid.UUID][]model.KPISnapshot)
	for _, snapshot := range history {
		index[snapshot.KPIID] = append(index[snapshot.KPIID], snapshot)
	}
	items := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		points := index[definition.ID]
		var latest model.KPISnapshot
		if len(points) > 0 {
			latest = points[0]
		}
		items = append(items, map[string]any{
			"kpi_id":          definition.ID,
			"name":            definition.Name,
			"value":           latest.Value,
			"status":          latest.Status,
			"target_value":    definition.TargetValue,
			"warning":         definition.WarningThreshold,
			"critical":        definition.CriticalThreshold,
			"trend_points":    points,
			"direction":       definition.Direction,
			"higher_is_better": definition.Direction == model.KPIDirectionHigherIsBetter,
		})
	}
	return map[string]any{
		"available": true,
		"kpis":      items,
	}, ""
}
