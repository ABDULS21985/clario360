package explainer

import (
	"fmt"
	"sort"
	"strings"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func humanizeFactors(factors []aigovmodel.Factor) string {
	if len(factors) == 0 {
		return "The output was generated from the configured transparent model logic."
	}
	sorted := append([]aigovmodel.Factor(nil), factors...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return abs(sorted[i].Impact) > abs(sorted[j].Impact)
	})
	parts := make([]string, 0, min(len(sorted), 3))
	for idx, factor := range sorted {
		if idx >= 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", factor.Name, factor.Value))
	}
	return "Primary drivers: " + strings.Join(parts, ", ") + "."
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
