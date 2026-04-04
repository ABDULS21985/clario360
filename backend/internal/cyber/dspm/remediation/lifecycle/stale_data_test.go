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

func timePtr(t time.Time) *time.Time { return &t }

func newTestStaleDataDetector(assets []*cybermodel.DSPMDataAsset) *StaleDataDetector {
	logger := zerolog.Nop()
	lister := &mockAssetLister{assets: assets}
	return NewStaleDataDetector(lister, logger)
}

func TestDetectFindsStaleAssets(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "stale-asset",
			DataClassification: "confidential",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -120)), // 120 days ago
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "very-stale-asset",
			DataClassification: "internal",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -200)), // 200 days ago
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "fresh-asset",
			DataClassification: "public",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -10)), // 10 days ago
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
	}

	detector := newTestStaleDataDetector(assets)

	findings, err := detector.Detect(context.Background(), tenantID)
	require.NoError(t, err)

	assert.Len(t, findings, 2, "should detect 2 stale assets")

	names := make(map[string]bool)
	for _, f := range findings {
		names[f.AssetName] = true
		assert.GreaterOrEqual(t, f.DaysStale, 90)
	}
	assert.True(t, names["stale-asset"])
	assert.True(t, names["very-stale-asset"])
}

func TestDetectNeverScannedAsset(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "never-scanned",
			DataClassification: "restricted",
			LastScannedAt:      nil, // never scanned
			CreatedAt:          time.Now().AddDate(0, 0, -200),
		},
	}

	detector := newTestStaleDataDetector(assets)

	findings, err := detector.Detect(context.Background(), tenantID)
	require.NoError(t, err)

	require.Len(t, findings, 1)
	assert.Equal(t, "never-scanned", findings[0].AssetName)
	assert.GreaterOrEqual(t, findings[0].DaysStale, 199) // based on creation date
}

func TestDetectNoStaleAssets(t *testing.T) {
	tenantID := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "recent-1",
			DataClassification: "confidential",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -5)),
			CreatedAt:          time.Now().AddDate(0, 0, -30),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "recent-2",
			DataClassification: "internal",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -30)),
			CreatedAt:          time.Now().AddDate(0, 0, -60),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "recent-3",
			DataClassification: "public",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -89)), // just under threshold
			CreatedAt:          time.Now().AddDate(0, 0, -180),
		},
	}

	detector := newTestStaleDataDetector(assets)

	findings, err := detector.Detect(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Empty(t, findings, "recently scanned assets should not be flagged")
}

func TestStaleConfidence(t *testing.T) {
	tests := []struct {
		name           string
		neverScanned   bool
		daysStale      int
		expectedConf   string
	}{
		{"never_scanned_is_high", true, 50, "high"},
		{"180_days_is_high", false, 200, "high"},
		{"90_days_is_medium", false, 95, "medium"},
		{"exactly_180_is_high", false, 180, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := staleConfidence(tt.neverScanned, tt.daysStale)
			assert.Equal(t, tt.expectedConf, conf)
		})
	}
}

func TestStaleConfidenceFromDetect(t *testing.T) {
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "never-scanned-asset",
			DataClassification: "confidential",
			LastScannedAt:      nil,
			CreatedAt:          time.Now().AddDate(0, 0, -200),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "long-stale-asset",
			DataClassification: "internal",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -200)),
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
		{
			ID:                 uuid.New(),
			AssetName:          "short-stale-asset",
			DataClassification: "public",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -95)),
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
	}

	detector := newTestStaleDataDetector(assets)

	findings, err := detector.Detect(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, findings, 3)

	confMap := make(map[string]string)
	for _, f := range findings {
		confMap[f.AssetName] = f.Confidence
	}

	assert.Equal(t, "high", confMap["never-scanned-asset"])
	assert.Equal(t, "high", confMap["long-stale-asset"])
	assert.Equal(t, "medium", confMap["short-stale-asset"])
}

func TestStaleRecommendation(t *testing.T) {
	tests := []struct {
		name           string
		classification string
		neverScanned   bool
		daysStale      int
		expectContains string
	}{
		{"public_never_scanned", "public", true, 200, "Delete unmanaged public"},
		{"public_long_stale", "public", false, 200, "Delete unmanaged public"},
		{"public_short_stale", "public", false, 95, "Re-scan public"},
		{"internal_never_scanned", "internal", true, 200, "Immediately scan and classify"},
		{"internal_long_stale", "internal", false, 200, "Archive internal"},
		{"internal_short_stale", "internal", false, 95, "Re-scan internal"},
		{"confidential_never_scanned", "confidential", true, 200, "Urgent: scan and classify"},
		{"confidential_stale", "confidential", false, 95, "Re-scan confidential"},
		{"restricted_never_scanned", "restricted", true, 200, "Critical: immediately scan"},
		{"restricted_stale", "restricted", false, 95, "Re-scan restricted"},
		{"unknown_never_scanned", "unknown", true, 200, "Scan unclassified"},
		{"unknown_stale", "unknown", false, 95, "Re-scan data asset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := staleRecommendation(tt.classification, tt.neverScanned, tt.daysStale)
			assert.Contains(t, rec, tt.expectContains)
		})
	}
}

func TestStaleRecommendationFromDetect(t *testing.T) {
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "stale-confidential",
			DataClassification: "confidential",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -100)),
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
	}

	detector := newTestStaleDataDetector(assets)

	findings, err := detector.Detect(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, findings, 1)

	assert.NotEmpty(t, findings[0].Recommendation)
	assert.Contains(t, findings[0].Recommendation, "Re-scan confidential")
}

func TestDetectEmptyAssets(t *testing.T) {
	detector := newTestStaleDataDetector(nil)

	findings, err := detector.Detect(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestDetectAssetListerError(t *testing.T) {
	logger := zerolog.Nop()
	lister := &mockAssetLister{err: assert.AnError}
	detector := NewStaleDataDetector(lister, logger)

	findings, err := detector.Detect(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, findings)
	assert.Contains(t, err.Error(), "list assets")
}

func TestDetectFindingFields(t *testing.T) {
	assetID := uuid.New()
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 assetID,
			AssetName:          "detailed-stale-asset",
			DataClassification: "restricted",
			LastScannedAt:      timePtr(time.Now().AddDate(0, 0, -100)),
			CreatedAt:          time.Now().AddDate(0, 0, -365),
		},
	}

	detector := newTestStaleDataDetector(assets)

	findings, err := detector.Detect(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, findings, 1)

	f := findings[0]
	assert.Equal(t, assetID, f.AssetID)
	assert.Equal(t, "detailed-stale-asset", f.AssetName)
	assert.Equal(t, "restricted", f.Classification)
	assert.GreaterOrEqual(t, f.DaysStale, 99)
	assert.NotEmpty(t, f.Confidence)
	assert.NotEmpty(t, f.Recommendation)
}
