package security_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// newTestFileUploadValidator creates a FileUploadValidator with sensible test defaults.
func newTestFileUploadValidator(maxSize int64, mimeTypes, extensions []string, opts ...security.FileUploadOption) *security.FileUploadValidator {
	cfg := &security.Config{
		MaxUploadSize:   maxSize,
		AllowedMIMETypes: mimeTypes,
		AllowedExtensions: extensions,
	}
	sanitizer := security.NewSanitizer()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	return security.NewFileUploadValidator(cfg, sanitizer, metrics, logger, opts...)
}

// fakeMultipartFile wraps a bytes.Reader to satisfy multipart.File.
type fakeMultipartFile struct {
	*bytes.Reader
}

func (f *fakeMultipartFile) Close() error { return nil }

// makeMultipart creates a multipart.File and *multipart.FileHeader for testing.
func makeMultipart(filename string, data []byte) (multipart.File, *multipart.FileHeader) {
	header := &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(data)),
	}
	file := &fakeMultipartFile{bytes.NewReader(data)}
	return file, header
}

func TestFileUpload_OversizedFile(t *testing.T) {
	v := newTestFileUploadValidator(1024, []string{"application/pdf"}, []string{".pdf"})
	data := bytes.Repeat([]byte("x"), 2048) // exceeds 1024 limit
	file, header := makeMultipart("report.pdf", data)

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}
	if !errors.Is(err, security.ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got: %v", err)
	}
}

func TestFileUpload_DangerousExtension_Exe(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"application/pdf"}, []string{".pdf"})
	file, header := makeMultipart("malware.exe", []byte("MZ fake exe content"))

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected error for .exe extension, got nil")
	}
}

func TestFileUpload_DangerousExtension_PHP(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"text/plain"}, []string{".txt"})
	file, header := makeMultipart("shell.php", []byte("<?php echo 'hacked'; ?>"))

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected error for .php extension, got nil")
	}
}

func TestFileUpload_DangerousExtension_SH(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"text/plain"}, []string{".txt"})
	file, header := makeMultipart("exploit.sh", []byte("#!/bin/bash\nrm -rf /"))

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected error for .sh extension, got nil")
	}
}

func TestFileUpload_DoubleExtension(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"application/pdf"}, []string{".pdf"})
	file, header := makeMultipart("report.pdf.exe", []byte("MZ fake"))

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected error for double extension .pdf.exe, got nil")
	}
}

func TestFileUpload_HiddenFile(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"text/plain"}, []string{".txt"})
	file, header := makeMultipart(".htaccess", []byte("Options +ExecCGI"))

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected error for hidden file, got nil")
	}
}

func TestFileUpload_PathTraversal(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"text/plain"}, []string{".txt"})
	// Path traversal in filename should be sanitized to the basename
	data := []byte("harmless content")
	file, header := makeMultipart("../../etc/passwd", data)

	result, err := v.ValidateFile(file, header)
	// The sanitizer strips path components. If it results in an error, that is also
	// acceptable. If it succeeds, the sanitized name must not contain path traversal.
	if err != nil {
		// Acceptable: the path traversal was blocked
		return
	}
	if strings.Contains(result.SanitizedName, "..") {
		t.Fatalf("path traversal not sanitized: %q", result.SanitizedName)
	}
	if strings.Contains(result.SanitizedName, "/") {
		t.Fatalf("path separator not sanitized: %q", result.SanitizedName)
	}
}

func TestFileUpload_NullBytesInFilename(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024, []string{"text/plain"}, []string{".txt"})
	data := []byte("content")
	file, header := makeMultipart("file\x00.txt", data)

	result, err := v.ValidateFile(file, header)
	if err != nil {
		// Acceptable: null bytes rejected entirely
		return
	}
	if strings.ContainsRune(result.SanitizedName, '\x00') {
		t.Fatal("null byte not removed from sanitized filename")
	}
}

func TestFileUpload_ValidPDF(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"application/pdf"},
		[]string{".pdf"})

	// PDF magic bytes: %PDF-
	pdfData := []byte("%PDF-1.4 fake pdf content that is long enough for detection to work properly with enough bytes here")
	file, header := makeMultipart("report.pdf", pdfData)

	result, err := v.ValidateFile(file, header)
	if err != nil {
		t.Fatalf("expected valid PDF to pass, got: %v", err)
	}
	if result.SanitizedName != "report.pdf" {
		t.Fatalf("expected sanitized name 'report.pdf', got: %q", result.SanitizedName)
	}
}

