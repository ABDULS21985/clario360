package drift

import (
	"fmt"
	"math"
	"sort"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func CalculatePSI(reference, current []float64, bins int) (float64, error) {
	if len(reference) == 0 || len(current) == 0 {
		return 0, fmt.Errorf("reference and current data are required")
	}
	if bins <= 1 {
		bins = 10
	}
	ref := append([]float64(nil), reference...)
	cur := append([]float64(nil), current...)
	sort.Float64s(ref)
	sort.Float64s(cur)
	edges := quantileEdges(ref, bins)
	refCounts := bucketCounts(ref, edges)
	curCounts := bucketCounts(cur, edges)
	total := 0.0
	for idx := range refCounts {
		refPct := float64(refCounts[idx]) / float64(len(ref))
		curPct := float64(curCounts[idx]) / float64(len(cur))
		if refPct == 0 {
			refPct = 0.001
		}
		if curPct == 0 {
			curPct = 0.001
		}
		total += (curPct - refPct) * math.Log(curPct/refPct)
	}
	return total, nil
}

func LevelForPSI(value float64) aigovmodel.DriftLevel {
	switch {
	case value > 0.25:
		return aigovmodel.DriftLevelSignificant
	case value >= 0.10:
		return aigovmodel.DriftLevelModerate
	case value > 0:
		return aigovmodel.DriftLevelLow
	default:
		return aigovmodel.DriftLevelNone
	}
}

func quantileEdges(values []float64, bins int) []float64 {
	edges := make([]float64, 0, bins+1)
	for idx := 0; idx <= bins; idx++ {
		position := float64(idx) / float64(bins)
		source := int(math.Round(position * float64(len(values)-1)))
		if source < 0 {
			source = 0
		}
		if source >= len(values) {
			source = len(values) - 1
		}
		edges = append(edges, values[source])
	}
	return edges
}

func bucketCounts(values, edges []float64) []int {
	counts := make([]int, len(edges)-1)
	for _, value := range values {
		idx := len(counts) - 1
		for edgeIdx := 0; edgeIdx < len(edges)-1; edgeIdx++ {
			upper := edges[edgeIdx+1]
			if edgeIdx == len(edges)-2 {
				if value >= edges[edgeIdx] && value <= upper {
					idx = edgeIdx
					break
				}
				continue
			}
			if value >= edges[edgeIdx] && value < upper {
				idx = edgeIdx
				break
			}
		}
		counts[idx]++
	}
	return counts
}
