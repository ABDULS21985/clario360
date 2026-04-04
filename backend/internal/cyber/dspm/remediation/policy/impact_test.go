package policy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// mockAssetLister implements AssetLister for tests.
type mockAssetLister struct {
	assets []*cybermodel.DSPMDataAsset
	err    error
}

func (m *mockAssetLister) ListAllActive(_ context.Context, _ uuid.UUID) ([]*cybermodel.DSPMDataAsset, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.assets, nil
}

func newTestImpactAnalyzer(assets []*cybermodel.DSPMDataAsset) *ImpactAnalyzer {
	logger := zerolog.Nop()
	lister := &mockAssetLister{assets: assets}
	return NewImpactAnalyzer(lister, logger)
}

func TestAnalyzeNoViolations(t *testing.T) {
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "encrypted-asset-1",
			DataClassification: "confidential",
			EncryptedAtRest:    boolPtr(true),
			EncryptedInTransit: boolPtr(true),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "encrypted-asset-2",
			DataClassification: "restricted",
			EncryptedAtRest:    boolPtr(true),
			EncryptedInTransit: boolPtr(true),
		},
	}

	analyzer := newTestImpactAnalyzer(assets)

	encryptionRule := model.EncryptionRule{
		RequireAtRest:    true,
		RequireInTransit: true,
	}
	ruleJSON, err := json.Marshal(encryptionRule)
	require.NoError(t, err)

	policy := &model.DataPolicy{
		ID:       uuid.New(),
		Name:     "require-encryption",
		Category: model.PolicyCategoryEncryption,
		Rule:     ruleJSON,
		Severity: "high",
	}

	impact, err := analyzer.Analyze(context.Background(), uuid.New(), policy)
	require.NoError(t, err)
	require.NotNil(t, impact)

	assert.Equal(t, 2, impact.TotalAssetsEvaluated)
	assert.Equal(t, 0, impact.ViolationsFound)
	assert.Empty(t, impact.AffectedAssets)
}

func TestAnalyzeWithViolations(t *testing.T) {
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "unencrypted-asset",
			DataClassification: "confidential",
			EncryptedAtRest:    boolPtr(false),
			EncryptedInTransit: boolPtr(true),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "encrypted-asset",
			DataClassification: "confidential",
			EncryptedAtRest:    boolPtr(true),
			EncryptedInTransit: boolPtr(true),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "nil-encryption-asset",
			DataClassification: "restricted",
			EncryptedAtRest:    nil,
			EncryptedInTransit: boolPtr(true),
		},
	}

	analyzer := newTestImpactAnalyzer(assets)

	encryptionRule := model.EncryptionRule{
		RequireAtRest:    true,
		RequireInTransit: true,
	}
	ruleJSON, err := json.Marshal(encryptionRule)
	require.NoError(t, err)

	policy := &model.DataPolicy{
		ID:                   uuid.New(),
		Name:                 "require-encryption",
		Category:             model.PolicyCategoryEncryption,
		Rule:                 ruleJSON,
		Severity:             "high",
		Enforcement:          model.EnforcementAlert,
		ComplianceFrameworks: []string{"SOC2", "ISO27001"},
	}

	impact, err := analyzer.Analyze(context.Background(), uuid.New(), policy)
	require.NoError(t, err)
	require.NotNil(t, impact)

	assert.Equal(t, 3, impact.TotalAssetsEvaluated)
	assert.Equal(t, 2, impact.ViolationsFound)
	assert.Len(t, impact.AffectedAssets, 2)

	// Verify violation fields are populated correctly.
	for _, v := range impact.AffectedAssets {
		assert.Equal(t, policy.ID, v.PolicyID)
		assert.Equal(t, policy.Name, v.PolicyName)
		assert.Equal(t, "encryption", v.Category)
		assert.Equal(t, "high", v.Severity)
		assert.NotEmpty(t, v.Description)
		assert.Contains(t, v.Description, "not encrypted at rest")
	}
}

