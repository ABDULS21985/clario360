package sqlutil

import (
	"fmt"
	"regexp"
	"strings"
)

var forbiddenPattern = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|create|truncate|grant|revoke|copy|call|execute|merge)\b`)

func ValidateReadOnlySQL(query string) error {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return fmt.Errorf("query is required")
	}
	compact := strings.ToLower(normalized)
	if !(strings.HasPrefix(compact, "select") || strings.HasPrefix(compact, "with")) {
		return fmt.Errorf("query must start with SELECT or WITH")
	}
	if strings.Contains(normalized, ";") {
		return fmt.Errorf("multiple statements are not allowed")
	}
	if strings.Contains(normalized, "--") || strings.Contains(normalized, "/*") || strings.Contains(normalized, "*/") {
		return fmt.Errorf("SQL comments are not allowed")
	}
	if forbiddenPattern.MatchString(normalized) {
		return fmt.Errorf("query contains forbidden write or DDL operations")
	}
	return nil
}

