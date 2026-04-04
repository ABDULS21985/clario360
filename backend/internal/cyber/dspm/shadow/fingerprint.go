package shadow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TableFingerprint represents the structural identity of a database table.
type TableFingerprint struct {
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	TableName  string `json:"table_name"`
	Hash       string `json:"hash"`
	Columns    []ColumnDef `json:"columns"`
}

// ColumnDef is a column name + type pair used for fingerprinting.
type ColumnDef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ComputeFingerprint generates a SHA-256 hash from sorted column names and types.
// The fingerprint is deterministic: same columns+types always produce the same hash.
func ComputeFingerprint(columns []ColumnDef) string {
	if len(columns) == 0 {
		return ""
	}

	// Sort columns by name for deterministic ordering
	sorted := make([]ColumnDef, len(columns))
	copy(sorted, columns)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	// Build canonical representation: "col1:type1|col2:type2|..."
	parts := make([]string, len(sorted))
	for i, col := range sorted {
		parts[i] = fmt.Sprintf("%s:%s", strings.ToLower(col.Name), strings.ToLower(col.Type))
	}
	canonical := strings.Join(parts, "|")

	hash := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", hash)
}

// ExtractColumnsFromSchema parses column definitions from a schema info map.
// Supports multiple schema formats found in the platform.
func ExtractColumnsFromSchema(schemaInfo map[string]interface{}) []TableFingerprint {
	var results []TableFingerprint

	tables, ok := schemaInfo["tables"].([]interface{})
	if !ok {
		return results
	}

	for _, t := range tables {
		tbl, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		tableName, _ := tbl["name"].(string)
		if tableName == "" {
			continue
		}

		cols := extractColumnDefs(tbl)
		if len(cols) == 0 {
			continue
		}

		results = append(results, TableFingerprint{
			TableName: tableName,
			Hash:      ComputeFingerprint(cols),
			Columns:   cols,
		})
	}

	return results
}

// ExtractColumnsFromJSON parses column definitions from a JSON raw message schema.
func ExtractColumnsFromJSON(data json.RawMessage) []TableFingerprint {
	if len(data) == 0 {
		return nil
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil
	}
	return ExtractColumnsFromSchema(schema)
}

func extractColumnDefs(table map[string]interface{}) []ColumnDef {
	var cols []ColumnDef

	columns, ok := table["columns"].([]interface{})
	if !ok {
		return cols
	}

	for _, c := range columns {
		switch col := c.(type) {
		case map[string]interface{}:
			name, _ := col["name"].(string)
			colType, _ := col["data_type"].(string)
			if colType == "" {
				colType, _ = col["type"].(string)
			}
			if name != "" {
				cols = append(cols, ColumnDef{Name: name, Type: colType})
			}
		case string:
			cols = append(cols, ColumnDef{Name: col, Type: "unknown"})
		}
	}

	return cols
}
