package kpi

import "github.com/clario360/platform/internal/visus/model"

type ThresholdEvaluator struct{}

func NewThresholdEvaluator() *ThresholdEvaluator {
	return &ThresholdEvaluator{}
}

func (t *ThresholdEvaluator) Evaluate(kpi *model.KPIDefinition, value float64) model.KPIStatus {
	if kpi == nil {
		return model.KPIStatusUnknown
	}
	if kpi.Direction == model.KPIDirectionHigherIsBetter {
		if kpi.CriticalThreshold != nil && value <= *kpi.CriticalThreshold {
			return model.KPIStatusCritical
		}
		if kpi.WarningThreshold != nil && value <= *kpi.WarningThreshold {
			return model.KPIStatusWarning
		}
		return model.KPIStatusNormal
	}
	if kpi.CriticalThreshold != nil && value >= *kpi.CriticalThreshold {
		return model.KPIStatusCritical
	}
	if kpi.WarningThreshold != nil && value >= *kpi.WarningThreshold {
		return model.KPIStatusWarning
	}
	return model.KPIStatusNormal
}
