package policy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// classificationRank maps a classification label to its ordinal rank.
// Higher rank means more sensitive data.
func classificationRank(classification string) int {
	switch strings.ToLower(classification) {
	case "public":
		return 0
	case "internal":
		return 1
	case "confidential":
		return 2
	case "restricted":
		return 3
	default:
		return -1
	}
}

// exposureRank maps a network exposure level to its ordinal rank.
// Higher rank means broader exposure.
func exposureRank(exposure string) int {
	switch strings.ToLower(exposure) {
	case "internal_only":
		return 0
	case "vpn_accessible":
		return 1
	case "internet_facing":
		return 2
	default:
		return -1
	}
}

// EvaluateRule dispatches to the appropriate category-specific rule evaluator.
// It returns (true, description) when the asset violates the policy, or
// (false, "") when the asset is compliant.
func EvaluateRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	switch policy.Category {
	case model.PolicyCategoryEncryption:
		return evaluateEncryptionRule(asset, policy)
	case model.PolicyCategoryClassification:
		return evaluateClassificationRule(asset, policy)
	case model.PolicyCategoryRetention:
		return evaluateRetentionRule(asset, policy)
	case model.PolicyCategoryExposure:
		return evaluateExposureRule(asset, policy)
	case model.PolicyCategoryPIIProtection:
		return evaluatePIIProtectionRule(asset, policy)
	case model.PolicyCategoryAccessReview:
		return evaluateAccessReviewRule(asset, policy)
	case model.PolicyCategoryBackup:
		return evaluateBackupRule(asset, policy)
	case model.PolicyCategoryAuditLogging:
		return evaluateAuditLoggingRule(asset, policy)
	default:
		return false, ""
	}
}

// evaluateEncryptionRule checks whether the asset meets encryption requirements.
// The rule's ClassificationMin field gates enforcement: assets below the
// minimum classification level are considered compliant regardless of their
// encryption state.
func evaluateEncryptionRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.EncryptionRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	// If a minimum classification is set, skip assets below it.
	if rule.ClassificationMin != "" {
		if classificationRank(asset.DataClassification) < classificationRank(rule.ClassificationMin) {
			return false, ""
		}
	}

	var issues []string

	if rule.RequireAtRest {
		if asset.EncryptedAtRest == nil || !*asset.EncryptedAtRest {
			issues = append(issues, "not encrypted at rest")
		}
	}

	if rule.RequireInTransit {
		if asset.EncryptedInTransit == nil || !*asset.EncryptedInTransit {
			issues = append(issues, "not encrypted in transit")
		}
	}

	if len(issues) == 0 {
		return false, ""
	}

	return true, fmt.Sprintf(
		"Asset %q (%s) violates encryption policy: %s",
		asset.AssetName, asset.DataClassification, strings.Join(issues, "; "),
	)
}

// evaluateClassificationRule verifies that PII-containing assets meet a
// minimum classification level when auto-escalation is enabled.
func evaluateClassificationRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.ClassificationRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	if !rule.AutoEscalate || rule.PIIImpliesMin == "" {
		return false, ""
	}

	// Only evaluate assets that contain PII.
	if !asset.ContainsPII {
		return false, ""
	}

	if classificationRank(asset.DataClassification) < classificationRank(rule.PIIImpliesMin) {
		return true, fmt.Sprintf(
			"Asset %q contains PII but is classified as %q; minimum required classification is %q",
			asset.AssetName, asset.DataClassification, rule.PIIImpliesMin,
		)
	}

	return false, ""
}

// evaluateRetentionRule checks whether a data asset has exceeded its maximum
// retention period. The optional ClassificationScope restricts enforcement to
// specific classification levels.
func evaluateRetentionRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.RetentionRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	if rule.MaxDays <= 0 {
		return false, ""
	}

	// If a classification scope is defined, skip assets not in the list.
	if len(rule.ClassificationScope) > 0 {
		if !stringInSlice(asset.DataClassification, rule.ClassificationScope) {
			return false, ""
		}
	}

	ageInDays := int(time.Since(asset.CreatedAt).Hours() / 24)
	if ageInDays <= rule.MaxDays {
		return false, ""
	}

	return true, fmt.Sprintf(
		"Asset %q (%s) is %d days old, exceeding the %d-day retention limit (action: %s)",
		asset.AssetName, asset.DataClassification, ageInDays, rule.MaxDays, rule.Action,
	)
}

