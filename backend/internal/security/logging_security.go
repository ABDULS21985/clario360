package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// SecurityEventType categorizes security events.
type SecurityEventType string

const (
	EventCSRFFailure       SecurityEventType = "csrf_failure"
	EventSQLInjection      SecurityEventType = "sql_injection_attempt"
	EventXSSAttempt        SecurityEventType = "xss_attempt"
	EventRateLimited       SecurityEventType = "rate_limited"
	EventAccountLockout    SecurityEventType = "account_lockout"
	EventBOLAAttempt       SecurityEventType = "bola_attempt"
	EventBFLAAttempt       SecurityEventType = "bfla_attempt"
	EventMassAssignment    SecurityEventType = "mass_assignment_attempt"
	EventSSRFBlocked       SecurityEventType = "ssrf_blocked"
	EventPathTraversal     SecurityEventType = "path_traversal_attempt"
	EventFileBlocked       SecurityEventType = "file_upload_blocked"
	EventSessionFixation   SecurityEventType = "session_fixation_attempt"
	EventBruteForce        SecurityEventType = "brute_force_detected"
	EventInvalidToken      SecurityEventType = "invalid_token"
	EventEscalation        SecurityEventType = "security_escalation"
	EventAuthFailure       SecurityEventType = "auth_failure"
	EventCryptoViolation   SecurityEventType = "crypto_violation"
	EventContentTypeReject SecurityEventType = "content_type_rejected"
)

// SecurityEventSeverity categorizes event severity.
type SecurityEventSeverity string

const (
	SeverityLow      SecurityEventSeverity = "low"
	SeverityMedium   SecurityEventSeverity = "medium"
	SeverityHigh     SecurityEventSeverity = "high"
	SeverityCritical SecurityEventSeverity = "critical"
)

// SecurityEvent represents a structured security event for logging and alerting.
type SecurityEvent struct {
	Timestamp   time.Time             `json:"timestamp"`
	EventType   SecurityEventType     `json:"event_type"`
	Severity    SecurityEventSeverity `json:"severity"`
	RequestID   string                `json:"request_id,omitempty"`
	ClientIP    string                `json:"client_ip,omitempty"`
	UserID      string                `json:"user_id,omitempty"`
	TenantID    string                `json:"tenant_id,omitempty"`
	Path        string                `json:"path,omitempty"`
	Method      string                `json:"method,omitempty"`
	Rule        string                `json:"rule,omitempty"`
	Description string                `json:"description,omitempty"`
	FieldName   string                `json:"field_name,omitempty"`
	Category    string                `json:"category,omitempty"`
	Blocked     bool                  `json:"blocked"`
	EntryHash   string                `json:"entry_hash,omitempty"`
}

// SecurityLogger provides structured, tamper-evident security event logging.
type SecurityLogger struct {
	logger       zerolog.Logger
	metrics      *Metrics
	lastHash     string
	enableChain  bool
}

// NewSecurityLogger creates a security logger with optional tamper-evident chaining.
func NewSecurityLogger(logger zerolog.Logger, metrics *Metrics, enableTamperProof bool) *SecurityLogger {
	return &SecurityLogger{
		logger:      logger.With().Str("component", "security").Logger(),
		metrics:     metrics,
		enableChain: enableTamperProof,
	}
}

// LogEvent logs a structured security event and increments metrics.
func (sl *SecurityLogger) LogEvent(event *SecurityEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if sl.enableChain {
		event.EntryHash = sl.computeHash(event)
		sl.lastHash = event.EntryHash
	}

	logEvent := sl.logger.Warn().
		Str("event_type", string(event.EventType)).
		Str("severity", string(event.Severity)).
		Bool("blocked", event.Blocked).
		Time("event_time", event.Timestamp)

	if event.RequestID != "" {
		logEvent = logEvent.Str("request_id", event.RequestID)
	}
	if event.ClientIP != "" {
		logEvent = logEvent.Str("client_ip", event.ClientIP)
	}
	if event.UserID != "" {
		logEvent = logEvent.Str("user_id", event.UserID)
	}
	if event.TenantID != "" {
		logEvent = logEvent.Str("tenant_id", event.TenantID)
	}
	if event.Path != "" {
		logEvent = logEvent.Str("path", event.Path)
	}
	if event.Method != "" {
		logEvent = logEvent.Str("method", event.Method)
	}
	if event.Rule != "" {
		logEvent = logEvent.Str("rule", event.Rule)
	}
	if event.FieldName != "" {
		logEvent = logEvent.Str("field_name", event.FieldName)
	}
	if event.Category != "" {
		logEvent = logEvent.Str("category", event.Category)
	}
	if event.EntryHash != "" {
		logEvent = logEvent.Str("entry_hash", event.EntryHash)
	}

	logEvent.Msg(event.Description)

	// Increment metrics
	if sl.metrics != nil {
		sl.metrics.SecurityEventsTotal.WithLabelValues(string(event.Severity), string(event.EventType)).Inc()
		if event.Blocked {
			sl.metrics.BlockedRequests.WithLabelValues(string(event.EventType), event.Category).Inc()
		}
	}
}

