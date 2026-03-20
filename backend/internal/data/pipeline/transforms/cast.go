package transforms

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CastConfig struct {
	Column string `json:"column"`
	ToType string `json:"to_type"`
}

func ApplyCast(data []map[string]interface{}, cfg CastConfig) ([]map[string]interface{}, *TransformStats, error) {
	if cfg.Column == "" || cfg.ToType == "" {
		return nil, nil, fmt.Errorf("cast transform requires column and to_type")
	}
	rows := cloneRows(data)
	stats := &TransformStats{InputRows: len(data), OutputRows: len(data)}
	for _, row := range rows {
		value, ok := row[cfg.Column]
		if !ok || value == nil {
			row[cfg.Column] = nil
			continue
		}
		castValue, ok := castValue(value, cfg.ToType)
		if !ok {
			row[cfg.Column] = nil
			stats.ErrorRows++
			continue
		}
		row[cfg.Column] = castValue
	}
	return rows, stats, nil
}

func castValue(value any, toType string) (any, bool) {
	switch strings.ToLower(strings.TrimSpace(toType)) {
	case "string":
		return fmt.Sprint(value), true
	case "integer":
		if number, ok := toFloat(value); ok {
			return int64(number), true
		}
		return nil, false
	case "float":
		if number, ok := toFloat(value); ok {
			return number, true
		}
		return nil, false
	case "boolean":
		switch strings.ToLower(strings.TrimSpace(fmt.Sprint(value))) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		default:
			return nil, false
		}
	case "datetime":
		if tm, ok := toTime(value); ok {
			return tm.UTC(), true
		}
		if parsed, err := time.Parse(time.RFC3339, fmt.Sprint(value)); err == nil {
			return parsed.UTC(), true
		}
		if unix, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(value)), 10, 64); err == nil {
			return time.Unix(unix, 0).UTC(), true
		}
		return nil, false
	default:
		return nil, false
	}
}

