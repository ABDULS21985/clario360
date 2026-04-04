package engine

import (
	"testing"
	"time"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

func TestDriftDetectorNoDrift(t *testing.T) {
	t.Parallel()

	detector := NewDriftDetector()
	alert := detector.AccuracyDrift(predictmodel.PredictionTypeAssetRisk, []float64{0.8, 0.81, 0.79}, []float64{0.8, 0.82, 0.78}, time.Now().UTC())
	if alert != nil {
		t.Fatalf("unexpected drift alert: %+v", alert)
	}
}

func TestDriftDetectorMildDrift(t *testing.T) {
	t.Parallel()

	detector := NewDriftDetector()
	alert := detector.AccuracyDrift(predictmodel.PredictionTypeAssetRisk, []float64{0.9, 0.91, 0.92}, []float64{0.7, 0.69, 0.68}, time.Now().UTC())
	if alert == nil {
		t.Fatal("expected drift alert")
	}
}

func TestDriftDetectorSevereDrift(t *testing.T) {
	t.Parallel()

	detector := NewDriftDetector()
	alert := detector.AccuracyDrift(predictmodel.PredictionTypeVulnerabilityExploit, []float64{0.95, 0.96, 0.94}, []float64{0.4, 0.35, 0.3}, time.Now().UTC())
	if alert == nil || alert.Severity != "critical" {
		t.Fatalf("expected critical drift alert, got %+v", alert)
	}
}
