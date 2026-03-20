package transforms

import (
	"fmt"

	dataexpr "github.com/clario360/platform/internal/data/expression"
)

type FilterConfig struct {
	Expression string `json:"expression"`
}

func ApplyFilter(data []map[string]interface{}, cfg FilterConfig) ([]map[string]interface{}, *TransformStats, error) {
	if cfg.Expression == "" {
		return nil, nil, fmt.Errorf("filter transform requires expression")
	}
	compiled, err := dataexpr.Compile(cfg.Expression)
	if err != nil {
		return nil, nil, err
	}
	rows := make([]map[string]interface{}, 0, len(data))
	stats := &TransformStats{InputRows: len(data)}
	for _, row := range cloneRows(data) {
		value, err := compiled.Evaluate(row)
		if err != nil {
			return nil, nil, err
		}
		if boolValue, ok := value.(bool); ok && boolValue {
			rows = append(rows, row)
			continue
		}
		stats.FilteredRows++
	}
	stats.OutputRows = len(rows)
	return rows, stats, nil
}