func TestAnalyzeScopeFiltering(t *testing.T) {
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "confidential-asset",
			AssetType:          "database",
			DataClassification: "confidential",
			EncryptedAtRest:    boolPtr(false),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "public-asset",
			AssetType:          "file_share",
			DataClassification: "public",
			EncryptedAtRest:    boolPtr(false),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "internal-asset",
			AssetType:          "database",
			DataClassification: "internal",
			EncryptedAtRest:    boolPtr(false),
		},
	}

	analyzer := newTestImpactAnalyzer(assets)

	encryptionRule := model.EncryptionRule{
		RequireAtRest: true,
	}
	ruleJSON, err := json.Marshal(encryptionRule)
	require.NoError(t, err)

	// Scope only to confidential databases.
	policy := &model.DataPolicy{
		ID:                  uuid.New(),
		Name:                "scoped-encryption",
		Category:            model.PolicyCategoryEncryption,
		Rule:                ruleJSON,
		Severity:            "medium",
		ScopeClassification: []string{"confidential"},
		ScopeAssetTypes:     []string{"database"},
	}

	impact, err := analyzer.Analyze(context.Background(), uuid.New(), policy)
	require.NoError(t, err)
	require.NotNil(t, impact)

	// Only the confidential database should be evaluated.
	assert.Equal(t, 1, impact.TotalAssetsEvaluated)
	assert.Equal(t, 1, impact.ViolationsFound)
	assert.Len(t, impact.AffectedAssets, 1)
	assert.Equal(t, "confidential-asset", impact.AffectedAssets[0].AssetName)
}

func TestAnalyzeEmptyAssets(t *testing.T) {
	analyzer := newTestImpactAnalyzer(nil)

	encryptionRule := model.EncryptionRule{
		RequireAtRest: true,
	}
	ruleJSON, err := json.Marshal(encryptionRule)
	require.NoError(t, err)

	policy := &model.DataPolicy{
		ID:       uuid.New(),
		Name:     "no-assets-policy",
		Category: model.PolicyCategoryEncryption,
		Rule:     ruleJSON,
		Severity: "low",
	}

	impact, err := analyzer.Analyze(context.Background(), uuid.New(), policy)
	require.NoError(t, err)
	require.NotNil(t, impact)

	assert.Equal(t, 0, impact.TotalAssetsEvaluated)
	assert.Equal(t, 0, impact.ViolationsFound)
	assert.Empty(t, impact.AffectedAssets)
}

func TestAnalyzeAssetListerError(t *testing.T) {
	logger := zerolog.Nop()
	lister := &mockAssetLister{
		err: assert.AnError,
	}
	analyzer := NewImpactAnalyzer(lister, logger)

	policy := &model.DataPolicy{
		ID:       uuid.New(),
		Category: model.PolicyCategoryEncryption,
		Rule:     json.RawMessage(`{}`),
	}

	impact, err := analyzer.Analyze(context.Background(), uuid.New(), policy)
	assert.Error(t, err)
	assert.Nil(t, impact)
	assert.Contains(t, err.Error(), "list assets")
}

func TestAnalyzeClassificationScopeOnly(t *testing.T) {
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "restricted-asset",
			AssetType:          "database",
			DataClassification: "restricted",
			EncryptedAtRest:    boolPtr(false),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "internal-asset",
			AssetType:          "database",
			DataClassification: "internal",
			EncryptedAtRest:    boolPtr(false),
		},
	}

	analyzer := newTestImpactAnalyzer(assets)

	encryptionRule := model.EncryptionRule{RequireAtRest: true}
	ruleJSON, err := json.Marshal(encryptionRule)
	require.NoError(t, err)

	policy := &model.DataPolicy{
		ID:                  uuid.New(),
		Name:                "restricted-only",
		Category:            model.PolicyCategoryEncryption,
		Rule:                ruleJSON,
		Severity:            "critical",
		ScopeClassification: []string{"restricted"},
	}

	impact, err := analyzer.Analyze(context.Background(), uuid.New(), policy)
	require.NoError(t, err)

	assert.Equal(t, 1, impact.TotalAssetsEvaluated)
	assert.Equal(t, 1, impact.ViolationsFound)
}
