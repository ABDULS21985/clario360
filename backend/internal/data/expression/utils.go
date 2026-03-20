package expression

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"02 Jan 2006",
	"02 Jan 2006 15:04:05",
}

func toFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case nil:
		return 0, false
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

func toBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case nil:
		return false, false
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		default:
			return false, false
		}
	case int, int8, int16, int32, int64:
		number, ok := toFloat(typed)
		return number != 0, ok
	case uint, uint8, uint16, uint32, uint64:
		number, ok := toFloat(typed)
		return number != 0, ok
	case float32, float64:
		number, ok := toFloat(typed)
		return number != 0, ok
	default:
		return false, false
	}
}

func toString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	default:
		return fmt.Sprint(typed)
	}
}

func toTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC(), true
	case *time.Time:
		if typed == nil {
			return time.Time{}, false
		}
		return typed.UTC(), true
	case string:
		for _, format := range timeFormats {
			if parsed, err := time.Parse(format, strings.TrimSpace(typed)); err == nil {
				return parsed.UTC(), true
			}
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}
