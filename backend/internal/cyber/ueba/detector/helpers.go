package detector

import "strings"

func normalizeQueryType(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "select", "insert", "update", "delete":
		return strings.ToLower(strings.TrimSpace(action))
	case "create", "alter", "drop":
		return "ddl"
	default:
		return "select"
	}
}

func qualifiedTableName(schemaName, tableName string) string {
	schemaName = strings.TrimSpace(schemaName)
	tableName = strings.TrimSpace(tableName)
	switch {
	case schemaName == "" && tableName == "":
		return ""
	case schemaName == "":
		return tableName
	case tableName == "":
		return schemaName
	default:
		return schemaName + "." + tableName
	}
}
