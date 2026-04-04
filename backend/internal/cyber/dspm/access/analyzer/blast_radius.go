package analyzer

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/mapper"
	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// AssetWeightProvider returns sensitivity weights for the full tenant asset pool.
type AssetWeightProvider interface {
	AllAssetWeights(ctx context.Context, tenantID uuid.UUID) ([]float64, error)
}

// BlastRadiusAnalyzer calculates the impact if an identity's credentials are compromised.
// A user with read access to 3 restricted databases has a vastly different compromise
// impact than a user with read access to 50 public tables. Blast radius quantifies
// "what's the damage?" in a single sensitivity-weighted number.
type BlastRadiusAnalyzer struct {
	effectiveAccess *mapper.EffectiveAccessResolver
	scorer          *mapper.SensitivityScorer
	weightProvider  AssetWeightProvider
	logger          zerolog.Logger
}

// NewBlastRadiusAnalyzer creates a new blast radius calculator.
func NewBlastRadiusAnalyzer(
	effectiveAccess *mapper.EffectiveAccessResolver,
	scorer *mapper.SensitivityScorer,
	weightProvider AssetWeightProvider,
	logger zerolog.Logger,
) *BlastRadiusAnalyzer {
	return &BlastRadiusAnalyzer{
		effectiveAccess: effectiveAccess,
		scorer:          scorer,
		weightProvider:  weightProvider,
		logger:          logger.With().Str("analyzer", "blast_radius").Logger(),
	}
}

// Calculate computes the blast radius for a single identity.
func (a *BlastRadiusAnalyzer) Calculate(ctx context.Context, tenantID uuid.UUID, identityID string) (*model.BlastRadius, error) {
	effective, err := a.effectiveAccess.Resolve(ctx, tenantID, identityID)
	if err != nil {
		return nil, err
	}

	if effective == nil || len(effective.Assets) == 0 {
		return &model.BlastRadius{
			IdentityID:             identityID,
			ExposedClassifications: make(map[string]int),
			Level:                  "low",
		}, nil
	}

	// Get max possible blast radius for the tenant.
	weights, err := a.weightProvider.AllAssetWeights(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	maxPossible := a.scorer.MaxPossibleScore(weights)

	// Compute score.
	score := a.scorer.Score(effective.Assets, maxPossible)
	level := model.RiskLevel(score)

	// Build classification breakdown.
	classBreakdown := make(map[string]int)
	sensitiveCount := 0
	for _, asset := range effective.Assets {
		classBreakdown[asset.DataClassification]++
		if asset.DataClassification == "restricted" || asset.DataClassification == "confidential" {
			sensitiveCount++
		}
	}

	// Build top risky assets (sorted by risk contribution).
	exposures := make([]model.AssetExposure, 0, len(effective.Assets))
	for _, asset := range effective.Assets {
		contribution := asset.SensitivityWeight * model.PermissionBreadth(asset.MaxPermissionLevel)
		exposures = append(exposures, model.AssetExposure{
			DataAssetID:        asset.DataAssetID,
			DataAssetName:      asset.DataAssetName,
			DataClassification: asset.DataClassification,
			MaxPermission:      asset.MaxPermissionLevel,
			SensitivityWeight:  asset.SensitivityWeight,
			RiskContribution:   contribution,
		})
	}
	sort.Slice(exposures, func(i, j int) bool {
		return exposures[i].RiskContribution > exposures[j].RiskContribution
	})
	topRisky := exposures
	if len(topRisky) > 10 {
		topRisky = topRisky[:10]
	}

	// Generate recommendations.
	var recommendations []string
	if sensitiveCount > 5 {
		recommendations = append(recommendations, "Review access to sensitive assets — identity has access to more than 5 restricted/confidential assets")
	}
	if level == "critical" || level == "high" {
		recommendations = append(recommendations, "Consider time-bounding access to reduce blast radius")
	}
	if score > 75 {
		recommendations = append(recommendations, "Apply least-privilege principle — current access exceeds recommended blast radius threshold")
	}

	return &model.BlastRadius{
		IdentityID:              effective.IdentityID,
		IdentityName:            effective.IdentityName,
		IdentityType:            effective.IdentityType,
		TotalAssetsExposed:      len(effective.Assets),
		SensitiveAssets:         sensitiveCount,
		WeightedScore:           score,
		Level:                   level,
		ExposedClassifications:  classBreakdown,
		TopRiskyAssets:          topRisky,
		RecommendedActions:      recommendations,
	}, nil
}
