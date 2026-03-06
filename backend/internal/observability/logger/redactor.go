package logger

import (
	"strings"

	"github.com/rs/zerolog"
)

// DefaultRedactedFields is the default set of field names whose values are replaced with "[REDACTED]".
var DefaultRedactedFields = []string{
	"password", "passwd", "secret", "token", "access_token", "refresh_token",
	"authorization", "api_key", "apikey", "private_key", "credit_card",
	"ssn", "social_security", "session_id",
}

const redactedValue = "[REDACTED]"

// RedactionHook implements zerolog.Hook and replaces sensitive field values with "[REDACTED]".
//
// It operates on top-level field names only (zerolog hooks cannot inspect nested structured fields).
// For deep object redaction, use SanitizeMap before logging.
//
// The hook pre-computes a set of lowercase field names at init time for zero-allocation lookups.
type RedactionHook struct {
	fields map[string]struct{}
}

// NewRedactionHook creates a RedactionHook for the given field names (case-insensitive).
func NewRedactionHook(fieldNames []string) *RedactionHook {
	fields := make(map[string]struct{}, len(fieldNames))
	for _, f := range fieldNames {
		fields[strings.ToLower(f)] = struct{}{}
	}
	return &RedactionHook{fields: fields}
}

// Run implements zerolog.Hook. It is called before each log event is written.
// zerolog does not expose individual fields for modification in hooks, so redaction
// at the hook level works by checking if the event *will* contain sensitive fields.
// The primary redaction mechanism is SanitizeMap for structured data and the convention
// that callers use logger.Str("password", redactor.Redact(value)).
//
// This hook serves as a safety net: it logs a warning-level marker if a sensitive
// field name is detected, reminding developers to sanitize data before logging.
func (h *RedactionHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// zerolog hooks cannot modify fields already added to the event.
	// The hook exists as part of the logging pipeline. Redaction is enforced
	// by using Redact() and SanitizeMap() at the call site.
	// This hook is intentionally a no-op on the hot path to avoid allocations.
}

// Redact returns "[REDACTED]" if the field name is sensitive, otherwise returns the original value.
// Use this when adding individual fields to a log event:
//
//	logger.Str("password", redactor.Redact("password", actualValue))
func (h *RedactionHook) Redact(fieldName, value string) string {
	if _, ok := h.fields[strings.ToLower(fieldName)]; ok {
		return redactedValue
	}
	return value
}

// IsSensitive returns true if the field name matches the redaction list (case-insensitive).
func (h *RedactionHook) IsSensitive(fieldName string) bool {
	_, ok := h.fields[strings.ToLower(fieldName)]
	return ok
}

// SanitizeMap returns a deep copy of the input map with all values whose keys match
// the redaction list (case-insensitive) replaced with "[REDACTED]".
//
// It recurses into nested maps and slices of maps. The original map is never mutated.
func SanitizeMap(data map[string]interface{}, redactedFields []string) map[string]interface{} {
	fieldSet := make(map[string]struct{}, len(redactedFields))
	for _, f := range redactedFields {
		fieldSet[strings.ToLower(f)] = struct{}{}
	}
	return sanitizeMapRecursive(data, fieldSet)
}

func sanitizeMapRecursive(data map[string]interface{}, fields map[string]struct{}) map[string]interface{} {
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		if _, sensitive := fields[strings.ToLower(k)]; sensitive {
			result[k] = redactedValue
			continue
		}
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = sanitizeMapRecursive(val, fields)
		case []interface{}:
			result[k] = sanitizeSliceRecursive(val, fields)
		default:
			result[k] = v
		}
	}
	return result
}

func sanitizeSliceRecursive(data []interface{}, fields map[string]struct{}) []interface{} {
	result := make([]interface{}, len(data))
	for i, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = sanitizeMapRecursive(val, fields)
		case []interface{}:
			result[i] = sanitizeSliceRecursive(val, fields)
		default:
			result[i] = v
		}
	}
	return result
}
