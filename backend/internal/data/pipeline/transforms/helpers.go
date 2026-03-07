package transforms

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var parseTimeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

func cloneRows(data []map[string]interface{}) []map[string]interface{} {
	cloned := make([]map[string]interface{}, 0, len(data))
	for _, row := range data {
		copyRow := make(map[string]interface{}, len(row))
		for key, value := range row {
			copyRow[key] = value
		}
		cloned = append(cloned, copyRow)
	}
	return cloned
}

func toFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func toTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case *time.Time:
		if typed == nil {
			return time.Time{}, false
		}
		return *typed, true
	case string:
		for _, format := range parseTimeFormats {
			if parsed, err := time.Parse(format, strings.TrimSpace(typed)); err == nil {
				return parsed, true
			}
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

func compareValues(a, b any) int {
	if at, ok := toTime(a); ok {
		if bt, ok := toTime(b); ok {
			switch {
			case at.Before(bt):
				return -1
			case at.After(bt):
				return 1
			default:
				return 0
			}
		}
	}
	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			default:
				return 0
			}
		}
	}
	as := fmt.Sprint(a)
	bs := fmt.Sprint(b)
	switch {
	case as < bs:
		return -1
	case as > bs:
		return 1
	default:
		return 0
	}
}

