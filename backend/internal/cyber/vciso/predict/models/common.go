package models

import (
	"math"
	"sort"
)

func logistic(value float64) float64 {
	return 1 / (1 + math.Exp(-value))
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
	sort.Float64s(clone)
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

func slope(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	xMean := float64(len(values)-1) / 2
	yMean := mean(values)
	num := 0.0
	den := 0.0
	for idx, value := range values {
		x := float64(idx)
		num += (x - xMean) * (value - yMean)
		den += (x - xMean) * (x - xMean)
	}
	if den == 0 {
		return 0
	}
	return num / den
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	dot := 0.0
	normA := 0.0
	normB := 0.0
	for idx := range a {
		dot += a[idx] * b[idx]
		normA += a[idx] * a[idx]
		normB += b[idx] * b[idx]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func rollingEMA(values []float64, alpha float64) float64 {
	if len(values) == 0 {
		return 0
	}
	alpha = clamp(alpha, 0.05, 0.95)
	value := values[0]
	for idx := 1; idx < len(values); idx++ {
		value = alpha*values[idx] + (1-alpha)*value
	}
	return value
}
