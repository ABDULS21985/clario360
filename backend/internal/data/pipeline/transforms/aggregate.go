package transforms

import (
	"fmt"
	"strings"
)

type AggregateConfig struct {
	GroupBy      []string               `json:"group_by"`
	Aggregations []AggregateDefinition  `json:"aggregations"`
}

type AggregateDefinition struct {
	Column   string `json:"column"`
	Function string `json:"function"`
	Alias    string `json:"alias"`
}

func ApplyAggregate(data []map[string]interface{}, cfg AggregateConfig) ([]map[string]interface{}, *TransformStats, error) {
	if len(cfg.Aggregations) == 0 {
		return nil, nil, fmt.Errorf("aggregate transform requires aggregations")
	}
	grouped := make(map[string][]map[string]interface{})
	for _, row := range cloneRows(data) {
		keyParts := make([]string, 0, len(cfg.GroupBy))
		for _, column := range cfg.GroupBy {
			keyParts = append(keyParts, fmt.Sprint(row[column]))
		}
		grouped[strings.Join(keyParts, "|")] = append(grouped[strings.Join(keyParts, "|")], row)
	}
	result := make([]map[string]interface{}, 0, len(grouped))
	for _, rows := range grouped {
		out := make(map[string]interface{})
		if len(rows) > 0 {
			for _, column := range cfg.GroupBy {
				out[column] = rows[0][column]
			}
		}
		for _, agg := range cfg.Aggregations {
			value, err := aggregateRows(rows, agg)
			if err != nil {
				return nil, nil, err
			}
			alias := agg.Alias
			if alias == "" {
				alias = agg.Function + "_" + agg.Column
			}
			out[alias] = value
		}
		result = append(result, out)
	}
	return result, &TransformStats{InputRows: len(data), OutputRows: len(result)}, nil
}

func aggregateRows(rows []map[string]interface{}, agg AggregateDefinition) (interface{}, error) {
	function := strings.ToLower(strings.TrimSpace(agg.Function))
	switch function {
	case "count":
		return float64(len(rows)), nil
	case "count_distinct":
		seen := make(map[string]struct{})
		for _, row := range rows {
			seen[fmt.Sprint(row[agg.Column])] = struct{}{}
		}
		return float64(len(seen)), nil
	case "sum", "avg":
		sum := 0.0
		count := 0.0
		for _, row := range rows {
			if number, ok := toFloat(row[agg.Column]); ok {
				sum += number
				count++
			}
		}
		if function == "sum" {
			return sum, nil
		}
		if count == 0 {
			return nil, nil
		}
		return sum / count, nil
	case "min", "max":
		if len(rows) == 0 {
			return nil, nil
		}
		best := rows[0][agg.Column]
		for _, row := range rows[1:] {
			cmp := compareValues(row[agg.Column], best)
			if function == "min" && cmp < 0 {
				best = row[agg.Column]
			}
			if function == "max" && cmp > 0 {
				best = row[agg.Column]
			}
		}
		return best, nil
	default:
		return nil, fmt.Errorf("unsupported aggregation function %q", agg.Function)
	}
}
