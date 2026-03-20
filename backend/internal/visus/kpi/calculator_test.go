package kpi

import (
	"testing"
	"time"

	"github.com/clario360/platform/internal/visus/model"
)

func TestCalc_Direct(t *testing.T) {
	calculator := NewCalculator()
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcDirect}
	if got := calculator.Calculate(kpi, 42, nil); got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
}

func TestCalc_Delta(t *testing.T) {
	calculator := NewCalculator()
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcDelta}
	history := []model.KPISnapshot{{Value: 45}}
	if got := calculator.Calculate(kpi, 50, history); got != 5 {
		t.Fatalf("expected 5, got %v", got)
	}
}

func TestCalc_Delta_NoHistory(t *testing.T) {
	calculator := NewCalculator()
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcDelta}
	if got := calculator.Calculate(kpi, 50, nil); got != 50 {
		t.Fatalf("expected 50, got %v", got)
	}
}

func TestCalc_PercentageChange(t *testing.T) {
	calculator := NewCalculator()
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcPercentageChange}
	history := []model.KPISnapshot{{Value: 50}}
	if got := calculator.Calculate(kpi, 55, history); got != 10 {
		t.Fatalf("expected 10, got %v", got)
	}
}

func TestCalc_PercentageChange_ZeroPrevious(t *testing.T) {
	calculator := NewCalculator()
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcPercentageChange}
	history := []model.KPISnapshot{{Value: 0}}
	if got := calculator.Calculate(kpi, 55, history); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestCalc_AverageOverPeriod(t *testing.T) {
	calculator := NewCalculator()
	window := "7d"
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcAverageOverPeriod, CalculationWindow: &window}
	history := snapshotsWithValues([]float64{50, 50, 50, 50, 50, 50, 50})
	if got := calculator.Calculate(kpi, 99, history); got != 50 {
		t.Fatalf("expected 50, got %v", got)
	}
}

func TestCalc_SumOverPeriod(t *testing.T) {
	calculator := NewCalculator()
	window := "7d"
	kpi := &model.KPIDefinition{CalculationType: model.KPICalcSumOverPeriod, CalculationWindow: &window}
	history := snapshotsWithValues([]float64{50, 50, 50, 50, 50, 50, 50})
	if got := calculator.Calculate(kpi, 99, history); got != 350 {
		t.Fatalf("expected 350, got %v", got)
	}
}

func snapshotsWithValues(values []float64) []model.KPISnapshot {
	now := time.Now().UTC()
	out := make([]model.KPISnapshot, 0, len(values))
	for idx, value := range values {
		out = append(out, model.KPISnapshot{
			Value:     value,
			CreatedAt: now.Add(-time.Duration(idx) * 24 * time.Hour),
		})
	}
	return out
}
