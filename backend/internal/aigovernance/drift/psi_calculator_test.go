package drift

import (
	"testing"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestPSIIdenticalDistributions(t *testing.T) {
	reference := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	current := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	psi, err := CalculatePSI(reference, current, 5)
	if err != nil {
		t.Fatalf("CalculatePSI() error = %v", err)
	}
	if psi != 0 {
		t.Fatalf("psi = %v, want 0", psi)
	}
	if level := LevelForPSI(psi); level != aigovmodel.DriftLevelNone {
		t.Fatalf("level = %s, want none", level)
	}
}

func TestPSISlightShift(t *testing.T) {
	reference := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	current := []float64{10, 11, 12, 13, 14, 15, 16, 18, 19, 20}
	psi, err := CalculatePSI(reference, current, 5)
	if err != nil {
		t.Fatalf("CalculatePSI() error = %v", err)
	}
	if psi >= 0.10 {
		t.Fatalf("psi = %v, want < 0.10", psi)
	}
}

func TestPSIMajorShift(t *testing.T) {
	reference := []float64{1, 1, 2, 2, 3, 3, 4, 4, 5, 5}
	current := []float64{20, 20, 21, 21, 22, 22, 23, 23, 24, 24}
	psi, err := CalculatePSI(reference, current, 5)
	if err != nil {
		t.Fatalf("CalculatePSI() error = %v", err)
	}
	if psi <= 0.25 {
		t.Fatalf("psi = %v, want > 0.25", psi)
	}
	if level := LevelForPSI(psi); level != aigovmodel.DriftLevelSignificant {
		t.Fatalf("level = %s, want significant", level)
	}
}

func TestPSIEmptyData(t *testing.T) {
	if _, err := CalculatePSI(nil, []float64{1, 2, 3}, 5); err == nil {
		t.Fatal("expected error for empty reference")
	}
}

func TestPSIZeroBinSmoothing(t *testing.T) {
	reference := []float64{1, 1, 1, 1, 1, 2, 2, 2, 2, 2}
	current := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	psi, err := CalculatePSI(reference, current, 5)
	if err != nil {
		t.Fatalf("CalculatePSI() error = %v", err)
	}
	if psi < 0 {
		t.Fatalf("psi = %v, want non-negative", psi)
	}
}
