package transforms

import "fmt"

type MapValuesConfig struct {
	Column  string         `json:"column"`
	Mapping map[string]any `json:"mapping"`
	Default any            `json:"default"`
}

func ApplyMapValues(data []map[string]interface{}, cfg MapValuesConfig) ([]map[string]interface{}, *TransformStats, error) {
	if cfg.Column == "" {
		return nil, nil, fmt.Errorf("map_values transform requires column")
	}
	rows := cloneRows(data)
	for _, row := range rows {
		original := fmt.Sprint(row[cfg.Column])
		if mapped, ok := cfg.Mapping[original]; ok {
			row[cfg.Column] = mapped
		} else if cfg.Default != nil {
			row[cfg.Column] = cfg.Default
		}
	}
	return rows, &TransformStats{InputRows: len(data), OutputRows: len(rows)}, nil
}

