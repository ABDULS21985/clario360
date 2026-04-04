package profiler

import "testing"

func TestEMA_BasicUpdate(t *testing.T) {
	if got := EMA(100, 200, 0.05); got != 105 {
		t.Fatalf("EMA() = %v, want 105", got)
	}
}

func TestEMA_Convergence(t *testing.T) {
	value := 100.0
	for i := 0; i < 100; i++ {
		value = EMA(value, 200, 0.05)
	}
	if value < 199 {
		t.Fatalf("EMA did not converge sufficiently: %v", value)
	}
}

func TestEMA_HalfLife(t *testing.T) {
	weight := 1.0
	for i := 0; i < 14; i++ {
		weight *= 1 - 0.05
	}
	if weight >= 0.5 {
		t.Fatalf("expected old observation weight < 0.5, got %v", weight)
	}
}
