package integration

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

// DLPRule defines a Data Loss Prevention rule generated from PII classification
// analysis of DSPM data assets.
type DLPRule struct {
	ID             string   `json:"id"`
	DataIdentifier string   `json:"data_identifier"`
	PIIType        string   `json:"pii_type"`
	Channels       []string `json:"channels"`
	Actions        []string `json:"actions"`
	Classification string   `json:"classification"`
	Description    string   `json:"description"`
}

// DLPPolicyGenerator analyses DSPM data assets and produces DLP rules
// tailored to the PII types found in each asset.
type DLPPolicyGenerator struct {
	logger zerolog.Logger
}

// NewDLPPolicyGenerator constructs a DLPPolicyGenerator.
func NewDLPPolicyGenerator(logger zerolog.Logger) *DLPPolicyGenerator {
	return &DLPPolicyGenerator{
		logger: logger.With().Str("component", "dlp_policy_generator").Logger(),
	}
}

// Generate analyses the given data assets and produces DLP rules for any
// asset that contains PII. Each distinct PII type found across all assets
// generates a dedicated rule with channels and enforcement actions
// appropriate to the data sensitivity:
//
//   - credit_card  → block external, require masking in non-prod
//   - ssn          → block all unencrypted, require tokenization
//   - email/phone  → allow internal, block external without consent
//   - national_id  → block external, require encryption
//   - passport     → block external, require encryption
//   - health_data  → block all unencrypted, require compliance controls
func (dg *DLPPolicyGenerator) Generate(ctx context.Context, tenantID uuid.UUID, assets []cybermodel.DSPMDataAsset) ([]DLPRule, error) {
	if len(assets) == 0 {
		return nil, nil
	}

	dg.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("asset_count", len(assets)).
		Msg("generating DLP rules from asset PII classifications")

	// Collect unique PII types across all assets, tracking the highest
	// classification level seen for each type.
	piiClassifications := make(map[string]string)
	for i := range assets {
		asset := &assets[i]
		if !asset.ContainsPII || len(asset.PIITypes) == 0 {
			continue
		}
		for _, piiType := range asset.PIITypes {
			normalised := strings.ToLower(strings.TrimSpace(piiType))
			if normalised == "" {
				continue
			}
			existing, ok := piiClassifications[normalised]
			if !ok || classificationRank(asset.DataClassification) > classificationRank(existing) {
				piiClassifications[normalised] = asset.DataClassification
			}
		}
	}

	if len(piiClassifications) == 0 {
		dg.logger.Info().
			Str("tenant_id", tenantID.String()).
			Msg("no PII types found across assets; no DLP rules generated")
		return nil, nil
	}

	rules := make([]DLPRule, 0, len(piiClassifications))
	ruleIndex := 0

	for piiType, classification := range piiClassifications {
		rule := buildDLPRule(tenantID, piiType, classification, ruleIndex)
		rules = append(rules, rule)
		ruleIndex++
	}

	dg.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("rules_generated", len(rules)).
		Msg("DLP rule generation complete")

	return rules, nil
}

