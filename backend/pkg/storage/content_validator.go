package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// blockedTypes are never allowed regardless of suite.
var blockedTypes = map[string]bool{
	"application/x-executable":                        true,
	"application/x-dosexec":                           true,
	"application/x-shellscript":                       true,
	"application/java-archive":                        true,
	"application/x-msdownload":                        true,
	"application/vnd.microsoft.portable-executable":   true,
}

// AllowedTypes maps suite names to permitted MIME types.
var AllowedTypes = map[string][]string{
	"cyber":    {"application/pdf", "application/json", "text/csv", "text/plain", "image/png", "image/jpeg"},
	"data":     {"application/pdf", "application/json", "text/csv", "text/plain", "application/sql", "application/gzip", "application/x-gzip", "application/zip"},
	"acta":     {"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "image/png", "image/jpeg"},
	"lex":      {"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "text/plain", "image/png", "image/jpeg"},
	"visus":    {"application/pdf", "application/json", "text/csv", "image/png", "image/svg+xml"},
	"platform": {"application/json", "text/csv", "application/gzip", "application/x-gzip"},
	"models":   {"application/octet-stream", "application/gzip", "application/x-gzip", "application/zip", "application/json"},
}

// magic byte signatures for improved detection
var magicSignatures = []struct {
	magic       []byte
	contentType string
}{
	{[]byte("%PDF"), "application/pdf"},
	{[]byte("\x89PNG\r\n\x1a\n"), "image/png"},
	{[]byte("\xff\xd8\xff"), "image/jpeg"},
	{[]byte("PK\x03\x04"), "application/zip"},
	{[]byte("\x1f\x8b"), "application/gzip"},
	{[]byte("\x7fELF"), "application/x-executable"},
	{[]byte("MZ"), "application/x-dosexec"},
	{[]byte("#!/"), "application/x-shellscript"},
}

// ContentValidationResult holds the outcome of content validation.
type ContentValidationResult struct {
	DeclaredType string
	DetectedType string
	Mismatch     bool
	Blocked      bool
	Allowed      bool
	Reader       io.Reader // replayed reader (header + remaining)
}

// ValidateContent checks a file's content type via magic bytes.
// It reads the first 512 bytes for detection, then returns a replayed reader.
func ValidateContent(body io.Reader, declaredType, suite string) (*ContentValidationResult, error) {
	header := make([]byte, 512)
	n, err := io.ReadAtLeast(body, header, 1)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("reading file header: %w", err)
	}
	header = header[:n]

	// Detect via magic bytes first, then fall back to http.DetectContentType
	detected := detectByMagic(header)
	if detected == "" {
		detected = http.DetectContentType(header)
	}

	// Normalize detected type (strip parameters)
	if idx := strings.IndexByte(detected, ';'); idx != -1 {
		detected = strings.TrimSpace(detected[:idx])
	}

	result := &ContentValidationResult{
		DeclaredType: declaredType,
		DetectedType: detected,
		Reader:       io.MultiReader(bytes.NewReader(header), body),
	}

	// Check blocked types first
	if blockedTypes[detected] {
		result.Blocked = true
		return result, nil
	}

	// Check mismatch between declared and detected
	if declaredType != "" && !typesCompatible(declaredType, detected) {
		result.Mismatch = true
	}

	// Check if allowed for suite
	allowed, ok := AllowedTypes[suite]
	if !ok {
		result.Allowed = true // unknown suite: allow all non-blocked
		return result, nil
	}

	for _, a := range allowed {
		if a == detected || a == declaredType {
			result.Allowed = true
			return result, nil
		}
	}

	return result, nil
}

// detectByMagic checks custom magic byte signatures.
func detectByMagic(header []byte) string {
	for _, sig := range magicSignatures {
		if len(header) >= len(sig.magic) && bytes.Equal(header[:len(sig.magic)], sig.magic) {
			return sig.contentType
		}
	}
	// JSON detection (starts with { or [)
	trimmed := bytes.TrimSpace(header)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return "application/json"
	}
	return ""
}

// typesCompatible checks if two MIME types are considered compatible.
func typesCompatible(declared, detected string) bool {
	if declared == detected {
		return true
	}
	// application/octet-stream is generic — always compatible
	if declared == "application/octet-stream" || detected == "application/octet-stream" {
		return true
	}
	// text/plain is generic for text detection
	if detected == "text/plain" {
		return true
	}
	// gzip variants
	if (declared == "application/gzip" || declared == "application/x-gzip") &&
		(detected == "application/gzip" || detected == "application/x-gzip") {
		return true
	}
	return false
}
