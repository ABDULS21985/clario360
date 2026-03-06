package storage

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateKey_Format(t *testing.T) {
	key := GenerateStorageKey("tenant-abc", "cyber", "report.pdf")

	// Expected: {tenant}/{suite}/{YYYY}/{MM}/{uuid}.{ext}
	// Example: tenant-abc/cyber/2026/03/550e8400-e29b-41d4-a716-446655440000.pdf
	pattern := `^tenant-abc/cyber/\d{4}/\d{2}/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.pdf$`
	matched, err := regexp.MatchString(pattern, key)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("key does not match expected format: %s", key)
	}

	// Verify the year/month are current
	now := time.Now().UTC()
	expectedPrefix := fmt.Sprintf("tenant-abc/cyber/%04d/%02d/", now.Year(), int(now.Month()))
	if !strings.HasPrefix(key, expectedPrefix) {
		t.Fatalf("key prefix mismatch: got %s, expected prefix %s", key, expectedPrefix)
	}
}

func TestGenerateKey_NoPathTraversal(t *testing.T) {
	keys := []string{
		GenerateStorageKey("tenant-1", "cyber", "../../etc/passwd"),
		GenerateStorageKey("tenant-2", "data", "../secret.txt"),
		GenerateStorageKey("tenant-3", "acta", "..\\windows\\system32\\config"),
	}

	for _, key := range keys {
		if strings.Contains(key, "..") {
			t.Fatalf("key contains path traversal '..': %s", key)
		}
		if strings.HasPrefix(key, "/") {
			t.Fatalf("key contains absolute path: %s", key)
		}
	}
}

func TestGenerateKey_NoOriginalFilename(t *testing.T) {
	originalName := "super-secret-document-name.pdf"
	key := GenerateStorageKey("tenant-1", "cyber", originalName)

	// The key should NOT contain "super-secret-document-name"
	if strings.Contains(key, "super-secret-document-name") {
		t.Fatalf("key contains original filename: %s", key)
	}
}

func TestGenerateKey_LongExtension(t *testing.T) {
	// Extensions >10 chars should default to .bin
	key := GenerateStorageKey("tenant-1", "cyber", "file.verylongextension")
	if !strings.HasSuffix(key, ".bin") {
		t.Fatalf("expected .bin suffix for long extension, got key: %s", key)
	}

	// Short extension should be preserved
	key = GenerateStorageKey("tenant-1", "cyber", "file.csv")
	if !strings.HasSuffix(key, ".csv") {
		t.Fatalf("expected .csv suffix, got key: %s", key)
	}
}

func TestSanitizeFilename_NullBytes(t *testing.T) {
	name := SanitizeFilename("hello\x00world.txt")
	if strings.ContainsRune(name, 0) {
		t.Fatalf("sanitized name still contains null bytes: %q", name)
	}
	if name != "helloworld.txt" {
		t.Fatalf("unexpected sanitized name: %q", name)
	}
}

func TestSanitizeFilename_PathComponents(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/etc/passwd", "passwd"},
		{"../../secret.txt", "secret.txt"},
		{"path/to/file.txt", "file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Fatalf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}

	// Windows-style backslash paths: on non-Windows, filepath.Base doesn't split on backslash.
	// SanitizeFilename strips backslashes, so the result has no path separators.
	t.Run("backslash stripped", func(t *testing.T) {
		result := SanitizeFilename("C:\\Windows\\system32\\cmd.exe")
		if strings.Contains(result, "\\") || strings.Contains(result, "/") {
			t.Fatalf("SanitizeFilename should strip path separators, got %q", result)
		}
	})
}

func TestSanitizeFilename_Empty(t *testing.T) {
	tests := []string{"", "   ", "\x00"}
	for _, input := range tests {
		result := SanitizeFilename(input)
		if result != "unnamed" {
			t.Fatalf("SanitizeFilename(%q) = %q, want %q", input, result, "unnamed")
		}
	}
}

func TestSanitizeFilename_TooLong(t *testing.T) {
	// Create a name with 300 runes
	long := strings.Repeat("a", 300)
	result := SanitizeFilename(long)
	runes := []rune(result)
	if len(runes) > 255 {
		t.Fatalf("expected max 255 runes, got %d", len(runes))
	}
	if len(runes) != 255 {
		t.Fatalf("expected exactly 255 runes for truncation, got %d", len(runes))
	}
}