// buildDLPRule creates a single DLP rule for a given PII type and
// classification level, applying enforcement policies specific to the
// sensitivity of the data.
func buildDLPRule(tenantID uuid.UUID, piiType, classification string, index int) DLPRule {
	ruleID := fmt.Sprintf("dlp-%s-%s-%d", tenantID.String()[:8], piiType, index)

	switch piiType {
	case "credit_card":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "credit-card-number",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "removable_media", "web_upload", "api"},
			Actions: []string{
				"block_external_transfer",
				"require_masking_non_production",
				"alert_security_team",
				"encrypt_at_rest",
				"log_access",
			},
			Classification: classification,
			Description:    "Block external transmission of credit card numbers. Require masking in non-production environments. All access must be encrypted and logged for PCI DSS compliance.",
		}

	case "ssn":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "social-security-number",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "removable_media", "web_upload", "api", "database_export", "print"},
			Actions: []string{
				"block_all_unencrypted",
				"require_tokenization",
				"alert_security_team",
				"quarantine_on_violation",
				"log_access",
			},
			Classification: classification,
			Description:    "Block all unencrypted transmission or storage of Social Security Numbers. Require tokenization for all processing. Quarantine assets on policy violation.",
		}

	case "email":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "email-address",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "web_upload", "api"},
			Actions: []string{
				"allow_internal_transfer",
				"block_external_without_consent",
				"log_access",
				"require_pseudonymization_analytics",
			},
			Classification: classification,
			Description:    "Allow internal transfer of email addresses. Block external sharing without documented consent. Require pseudonymization for analytics and non-operational use.",
		}

	case "phone":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "phone-number",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "web_upload", "api"},
			Actions: []string{
				"allow_internal_transfer",
				"block_external_without_consent",
				"log_access",
				"require_pseudonymization_analytics",
			},
			Classification: classification,
			Description:    "Allow internal transfer of phone numbers. Block external sharing without documented consent. Require pseudonymization for analytics workloads.",
		}

	case "national_id":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "national-identification-number",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "removable_media", "web_upload", "api", "database_export"},
			Actions: []string{
				"block_external_transfer",
				"require_encryption",
				"alert_security_team",
				"log_access",
				"require_access_justification",
			},
			Classification: classification,
			Description:    "Block external transfer of national identification numbers. Require encryption for storage and transmission. All access requires documented justification.",
		}

	case "passport":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "passport-number",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "removable_media", "web_upload", "api", "database_export"},
			Actions: []string{
				"block_external_transfer",
				"require_encryption",
				"alert_security_team",
				"log_access",
				"require_access_justification",
			},
			Classification: classification,
			Description:    "Block external transfer of passport numbers. Require encryption at rest and in transit. Access must be justified and logged.",
		}

	case "health_data":
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: "protected-health-information",
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "removable_media", "web_upload", "api", "database_export", "print", "screen_capture"},
			Actions: []string{
				"block_all_unencrypted",
				"require_hipaa_compliance_controls",
				"require_baa_verification",
				"alert_security_team",
				"alert_compliance_team",
				"quarantine_on_violation",
				"log_access",
			},
			Classification: classification,
			Description:    "Block all unencrypted transmission or storage of health data. Require HIPAA compliance controls and Business Associate Agreement verification. Quarantine on violation and notify both security and compliance teams.",
		}

	default:
		// Generic rule for unrecognised PII types — apply conservative controls.
		return DLPRule{
			ID:             ruleID,
			DataIdentifier: fmt.Sprintf("pii-%s", piiType),
			PIIType:        piiType,
			Channels:       []string{"email", "cloud_storage", "web_upload", "api"},
			Actions:        dlpActionsForClassification(classification),
			Classification: classification,
			Description: fmt.Sprintf(
				"DLP rule for PII type %q at %s classification. Apply controls proportional to data sensitivity.",
				piiType, classification,
			),
		}
	}
}

// dlpActionsForClassification returns default DLP actions based on the data
// classification level, used as a fallback for unrecognised PII types.
func dlpActionsForClassification(classification string) []string {
	switch strings.ToLower(classification) {
	case "restricted":
		return []string{
			"block_all_unencrypted",
			"require_encryption",
			"alert_security_team",
			"quarantine_on_violation",
			"log_access",
			"require_access_justification",
		}
	case "confidential":
		return []string{
			"block_external_transfer",
			"require_encryption",
			"alert_security_team",
			"log_access",
		}
	case "internal":
		return []string{
			"allow_internal_transfer",
			"block_external_without_approval",
			"log_access",
		}
	default:
		return []string{
			"log_access",
			"monitor_external_transfer",
		}
	}
}

// classificationRank returns a numeric rank for a data classification level.
// Higher values indicate more sensitive data.
func classificationRank(classification string) int {
	switch strings.ToLower(classification) {
	case "restricted":
		return 4
	case "confidential":
		return 3
	case "internal":
		return 2
	case "public":
		return 1
	default:
		return 0
	}
}
