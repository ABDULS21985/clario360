package security

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Sanitizer provides defence-in-depth input validation.
// PRIMARY defences: parameterized SQL queries, output encoding, RLS.
// This sanitizer provides SECONDARY defence — catching obvious attack payloads
// before they reach business logic.
type Sanitizer struct {
	maxStringLength   int
	maxJSONDepth      int
	maxJSONSize       int
	maxFilenameLength int
	sqlPatterns       []*compiledPattern
	xssPatterns       []*compiledPattern
	pathPatterns      []*compiledPattern
}

type compiledPattern struct {
	regex    *regexp.Regexp
	category string
}

// SanitizerOption configures the Sanitizer.
type SanitizerOption func(*Sanitizer)

// WithMaxStringLength sets the maximum string length.
func WithMaxStringLength(n int) SanitizerOption {
	return func(s *Sanitizer) { s.maxStringLength = n }
}

// WithMaxJSONDepth sets the maximum JSON nesting depth.
func WithMaxJSONDepth(n int) SanitizerOption {
	return func(s *Sanitizer) { s.maxJSONDepth = n }
}

// WithMaxJSONSize sets the maximum JSON size in bytes.
func WithMaxJSONSize(n int) SanitizerOption {
	return func(s *Sanitizer) { s.maxJSONSize = n }
}

// WithMaxFilenameLength sets the maximum filename length.
func WithMaxFilenameLength(n int) SanitizerOption {
	return func(s *Sanitizer) { s.maxFilenameLength = n }
}

// NewSanitizer creates a new Sanitizer with pre-compiled regex patterns.
func NewSanitizer(opts ...SanitizerOption) *Sanitizer {
	s := &Sanitizer{
		maxStringLength:   10000,
		maxJSONDepth:      10,
		maxJSONSize:       1 * 1024 * 1024,
		maxFilenameLength: 255,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.sqlPatterns = compileSQLPatterns()
	s.xssPatterns = compileXSSPatterns()
	s.pathPatterns = compilePathPatterns()

	return s
}

// SanitizeString applies the sanitization pipeline to an input string.
// Used for display values — follows OWASP "encode output, not input" principle.
func (s *Sanitizer) SanitizeString(input string) string {
	// 1. Trim whitespace
	result := strings.TrimSpace(input)

	// 2. Remove null bytes
	result = strings.ReplaceAll(result, "\x00", "")

	// 3. Remove control characters (except \t \n \r)
	result = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, result)

	// 4. Normalize Unicode to NFC
	result = norm.NFC.String(result)

	// 5. Truncate to maxStringLength
	if len([]rune(result)) > s.maxStringLength {
		runes := []rune(result)
		result = string(runes[:s.maxStringLength])
	}

	// 6. HTML-encode dangerous characters
	result = html.EscapeString(result)

	return result
}

// SanitizeStringStrict applies strict sanitization — strips all HTML tags.
// Used for search queries, filter values, enum-like string inputs.
func (s *Sanitizer) SanitizeStringStrict(input string) string {
	// First apply base sanitization
	result := strings.TrimSpace(input)
	result = strings.ReplaceAll(result, "\x00", "")
	result = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, result)
	result = norm.NFC.String(result)

	// Strip all HTML tags
	result = stripHTMLTags(result)

	if len([]rune(result)) > s.maxStringLength {
		runes := []rune(result)
		result = string(runes[:s.maxStringLength])
	}

	return result
}

// htmlTagRegex matches HTML tags for stripping.
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags removes all HTML tags from input.
func stripHTMLTags(input string) string {
	return htmlTagRegex.ReplaceAllString(input, "")
}

// ValidateJSONField validates JSONB fields before they reach the database.
func (s *Sanitizer) ValidateJSONField(input json.RawMessage) error {
	if !json.Valid(input) {
		return ErrInvalidJSON
	}

	if len(input) > s.maxJSONSize {
		return ErrJSONTooLarge
	}

	var parsed interface{}
	if err := json.Unmarshal(input, &parsed); err != nil {
		return ErrInvalidJSON
	}

	checker := &jsonDepthChecker{maxDepth: s.maxJSONDepth}
	if err := checker.check(parsed, 0); err != nil {
		return err
	}

	return s.validateJSONValues(parsed)
}

type jsonDepthChecker struct {
	maxDepth int
}

