package verification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// ConfigVerifier verifies configuration-change remediations by checking that
// the asset metadata in the database matches each key/value declared in
// plan.target_config.
type ConfigVerifier struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewConfigVerifier creates a ConfigVerifier.
func NewConfigVerifier(db *pgxpool.Pool, logger zerolog.Logger) *ConfigVerifier {
	return &ConfigVerifier{
		db:     db,
		logger: logger.With().Str("component", "config-verifier").Logger(),
	}
}

// Verify checks that asset metadata reflects the desired configuration keys
// declared in action.Plan.TargetConfig for every affected asset.
func (c *ConfigVerifier) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	result := &model.VerificationResult{Checks: make([]model.VerificationCheck, 0)}
	allPassed := true

	if len(action.Plan.TargetConfig) == 0 {
		result.Verified = true
		result.Checks = append(result.Checks, model.VerificationCheck{
			Name:     "Configuration target check",
			Passed:   true,
			Expected: "no target_config keys declared",
			Actual:   "no target_config keys declared",
		})
		return result, nil
	}

	for _, assetID := range action.AffectedAssetIDs {
		var rawMeta []byte
		err := c.db.QueryRow(ctx,
			"SELECT COALESCE(metadata, '{}') FROM assets WHERE id=$1 AND tenant_id=$2",
			assetID, action.TenantID,
		).Scan(&rawMeta)
		if err != nil {
			allPassed = false
			c.logger.Warn().Err(err).Str("asset_id", assetID.String()).Msg("failed to fetch asset metadata")
			result.Checks = append(result.Checks, model.VerificationCheck{
				Name:     fmt.Sprintf("Config check on asset %s", assetID),
				Expected: "metadata readable",
				Actual:   fmt.Sprintf("query error: %v", err),
				Passed:   false,
			})
			continue
		}

		var meta map[string]interface{}
		if err := json.Unmarshal(rawMeta, &meta); err != nil {
			allPassed = false
			result.Checks = append(result.Checks, model.VerificationCheck{
				Name:     fmt.Sprintf("Config check on asset %s", assetID),
				Expected: "valid JSON metadata",
				Actual:   fmt.Sprintf("parse error: %v", err),
				Passed:   false,
			})
			continue
		}

		for key, wantRaw := range action.Plan.TargetConfig {
			wantStr := fmt.Sprintf("%v", wantRaw)
			gotRaw, exists := meta[key]
			gotStr := ""
			if exists {
				gotStr = fmt.Sprintf("%v", gotRaw)
			}
			passed := exists && gotStr == wantStr
			if !passed {
				allPassed = false
			}
			actual := gotStr
			if !exists {
				actual = "<key absent>"
			}
			result.Checks = append(result.Checks, model.VerificationCheck{
				Name:     fmt.Sprintf("Asset %s — config key '%s'", assetID, key),
				Expected: wantStr,
				Actual:   actual,
				Passed:   passed,
			})
		}
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more configuration keys do not match the desired state"
	}
	return result, nil
}
