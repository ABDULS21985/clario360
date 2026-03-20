package contradiction

import (
	"math"
	"strings"
	"time"

	"github.com/clario360/platform/internal/data/model"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
)

func ComputeConfidence(raw cruntime.RawContradiction, modelA, modelB *model.DataModel, sourceA, sourceB *model.DataSource) float64 {
	score := map[model.ContradictionType]float64{
		model.ContradictionTypeLogical:    0.80,
		model.ContradictionTypeSemantic:   0.70,
		model.ContradictionTypeTemporal:   0.60,
		model.ContradictionTypeAnalytical: 0.75,
	}[raw.Type]

	if raw.AffectedRecords > 100 {
		score += 0.10
	}
	if sourceA.Status == model.DataSourceStatusActive && sourceB.Status == model.DataSourceStatusActive {
		score += 0.05
	}
	if columnLooksLikePII(raw.Column, modelA, modelB) {
		score += 0.05
	}
	if syncedFarApart(sourceA.LastSyncedAt, sourceB.LastSyncedAt) {
		score -= 0.10
	}
	if strings.Contains(strings.ToLower(raw.Column), "updated") || strings.Contains(strings.ToLower(raw.Column), "modified") {
		score -= 0.05
	}
	if raw.NumericDeltaPct > 0 && raw.NumericDeltaPct < 1 {
		score -= 0.15
	}
	return math.Min(0.99, math.Max(0.10, score))
}

func syncedFarApart(a, b *time.Time) bool {
	if a == nil || b == nil {
		return false
	}
	diff := a.Sub(*b)
	if diff < 0 {
		diff = -diff
	}
	return diff > 24*time.Hour
}

func columnLooksLikePII(column string, models ...*model.DataModel) bool {
	for _, item := range models {
		for _, pii := range item.PIIColumns {
			if strings.EqualFold(pii, column) {
				return true
			}
		}
	}
	return false
}
