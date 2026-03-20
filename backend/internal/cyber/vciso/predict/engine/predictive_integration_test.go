package engine

import (
	"testing"
	"time"

	predictexplainer "github.com/clario360/platform/internal/cyber/vciso/predict/explainer"
	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
	predictmodels "github.com/clario360/platform/internal/cyber/vciso/predict/models"
)

func TestPredictivePipelineAlertVolumeForecast(t *testing.T) {
	t.Parallel()

	model := predictmodels.NewAlertVolumeForecaster("")
	samples := make([]predictmodels.AlertVolumeSample, 0, 14)
	for idx := 0; idx < 14; idx++ {
		samples = append(samples, predictmodels.AlertVolumeSample{
			Timestamp:      time.Date(2026, 1, idx+1, 0, 0, 0, 0, time.UTC),
			AlertCount:     float64(20 + idx),
			ThreatActivity: float64(idx % 4),
		})
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	forecast, featureTotals := model.Forecast(7, nil)
	narrator := predictexplainer.NewPredictionNarrator()
	text, steps := narrator.Explain(predictmodel.PredictionTypeAlertVolumeForecast, 0.8, predictmodel.ConfidenceInterval{P10: 10, P50: 12, P90: 14}, mapContributions(featureTotals), "")
	if len(forecast.Points) != 7 || text == "" || len(steps) == 0 {
		t.Fatalf("pipeline output incomplete")
	}
}

func TestPredictivePipelineAssetRiskRanking(t *testing.T) {
	t.Parallel()

	model := predictmodels.NewAssetRiskPredictor("")
	samples := []predictmodels.AssetRiskSample{
		{AssetName: "high", CriticalityScore: 1, OpenCritical: 4, InternetFacing: 1, TargetedLabel: 1},
		{AssetName: "low", CriticalityScore: 0.25, OpenCritical: 0, InternetFacing: 0, TargetedLabel: 0},
		{AssetName: "mid", CriticalityScore: 0.5, OpenCritical: 1, InternetFacing: 0, TargetedLabel: 0},
		{AssetName: "high-2", CriticalityScore: 1, OpenCritical: 5, InternetFacing: 1, TargetedLabel: 1},
		{AssetName: "low-2", CriticalityScore: 0.25, OpenCritical: 0, InternetFacing: 0, TargetedLabel: 0},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	shap := predictexplainer.NewSHAPExplainer()
	top := shap.TopN(shap.FromWeights(assetRiskValues(samples[0]), model.Baseline, model.Weights, assetRiskRaw(samples[0])), 5)
	if len(top) == 0 || model.Predict(samples[0]) <= model.Predict(samples[1]) {
		t.Fatalf("asset risk pipeline did not rank correctly")
	}
}

func TestPredictivePipelineVulnerabilityPriority(t *testing.T) {
	t.Parallel()

	model := predictmodels.NewVulnerabilityExploitPredictor("")
	samples := []predictmodels.VulnerabilitySample{
		{CVEID: "CVE-1", CVSS: 9.8, EPSS: 0.9, KEV: 1, ExploitedLabel: 1},
		{CVEID: "CVE-2", CVSS: 4.0, EPSS: 0.1, KEV: 0, ExploitedLabel: 0},
		{CVEID: "CVE-3", CVSS: 8.1, EPSS: 0.7, KEV: 1, ExploitedLabel: 1},
		{CVEID: "CVE-4", CVSS: 5.0, EPSS: 0.2, KEV: 0, ExploitedLabel: 0},
		{CVEID: "CVE-5", CVSS: 9.0, EPSS: 0.8, KEV: 1, ExploitedLabel: 1},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	if model.Predict(samples[0]) <= model.Predict(samples[1]) {
		t.Fatalf("vulnerability pipeline did not rank correctly")
	}
}

func TestPredictivePipelineTechniqueTrend(t *testing.T) {
	t.Parallel()

	model := predictmodels.NewTechniqueTrendAnalyzer("")
	samples := []predictmodels.TechniqueTrendSample{
		{TechniqueID: "T1566", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), InternalCount: 1, IndustryCount: 2},
		{TechniqueID: "T1566", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), InternalCount: 3, IndustryCount: 3},
		{TechniqueID: "T1566", Timestamp: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), InternalCount: 5, IndustryCount: 4},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	items := model.Predict(30)
	if len(items) == 0 || items[0].Trend != "increasing" {
		t.Fatalf("technique pipeline trend mismatch")
	}
}

func TestPredictivePipelineInsiderTrajectory(t *testing.T) {
	t.Parallel()

	model := predictmodels.NewInsiderThreatTrajectoryModel("")
	series := []predictmodels.InsiderThreatSample{
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RiskScore: 45},
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), RiskScore: 55, LoginAnomalies: 1},
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), RiskScore: 68, LoginAnomalies: 2, PolicyViolations: 1},
	}
	if err := model.Train(map[string][]predictmodels.InsiderThreatSample{"u1": series}); err != nil {
		t.Fatalf("train error: %v", err)
	}
	projected, _, _ := model.Predict(series, 7, 80)
	if projected <= series[len(series)-1].RiskScore {
		t.Fatalf("insider pipeline projected %.2f <= current %.2f", projected, series[len(series)-1].RiskScore)
	}
}

func TestPredictivePipelineCampaignDetection(t *testing.T) {
	t.Parallel()

	model := predictmodels.NewCampaignDetector("")
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	samples := []predictmodels.CampaignAlertSample{
		{Title: "a", Timestamp: base, Embedding: []float64{1, 0}, IOCs: []string{"ioc"}, Techniques: []string{"T1"}, TargetAssets: []string{"asset-a"}},
		{Title: "b", Timestamp: base.Add(time.Hour), Embedding: []float64{0.95, 0.05}, IOCs: []string{"ioc"}, Techniques: []string{"T1"}, TargetAssets: []string{"asset-a"}},
		{Title: "c", Timestamp: base.Add(2 * time.Hour), Embedding: []float64{0.9, 0.1}, IOCs: []string{"ioc"}, Techniques: []string{"T1"}, TargetAssets: []string{"asset-b"}},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	clusters := model.Detect(samples)
	if len(clusters) == 0 {
		t.Fatal("expected campaign clusters")
	}
}
