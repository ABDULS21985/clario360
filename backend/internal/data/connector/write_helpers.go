package connector

import (
	"sort"
)

func writeColumns(rows []map[string]any) []string {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	columns := make([]string, 0)
	for _, row := range rows {
		for key := range row {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			columns = append(columns, key)
		}
	}
	sort.Strings(columns)
	return columns
}

func rowValues(row map[string]any, columns []string) []any {
	values := make([]any, 0, len(columns))
	for _, column := range columns {
		values = append(values, row[column])
	}
	return values
}
