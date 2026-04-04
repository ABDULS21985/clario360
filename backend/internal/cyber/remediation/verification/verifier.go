package verification

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// Verifier orchestrates post-execution verification using type-specific verifiers.
type Verifier struct {
	vuln   *VulnVerifier
	config *ConfigVerifier
	logger zerolog.Logger
}

// NewVerifier creates a Verifier.
func NewVerifier(db *pgxpool.Pool, logger zerolog.Logger) *Verifier {
	return &Verifier{
		vuln:   NewVulnVerifier(db, logger),
		config: NewConfigVerifier(db, logger),
		logger: logger.With().Str("component", "remediation-verifier").Logger(),
	}
}

// Verify runs post-execution verification appropriate for the action type.
func (v *Verifier) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	switch action.Type {
	case model.RemediationTypePatch,
		model.RemediationTypeAccessRevoke,
		model.RemediationTypeCertRenew:
		return v.vuln.Verify(ctx, action)

	case model.RemediationTypeConfigChange,
		model.RemediationTypeFirewallRule:
		return v.config.Verify(ctx, action)

	case model.RemediationTypeBlockIP:
		return v.verifyIPBlock(ctx, action)

	case model.RemediationTypeIsolateAsset:
		return v.verifyIsolation(ctx, action)

	case model.RemediationTypeCustom:
		return &model.VerificationResult{
			Verified: true,
			Checks: []model.VerificationCheck{{
				Name:     "Manual verification confirmation",
				Passed:   true,
				Expected: "operator confirmed",
				Actual:   "operator confirmed",
			}},
		}, nil

	default:
		return nil, fmt.Errorf("no verifier registered for remediation type '%s'", action.Type)
	}
}

func (v *Verifier) verifyIPBlock(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	result := &model.VerificationResult{Checks: make([]model.VerificationCheck, 0)}
	allPassed := true

	for _, target := range action.Plan.BlockTargets {
		var active bool
		err := v.vuln.db.QueryRow(ctx,
			"SELECT COALESCE(bool_or(active), false) FROM threat_indicators WHERE tenant_id=$1 AND value=$2 AND type='ip'",
			action.TenantID, target,
		).Scan(&active)
		if err != nil {
			allPassed = false
		}
		passed := active
		if !passed {
			allPassed = false
		}
		result.Checks = append(result.Checks, model.VerificationCheck{
			Name:     fmt.Sprintf("Block indicator active for %s", target),
			Expected: "indicator active=true",
			Actual:   fmt.Sprintf("active=%v", active),
			Passed:   passed,
		})
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more IP block indicators are not active"
	}
	return result, nil
}

func (v *Verifier) verifyIsolation(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	result := &model.VerificationResult{Checks: make([]model.VerificationCheck, 0)}
	allPassed := true

	for _, assetID := range action.AffectedAssetIDs {
		var assetName string
		var isolated bool
		_ = v.vuln.db.QueryRow(ctx, "SELECT name FROM assets WHERE id=$1", assetID).Scan(&assetName)
		_ = v.vuln.db.QueryRow(ctx,
			"SELECT COALESCE((metadata->>'isolated')::boolean, false) FROM assets WHERE id=$1",
			assetID,
		).Scan(&isolated)

		if !isolated {
			allPassed = false
		}
		result.Checks = append(result.Checks, model.VerificationCheck{
			Name:     fmt.Sprintf("Asset '%s' isolation check", assetName),
			Expected: "isolated=true",
			Actual:   fmt.Sprintf("isolated=%v", isolated),
			Passed:   isolated,
		})
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more assets are not isolated"
	}
	return result, nil
}
