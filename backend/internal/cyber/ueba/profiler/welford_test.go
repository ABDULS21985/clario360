package profiler

import (
	"math"
	"testing"
)

func TestWelford_KnownDistribution(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50}
	var mean, m2, stddev float64
	for i, value := range values {
		mean, m2, stddev = WelfordUpdate(int64(i), mean, m2, value)
	}
	if math.Abs(mean-30) > 0.001 {
		t.Fatalf("mean = %v, want 30", mean)
	}
	if math.Abs(stddev-15.8113883) > 0.001 {
		t.Fatalf("stddev = %v, want 15.8113883", stddev)
	}
}

func TestWelford_SingleValue(t *testing.T) {
	mean, _, stddev := WelfordUpdate(0, 0, 0, 42)
	if mean != 42 {
		t.Fatalf("mean = %v, want 42", mean)
	}
	if stddev != 0 {
		t.Fatalf("stddev = %v, want 0", stddev)
	}
}
