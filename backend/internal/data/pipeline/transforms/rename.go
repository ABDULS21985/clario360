package transforms

import "fmt"

type RenameConfig struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func ApplyRename(data []map[string]interface{}, cfg RenameConfig) ([]map[string]interface{}, *TransformStats, error) {
	if cfg.From == "" || cfg.To == "" {
		return nil, nil, fmt.Errorf("rename transform requires from and to")
	}
	if len(data) > 0 {
		if _, ok := data[0][cfg.From]; !ok {
			return nil, nil, fmt.Errorf("column %q not found", cfg.From)
		}
	}
	rows := cloneRows(data)
	for _, row := range rows {
		row[cfg.To] = row[cfg.From]
		delete(row, cfg.From)
	}
	return rows, &TransformStats{InputRows: len(data), OutputRows: len(rows)}, nil
}

