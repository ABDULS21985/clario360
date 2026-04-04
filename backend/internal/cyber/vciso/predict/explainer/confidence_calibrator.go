package explainer

import (
	"math"
	"sort"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type ConfidenceCalibrator struct {
	minWidth float64
}

func NewConfidenceCalibrator() *ConfidenceCalibrator {
	return &ConfidenceCalibrator{minWidth: 0.05}
}

func (c *ConfidenceCalibrator) CalibrateProbability(probability float64, residuals []float64) (float64, predictmodel.ConfidenceInterval) {
	probability = clamp(probability, 0.01, 0.99)
	spread := c.estimateSpread(residuals, 0.10)
	return probability, predictmodel.ConfidenceInterval{
		P10: clamp(probability-spread, 0, 1),
		P50: probability,
		P90: clamp(probability+spread, 0, 1),
	}
}

func (c *ConfidenceCalibrator) CalibrateValue(value float64, residuals []float64) predictmodel.ConfidenceInterval {
	spread := c.estimateSpread(residuals, math.Max(math.Abs(value)*0.15, 1))
	return predictmodel.ConfidenceInterval{
		P10: value - spread,
		P50: value,
		P90: value + spread,
	}
}

func (c *ConfidenceCalibrator) estimateSpread(residuals []float64, fallback float64) float64 {
	if len(residuals) == 0 {
		return math.Max(fallback, c.minWidth)
	}
	values := make([]float64, 0, len(residuals))
	for _, item := range residuals {
		values = append(values, math.Abs(item))
	}
	sort.Float64s(values)
	idx := int(math.Round(float64(len(values)-1) * 0.90))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return math.Max(values[idx], c.minWidth)
}

func clamp(value, floor, ceiling float64) float64 {
	if value < floor {
		return floor
	}
	if value > ceiling {
		return ceiling
	}
	return value
}
