package verification

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// VulnVerifier verifies patch and access remediation outcomes by re-checking
// whether the targeted vulnerabilities / CVEs are still present post-execution.
type VulnVerifier struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewVulnVerifier creates a VulnVerifier.
func NewVulnVerifier(db *pgxpool.Pool, logger zerolog.Logger) *VulnVerifier {
	return &VulnVerifier{
		db:     db,
		logger: logger.With().Str("component", "vuln-verifier").Logger(),
	}
}

// Verify checks that targeted CVEs/vulnerabilities are no longer active on the
// affected assets. A check is emitted for each asset × CVE combination.
func (v *VulnVerifier) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	result := &model.VerificationResult{Checks: make([]model.VerificationCheck, 0)}
	allPassed := true

	cves := extractCVETargets(action)
	if len(cves) == 0 {
		// Nothing to verify — count as a trivial pass.
		result.Verified = true
		result.Checks = append(result.Checks, model.VerificationCheck{
			Name:     "CVE target check",
			Passed:   true,
			Expected: "no CVE targets specified",
			Actual:   "no CVE targets specified",
		})
		return result, nil
	}

	for _, assetID := range action.AffectedAssetIDs {
		for _, cve := range cves {
			var count int
			err := v.db.QueryRow(ctx,
				`SELECT COUNT(*) FROM vulnerabilities
				 WHERE asset_id = $1
				   AND tenant_id = $2
				   AND cve_id = $3
				   AND status NOT IN ('remediated','accepted','false_positive')`,
				assetID, action.TenantID, cve,
			).Scan(&count)
			if err != nil {
				allPassed = false
				v.logger.Warn().Err(err).
					Str("asset_id", assetID.String()).
					Str("cve", cve).
					Msg("vuln verification query failed")
				result.Checks = append(result.Checks, model.VerificationCheck{
					Name:     fmt.Sprintf("CVE %s on asset %s", cve, assetID),
					Expected: "vuln not active",
					Actual:   fmt.Sprintf("query error: %v", err),
					Passed:   false,
				})
				continue
			}
			passed := count == 0
			if !passed {
				allPassed = false
			}
			actual := "not present"
			if count > 0 {
				actual = fmt.Sprintf("%d active record(s) found", count)
			}
			result.Checks = append(result.Checks, model.VerificationCheck{
				Name:     fmt.Sprintf("CVE %s on asset %s", cve, assetID),
				Expected: "vuln not active",
				Actual:   actual,
				Passed:   passed,
			})
		}
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more CVEs remain active after remediation"
	}
	return result, nil
}

// extractCVETargets returns CVE/target identifiers from the remediation plan.
// TargetVersion is used to store the primary CVE ID for patch-type remediations.
func extractCVETargets(action *model.RemediationAction) []string {
	if action.Plan.TargetVersion == "" {
		return nil
	}
	return []string{action.Plan.TargetVersion}
}