// evaluateExposureRule checks whether the asset's network exposure level
// exceeds the policy maximum, gated by an optional minimum classification.
func evaluateExposureRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.ExposureRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	// If a minimum classification is set, skip assets below it.
	if rule.ClassificationMin != "" {
		if classificationRank(asset.DataClassification) < classificationRank(rule.ClassificationMin) {
			return false, ""
		}
	}

	if asset.NetworkExposure == nil {
		return false, ""
	}

	assetRank := exposureRank(*asset.NetworkExposure)
	maxRank := exposureRank(rule.MaxExposure)

	// Unknown ranks are not actionable.
	if assetRank < 0 || maxRank < 0 {
		return false, ""
	}

	if assetRank <= maxRank {
		return false, ""
	}

	return true, fmt.Sprintf(
		"Asset %q (%s) has network exposure %q which exceeds the maximum allowed %q",
		asset.AssetName, asset.DataClassification, *asset.NetworkExposure, rule.MaxExposure,
	)
}

// evaluatePIIProtectionRule verifies that PII-bearing assets have adequate
// encryption, access controls, and audit logging.
func evaluatePIIProtectionRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.PIIProtectionRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	// Only applies to assets that contain PII.
	if !asset.ContainsPII {
		return false, ""
	}

	var issues []string

	if rule.RequireEncryption {
		atRest := asset.EncryptedAtRest != nil && *asset.EncryptedAtRest
		inTransit := asset.EncryptedInTransit != nil && *asset.EncryptedInTransit
		if !atRest || !inTransit {
			issues = append(issues, "missing full encryption (at-rest and in-transit required)")
		}
	}

	if rule.RequireAccessControl != "" {
		if asset.AccessControlType == nil || *asset.AccessControlType == "" {
			issues = append(issues, fmt.Sprintf("no access control configured (required: %s)", rule.RequireAccessControl))
		} else if !strings.EqualFold(*asset.AccessControlType, rule.RequireAccessControl) {
			issues = append(issues, fmt.Sprintf(
				"access control type %q does not meet required %q",
				*asset.AccessControlType, rule.RequireAccessControl,
			))
		}
	}

	if rule.RequireAudit {
		if asset.AuditLogging == nil || !*asset.AuditLogging {
			issues = append(issues, "audit logging not enabled")
		}
	}

	if len(issues) == 0 {
		return false, ""
	}

	return true, fmt.Sprintf(
		"PII asset %q (types: %s) violates PII protection policy: %s",
		asset.AssetName, strings.Join(asset.PIITypes, ", "), strings.Join(issues, "; "),
	)
}

// evaluateAccessReviewRule checks whether the asset's last access review is
// within the required interval, gated by an optional minimum classification.
func evaluateAccessReviewRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.AccessReviewRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	if rule.ReviewIntervalDays <= 0 {
		return false, ""
	}

	// If a minimum classification is set, skip assets below it.
	if rule.ClassificationMin != "" {
		if classificationRank(asset.DataClassification) < classificationRank(rule.ClassificationMin) {
			return false, ""
		}
	}

	if asset.LastAccessReview == nil {
		return true, fmt.Sprintf(
			"Asset %q (%s) has never had an access review; policy requires review every %d days",
			asset.AssetName, asset.DataClassification, rule.ReviewIntervalDays,
		)
	}

	daysSinceReview := int(time.Since(*asset.LastAccessReview).Hours() / 24)
	if daysSinceReview <= rule.ReviewIntervalDays {
		return false, ""
	}

	return true, fmt.Sprintf(
		"Asset %q (%s) last access review was %d days ago; policy requires review every %d days",
		asset.AssetName, asset.DataClassification, daysSinceReview, rule.ReviewIntervalDays,
	)
}

// evaluateBackupRule checks whether assets with the specified classifications
// have backup configured.
func evaluateBackupRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.BackupRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	// If specific classifications are listed, only check those.
	if len(rule.RequiredFor) > 0 {
		if !stringInSlice(asset.DataClassification, rule.RequiredFor) {
			return false, ""
		}
	}

	if asset.BackupConfigured != nil && *asset.BackupConfigured {
		return false, ""
	}

	return true, fmt.Sprintf(
		"Asset %q (%s) does not have backup configured",
		asset.AssetName, asset.DataClassification,
	)
}

// evaluateAuditLoggingRule checks whether assets with the specified
// classifications have audit logging enabled.
func evaluateAuditLoggingRule(asset *cybermodel.DSPMDataAsset, policy *model.DataPolicy) (bool, string) {
	var rule model.AuditLoggingRule
	if err := json.Unmarshal(policy.Rule, &rule); err != nil {
		return false, ""
	}

	// If specific classifications are listed, only check those.
	if len(rule.RequiredFor) > 0 {
		if !stringInSlice(asset.DataClassification, rule.RequiredFor) {
			return false, ""
		}
	}

	if asset.AuditLogging != nil && *asset.AuditLogging {
		return false, ""
	}

	return true, fmt.Sprintf(
		"Asset %q (%s) does not have audit logging enabled",
		asset.AssetName, asset.DataClassification,
	)
}
