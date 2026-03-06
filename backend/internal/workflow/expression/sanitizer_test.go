package expression

import (
	"strings"
	"testing"
)

func TestSanitizer_ValidExpressions(t *testing.T) {
	sanitizer := NewSanitizer()

	validExpressions := []string{
		"a == 1",
		"a != 'hello'",
		"a > 5 && b < 10",
		"steps.triage.output.is_valid == true",
		"variables.severity in ['critical', 'high']",
		"(a == 1 || b == 2) && c == 3",
		"!a",
		"a >= 5",
		"a <= 10",
	}

	for _, expr := range validExpressions {
		t.Run(expr, func(t *testing.T) {
			err := sanitizer.SanitizeExpression(expr)
			if err != nil {
				t.Errorf("SanitizeExpression(%q) returned error: %v", expr, err)
			}
		})
	}
}

func TestSanitizer_SQLInjection(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name string
		expr string
	}{
		{
			name: "SELECT keyword",
			expr: "SELECT * FROM users",
		},
		{
			name: "select lowercase",
			expr: "select * from users",
		},
		{
			name: "DROP keyword",
			expr: "DROP TABLE users",
		},
		{
			name: "INSERT keyword",
			expr: "INSERT INTO users VALUES (1)",
		},
		{
			name: "UPDATE keyword",
			expr: "UPDATE users SET name = 'x'",
		},
		{
			name: "DELETE keyword",
			expr: "DELETE FROM users",
		},
		{
			name: "UNION keyword",
			expr: "a == 1 UNION SELECT 1",
		},
		{
			name: "double dash comment",
			expr: "a == 1 -- comment",
		},
		{
			name: "semicolon",
			expr: "a == 1; DROP TABLE users",
		},
		{
			name: "mixed case SQL",
			expr: "SeLeCt * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizer.SanitizeExpression(tt.expr)
			if err == nil {
				t.Errorf("SanitizeExpression(%q) should have returned error", tt.expr)
			}
		})
	}
}

func TestSanitizer_SQLKeywordFalsePositives(t *testing.T) {
	sanitizer := NewSanitizer()

	// These should NOT be rejected because the SQL keywords appear
	// as substrings within larger identifiers.
	validExpressions := []string{
		"droplet == 'active'",
		"selected == true",
		"inserting == true",
		"updated_at == '2024-01-01'",
		"deleted == false",
		"reunion == 'scheduled'",
	}

	for _, expr := range validExpressions {
		t.Run(expr, func(t *testing.T) {
			err := sanitizer.SanitizeExpression(expr)
			if err != nil {
				t.Errorf("SanitizeExpression(%q) should not be rejected (false positive): %v", expr, err)
			}
		})
	}
}

func TestSanitizer_Backticks(t *testing.T) {
	sanitizer := NewSanitizer()

	err := sanitizer.SanitizeExpression("a == `hello`")
	if err == nil {
		t.Error("SanitizeExpression with backticks should return error")
	}
	if !strings.Contains(err.Error(), "backtick") {
		t.Errorf("expected backtick error, got: %v", err)
	}
}

func TestSanitizer_NullBytes(t *testing.T) {
	sanitizer := NewSanitizer()

	// Null bytes should be stripped; the remaining expression is valid.
	err := sanitizer.SanitizeExpression("a == 1\x00")
	if err != nil {
		t.Errorf("SanitizeExpression with null byte should strip it and succeed: %v", err)
	}
}

func TestSanitizer_DollarSign(t *testing.T) {
	sanitizer := NewSanitizer()

	t.Run("valid ${...} syntax accepted", func(t *testing.T) {
		err := sanitizer.SanitizeExpression("${variables.x} == 1")
		if err != nil {
			t.Errorf("SanitizeExpression with valid ${} should pass: %v", err)
		}
	})

	t.Run("standalone dollar sign rejected", func(t *testing.T) {
		err := sanitizer.SanitizeExpression("$a == 1")
		if err == nil {
			t.Error("SanitizeExpression with standalone $ should return error")
		}
		if !strings.Contains(err.Error(), "dollar") {
			t.Errorf("expected dollar sign error, got: %v", err)
		}
	})
}

func TestSanitizer_MaxLength(t *testing.T) {
	sanitizer := NewSanitizer()

	longExpr := strings.Repeat("a", 1001)
	err := sanitizer.SanitizeExpression(longExpr)
	if err == nil {
		t.Error("SanitizeExpression with expression over max length should return error")
	}
	if !strings.Contains(err.Error(), "maximum length") {
		t.Errorf("expected max length error, got: %v", err)
	}
}

func TestSanitizer_PathTraversal(t *testing.T) {
	sanitizer := NewSanitizer()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "double dot traversal",
			path: "variables..name",
		},
		{
			name: "parent directory traversal",
			path: "variables.../etc/passwd",
		},
		{
			name: "__proto__ traversal",
			path: "__proto__.polluted",
		},
		{
			name: "constructor traversal",
			path: "constructor.name",
		},
		{
			name: "empty path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizer.SanitizePath(tt.path)
			if err == nil {
				t.Errorf("SanitizePath(%q) should return error", tt.path)
			}
		})
	}
}

func TestSanitizer_ValidPaths(t *testing.T) {
	sanitizer := NewSanitizer()

	validPaths := []string{
		"variables.name",
		"steps.triage.output.is_valid",
		"trigger.data.id",
		"variables.alert_id",
		"steps.step_1.output.result",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			err := sanitizer.SanitizePath(path)
			if err != nil {
				t.Errorf("SanitizePath(%q) should not return error: %v", path, err)
			}
		})
	}
}

func TestSanitizer_PathWithNullBytes(t *testing.T) {
	sanitizer := NewSanitizer()

	// Path with null byte between segments, after stripping it becomes
	// "variables.name" which should be valid if the null byte is simply removed.
	// However "variables\x00.name" after stripping becomes "variables.name" which is valid.
	err := sanitizer.SanitizePath("variables\x00.name")
	if err != nil {
		t.Errorf("SanitizePath with stripped null byte should succeed: %v", err)
	}
}

func TestSanitizer_PathProtoPollution(t *testing.T) {
	sanitizer := NewSanitizer()

	t.Run("__proto__ case insensitive", func(t *testing.T) {
		err := sanitizer.SanitizePath("__PROTO__.x")
		if err == nil {
			t.Error("SanitizePath should reject __PROTO__ (case insensitive)")
		}
	})

	t.Run("Constructor case insensitive", func(t *testing.T) {
		err := sanitizer.SanitizePath("Constructor.prototype")
		if err == nil {
			t.Error("SanitizePath should reject Constructor (case insensitive)")
		}
	})
}

func TestSanitizer_UnterminatedPlaceholder(t *testing.T) {
	sanitizer := NewSanitizer()

	err := sanitizer.SanitizeExpression("${variables.x")
	if err == nil {
		t.Error("SanitizeExpression with unterminated ${} should return error")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("expected unterminated error, got: %v", err)
	}
}
