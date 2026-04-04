package security

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// FileUploadValidator provides file upload security: magic byte validation,
// size limits, path traversal prevention, and virus scan integration.
type FileUploadValidator struct {
	maxSize         int64
	allowedMIME     map[string]bool
	allowedExt      map[string]bool
	sanitizer       *Sanitizer
	metrics         *Metrics
	logger          zerolog.Logger
	virusScanHook   func(data []byte, filename string) error
}

// FileUploadOption configures the FileUploadValidator.
type FileUploadOption func(*FileUploadValidator)

// WithVirusScanHook sets a virus scanning callback.
func WithVirusScanHook(hook func(data []byte, filename string) error) FileUploadOption {
	return func(v *FileUploadValidator) { v.virusScanHook = hook }
}

// NewFileUploadValidator creates a new file upload validator.
func NewFileUploadValidator(cfg *Config, sanitizer *Sanitizer, metrics *Metrics, logger zerolog.Logger, opts ...FileUploadOption) *FileUploadValidator {
	allowedMIME := make(map[string]bool, len(cfg.AllowedMIMETypes))
	for _, m := range cfg.AllowedMIMETypes {
		allowedMIME[strings.ToLower(m)] = true
	}

	allowedExt := make(map[string]bool, len(cfg.AllowedExtensions))
	for _, e := range cfg.AllowedExtensions {
		allowedExt[strings.ToLower(e)] = true
	}

	v := &FileUploadValidator{
		maxSize:     cfg.MaxUploadSize,
		allowedMIME: allowedMIME,
		allowedExt:  allowedExt,
		sanitizer:   sanitizer,
		metrics:     metrics,
		logger:      logger.With().Str("component", "file_upload").Logger(),
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// ValidatedFile represents a file that has passed all security checks.
type ValidatedFile struct {
	OriginalName  string
	SanitizedName string
	Size          int64
	ContentType   string
	Data          []byte
}

// ValidateFile validates an uploaded file for security.
func (v *FileUploadValidator) ValidateFile(file multipart.File, header *multipart.FileHeader) (*ValidatedFile, error) {
	// 1. Size check
	if header.Size > v.maxSize {
		if v.metrics != nil {
			v.metrics.FileUploadBlocked.WithLabelValues("too_large").Inc()
		}
		return nil, ErrFileTooLarge
	}

	// 2. Sanitize filename
	sanitizedName, err := v.sanitizer.ValidateFileName(header.Filename)
	if err != nil {
		if v.metrics != nil {
			v.metrics.FileUploadBlocked.WithLabelValues("bad_filename").Inc()
		}
		return nil, fmt.Errorf("invalid filename: %w", err)
	}

	// 3. Check extension
	ext := strings.ToLower(fileExtension(sanitizedName))
	if ext != "" && !v.allowedExt[ext] {
		if v.metrics != nil {
			v.metrics.FileUploadBlocked.WithLabelValues("bad_extension").Inc()
		}
		return nil, fmt.Errorf("%w: extension %s not allowed", ErrInvalidMIMEType, ext)
	}

	// 4. Read file data (with size limit)
	data, err := io.ReadAll(io.LimitReader(file, v.maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if int64(len(data)) > v.maxSize {
		if v.metrics != nil {
			v.metrics.FileUploadBlocked.WithLabelValues("too_large").Inc()
		}
		return nil, ErrFileTooLarge
	}

	// 5. Magic byte validation — detect actual content type
	detectedType := http.DetectContentType(data)
	if !v.isAllowedContentType(detectedType) {
		if v.metrics != nil {
			v.metrics.FileUploadBlocked.WithLabelValues("bad_content_type").Inc()
		}
		return nil, fmt.Errorf("%w: detected content type %s", ErrInvalidMIMEType, detectedType)
	}

	// 6. Check for polyglot files (files that are valid in multiple formats)
	if isPolyglotFile(data) {
		if v.metrics != nil {
			v.metrics.FileUploadBlocked.WithLabelValues("polyglot").Inc()
		}
		return nil, fmt.Errorf("%w: polyglot file detected", ErrMagicByteMismatch)
	}

	// 7. Virus scan hook
	if v.virusScanHook != nil {
		if err := v.virusScanHook(data, sanitizedName); err != nil {
			if v.metrics != nil {
				v.metrics.FileUploadBlocked.WithLabelValues("malware").Inc()
			}
			return nil, ErrMalwareDetected
		}
		if v.metrics != nil {
			v.metrics.FileUploadScanned.Inc()
		}
	}

	return &ValidatedFile{
		OriginalName:  header.Filename,
		SanitizedName: sanitizedName,
		Size:          int64(len(data)),
		ContentType:   detectedType,
		Data:          data,
	}, nil
}

// isAllowedContentType checks if a detected content type is allowed.
func (v *FileUploadValidator) isAllowedContentType(contentType string) bool {
	// Normalize: http.DetectContentType returns type/subtype; drop params
	ct := strings.SplitN(contentType, ";", 2)[0]
	ct = strings.TrimSpace(strings.ToLower(ct))

	if v.allowedMIME[ct] {
		return true
	}

	// application/octet-stream is the fallback — check extension instead
	if ct == "application/octet-stream" {
		return true // Extension already validated
	}

	// text/plain is detected for CSV and plain text files
	if ct == "text/plain" {
		return true
	}

	return false
}

// isPolyglotFile detects files that are valid in multiple formats (e.g., GIFAR).
func isPolyglotFile(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	// Check for Java archive embedded in image
	hasJARSignature := bytes.Contains(data, []byte("PK\x03\x04")) && len(data) > 100
	hasImageHeader := bytes.HasPrefix(data, []byte("\x89PNG")) ||
		bytes.HasPrefix(data, []byte("\xff\xd8\xff")) ||
		bytes.HasPrefix(data, []byte("GIF8"))

	if hasJARSignature && hasImageHeader {
		return true
	}

	// Check for HTML embedded in other file types
	if hasImageHeader {
		lower := bytes.ToLower(data)
		if bytes.Contains(lower, []byte("<script")) || bytes.Contains(lower, []byte("<html")) {
			return true
		}
	}

	return false
}

// fileExtension extracts the file extension including the dot.
func fileExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return strings.ToLower(name[i:])
		}
	}
	return ""
}

// FileUploadMiddleware returns middleware that validates file uploads.
func FileUploadMiddleware(validator *FileUploadValidator, secLogger *SecurityLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost && r.Method != http.MethodPut {
				next.ServeHTTP(w, r)
				return
			}

			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "multipart/form-data") {
				next.ServeHTTP(w, r)
				return
			}

			// File validation is done in the handler using the validator directly
			// This middleware ensures multipart body size limits
			r.Body = http.MaxBytesReader(w, r.Body, validator.maxSize+1024*1024) // +1MB for form fields

			next.ServeHTTP(w, r)
		})
	}
}
