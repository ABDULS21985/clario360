package engine

import "math"

func clamp(value, floor, ceiling float64) float64 {
	if value < floor {
		return floor
	}
	if value > ceiling {
		return ceiling
	}
	return value
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	clone := append([]float64(nil), values...)
	sortFloat64s(clone)
	if p <= 0 {
		return clone[0]
	}
	if p >= 1 {
		return clone[len(clone)-1]
	}
	pos := p * float64(len(clone)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return clone[lower]
	}
	weight := pos - float64(lower)
	return clone[lower] + (clone[upper]-clone[lower])*weight
}

func absResiduals(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, value := range values {
		out = append(out, math.Abs(value))
	}
	return out
}

func sortFloat64s(values []float64) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
