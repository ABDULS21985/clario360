package models

import (
	"testing"
	"time"
)

func TestTechniqueTrendAnalyzerDetectsIncreasingTrend(t *testing.T) {
	t.Parallel()

	model := NewTechniqueTrendAnalyzer("")
	samples := make([]TechniqueTrendSample, 0, 8)
	for idx := 0; idx < 8; idx++ {
		samples = append(samples, TechniqueTrendSample{
			TechniqueID:   "T1566",
			TechniqueName: "Phishing",
			Timestamp:     time.Date(2026, 1, idx+1, 0, 0, 0, 0, time.UTC),
			InternalCount: float64(2 + idx),
			IndustryCount: float64(1 + idx/2),
		})
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	items := model.Predict(30)
	if len(items) == 0 {
		t.Fatal("expected technique predictions")
	}
	if items[0].Trend != "increasing" {
		t.Fatalf("trend = %q, want increasing", items[0].Trend)
	}
}

func TestTechniqueTrendAnalyzerSerializeRoundTrip(t *testing.T) {
	t.Parallel()

	model := NewTechniqueTrendAnalyzer("technique-vtest")
	samples := []TechniqueTrendSample{
		{TechniqueID: "T1", Timestamp: time.Now().UTC(), InternalCount: 2, IndustryCount: 1},
		{TechniqueID: "T1", Timestamp: time.Now().UTC().AddDate(0, 0, 1), InternalCount: 3, IndustryCount: 2},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	payload, err := model.Serialize()
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	loaded := NewTechniqueTrendAnalyzer("")
	if err := loaded.Deserialize(payload); err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if loaded.ModelVersion != "technique-vtest" {
		t.Fatalf("version = %q", loaded.ModelVersion)
	}
}
