package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestAssetRiskPredictorRanksHighRiskAboveLowRisk(t *testing.T) {
	t.Parallel()

	model := NewAssetRiskPredictor("")
	high := AssetRiskSample{
		AssetID:              uuid.New(),
		AssetName:            "prod-web-1",
		CriticalityScore:     1,
		OpenCritical:         4,
		OpenHigh:             6,
		PatchAgeDays:         45,
		InternetFacing:       1,
		HistoricalAlerts:     10,
		UserAccessCount:      20,
		DataSensitivity:      1,
		IndustrySignal:       0.9,
		TechniqueCoverageGap: 0.8,
		TargetedLabel:        1,
	}
	low := AssetRiskSample{
		AssetID:              uuid.New(),
		AssetName:            "dev-box-1",
		CriticalityScore:     0.25,
		OpenCritical:         0,
		OpenHigh:             0,
		PatchAgeDays:         3,
		InternetFacing:       0,
		HistoricalAlerts:     0,
		UserAccessCount:      1,
		DataSensitivity:      0.25,
		IndustrySignal:       0.1,
		TechniqueCoverageGap: 0.1,
		TargetedLabel:        0,
	}
	samples := []AssetRiskSample{high, low, high, low, high, low}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	if model.Predict(high) <= model.Predict(low) {
		t.Fatalf("expected high risk score > low risk score")
	}
}

func TestAssetRiskPredictorSerializeRoundTrip(t *testing.T) {
	t.Parallel()

	model := NewAssetRiskPredictor("asset-risk-vtest")
	samples := []AssetRiskSample{
		{AssetID: uuid.New(), AssetName: "a", CriticalityScore: 1, OpenCritical: 3, InternetFacing: 1, TargetedLabel: 1},
		{AssetID: uuid.New(), AssetName: "b", CriticalityScore: 0.25, OpenCritical: 0, InternetFacing: 0, TargetedLabel: 0},
		{AssetID: uuid.New(), AssetName: "c", CriticalityScore: 0.75, OpenCritical: 1, InternetFacing: 1, TargetedLabel: 1},
		{AssetID: uuid.New(), AssetName: "d", CriticalityScore: 0.50, OpenCritical: 0, InternetFacing: 0, TargetedLabel: 0},
		{AssetID: uuid.New(), AssetName: "e", CriticalityScore: 1, OpenCritical: 2, InternetFacing: 1, TargetedLabel: 1},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	payload, err := model.Serialize()
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	loaded := NewAssetRiskPredictor("")
	if err := loaded.Deserialize(payload); err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if loaded.ModelVersion != "asset-risk-vtest" {
		t.Fatalf("version = %q", loaded.ModelVersion)
	}
}
