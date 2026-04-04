package metrics

import (
	"sort"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type ScoredSample struct {
	Score          float64
	ActualPositive bool
}

func DefaultThresholds() []float64 {
	out := make([]float64, 0, 19)
	for threshold := 0.05; threshold < 1.0; threshold += 0.05 {
		out = append(out, roundThreshold(threshold))
	}
	return out
}

func BuildROCCurve(samples []ScoredSample, thresholds []float64) ([]aigovmodel.ROCPoint, float64) {
	normalized := normalizeThresholds(thresholds)
	if len(normalized) == 0 {
		normalized = []float64{1, 0}
	}
	points := make([]aigovmodel.ROCPoint, 0, len(normalized))
	for _, threshold := range normalized {
		matrix := thresholdMatrix(samples, threshold)
		points = append(points, aigovmodel.ROCPoint{
			Threshold: threshold,
			FPR:       FalsePositiveRate(matrix.FP, matrix.TN),
			TPR:       Recall(matrix.TP, matrix.FN),
		})
	}
	return points, AUC(points)
}

func AUC(points []aigovmodel.ROCPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	ordered := make([]aigovmodel.ROCPoint, len(points))
	copy(ordered, points)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].FPR == ordered[j].FPR {
			return ordered[i].TPR < ordered[j].TPR
		}
		return ordered[i].FPR < ordered[j].FPR
	})
	area := 0.0
	for idx := 0; idx < len(ordered)-1; idx++ {
		width := ordered[idx+1].FPR - ordered[idx].FPR
		height := (ordered[idx].TPR + ordered[idx+1].TPR) / 2
		area += width * height
	}
	switch {
	case area < 0:
		return 0
	case area > 1:
		return 1
	default:
		return area
	}
}

func normalizeThresholds(thresholds []float64) []float64 {
	set := map[float64]struct{}{
		1.0: {},
		0.0: {},
	}
	for _, threshold := range thresholds {
		if threshold < 0 {
			threshold = 0
		}
		if threshold > 1 {
			threshold = 1
		}
		set[roundThreshold(threshold)] = struct{}{}
	}
	out := make([]float64, 0, len(set))
	for threshold := range set {
		out = append(out, threshold)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] > out[j]
	})
	return out
}

func thresholdMatrix(samples []ScoredSample, threshold float64) ConfusionMatrix {
	binary := make([]BinarySample, 0, len(samples))
	for _, sample := range samples {
		binary = append(binary, BinarySample{
			PredictedPositive: sample.Score >= threshold,
			ActualPositive:    sample.ActualPositive,
		})
	}
	return CalculateConfusionMatrix(binary)
}

func roundThreshold(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}
