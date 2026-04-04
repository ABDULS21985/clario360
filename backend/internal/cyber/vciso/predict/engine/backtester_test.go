package engine

import "testing"

func TestBacktesterClassification(t *testing.T) {
	t.Parallel()

	backtester := NewBacktester()
	metrics, err := backtester.Classification([]float64{0.9, 0.8, 0.1, 0.2}, []float64{1, 1, 0, 0}, 0.5)
	if err != nil {
		t.Fatalf("classification error: %v", err)
	}
	if metrics.Accuracy < 0.9 {
		t.Fatalf("accuracy = %.2f, want >= 0.9", metrics.Accuracy)
	}
}

func TestBacktesterRegression(t *testing.T) {
	t.Parallel()

	backtester := NewBacktester()
	metrics, err := backtester.Regression([]float64{10, 12, 14}, []float64{11, 12, 15})
	if err != nil {
		t.Fatalf("regression error: %v", err)
	}
	if metrics.MAPE <= 0 {
		t.Fatalf("expected positive MAPE")
	}
}

func TestBacktesterClusterQuality(t *testing.T) {
	t.Parallel()

	backtester := NewBacktester()
	metrics, err := backtester.ClusterQuality([]float64{0.9, 0.8, 0.85})
	if err != nil {
		t.Fatalf("cluster quality error: %v", err)
	}
	if metrics.Accuracy < 0.8 {
		t.Fatalf("accuracy = %.2f, want >= 0.8", metrics.Accuracy)
	}
}
