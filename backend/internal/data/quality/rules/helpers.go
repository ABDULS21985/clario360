package rules

import (
	"fmt"
	"regexp"
	"time"
)

func passRate(checked, failed int64) float64 {
	if checked == 0 {
		return 100
	}
	return (float64(checked-failed) / float64(checked)) * 100
}

func limitedSamples(rows []map[string]interface{}, limit int) []map[string]interface{} {
	if len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func asFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		var parsed float64
		_, err := fmt.Sscanf(typed, "%f", &parsed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func asTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case string:
		formats := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"}
		for _, format := range formats {
			if parsed, err := time.Parse(format, typed); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func compileRegex(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile(pattern)
}

