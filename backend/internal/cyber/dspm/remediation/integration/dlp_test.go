package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

func newTestDLPGenerator() *DLPPolicyGenerator {
	logger := zerolog.Nop()
	return NewDLPPolicyGenerator(logger)
}

func TestGenerateCreditCard(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "payment-db",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"credit_card"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "credit_card", rule.PIIType)
	assert.Equal(t, "credit-card-number", rule.DataIdentifier)
	assert.Equal(t, "restricted", rule.Classification)
	assert.Contains(t, rule.Channels, "email")
	assert.Contains(t, rule.Channels, "cloud_storage")
	assert.Contains(t, rule.Channels, "removable_media")
	assert.Contains(t, rule.Actions, "block_external_transfer")
	assert.Contains(t, rule.Actions, "require_masking_non_production")
	assert.Contains(t, rule.Actions, "encrypt_at_rest")
	assert.NotEmpty(t, rule.Description)
	assert.Contains(t, rule.Description, "credit card")
}

func TestGenerateSSN(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "employee-db",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"ssn"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "ssn", rule.PIIType)
	assert.Equal(t, "social-security-number", rule.DataIdentifier)
	assert.Contains(t, rule.Actions, "block_all_unencrypted")
	assert.Contains(t, rule.Actions, "require_tokenization")
	assert.Contains(t, rule.Actions, "quarantine_on_violation")
	assert.Contains(t, rule.Channels, "database_export")
	assert.Contains(t, rule.Channels, "print")
	assert.Contains(t, rule.Description, "Social Security Numbers")
}

func TestGenerateEmail(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "customer-db",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{"email"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "email", rule.PIIType)
	assert.Equal(t, "email-address", rule.DataIdentifier)
	assert.Equal(t, "confidential", rule.Classification)
	assert.Contains(t, rule.Actions, "allow_internal_transfer")
	assert.Contains(t, rule.Actions, "block_external_without_consent")
	assert.Contains(t, rule.Actions, "require_pseudonymization_analytics")
	assert.Contains(t, rule.Description, "email addresses")
}

func TestGeneratePhone(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "contacts-db",
			DataClassification: "internal",
			ContainsPII:        true,
			PIITypes:           []string{"phone"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "phone", rule.PIIType)
	assert.Equal(t, "phone-number", rule.DataIdentifier)
	assert.Contains(t, rule.Actions, "allow_internal_transfer")
	assert.Contains(t, rule.Actions, "block_external_without_consent")
}

func TestGenerateNationalID(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "id-db",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"national_id"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "national_id", rule.PIIType)
	assert.Equal(t, "national-identification-number", rule.DataIdentifier)
	assert.Contains(t, rule.Actions, "block_external_transfer")
	assert.Contains(t, rule.Actions, "require_encryption")
	assert.Contains(t, rule.Actions, "require_access_justification")
}

func TestGeneratePassport(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "travel-db",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"passport"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "passport", rule.PIIType)
	assert.Equal(t, "passport-number", rule.DataIdentifier)
	assert.Contains(t, rule.Actions, "block_external_transfer")
	assert.Contains(t, rule.Actions, "require_encryption")
}

func TestGenerateHealthData(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "health-db",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"health_data"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "health_data", rule.PIIType)
	assert.Equal(t, "protected-health-information", rule.DataIdentifier)
	assert.Contains(t, rule.Actions, "block_all_unencrypted")
	assert.Contains(t, rule.Actions, "require_hipaa_compliance_controls")
	assert.Contains(t, rule.Actions, "require_baa_verification")
	assert.Contains(t, rule.Actions, "quarantine_on_violation")
	assert.Contains(t, rule.Channels, "screen_capture")
	assert.Contains(t, rule.Channels, "print")
}

func TestGenerateMultiplePII(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "combined-db",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"credit_card", "ssn", "email"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	assert.Len(t, rules, 3, "should generate one rule per PII type")

	piiTypes := make(map[string]bool)
	for _, r := range rules {
		piiTypes[r.PIIType] = true
		assert.NotEmpty(t, r.ID)
		assert.NotEmpty(t, r.DataIdentifier)
		assert.NotEmpty(t, r.Actions)
		assert.NotEmpty(t, r.Channels)
	}
	assert.True(t, piiTypes["credit_card"])
	assert.True(t, piiTypes["ssn"])
	assert.True(t, piiTypes["email"])
}

func TestGenerateMultipleAssetsDeduplicates(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "db-1",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{"email"},
		},
		{
			ID:                 uuid.New(),
			AssetName:          "db-2",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"email"}, // same PII type
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)

	// Should only produce one rule for "email" with the highest classification.
	assert.Len(t, rules, 1)
	assert.Equal(t, "email", rules[0].PIIType)
	// The highest classification (restricted > confidential) should be used.
	assert.Equal(t, "restricted", rules[0].Classification)
}

func TestGenerateNoPII(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "no-pii-db",
			DataClassification: "public",
			ContainsPII:        false,
			PIITypes:           nil,
		},
		{
			ID:                 uuid.New(),
			AssetName:          "another-no-pii",
			DataClassification: "internal",
			ContainsPII:        false,
			PIITypes:           []string{},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	assert.Nil(t, rules, "no PII assets should generate no rules")
}

func TestGenerateEmptyAssets(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	rules, err := gen.Generate(context.Background(), tenantID, nil)
	require.NoError(t, err)
	assert.Nil(t, rules, "empty assets should generate no rules")
}

func TestGenerateUnknownPIIType(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "custom-pii-db",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{"biometric_data"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "biometric_data", rule.PIIType)
	assert.Equal(t, "pii-biometric_data", rule.DataIdentifier)
	assert.Contains(t, rule.Description, "biometric_data")
	// For confidential classification, should have block_external_transfer.
	assert.Contains(t, rule.Actions, "block_external_transfer")
	assert.Contains(t, rule.Actions, "require_encryption")
}

func TestGenerateRuleIDFormat(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "test-db",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{"ssn"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Rule ID format: dlp-{first 8 chars of tenant ID}-{pii_type}-{index}
	expectedPrefix := "dlp-" + tenantID.String()[:8]
	assert.Contains(t, rules[0].ID, expectedPrefix)
}

func TestGenerateWhitespaceAndCasePIITypes(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "messy-pii-db",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{" Email ", "EMAIL"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)

	// Both " Email " and "EMAIL" should normalize to "email" and produce a single rule.
	assert.Len(t, rules, 1)
	assert.Equal(t, "email", rules[0].PIIType)
}

func TestGenerateEmptyPIITypeString(t *testing.T) {
	gen := newTestDLPGenerator()
	tenantID := uuid.New()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "empty-pii-type-db",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{"", "  ", "email"},
		},
	}

	rules, err := gen.Generate(context.Background(), tenantID, assets)
	require.NoError(t, err)

	// Empty and whitespace-only PII types should be skipped.
	assert.Len(t, rules, 1)
	assert.Equal(t, "email", rules[0].PIIType)
}
