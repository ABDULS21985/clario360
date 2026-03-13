package engine

import (
	"math"
	"sort"
	"time"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type DriftDetector struct {
	mildThreshold   float64
	severeThreshold float64
}

func NewDriftDetector() *DriftDetector {
	return &DriftDetector{
		mildThreshold:   0.10,
		severeThreshold: 0.25,
	}
}

func (d *DriftDetector) AccuracyDrift(modelType predictmodel.PredictionType, baseline, recent []float64, now time.Time) *predictmodel.ModelDriftAlert {
	score := d.accuracyShiftScore(baseline, recent)
	if score < d.mildThreshold {
		return nil
	}
	severity := "warning"
	recommendation := "Monitor recent accuracy and refresh features on the next scheduled retrain."
	if score >= d.severeThreshold {
		severity = "critical"
		recommendation = "Trigger emergency retraining because predictive quality has materially shifted."
	}
	return &predictmodel.ModelDriftAlert{
		ModelType:      modelType,
		Severity:       severity,
		DriftScore:     score,
		ObservedAt:     now,
		Recommendation: recommendation,
	}
}

func (d *DriftDetector) accuracyShiftScore(baseline, recent []float64) float64 {
	if len(baseline) == 0 || len(recent) == 0 {
		return 0
	}
	if len(baseline) < 25 || len(recent) < 25 {
		return math.Abs(meanFloat(baseline) - meanFloat(recent))
	}
	return d.PSIDrift(baseline, recent, 10)
}

func (d *DriftDetector) PSIDrift(baseline, recent []float64, buckets int) float64 {
	if len(baseline) == 0 || len(recent) == 0 {
		return 0
	}
	if buckets <= 1 {
		buckets = 10
	}
	minValue := minFloat(sliceMin(baseline), sliceMin(recent))
	maxValue := maxFloat(sliceMax(baseline), sliceMax(recent))
	if minValue == maxValue {
		return 0
	}
	width := (maxValue - minValue) / float64(buckets)
	psi := 0.0
	for bucket := 0; bucket < buckets; bucket++ {
		low := minValue + float64(bucket)*width
		high := low + width
		expected := bucketShare(baseline, low, high, bucket == buckets-1)
		actual := bucketShare(recent, low, high, bucket == buckets-1)
		expected = math.Max(expected, 0.0001)
		actual = math.Max(actual, 0.0001)
		psi += (actual - expected) * math.Log(actual/expected)
	}
	return psi
}

func (d *DriftDetector) KSStatistic(baseline, recent []float64) float64 {
	if len(baseline) == 0 || len(recent) == 0 {
		return 0
	}
	base := append([]float64(nil), baseline...)
	cur := append([]float64(nil), recent...)
	sort.Float64s(base)
	sort.Float64s(cur)
	maxDiff := 0.0
	for _, value := range append([]float64(nil), base...) {
		diff := math.Abs(cdf(base, value) - cdf(cur, value))
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}

func bucketShare(values []float64, low, high float64, inclusiveHigh bool) float64 {
	if len(values) == 0 {
		return 0
	}
	count := 0.0
	for _, value := range values {
		if value < low {
			continue
		}
		if inclusiveHigh {
			if value <= high {
				count++
			}
			continue
		}
		if value < high {
			count++
		}
	}
	return count / float64(len(values))
}

func cdf(values []float64, x float64) float64 {
	if len(values) == 0 {
		return 0
	}
	idx := sort.Search(len(values), func(i int) bool { return values[i] > x })
	return float64(idx) / float64(len(values))
}

func sliceMin(values []float64) float64 {
	minValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
	}
	return minValue
}

func sliceMax(values []float64) float64 {
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}