func (c *jsonDepthChecker) check(data interface{}, depth int) error {
	if depth > c.maxDepth {
		return ErrJSONTooDeep
	}

	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			// Validate key
			if containsDangerousContent(key) {
				return fmt.Errorf("%w: dangerous JSON key detected", ErrMaliciousInput)
			}
			if err := c.check(val, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, val := range v {
			if err := c.check(val, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateJSONValues checks string values in JSON for injection patterns.
func (s *Sanitizer) validateJSONValues(data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		for _, val := range v {
			if err := s.validateJSONValues(val); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, val := range v {
			if err := s.validateJSONValues(val); err != nil {
				return err
			}
		}
	case string:
		if len(v) > s.maxStringLength {
			return ErrStringTooLong
		}
		if err := s.ValidateNoSQLInjection(v); err != nil {
			return err
		}
	}
	return nil
}

// containsDangerousContent checks if a JSON key contains SQL/XSS patterns.
func containsDangerousContent(key string) bool {
	lower := strings.ToLower(key)
	dangerous := []string{
		"<script", "onerror=", "onload=", "javascript:",
		"select ", "drop ", "delete from", "insert into",
		"update ", "union ", "--", "/*",
	}
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return true
		}
	}
	return false
}

// ValidateFilePath prevents path traversal attacks.
func (s *Sanitizer) ValidateFilePath(path string, baseDir string) error {
	// 1. Reject null bytes
	if strings.ContainsRune(path, '\x00') {
		return ErrPathTraversalDetected
	}

	// 2. Reject control characters
	for _, r := range path {
		if unicode.IsControl(r) && r != '\t' {
			return ErrPathTraversalDetected
		}
	}

	// 3. Reject ".." path components
	if strings.Contains(path, "..") {
		return ErrPathTraversalDetected
	}

	// 4. Reject absolute paths
	if strings.HasPrefix(path, "/") {
		return ErrPathTraversalDetected
	}

	// 5. Reject backslashes
	if strings.Contains(path, "\\") {
		return ErrPathTraversalDetected
	}

	// 6. Reject home directory expansion
	if strings.HasPrefix(path, "~") {
		return ErrPathTraversalDetected
	}

	// 7. Reject URL-encoded traversal
	decoded, err := url.PathUnescape(path)
	if err == nil && decoded != path {
		if strings.Contains(decoded, "..") || strings.Contains(decoded, "\\") {
			return ErrPathTraversalDetected
		}
	}

	// 8. Clean the path
	cleaned := filepath.Clean(path)

	// 9. Verify path is within baseDir
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("security: invalid base directory: %w", err)
	}
	absPath, err := filepath.Abs(filepath.Join(baseDir, cleaned))
	if err != nil {
		return ErrPathTraversalDetected
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return ErrPathTraversalDetected
	}

	return nil
}

// dangerousExtensions are file extensions that can execute code.
var dangerousExtensions = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".sh": true,
	".php": true, ".jsp": true, ".asp": true, ".aspx": true,
	".cgi": true, ".py": true, ".rb": true, ".pl": true,
	".com": true, ".scr": true, ".pif": true, ".vbs": true,
	".js": true, ".msi": true, ".dll": true, ".so": true,
}

// safeExtensions are expected document extensions.
var safeExtensions = map[string]bool{
	".pdf": true, ".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".csv": true, ".xlsx": true, ".docx": true,
	".txt": true, ".svg": true, ".webp": true, ".bmp": true,
}

// ValidateFileName validates and sanitizes uploaded file names.
func (s *Sanitizer) ValidateFileName(filename string) (string, error) {
	// 1. Extract basename
	name := filepath.Base(filename)

	// 2. Remove null bytes
	name = strings.ReplaceAll(name, "\x00", "")

	// 3. Replace unsafe characters
	name = sanitizeFileNameChars(name)

	// 4. Check for double extensions hiding executables
	parts := strings.Split(name, ".")
	if len(parts) >= 3 {
		lastExt := "." + strings.ToLower(parts[len(parts)-1])
		secondLastExt := "." + strings.ToLower(parts[len(parts)-2])
		if safeExtensions[secondLastExt] && dangerousExtensions[lastExt] {
			return "", ErrDangerousFileExtension
		}
	}

	// Check single dangerous extension
	ext := strings.ToLower(filepath.Ext(name))
	if dangerousExtensions[ext] {
		return "", ErrDangerousFileExtension
	}

	// 5. Reject hidden files
	if strings.HasPrefix(name, ".") {
		return "", ErrHiddenFile
	}

	// 6. Truncate to max length
	if len(name) > s.maxFilenameLength {
		ext := filepath.Ext(name)
		base := name[:s.maxFilenameLength-len(ext)]
		name = base + ext
	}

	// 7. Handle empty filename
	if name == "" || name == "." {
		name = "unnamed_upload"
	}

	return name, nil
}

// sanitizeFileNameChars replaces characters not in the safe set with underscores.
func sanitizeFileNameChars(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// compilePathPatterns compiles path traversal detection patterns.
func compilePathPatterns() []*compiledPattern {
	patterns := []struct {
		pattern  string
		category string
	}{
		{`\.\.`, "path_traversal"},
		{`%2e%2e`, "encoded_traversal"},
		{`%2f`, "encoded_separator"},
		{`%5c`, "encoded_backslash"},
	}

	compiled := make([]*compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, &compiledPattern{
			regex:    regexp.MustCompile("(?i)" + p.pattern),
			category: p.category,
		})
	}
	return compiled
}
