package expression

import (
	"fmt"
	"strings"
)

// Sanitizer validates and sanitizes expressions and variable references.
// It provides defense-in-depth against injection attacks and path traversal.
type Sanitizer struct{}

// NewSanitizer creates a new Sanitizer.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// sqlKeywords are SQL keywords that should be rejected in expressions.
var sqlKeywords = []string{
	"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "UNION",
}

// SanitizeExpression checks an expression for injection attempts.
// It enforces the following rules:
//   - Strips null bytes
//   - Rejects SQL keywords (case-insensitive)
//   - Rejects backticks
//   - Rejects standalone dollar signs (outside ${...} syntax)
//   - Enforces maximum expression length of 1000 characters
//   - Rejects double-dash SQL comments (--)
//   - Rejects semicolons
func (s *Sanitizer) SanitizeExpression(expr string) error {
	// Strip null bytes.
	cleaned := strings.ReplaceAll(expr, "\x00", "")

	// Enforce maximum length.
	if len(cleaned) > 1000 {
		return fmt.Errorf("expression exceeds maximum length of 1000 characters")
	}

	// Reject backticks.
	if strings.Contains(cleaned, "`") {
		return fmt.Errorf("expression contains forbidden character: backtick")
	}

	// Check for SQL keywords (case-insensitive).
	upper := strings.ToUpper(cleaned)
	for _, keyword := range sqlKeywords {
		if containsWord(upper, keyword) {
			return fmt.Errorf("expression contains forbidden SQL keyword: %s", keyword)
		}
	}

	// Reject double-dash SQL comments.
	if strings.Contains(cleaned, "--") {
		return fmt.Errorf("expression contains forbidden sequence: --")
	}

	// Reject semicolons.
	if strings.Contains(cleaned, ";") {
		return fmt.Errorf("expression contains forbidden character: ;")
	}

	// Reject dollar signs outside ${...} syntax.
	if err := checkDollarSigns(cleaned); err != nil {
		return err
	}

	return nil
}

// SanitizePath checks a variable path for traversal attempts.
// It rejects:
//   - Path traversal sequences (..)
//   - Prototype pollution (__proto__)
//   - Constructor access (constructor)
//   - Empty paths
//   - Null bytes
func (s *Sanitizer) SanitizePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Strip null bytes.
	cleaned := strings.ReplaceAll(path, "\x00", "")
	if cleaned == "" {
		return fmt.Errorf("path is empty after stripping null bytes")
	}

	// Reject path traversal.
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path contains traversal sequence: ..")
	}

	// Check each segment for forbidden values.
	segments := strings.Split(cleaned, ".")
	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf("path contains empty segment")
		}
		lower := strings.ToLower(seg)
		if lower == "__proto__" {
			return fmt.Errorf("path contains forbidden segment: __proto__")
		}
		if lower == "constructor" {
			return fmt.Errorf("path contains forbidden segment: constructor")
		}
	}

	return nil
}

// containsWord checks if the haystack contains the word as a standalone word
// (not part of a larger identifier). This prevents false positives like
// "droplet" matching "DROP".
func containsWord(haystack, word string) bool {
	idx := 0
	for {
		pos := strings.Index(haystack[idx:], word)
		if pos == -1 {
			return false
		}
		absPos := idx + pos
		endPos := absPos + len(word)

		// Check if it's a standalone word (not part of a larger identifier).
		before := true
		after := true
		if absPos > 0 {
			ch := haystack[absPos-1]
			if isIdentChar(ch) {
				before = false
			}
		}
		if endPos < len(haystack) {
			ch := haystack[endPos]
			if isIdentChar(ch) {
				after = false
			}
		}

		if before && after {
			return true
		}

		idx = absPos + 1
		if idx >= len(haystack) {
			return false
		}
	}
}

// isIdentChar returns true if ch is a letter, digit, or underscore.
func isIdentChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// checkDollarSigns rejects standalone dollar signs outside ${...} syntax.
func checkDollarSigns(s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == '$' {
			// Check if this is the start of a ${...} placeholder.
			if i+1 < len(s) && s[i+1] == '{' {
				// Skip to closing brace.
				j := strings.Index(s[i:], "}")
				if j == -1 {
					return fmt.Errorf("unterminated ${...} placeholder")
				}
				i = i + j
				continue
			}
			// Standalone dollar sign.
			return fmt.Errorf("expression contains forbidden standalone dollar sign at position %d", i)
		}
	}
	return nil
}
