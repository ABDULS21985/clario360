package metrics

import (
	"math"
	"testing"
)

func TestPrecision(t *testing.T) {
	got := Precision(80, 20)
	if math.Abs(got-0.8) > 0.0001 {
		t.Fatalf("precision = %.3f, want 0.800", got)
	}
}

func TestRecall(t *testing.T) {
	got := Recall(80, 10)
	if math.Abs(got-0.8888888889) > 0.001 {
		t.Fatalf("recall = %.3f, want 0.889", got)
	}
}

func TestF1(t *testing.T) {
	got := F1(0.8, 0.8888888889)
	if math.Abs(got-0.8421052632) > 0.001 {
		t.Fatalf("f1 = %.3f, want 0.842", got)
	}
}

func TestFPR(t *testing.T) {
	got := FalsePositiveRate(20, 100)
	if math.Abs(got-0.1666666667) > 0.001 {
		t.Fatalf("fpr = %.3f, want 0.167", got)
	}
}

func TestDivisionByZero(t *testing.T) {
	if got := Precision(0, 0); got != 0 {
		t.Fatalf("precision = %.3f, want 0", got)
	}
	if got := Recall(0, 0); got != 0 {
		t.Fatalf("recall = %.3f, want 0", got)
	}
	if got := F1(0, 0); got != 0 {
		t.Fatalf("f1 = %.3f, want 0", got)
	}
}
