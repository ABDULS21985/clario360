package models

import (
	"testing"
	"time"
)

func TestAlertVolumeForecasterTrainRequiresMinimumSamples(t *testing.T) {
	t.Parallel()

	model := NewAlertVolumeForecaster("")
	err := model.Train([]AlertVolumeSample{{Timestamp: time.Now().UTC(), AlertCount: 4}})
	if err == nil {
		t.Fatal("expected training error for insufficient samples")
	}
}

func TestAlertVolumeForecasterForecastShape(t *testing.T) {
	t.Parallel()

	model := NewAlertVolumeForecaster("")
	samples := make([]AlertVolumeSample, 0, 14)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for idx := 0; idx < 14; idx++ {
		samples = append(samples, AlertVolumeSample{
			Timestamp:       start.AddDate(0, 0, idx),
			AlertCount:      float64(10 + idx),
			ThreatActivity:  float64(idx % 3),
			AssetOnboarding: float64(idx % 2),
		})
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	forecast, _ := model.Forecast(7, nil)
	if len(forecast.Points) != 7 {
		t.Fatalf("forecast points = %d, want 7", len(forecast.Points))
	}
	if forecast.Points[0].Bounds.P90 < forecast.Points[0].Bounds.P50 {
		t.Fatalf("invalid confidence interval: %+v", forecast.Points[0].Bounds)
	}
}

func TestAlertVolumeForecasterSerializeRoundTrip(t *testing.T) {
	t.Parallel()

	model := NewAlertVolumeForecaster("alert-volume-vtest")
	samples := make([]AlertVolumeSample, 0, 7)
	for idx := 0; idx < 7; idx++ {
		samples = append(samples, AlertVolumeSample{
			Timestamp:  time.Date(2026, 2, idx+1, 0, 0, 0, 0, time.UTC),
			AlertCount: float64(8 + idx),
		})
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	payload, err := model.Serialize()
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	loaded := NewAlertVolumeForecaster("")
	if err := loaded.Deserialize(payload); err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if loaded.ModelVersion != model.ModelVersion {
		t.Fatalf("version = %q, want %q", loaded.ModelVersion, model.ModelVersion)
	}
}
