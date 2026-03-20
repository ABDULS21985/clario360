package transforms

import (
	"fmt"
	"sort"
	"strings"
)

type DeduplicateConfig struct {
	KeyColumns []string `json:"key_columns"`
	Keep       string   `json:"keep"`
	OrderBy    string   `json:"order_by"`
}

func ApplyDeduplicate(data []map[string]interface{}, cfg DeduplicateConfig) ([]map[string]interface{}, *TransformStats, error) {
	if len(cfg.KeyColumns) == 0 {
		return nil, nil, fmt.Errorf("deduplicate transform requires key_columns")
	}
	keep := strings.ToLower(strings.TrimSpace(cfg.Keep))
	if keep == "" {
		keep = "latest"
	}
	if keep == "all" {
		return cloneRows(data), &TransformStats{InputRows: len(data), OutputRows: len(data)}, nil
	}
	grouped := make(map[string][]map[string]interface{})
	for _, row := range cloneRows(data) {
		keyParts := make([]string, 0, len(cfg.KeyColumns))
		for _, column := range cfg.KeyColumns {
			keyParts = append(keyParts, fmt.Sprint(row[column]))
		}
		grouped[strings.Join(keyParts, "|")] = append(grouped[strings.Join(keyParts, "|")], row)
	}
	result := make([]map[string]interface{}, 0, len(grouped))
	for _, rows := range grouped {
		sort.SliceStable(rows, func(i, j int) bool {
			if cfg.OrderBy == "" {
				return i < j
			}
			cmp := compareValues(rows[i][cfg.OrderBy], rows[j][cfg.OrderBy])
			if keep == "first" {
				return cmp < 0
			}
			return cmp > 0
		})
		result = append(result, rows[0])
	}
	stats := &TransformStats{
		InputRows:   len(data),
		OutputRows:  len(result),
		DedupedRows: len(data) - len(result),
	}
	return result, stats, nil
}

