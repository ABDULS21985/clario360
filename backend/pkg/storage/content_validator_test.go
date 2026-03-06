package storage

import (
	"bytes"
	"io"
	"testing"
)

func TestValidate_PDF(t *testing.T) {
	// PDF magic bytes: %PDF
	content := append([]byte("%PDF-1.7 fake pdf content"), make([]byte, 500)...)
	result, err := ValidateContent(bytes.NewReader(content), "application/pdf", "cyber")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if result.DetectedType != "application/pdf" {
		t.Fatalf("expected detected type application/pdf, got %s", result.DetectedType)
	}
	if result.Blocked {
		t.Fatal("PDF should not be blocked")
	}
	if result.Mismatch {
		t.Fatal("declared matches detected, should not be mismatch")
	}
	if !result.Allowed {
		t.Fatal("PDF should be allowed for cyber suite")
	}
}

func TestValidate_PNG(t *testing.T) {
	// PNG magic bytes: \x89PNG\r\n\x1a\n
	header := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	content := append(header, make([]byte, 500)...)
	result, err := ValidateContent(bytes.NewReader(content), "image/png", "cyber")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if result.DetectedType != "image/png" {
		t.Fatalf("expected detected type image/png, got %s", result.DetectedType)
	}
	if result.Blocked {
		t.Fatal("PNG should not be blocked")
	}
	if !result.Allowed {
		t.Fatal("PNG should be allowed for cyber suite")
	}
}

func TestValidate_JPEG(t *testing.T) {
	// JPEG magic bytes: \xff\xd8\xff
	header := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	content := append(header, make([]byte, 500)...)
	result, err := ValidateContent(bytes.NewReader(content), "image/jpeg", "cyber")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if result.DetectedType != "image/jpeg" {
		t.Fatalf("expected detected type image/jpeg, got %s", result.DetectedType)
	}
	if result.Blocked {
		t.Fatal("JPEG should not be blocked")
	}
	if !result.Allowed {
		t.Fatal("JPEG should be allowed for cyber suite")
	}
}

func TestValidate_JSON(t *testing.T) {
	content := []byte(`{"key": "value", "number": 42}`)
	result, err := ValidateContent(bytes.NewReader(content), "application/json", "platform")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if result.DetectedType != "application/json" {
		t.Fatalf("expected detected type application/json, got %s", result.DetectedType)
	}
	if result.Blocked {
		t.Fatal("JSON should not be blocked")
	}
	if !result.Allowed {
		t.Fatal("JSON should be allowed for platform suite")
	}
}

func TestValidate_GZIP(t *testing.T) {
	// GZIP magic bytes: \x1f\x8b
	header := []byte{0x1F, 0x8B, 0x08, 0x00}
	content := append(header, make([]byte, 500)...)
	result, err := ValidateContent(bytes.NewReader(content), "application/gzip", "data")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if result.DetectedType != "application/gzip" {
		t.Fatalf("expected detected type application/gzip, got %s", result.DetectedType)
	}
	if result.Blocked {
		t.Fatal("GZIP should not be blocked")
	}
	if !result.Allowed {
		t.Fatal("GZIP should be allowed for data suite")
	}
}

func TestValidate_ExecutableBlocked(t *testing.T) {
	// ELF binary magic bytes: \x7fELF
	header := []byte{0x7F, 0x45, 0x4C, 0x46}
	content := append(header, make([]byte, 500)...)

	// Test across multiple suites -- ELF should always be blocked
	suites := []string{"cyber", "data", "acta", "lex", "visus", "platform", "models", "unknown-suite"}
	for _, suite := range suites {
		t.Run(suite, func(t *testing.T) {
			result, err := ValidateContent(bytes.NewReader(content), "application/x-executable", suite)
			if err != nil {
				t.Fatalf("ValidateContent: %v", err)
			}
			if result.DetectedType != "application/x-executable" {
				t.Fatalf("expected detected type application/x-executable, got %s", result.DetectedType)
			}
			if !result.Blocked {
				t.Fatal("ELF binary should be blocked regardless of suite")
			}
		})
	}
}

func TestValidate_ContentTypeMismatch(t *testing.T) {
	// Declare as PDF but provide JPEG content
	header := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	content := append(header, make([]byte, 500)...)
	result, err := ValidateContent(bytes.NewReader(content), "application/pdf", "cyber")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if result.DeclaredType != "application/pdf" {
		t.Fatalf("expected declared type application/pdf, got %s", result.DeclaredType)
	}
	if result.DetectedType != "image/jpeg" {
		t.Fatalf("expected detected type image/jpeg, got %s", result.DetectedType)
	}
	if !result.Mismatch {
		t.Fatal("declared PDF vs detected JPEG should set Mismatch flag")
	}
}

func TestValidate_AllowedForSuite(t *testing.T) {
	// SQL content is allowed for "data" but not for "cyber"
	// SQL will be detected as text/plain by http.DetectContentType
	content := []byte("SELECT * FROM users WHERE id = 1;\n" + string(make([]byte, 500)))

	t.Run("allowed for data suite with matching declared type", func(t *testing.T) {
		result, err := ValidateContent(bytes.NewReader(content), "application/sql", "data")
		if err != nil {
			t.Fatalf("ValidateContent: %v", err)
		}
		if !result.Allowed {
			t.Fatal("application/sql should be allowed for data suite")
		}
	})

	t.Run("not allowed for platform suite", func(t *testing.T) {
		// PNG is not in platform allowed types
		header := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		pngContent := append(header, make([]byte, 500)...)
		result, err := ValidateContent(bytes.NewReader(pngContent), "image/png", "platform")
		if err != nil {
			t.Fatalf("ValidateContent: %v", err)
		}
		if result.Allowed {
			t.Fatal("PNG should NOT be allowed for platform suite")
		}
	})

	t.Run("unknown suite allows all non-blocked", func(t *testing.T) {
		result, err := ValidateContent(bytes.NewReader(content), "text/plain", "nonexistent-suite")
		if err != nil {
			t.Fatalf("ValidateContent: %v", err)
		}
		if !result.Allowed {
			t.Fatal("unknown suite should allow all non-blocked content")
		}
	})
}

func TestValidate_ReaderReplay(t *testing.T) {
	original := []byte("This is the full file content that should be completely readable after validation.")
	result, err := ValidateContent(bytes.NewReader(original), "text/plain", "cyber")
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}

	// The returned Reader should contain the full original content
	replayed, err := io.ReadAll(result.Reader)
	if err != nil {
		t.Fatalf("reading replayed reader: %v", err)
	}
	if !bytes.Equal(replayed, original) {
		t.Fatalf("replayed content mismatch:\n  got:  %q\n  want: %q", replayed, original)
	}
}
