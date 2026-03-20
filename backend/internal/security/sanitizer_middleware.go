package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// SanitizeRequestBody returns middleware that sanitizes all string fields
// in JSON request bodies before they reach handlers.
func SanitizeRequestBody(sanitizer *Sanitizer, secLogger *SecurityLogger, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only process JSON bodies on state-changing methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			if r.Body == nil || r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Read body
			body, err := io.ReadAll(io.LimitReader(r.Body, int64(sanitizer.maxJSONSize)+1))
			r.Body.Close()
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to read request body")
				return
			}

			// Check size limit
			if len(body) > sanitizer.maxJSONSize {
				secLogger.LogFromRequest(r, EventSQLInjection, SeverityMedium, "request body exceeds maximum size", true)
				writeJSONError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "request body exceeds maximum allowed size")
				return
			}

			// Parse JSON
			var data interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				// Not valid JSON — let the handler deal with it
				r.Body = io.NopCloser(bytes.NewReader(body))
				next.ServeHTTP(w, r)
				return
			}

			// Validate string fields for injection patterns
			violations := sanitizeJSONValue(sanitizer, data, "")
			if len(violations) > 0 {
				for _, v := range violations {
					secLogger.LogFieldViolation(r, v.eventType, v.field, v.category)
				}
				writeJSONError(w, http.StatusBadRequest, "MALICIOUS_INPUT", "request contains potentially malicious content")
				return
			}

			// Restore body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))

			next.ServeHTTP(w, r)
		})
	}
}

type violation struct {
	field     string
	category  string
	eventType SecurityEventType
}

// sanitizeJSONValue recursively checks JSON values for injection patterns.
func sanitizeJSONValue(s *Sanitizer, data interface{}, prefix string) []violation {
	var violations []violation

	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fieldPath := key
			if prefix != "" {
				fieldPath = prefix + "." + key
			}
			violations = append(violations, sanitizeJSONValue(s, val, fieldPath)...)
		}
	case []interface{}:
		for i, val := range v {
			fieldPath := fmt.Sprintf("%s[%d]", prefix, i)
			violations = append(violations, sanitizeJSONValue(s, val, fieldPath)...)
		}
	case string:
		if err := s.ValidateNoSQLInjection(v); err != nil {
			if injErr, ok := err.(*InjectionError); ok {
				violations = append(violations, violation{
					field:     prefix,
					category:  injErr.Category,
					eventType: EventSQLInjection,
				})
			}
		}
		if err := s.ValidateNoXSS(v); err != nil {
			if injErr, ok := err.(*InjectionError); ok {
				violations = append(violations, violation{
					field:     prefix,
					category:  injErr.Category,
					eventType: EventXSSAttempt,
				})
			}
		}
	}

	return violations
}

// writeJSONError writes a standard JSON error response.
func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}
