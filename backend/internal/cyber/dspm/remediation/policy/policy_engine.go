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

// AssetLister abstracts fetching active data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// ExceptionChecker determines whether a policy violation is covered by an active exception.
type ExceptionChecker interface {
	HasActiveException(ctx context.Context, tenantID uuid.UUID, assetID uuid.UUID, policyID uuid.UUID) (bool, error)
}

// PolicyEngine evaluates data policies against the DSPM asset inventory.
type PolicyEngine struct {
	assetLister      AssetLister
	exceptionChecker ExceptionChecker
	logger           zerolog.Logger
}

// NewPolicyEngine constructs a PolicyEngine with its required dependencies.
func NewPolicyEngine(assetLister AssetLister, exceptionChecker ExceptionChecker, logger zerolog.Logger) *PolicyEngine {
	return &PolicyEngine{
		assetLister:      assetLister,
		exceptionChecker: exceptionChecker,
		logger:           logger.With().Str("component", "policy_engine").Logger(),
	}
}

// EvaluateAll evaluates every enabled policy against all active assets for the
// given tenant. Violations covered by an active exception are silently skipped.
func (pe *PolicyEngine) EvaluateAll(ctx context.Context, tenantID uuid.UUID, policies []model.DataPolicy) ([]model.PolicyViolation, error) {
	assets, err := pe.assetLister.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("policy engine: list assets: %w", err)
	}

	pe.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("policy_count", len(policies)).
		Int("asset_count", len(assets)).
		Msg("starting policy evaluation")

	start := time.Now()
	var violations []model.PolicyViolation

	for i := range policies {
		policy := &policies[i]
		if !policy.Enabled {
			continue
		}

		policyViolations := pe.evaluatePolicyAgainstAssets(ctx, tenantID, policy, assets)
		violations = append(violations, policyViolations...)
	}

	pe.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("violations_found", len(violations)).
		Dur("duration", time.Since(start)).
		Msg("policy evaluation complete")

	return violations, nil
}

// DryRunPolicy evaluates a single policy against all active assets without
// persisting results or triggering enforcement. Useful for previewing the
// impact of a new or modified policy before enabling it.
func (pe *PolicyEngine) DryRunPolicy(ctx context.Context, tenantID uuid.UUID, policy *model.DataPolicy) (*model.PolicyImpact, error) {
	analyzer := NewImpactAnalyzer(pe.assetLister, pe.logger)
	return analyzer.Analyze(ctx, tenantID, policy)
}

// evaluatePolicyAgainstAssets checks one policy against every asset, filtering
// by scope and exceptions.
func (pe *PolicyEngine) evaluatePolicyAgainstAssets(
	ctx context.Context,
	tenantID uuid.UUID,
	policy *model.DataPolicy,
	assets []*cybermodel.DSPMDataAsset,
) []model.PolicyViolation {
	var violations []model.PolicyViolation

	for _, asset := range assets {

		if !pe.assetInScope(asset, policy) {
			continue
		}

		isViolation, description := EvaluateRule(asset, policy)
		if !isViolation {
			continue
		}

		// Check for an active exception before recording the violation.
		if pe.exceptionChecker != nil {
			hasException, err := pe.exceptionChecker.HasActiveException(ctx, tenantID, asset.ID, policy.ID)
			if err != nil {
				pe.logger.Warn().
					Err(err).
					Str("policy_id", policy.ID.String()).
					Str("asset_id", asset.ID.String()).
					Msg("exception check failed, treating as no exception")
			} else if hasException {
				pe.logger.Debug().
					Str("policy_id", policy.ID.String()).
					Str("asset_id", asset.ID.String()).
					Msg("violation suppressed by active exception")
				continue
			}
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

	return violations
}

// assetInScope returns true when the asset matches the policy's scope filters.
// An empty scope list means all values match.
func (pe *PolicyEngine) assetInScope(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) bool {
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

// stringInSlice is a simple membership test.
func stringInSlice(val string, list []string) bool {
	for _, item := range list {
		if item == val {
			return true
		}
	}
	return false
}
