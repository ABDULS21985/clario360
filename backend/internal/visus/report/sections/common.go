package sections

import (
	"fmt"

	"github.com/clario360/platform/internal/visus/aggregator"
)

func unavailable() map[string]any {
	return map[string]any{
		"available": false,
		"message":   "Data unavailable for this section.",
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func mustValue(payload map[string]any, path string) float64 {
	value, err := aggregator.ExtractValue(payload, path)
	if err != nil {
		return 0
	}
	return value
}

func mustString(payload map[string]any, path string) string {
	value, err := aggregator.Extract(payload, path)
	if err != nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprint(value)
}

func mitreCoverage(payload map[string]any) float64 {
	value, err := aggregator.Extract(payload, "$.data.mitre_heatmap.cells")
	if err != nil {
		return 0
	}
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return 0
	}
	covered := 0
	for _, item := range items {
		mapped, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if detection, ok := mapped["has_detection"].(bool); ok && detection {
			covered++
		}
	}
	return (float64(covered) / float64(len(items))) * 100
}

func trendWord(delta float64, higherIsBetter bool) string {
	adjusted := delta
	if !higherIsBetter {
		adjusted = -delta
	}
	switch {
	case adjusted > 5:
		return "significantly improved"
	case adjusted > 0:
		return "improved"
	case adjusted == 0:
		return "remained stable"
	case adjusted < -5:
		return "significantly declined"
	default:
		return "declined"
	}
}

func number(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	default:
		return 0
	}
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func available(section map[string]any) bool {
	if section == nil {
		return false
	}
	flag, ok := section["available"].(bool)
	return ok && flag
}
