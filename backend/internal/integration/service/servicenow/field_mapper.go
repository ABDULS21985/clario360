package servicenow

import (
	"strings"
)

func MapUrgency(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 1
	case "high", "medium":
		return 2
	default:
		return 3
	}
}

func MapImpact(assetCount int, criticalAsset bool) int {
	switch {
	case criticalAsset || assetCount > 5:
		return 1
	case assetCount >= 2:
		return 2
	default:
		return 3
	}
}

func MapStateToClario(mapping map[string]string, state string) string {
	if mapped, ok := mapping[state]; ok {
		return mapped
	}
	switch state {
	case "1", "New":
		return "new"
	case "2", "In Progress":
		return "investigating"
	case "6", "Resolved", "7", "Closed":
		return "resolved"
	case "8", "Canceled", "Cancelled":
		return "false_positive"
	default:
		return ""
	}
}
