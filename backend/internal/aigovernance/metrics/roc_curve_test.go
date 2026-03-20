package metrics

import (
	"math"
	"testing"
)

func TestROC_Shape(t *testing.T) {
	points, _ := BuildROCCurve([]ScoredSample{
		{Score: 0.95, ActualPositive: true},
		{Score: 0.90, ActualPositive: true},
		{Score: 0.80, ActualPositive: false},
		{Score: 0.30, ActualPositive: false},
	}, DefaultThresholds())

	for idx := 0; idx < len(points)-1; idx++ {
		if points[idx+1].FPR < points[idx].FPR {
			t.Fatalf("fpr decreased at %d: %.3f -> %.3f", idx, points[idx].FPR, points[idx+1].FPR)
		}
		if points[idx+1].TPR < points[idx].TPR {
			t.Fatalf("tpr decreased at %d: %.3f -> %.3f", idx, points[idx].TPR, points[idx+1].TPR)
		}
	}
}

func TestAUC_Perfect(t *testing.T) {
	_, auc := BuildROCCurve([]ScoredSample{
		{Score: 0.99, ActualPositive: true},
		{Score: 0.92, ActualPositive: true},
		{Score: 0.15, ActualPositive: false},
		{Score: 0.05, ActualPositive: false},
	}, DefaultThresholds())
	if math.Abs(auc-1.0) > 0.0001 {
		t.Fatalf("auc = %.4f, want 1.0", auc)
	}
}

func TestAUC_Random(t *testing.T) {
	_, auc := BuildROCCurve([]ScoredSample{
		{Score: 0.90, ActualPositive: true},
		{Score: 0.10, ActualPositive: true},
		{Score: 0.80, ActualPositive: false},
		{Score: 0.20, ActualPositive: false},
	}, DefaultThresholds())
	if math.Abs(auc-0.5) > 0.05 {
		t.Fatalf("auc = %.4f, want about 0.5", auc)
	}
}

func TestAUC_Trapezoidal(t *testing.T) {
	points, auc := BuildROCCurve([]ScoredSample{
		{Score: 0.90, ActualPositive: true},
		{Score: 0.60, ActualPositive: true},
		{Score: 0.80, ActualPositive: false},
		{Score: 0.30, ActualPositive: false},
	}, []float64{1.0, 0.9, 0.8, 0.6, 0.3, 0.0})

	if len(points) < 2 {
		t.Fatal("expected roc points")
	}
	if math.Abs(auc-0.75) > 0.001 {
		t.Fatalf("auc = %.4f, want 0.7500", auc)
	}
}
