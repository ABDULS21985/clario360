package compliance

import (
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// GapAnalyzer extracts and aggregates compliance gaps from posture evaluations
// across all frameworks.
type GapAnalyzer struct {
	logger zerolog.Logger
}

// NewGapAnalyzer creates a new gap analyzer instance.
func NewGapAnalyzer(logger zerolog.Logger) *GapAnalyzer {
	return &GapAnalyzer{
		logger: logger.With().Str("component", "gap_analyzer").Logger(),
	}
}

// Analyze extracts all non-compliant and partially-compliant controls from
// the given posture evaluations and aggregates them into a list of compliance
// gaps ordered by severity and asset count.
func (g *GapAnalyzer) Analyze(postures []model.CompliancePosture) []model.ComplianceGap {
	var gaps []model.ComplianceGap

	for _, posture := range postures {
		for _, detail := range posture.ControlDetails {
			if detail.Status == model.ControlCompliant || detail.Status == model.ControlNotApplicable {
				continue
			}

			severity := controlSeverity(detail)
			gap := model.ComplianceGap{
				Framework:   posture.Framework,
				ControlID:   detail.ControlID,
				ControlName: detail.Name,
				Severity:    severity,
				AssetCount:  detail.AssetsNonCompliant,
				Gaps:        detail.Gaps,
			}
			gaps = append(gaps, gap)
		}
	}

	// Sort gaps by severity (critical first), then by asset count (descending).
	sortGaps(gaps)

	g.logger.Info().
		Int("total_gaps", len(gaps)).
		Int("frameworks_analyzed", len(postures)).
		Msg("gap analysis complete")

	return gaps
}

// controlSeverity determines the severity of a control gap based on the
// compliance score and the number of affected assets.
func controlSeverity(detail model.ControlDetail) string {
	switch {
	case detail.Score < 25:
		return "critical"
	case detail.Score < 50:
		return "high"
	case detail.Score < 75:
		return "medium"
	default:
		return "low"
	}
}

// sortGaps sorts gaps by severity (critical > high > medium > low), then
// by asset count descending within the same severity level.
func sortGaps(gaps []model.ComplianceGap) {
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	for i := 1; i < len(gaps); i++ {
		key := gaps[i]
		j := i - 1
		for j >= 0 && shouldSwap(gaps[j], key, severityOrder) {
			gaps[j+1] = gaps[j]
			j--
		}
		gaps[j+1] = key
	}
}

// shouldSwap determines if gap a should come after gap b in sorted order.
func shouldSwap(a, b model.ComplianceGap, severityOrder map[string]int) bool {
	orderA := severityOrder[a.Severity]
	orderB := severityOrder[b.Severity]

	if orderA != orderB {
		return orderA > orderB
	}
	// Same severity: higher asset count should come first.
	return a.AssetCount < b.AssetCount
}