func TestFileUpload_ValidImage(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"image/png", "image/jpeg"},
		[]string{".png", ".jpg", ".jpeg"})

	// PNG magic bytes
	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	pngData := append(pngHeader, bytes.Repeat([]byte{0x00}, 100)...)
	file, header := makeMultipart("photo.png", pngData)

	result, err := v.ValidateFile(file, header)
	if err != nil {
		t.Fatalf("expected valid PNG to pass, got: %v", err)
	}
	if result.SanitizedName != "photo.png" {
		t.Fatalf("expected sanitized name 'photo.png', got: %q", result.SanitizedName)
	}
}

func TestFileUpload_PolyglotFile(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"image/png", "application/java-archive"},
		[]string{".png", ".jar"})

	// Construct a polyglot: PNG header + embedded JAR (PK\x03\x04)
	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	padding := bytes.Repeat([]byte{0x00}, 100)
	jarSignature := []byte("PK\x03\x04")
	polyglotData := append(pngHeader, padding...)
	polyglotData = append(polyglotData, jarSignature...)
	polyglotData = append(polyglotData, bytes.Repeat([]byte{0x00}, 50)...)

	file, header := makeMultipart("image.png", polyglotData)

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected polyglot file (PNG+JAR) to be rejected, got nil")
	}
	if !errors.Is(err, security.ErrMagicByteMismatch) {
		t.Fatalf("expected ErrMagicByteMismatch, got: %v", err)
	}
}

func TestFileUpload_ValidCSV(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"text/csv", "text/plain"},
		[]string{".csv"})

	csvData := []byte("name,email,role\nAlice,alice@example.com,admin\nBob,bob@example.com,user\n")
	file, header := makeMultipart("users.csv", csvData)

	result, err := v.ValidateFile(file, header)
	if err != nil {
		t.Fatalf("expected valid CSV to pass, got: %v", err)
	}
	if result.SanitizedName != "users.csv" {
		t.Fatalf("expected sanitized name 'users.csv', got: %q", result.SanitizedName)
	}
}

func TestFileUpload_NoExtension(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"text/plain"},
		[]string{".txt"})

	data := []byte("some content without an extension")
	file, header := makeMultipart("README", data)

	// A file with no extension: the extension check should pass (empty ext is not
	// in the forbidden list). The content type check determines the result.
	result, err := v.ValidateFile(file, header)
	if err != nil {
		// If the validator rejects files without extensions, that is acceptable
		// security-conservative behavior.
		t.Logf("file with no extension rejected (acceptable): %v", err)
		return
	}
	if result.SanitizedName == "" {
		t.Fatal("expected non-empty sanitized name")
	}
}

func TestFileUpload_EmptyFilename(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"text/plain"},
		[]string{".txt"})

	data := []byte("content")
	file, header := makeMultipart("", data)

	result, err := v.ValidateFile(file, header)
	if err != nil {
		// Acceptable if empty filename is rejected outright
		t.Logf("empty filename rejected (acceptable): %v", err)
		return
	}
	if result.SanitizedName != "unnamed_upload" {
		t.Fatalf("expected sanitized name 'unnamed_upload' for empty filename, got: %q", result.SanitizedName)
	}
}

func TestFileUpload_VirusScanHookTriggered(t *testing.T) {
	malwareErr := errors.New("ClamAV: Eicar-Signature FOUND")
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"text/plain"},
		[]string{".txt"},
		security.WithVirusScanHook(func(data []byte, filename string) error {
			return malwareErr
		}),
	)

	file, header := makeMultipart("innocent.txt", []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"))

	_, err := v.ValidateFile(file, header)
	if err == nil {
		t.Fatal("expected malware detection error, got nil")
	}
	if !errors.Is(err, security.ErrMalwareDetected) {
		t.Fatalf("expected ErrMalwareDetected, got: %v", err)
	}
}

func TestFileUpload_VirusScanHookPasses(t *testing.T) {
	v := newTestFileUploadValidator(10*1024*1024,
		[]string{"text/plain"},
		[]string{".txt"},
		security.WithVirusScanHook(func(data []byte, filename string) error {
			return nil // clean file
		}),
	)

	file, header := makeMultipart("clean.txt", []byte("perfectly safe content"))

	result, err := v.ValidateFile(file, header)
	if err != nil {
		t.Fatalf("expected clean file to pass virus scan, got: %v", err)
	}
	if result.SanitizedName != "clean.txt" {
		t.Fatalf("expected 'clean.txt', got: %q", result.SanitizedName)
	}
}
