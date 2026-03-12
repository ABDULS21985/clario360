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

func TestWelford_LargeN(t *testing.T) {
	// Feed 100,000 values drawn from a known uniform distribution [0, 100).
	// Expected mean ≈ 49.5, expected stddev ≈ 28.87 (100/√12).
	// Use a deterministic sequence so the test is reproducible.
	n := 100000
	var mean, m2, stddev float64
	// Simple LCG for deterministic pseudo-random values in [0, 100).
	seed := uint64(42)
	for i := 0; i < n; i++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		value := float64(seed>>33) / float64(1<<31) * 100
		mean, m2, stddev = WelfordUpdate(int64(i), mean, m2, value)
	}
	// With 100k samples the empirical mean and stddev should be within 1% of theoretical.
	expectedMean := 50.0
	expectedStddev := 100.0 / math.Sqrt(12)

	meanErr := math.Abs(mean-expectedMean) / expectedMean
	if meanErr > 0.01 {
		t.Fatalf("mean = %v, want ≈ %v (error %.4f%%)", mean, expectedMean, meanErr*100)
	}
	stddevErr := math.Abs(stddev-expectedStddev) / expectedStddev
	if stddevErr > 0.01 {
		t.Fatalf("stddev = %v, want ≈ %v (error %.4f%%)", stddev, expectedStddev, stddevErr*100)
	}
	_ = m2 // m2 is an internal accumulator, correctness validated via stddev
}
