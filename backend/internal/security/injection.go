package security

import (
	"fmt"
	"regexp"
	"strings"
)

// compileSQLPatterns compiles SQL injection detection patterns.
// Compiled ONCE at startup, not per-request.
func compileSQLPatterns() []*compiledPattern {
	patterns := []struct {
		pattern  string
		category string
	}{
		// UNION-based injection
		{`union\s+(all\s+)?select`, "union_select"},
		// Piggyback DROP
		{`;\s*drop\s+`, "piggyback_drop"},
		// Piggyback DELETE
		{`;\s*delete\s+from\s+`, "piggyback_delete"},
		// Piggyback UPDATE
		{`;\s*update\s+\S+\s+set\s+`, "piggyback_update"},
		// Piggyback INSERT
		{`;\s*insert\s+into\s+`, "piggyback_insert"},
		// Tautology (string-based)
		{`'\s*or\s+'[^']*'\s*=\s*'[^']*`, "tautology_string"},
		// Tautology (numeric)
		{`'\s*or\s+\d+\s*=\s*\d+`, "tautology_numeric"},
		// Comment terminator
		{`--\s*$`, "comment_terminator"},
		// Block comment
		{`/\*[\s\S]*?\*/`, "block_comment"},
		// Command execution
		{`xp_cmdshell|xp_regread|xp_servicecontrol`, "command_exec"},
		// EXEC/EXECUTE
		{`exec(ute)?\s*\(`, "exec_function"},
		// WAITFOR timing attack
		{`waitfor\s+delay\s+'`, "waitfor_delay"},
		// SLEEP timing attack
		{`sleep\s*\(\s*\d+\s*\)`, "sleep_function"},
		// INTO OUTFILE
		{`into\s+(out|dump)file\s+`, "file_write"},
		// LOAD_FILE
		{`load_file\s*\(`, "file_read"},
		// BENCHMARK
		{`benchmark\s*\(`, "benchmark"},
		// Long hex encoding (potential encoded payload)
		{`0x[0-9a-f]{8,}`, "hex_encoding"},
		// CHAR() encoding bypass
		{`char\s*\(\s*\d+(\s*,\s*\d+)+\s*\)`, "char_encoding"},
	}

	compiled := make([]*compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		r, err := regexp.Compile("(?i)" + p.pattern)
		if err != nil {
			continue
		}
		compiled = append(compiled, &compiledPattern{
			regex:    r,
			category: p.category,
		})
	}
	return compiled
}

// ValidateNoSQLInjection checks the input string against known SQL injection patterns.
// This is NOT the primary SQL injection defence (parameterized queries are).
// This is a belt-and-suspenders check that catches obvious malicious inputs.
func (s *Sanitizer) ValidateNoSQLInjection(input string) error {
	if input == "" {
		return nil
	}

	for _, pattern := range s.sqlPatterns {
		if pattern.regex.MatchString(input) {
			return &InjectionError{
				Category: pattern.category,
				Type:     "sql_injection",
			}
		}
	}

	return nil
}

// ValidateNoSQLInjectionBatch validates multiple fields and returns the first error.
func (s *Sanitizer) ValidateNoSQLInjectionBatch(fields map[string]string) error {
	for fieldName, value := range fields {
		if err := s.ValidateNoSQLInjection(value); err != nil {
			var injErr *InjectionError
			if asInjErr, ok := err.(*InjectionError); ok {
				injErr = asInjErr
			} else {
				injErr = &InjectionError{Category: "unknown", Type: "sql_injection"}
			}
			return fmt.Errorf("field %q: %w", fieldName, injErr)
		}
	}
	return nil
}

// InjectionError represents a detected injection attempt.
type InjectionError struct {
	Category string
	Type     string
	Field    string
}

func (e *InjectionError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("security: %s detected in field %q (category: %s)", e.Type, e.Field, e.Category)
	}
	return fmt.Sprintf("security: %s detected (category: %s)", e.Type, e.Category)
}

// ValidateIdentifier checks if a string is a safe SQL identifier (table/column name).
// Only allows alphanumeric characters and underscores.
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("security: identifier must not be empty")
	}
	if len(name) > 63 { // PostgreSQL identifier limit
		return fmt.Errorf("security: identifier exceeds maximum length")
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("security: identifier contains invalid character")
		}
	}
	// Must not start with a number
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("security: identifier must not start with a digit")
	}
	// Check against SQL reserved words that could be dangerous
	upper := strings.ToUpper(name)
	reservedDangerous := map[string]bool{
		"DROP": true, "DELETE": true, "TRUNCATE": true, "ALTER": true,
		"EXEC": true, "EXECUTE": true, "GRANT": true, "REVOKE": true,
	}
	if reservedDangerous[upper] {
		return fmt.Errorf("security: identifier matches reserved word")
	}
	return nil
}
