package kpi

import (
	"testing"

	"github.com/clario360/platform/internal/visus/model"
)

func TestThreshold_LowerIsBetter_Normal(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:         model.KPIDirectionLowerIsBetter,
		WarningThreshold:  floatPtr(60),
		CriticalThreshold: floatPtr(80),
	}
	if got := evaluator.Evaluate(kpi, 50); got != model.KPIStatusNormal {
		t.Fatalf("expected normal, got %s", got)
	}
}

func TestThreshold_LowerIsBetter_Warning(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:         model.KPIDirectionLowerIsBetter,
		WarningThreshold:  floatPtr(60),
		CriticalThreshold: floatPtr(80),
	}
	if got := evaluator.Evaluate(kpi, 65); got != model.KPIStatusWarning {
		t.Fatalf("expected warning, got %s", got)
	}
}

func TestThreshold_LowerIsBetter_Critical(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:         model.KPIDirectionLowerIsBetter,
		WarningThreshold:  floatPtr(60),
		CriticalThreshold: floatPtr(80),
	}
	if got := evaluator.Evaluate(kpi, 85); got != model.KPIStatusCritical {
		t.Fatalf("expected critical, got %s", got)
	}
}

func TestThreshold_HigherIsBetter_Normal(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:         model.KPIDirectionHigherIsBetter,
		WarningThreshold:  floatPtr(90),
		CriticalThreshold: floatPtr(80),
	}
	if got := evaluator.Evaluate(kpi, 95); got != model.KPIStatusNormal {
		t.Fatalf("expected normal, got %s", got)
	}
}

func TestThreshold_HigherIsBetter_Warning(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:         model.KPIDirectionHigherIsBetter,
		WarningThreshold:  floatPtr(90),
		CriticalThreshold: floatPtr(80),
	}
	if got := evaluator.Evaluate(kpi, 88); got != model.KPIStatusWarning {
		t.Fatalf("expected warning, got %s", got)
	}
}

func TestThreshold_HigherIsBetter_Critical(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:         model.KPIDirectionHigherIsBetter,
		WarningThreshold:  floatPtr(90),
		CriticalThreshold: floatPtr(80),
	}
	if got := evaluator.Evaluate(kpi, 75); got != model.KPIStatusCritical {
		t.Fatalf("expected critical, got %s", got)
	}
}

func TestThreshold_NoThresholds(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{Direction: model.KPIDirectionLowerIsBetter}
	if got := evaluator.Evaluate(kpi, 999); got != model.KPIStatusNormal {
		t.Fatalf("expected normal, got %s", got)
	}
}

func TestThreshold_OnlyWarning(t *testing.T) {
	evaluator := NewThresholdEvaluator()
	kpi := &model.KPIDefinition{
		Direction:        model.KPIDirectionLowerIsBetter,
		WarningThreshold: floatPtr(60),
	}
	if got := evaluator.Evaluate(kpi, 65); got != model.KPIStatusWarning {
		t.Fatalf("expected warning, got %s", got)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
