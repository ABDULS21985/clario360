package transforms

import (
	"fmt"

	dataexpr "github.com/clario360/platform/internal/data/expression"
)

type DeriveConfig struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

func ApplyDerive(data []map[string]interface{}, cfg DeriveConfig) ([]map[string]interface{}, *TransformStats, error) {
	if cfg.Name == "" || cfg.Expression == "" {
		return nil, nil, fmt.Errorf("derive transform requires name and expression")
	}
	compiled, err := dataexpr.Compile(cfg.Expression)
	if err != nil {
		return nil, nil, err
	}
	rows := cloneRows(data)
	stats := &TransformStats{InputRows: len(data), OutputRows: len(data)}
	for _, row := range rows {
		value, err := compiled.Evaluate(row)
		if err != nil {
			return nil, nil, err
		}
		row[cfg.Name] = value
	}
	return rows, stats, nil
}

