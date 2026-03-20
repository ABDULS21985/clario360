package discovery

import (
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/data/model"
)

func AnalyzeSamples(samples []string) model.SampleStats {
	stats := model.SampleStats{
		ObservedSamples: len(samples),
	}
	seen := make(map[string]struct{}, len(samples))

	var minValue *string
	var maxValue *string

	for _, sample := range samples {
		trimmed := strings.TrimSpace(sample)
		if trimmed == "" {
			stats.NullCount++
			continue
		}
		seen[trimmed] = struct{}{}
		if looksLikeEmail(trimmed) {
			stats.LooksLikeEmail = true
		}
		if looksLikePhone(trimmed) {
			stats.LooksLikePhone = true
		}
		if looksLikeCreditCard(trimmed) {
			stats.LooksLikeCard = true
		}
		if net.ParseIP(trimmed) != nil {
			stats.LooksLikeIP = true
		}

		if minValue == nil || trimmed < *minValue {
			value := trimmed
			minValue = &value
		}
		if maxValue == nil || trimmed > *maxValue {
			value := trimmed
			maxValue = &value
		}
	}

	stats.DistinctCount = len(seen)
	if len(seen) > 0 && len(seen) <= 20 {
		stats.EnumValues = make([]string, 0, len(seen))
		for value := range seen {
			stats.EnumValues = append(stats.EnumValues, value)
		}
		sort.Strings(stats.EnumValues)
	}
	stats.MinValue = minValue
	stats.MaxValue = maxValue
	return stats
}

func InferSampleType(samples []string) string {
	typeScore := map[string]int{
		"integer":  0,
		"float":    0,
		"boolean":  0,
		"datetime": 0,
		"string":   0,
	}

	for _, sample := range samples {
		value := strings.TrimSpace(sample)
		if value == "" {
			continue
		}
		switch {
		case isInteger(value):
			typeScore["integer"]++
		case isFloat(value):
			typeScore["float"]++
		case isBoolean(value):
			typeScore["boolean"]++
		case isDatetime(value):
			typeScore["datetime"]++
		default:
			typeScore["string"]++
		}
	}

	bestType := "string"
	bestScore := -1
	for kind, score := range typeScore {
		if score > bestScore {
			bestType = kind
			bestScore = score
		}
	}
	return bestType
}

func isInteger(value string) bool {
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}

func isFloat(value string) bool {
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func isBoolean(value string) bool {
	_, err := strconv.ParseBool(strings.ToLower(value))
	return err == nil
}

func isDatetime(value string) bool {
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}
	return false
}
