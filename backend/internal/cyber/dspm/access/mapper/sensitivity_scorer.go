package mapper

import (
	"math"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// SensitivityScorer calculates blast radius and sensitivity-weighted scores.
type SensitivityScorer struct{}

// NewSensitivityScorer creates a new scorer.
func NewSensitivityScorer() *SensitivityScorer {
	return &SensitivityScorer{}
}

// Score computes a normalized blast radius score (0-100) from accessible assets.
// blast_radius = Σ (sensitivity_weight × permission_breadth_factor) for each asset.
// Normalized: score = min(blast_radius / max_possible × 100, 100).
func (s *SensitivityScorer) Score(assets []model.AssetAccess, maxPossible float64) float64 {
	if len(assets) == 0 || maxPossible <= 0 {
		return 0
	}

	var rawScore float64
	for _, a := range assets {
		rawScore += a.SensitivityWeight * model.PermissionBreadth(a.MaxPermissionLevel)
	}

	score := rawScore / maxPossible * 100
	return math.Min(math.Round(score*100)/100, 100)
}

// MaxPossibleScore computes the theoretical maximum blast radius for a tenant.
// This is the sum of all assets × their sensitivity weight × max breadth (5.0 for full_control).
func (s *SensitivityScorer) MaxPossibleScore(assetWeights []float64) float64 {
	var total float64
	for _, w := range assetWeights {
		total += w * 5.0 // full_control breadth
	}
	if total == 0 {
		return 1 // Avoid division by zero.
	}
	return total
}

// WeightedRisk calculates the total weighted risk for a set of asset accesses.
func (s *SensitivityScorer) WeightedRisk(assets []model.AssetAccess) float64 {
	var total float64
	for _, a := range assets {
		total += a.SensitivityWeight * model.PermissionBreadth(a.MaxPermissionLevel)
	}
	return math.Round(total*100) / 100
}
