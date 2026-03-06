package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// GenerateStorageKey produces a safe storage key.
// Format: {tenant_id}/{suite}/{YYYY}/{MM}/{uuid}.{ext}
// The original filename is NEVER used in the key.
func GenerateStorageKey(tenantID, suite, originalName string) string {
	ext := safeExtension(originalName)
	now := time.Now().UTC()
	id := uuid.New().String()
	return fmt.Sprintf("%s/%s/%04d/%02d/%s%s", tenantID, suite, now.Year(), int(now.Month()), id, ext)
}

// SanitizeFilename strips dangerous characters from a filename.
// Returns a safe display name that can be stored as metadata.
func SanitizeFilename(name string) string {
	// Remove path components
	name = filepath.Base(name)

	// Remove null bytes
	name = strings.ReplaceAll(name, "\x00", "")

	// Remove path traversal sequences
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")

	// Trim whitespace
	name = strings.TrimSpace(name)

	// Default if empty
	if name == "" || name == "." {
		return "unnamed"
	}

	// Truncate to 255 characters (UTF-8 safe)
	if utf8.RuneCountInString(name) > 255 {
		runes := []rune(name)
		name = string(runes[:255])
	}

	return name
}

// safeExtension extracts and validates a file extension.
func safeExtension(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ".bin"
	}

	ext = strings.ToLower(ext)

	// Reject extensions > 10 chars or containing path traversal
	if len(ext) > 10 || strings.Contains(ext, "..") {
		return ".bin"
	}

	// Ensure only safe characters
	for _, r := range ext[1:] { // skip the leading dot
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return ".bin"
		}
	}

	return ext
}
