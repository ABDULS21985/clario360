package aggregator

import (
	"fmt"
	"strconv"
	"strings"
)

func ExtractValue(data interface{}, path string) (float64, error) {
	value, err := Extract(data, path)
	if err != nil {
		return 0, err
	}
	return toFloat64(value, path)
}

func Extract(data interface{}, path string) (interface{}, error) {
	if strings.TrimSpace(path) == "" {
		return data, nil
	}
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$.")
	if path == "$" || path == "" {
		return data, nil
	}
	current := data
	for _, segment := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			value, ok := typed[segment]
			if !ok {
				return nil, fmt.Errorf("path %q not found in response", "$."+path)
			}
			current = value
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("path %q not found in response", "$."+path)
			}
			if index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("path %q not found in response", "$."+path)
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("path %q not found in response", "$."+path)
		}
	}
	return current, nil
}

func toFloat64(value interface{}, path string) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case jsonNumber:
		out, err := typed.Float64()
		if err == nil {
			return out, nil
		}
	case string:
		out, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return out, nil
		}
	}
	return 0, fmt.Errorf("value at %q is not numeric", path)
}

type jsonNumber interface {
	Float64() (float64, error)
}
