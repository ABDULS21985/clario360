package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// ImpactAnalyzer performs dry-run policy evaluations to quantify the impact of
// a policy without triggering any enforcement actions.
type ImpactAnalyzer struct {
	assetLister AssetLister
	logger      zerolog.Logger
}

// NewImpactAnalyzer constructs an ImpactAnalyzer.
func NewImpactAnalyzer(assetLister AssetLister, logger zerolog.Logger) *ImpactAnalyzer {
	return &ImpactAnalyzer{
		assetLister: assetLister,
		logger:      logger.With().Str("component", "impact_analyzer").Logger(),
	}
}

// Analyze evaluates a single policy against all active assets for a tenant and
// returns a summary of how many assets would be affected. No enforcement
// actions are triggered and no violations are persisted.
func (ia *ImpactAnalyzer) Analyze(ctx context.Context, tenantID uuid.UUID, policy *model.DataPolicy) (*model.PolicyImpact, error) {
	assets, err := ia.assetLister.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("impact analyzer: list assets: %w", err)
	}

	ia.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("policy_id", policy.ID.String()).
		Str("category", string(policy.Category)).
		Int("asset_count", len(assets)).
		Msg("starting impact analysis")

	start := time.Now()

	// Filter to assets in scope, then evaluate.
	var evaluated int
	var violations []model.PolicyViolation

	for _, asset := range assets {

		if !ia.assetInScope(asset, policy) {
			continue
		}

		evaluated++

		isViolation, description := EvaluateRule(asset, policy)
		if !isViolation {
			continue
		}

		violation := model.PolicyViolation{
			PolicyID:             policy.ID,
			PolicyName:           policy.Name,
			Category:             string(policy.Category),
			AssetID:              asset.ID,
			AssetName:            asset.AssetName,
			AssetType:            asset.AssetType,
			Classification:       asset.DataClassification,
			Severity:             policy.Severity,
			Description:          description,
			Enforcement:          string(policy.Enforcement),
			ComplianceFrameworks: policy.ComplianceFrameworks,
		}

		violations = append(violations, violation)
	}

	ia.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("policy_id", policy.ID.String()).
		Int("evaluated", evaluated).
		Int("violations", len(violations)).
		Dur("duration", time.Since(start)).
		Msg("impact analysis complete")

	return &model.PolicyImpact{
		TotalAssetsEvaluated: evaluated,
		ViolationsFound:      len(violations),
		AffectedAssets:       violations,
	}, nil
}

// assetInScope returns true when the asset matches the policy's scope filters.
func (ia *ImpactAnalyzer) assetInScope(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) bool {
	if len(policy.ScopeClassification) > 0 {
		if !stringInSlice(asset.DataClassification, policy.ScopeClassification) {
			return false
		}
	}

	if len(policy.ScopeAssetTypes) > 0 {
		if !stringInSlice(asset.AssetType, policy.ScopeAssetTypes) {
			return false
		}
	}

	return true
}
