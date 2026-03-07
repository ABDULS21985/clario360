package kpi

import (
	"strings"
	"time"

	"github.com/clario360/platform/internal/visus/model"
)

type KPICalculator struct{}

func NewCalculator() *KPICalculator {
	return &KPICalculator{}
}

func (c *KPICalculator) Calculate(kpi *model.KPIDefinition, rawValue float64, history []model.KPISnapshot) float64 {
	if kpi == nil {
		return rawValue
	}
	switch kpi.CalculationType {
	case model.KPICalcDelta:
		if len(history) == 0 {
			return rawValue
		}
		return rawValue - history[0].Value
	case model.KPICalcPercentageChange:
		if len(history) == 0 || history[0].Value == 0 {
			return 0
		}
		return ((rawValue - history[0].Value) / history[0].Value) * 100
	case model.KPICalcAverageOverPeriod:
		values := valuesInWindow(history, kpi.CalculationWindow)
		if len(values) == 0 {
			return rawValue
		}
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	case model.KPICalcSumOverPeriod:
		values := valuesInWindow(history, kpi.CalculationWindow)
		if len(values) == 0 {
			return rawValue
		}
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total
	default:
		return rawValue
	}
}

func valuesInWindow(history []model.KPISnapshot, window *string) []float64 {
	if len(history) == 0 {
		return nil
	}
	duration := 0 * time.Second
	if window != nil {
		duration = parseWindow(*window)
	}
	if duration <= 0 {
		values := make([]float64, 0, len(history))
		for _, snapshot := range history {
			values = append(values, snapshot.Value)
		}
		return values
	}
	cutoff := time.Now().UTC().Add(-duration)
	values := make([]float64, 0, len(history))
	for _, snapshot := range history {
		if snapshot.CreatedAt.Before(cutoff) {
			continue
		}
		values = append(values, snapshot.Value)
	}
	return values
}

func parseWindow(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if strings.HasSuffix(raw, "d") {
		if days, err := time.ParseDuration(strings.TrimSuffix(raw, "d") + "24h"); err == nil {
			return days
		}
	}
	duration, _ := time.ParseDuration(raw)
	return duration
}
