package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
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

func newTestRetentionEnforcer(assets []*cybermodel.DSPMDataAsset) *RetentionEnforcer {
	logger := zerolog.Nop()
	lister := &mockAssetLister{assets: assets}
	return NewRetentionEnforcer(lister, logger)
}

func TestEvaluateFindsViolations(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "old-asset",
			DataClassification: "confidential",
			CreatedAt:          time.Now().AddDate(0, 0, -400), // 400 days old
		},
		{
			ID:                 uuid.New(),
			AssetName:          "very-old-asset",
			DataClassification: "confidential",
			CreatedAt:          time.Now().AddDate(0, 0, -600), // 600 days old
		},
		{
			ID:                 uuid.New(),
			AssetName:          "recent-asset",
			DataClassification: "confidential",
			CreatedAt:          time.Now().AddDate(0, 0, -10), // 10 days old
		},
	}

	enforcer := newTestRetentionEnforcer(assets)

	violations, err := enforcer.Evaluate(context.Background(), tenantID, 365, nil)
	require.NoError(t, err)

	assert.Len(t, violations, 2, "should find 2 violations for assets past retention period")

	// Verify the correct assets were flagged.
	names := make(map[string]bool)
	for _, v := range violations {
		names[v.AssetName] = true
		assert.Greater(t, v.DaysOverdue, 0, "days overdue should be positive")
	}
	assert.True(t, names["old-asset"])
	assert.True(t, names["very-old-asset"])
}

func TestEvaluateNoViolations(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "fresh-asset-1",
			DataClassification: "confidential",
			CreatedAt:          time.Now().AddDate(0, 0, -10),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "fresh-asset-2",
			DataClassification: "internal",
			CreatedAt:          time.Now().AddDate(0, 0, -30),
		},
	}

	enforcer := newTestRetentionEnforcer(assets)

	violations, err := enforcer.Evaluate(context.Background(), tenantID, 365, nil)
	require.NoError(t, err)
	assert.Empty(t, violations, "fresh assets should not trigger retention violations")
}

func TestSeverityEscalation(t *testing.T) {
	tests := []struct {
		name             string
		daysOld          int
		maxDays          int
		expectedSeverity string
	}{
		{
			name:             "low_severity_under_30_days_overdue",
			daysOld:          385,
			maxDays:          365,
			expectedSeverity: "low",
		},
		{
			name:             "medium_severity_30_to_89_days_overdue",
			daysOld:          395 + 30, // 60 days overdue (for safety in calculation rounding)
			maxDays:          365,
			expectedSeverity: "medium",
		},
		{
			name:             "high_severity_90_to_179_days_overdue",
			daysOld:          365 + 120,
			maxDays:          365,
			expectedSeverity: "high",
		},
		{
			name:             "critical_severity_180_plus_days_overdue",
			daysOld:          365 + 200,
			maxDays:          365,
			expectedSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assets := []*cybermodel.DSPMDataAsset{
				{
					ID:                 uuid.New(),
					AssetName:          "test-asset",
					DataClassification: "confidential",
					CreatedAt:          time.Now().AddDate(0, 0, -tt.daysOld),
				},
			}

			enforcer := newTestRetentionEnforcer(assets)

			violations, err := enforcer.Evaluate(context.Background(), uuid.New(), tt.maxDays, nil)
			require.NoError(t, err)
			require.Len(t, violations, 1)
			assert.Equal(t, tt.expectedSeverity, violations[0].Severity,
				"expected severity %q for %d days overdue, got %q",
				tt.expectedSeverity, violations[0].DaysOverdue, violations[0].Severity)
		})
	}
}

func TestRetentionDaysValidation(t *testing.T) {
	enforcer := newTestRetentionEnforcer(nil)

	tests := []struct {
		name    string
		maxDays int
	}{
		{"zero_days", 0},
		{"negative_days", -10},
		{"very_negative_days", -365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := enforcer.Evaluate(context.Background(), uuid.New(), tt.maxDays, nil)
			assert.Error(t, err)
			assert.Nil(t, violations)
			assert.Contains(t, err.Error(), "maxDays must be positive")
		})
	}
}

func TestEvaluateClassificationScope(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "confidential-old",
			DataClassification: "confidential",
			CreatedAt:          time.Now().AddDate(0, 0, -400),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "public-old",
			DataClassification: "public",
			CreatedAt:          time.Now().AddDate(0, 0, -400),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "internal-old",
			DataClassification: "internal",
			CreatedAt:          time.Now().AddDate(0, 0, -400),
		},
	}

	enforcer := newTestRetentionEnforcer(assets)

	// Only evaluate confidential assets.
	violations, err := enforcer.Evaluate(context.Background(), tenantID, 365, []string{"confidential"})
	require.NoError(t, err)

	assert.Len(t, violations, 1)
	assert.Equal(t, "confidential-old", violations[0].AssetName)
}

func TestEvaluateClassificationScopeCaseInsensitive(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "mixed-case-asset",
			DataClassification: "Confidential",
			CreatedAt:          time.Now().AddDate(0, 0, -400),
		},
	}

	enforcer := newTestRetentionEnforcer(assets)

	// Scope with lowercase should still match.
	violations, err := enforcer.Evaluate(context.Background(), tenantID, 365, []string{"confidential"})
	require.NoError(t, err)
	assert.Len(t, violations, 1)
}

func TestEvaluateAssetListerError(t *testing.T) {
	logger := zerolog.Nop()
	lister := &mockAssetLister{err: assert.AnError}
	enforcer := NewRetentionEnforcer(lister, logger)

	violations, err := enforcer.Evaluate(context.Background(), uuid.New(), 365, nil)
	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "list assets")
}

func TestEvaluateEmptyAssets(t *testing.T) {
	enforcer := newTestRetentionEnforcer(nil)

	violations, err := enforcer.Evaluate(context.Background(), uuid.New(), 365, nil)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestEvaluateViolationFields(t *testing.T) {
	assetID := uuid.New()
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 assetID,
			AssetName:          "detailed-asset",
			DataClassification: "restricted",
			CreatedAt:          time.Now().AddDate(0, 0, -400),
		},
	}

	enforcer := newTestRetentionEnforcer(assets)

	violations, err := enforcer.Evaluate(context.Background(), uuid.New(), 365, nil)
	require.NoError(t, err)
	require.Len(t, violations, 1)

	v := violations[0]
	assert.Equal(t, assetID, v.AssetID)
	assert.Equal(t, "detailed-asset", v.AssetName)
	assert.Equal(t, "restricted", v.Classification)
	assert.Greater(t, v.DaysOverdue, 0)
	assert.NotEmpty(t, v.Severity)
}
