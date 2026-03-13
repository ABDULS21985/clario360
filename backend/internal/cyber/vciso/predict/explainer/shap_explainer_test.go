package explainer

import "testing"

func TestSHAPExplainerSortsByAbsoluteContribution(t *testing.T) {
	t.Parallel()

	explainer := NewSHAPExplainer()
	items := explainer.FromWeights(
		map[string]float64{"a": 10, "b": 2, "c": 1},
		map[string]float64{"a": 0, "b": 0, "c": 0},
		map[string]float64{"a": 0.1, "b": 1.0, "c": -3.0},
		map[string]any{},
	)
	if items[0].Feature != "c" {
		t.Fatalf("top feature = %q, want c", items[0].Feature)
	}
}

func TestConfidenceCalibratorProducesOrderedIntervals(t *testing.T) {
	t.Parallel()

	calibrator := NewConfidenceCalibrator()
	_, interval := calibrator.CalibrateProbability(0.7, []float64{0.1, -0.05, 0.08})
	if interval.P10 > interval.P50 || interval.P50 > interval.P90 {
		t.Fatalf("invalid interval ordering: %+v", interval)
	}
}