// LogFromRequest creates and logs a security event from an HTTP request context.
func (sl *SecurityLogger) LogFromRequest(r *http.Request, eventType SecurityEventType, severity SecurityEventSeverity, description string, blocked bool) {
	event := &SecurityEvent{
		EventType:   eventType,
		Severity:    severity,
		Description: description,
		Blocked:     blocked,
		Path:        r.URL.Path,
		Method:      r.Method,
		ClientIP:    extractClientIP(r),
		RequestID:   r.Header.Get("X-Request-ID"),
	}

	if user := auth.UserFromContext(r.Context()); user != nil {
		event.UserID = user.ID
		event.TenantID = user.TenantID
	} else if tenantID := auth.TenantFromContext(r.Context()); tenantID != "" {
		event.TenantID = tenantID
	}

	sl.LogEvent(event)
}

// LogFieldViolation logs a field-specific security event.
func (sl *SecurityLogger) LogFieldViolation(r *http.Request, eventType SecurityEventType, fieldName, category string) {
	event := &SecurityEvent{
		EventType:   eventType,
		Severity:    SeverityMedium,
		Description: fmt.Sprintf("security violation detected on field"),
		Blocked:     true,
		Path:        r.URL.Path,
		Method:      r.Method,
		ClientIP:    extractClientIP(r),
		RequestID:   r.Header.Get("X-Request-ID"),
		FieldName:   fieldName,
		Category:    category,
	}

	if user := auth.UserFromContext(r.Context()); user != nil {
		event.UserID = user.ID
		event.TenantID = user.TenantID
	}

	sl.LogEvent(event)
}

// computeHash creates a SHA-256 hash chain entry for tamper evidence.
func (sl *SecurityLogger) computeHash(event *SecurityEvent) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		sl.lastHash,
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.Severity,
		event.RequestID,
		event.ClientIP,
		event.Description,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// extractClientIP extracts the real client IP from the request.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// ClientIPHash returns a SHA-256 hash of the client IP for privacy-safe storage.
func ClientIPHash(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}

// RequestContextFields extracts common fields from a request context for logging.
func RequestContextFields(ctx context.Context) map[string]string {
	fields := make(map[string]string)
	if user := auth.UserFromContext(ctx); user != nil {
		fields["user_id"] = user.ID
		fields["tenant_id"] = user.TenantID
	}
	if tenantID := auth.TenantFromContext(ctx); tenantID != "" {
		fields["tenant_id"] = tenantID
	}
	return fields
}

// piiFields lists fields that should never be logged in raw form.
var piiFields = map[string]bool{
	"email":         true,
	"phone":         true,
	"ssn":           true,
	"name":          true,
	"address":       true,
	"date_of_birth": true,
	"ip_address":    true,
	"password":      true,
	"credit_card":   true,
	"api_key":       true,
	"token":         true,
	"secret":        true,
	"mfa_secret":    true,
}

// IsPIIField returns true if the field name is a known PII field.
func IsPIIField(fieldName string) bool {
	return piiFields[strings.ToLower(fieldName)]
}

// RedactPII replaces PII field values with a hash prefix for correlation.
func RedactPII(fieldName, value string) string {
	if !IsPIIField(fieldName) {
		return value
	}
	hash := sha256.Sum256([]byte(value))
	return "[REDACTED:" + hex.EncodeToString(hash[:4]) + "]"
}
