package security_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	security "github.com/clario360/platform/internal/security"
)

func TestLogEvent_BasicFields(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	sl := security.NewSecurityLogger(logger, metrics, false)

	event := &security.SecurityEvent{
		EventType:   security.EventSQLInjection,
		Severity:    security.SeverityHigh,
		Description: "test injection attempt",
		Blocked:     true,
		ClientIP:    "10.0.0.1",
		RequestID:   "req-123",
		Path:        "/api/v1/users",
		Method:      "POST",
	}

	sl.LogEvent(event)

	output := buf.String()
	for _, expected := range []string{"sql_injection_attempt", "high", "test injection attempt", "10.0.0.1", "req-123"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected log to contain %q, got: %s", expected, output)
		}
	}
}

func TestLogEvent_TamperProofChaining(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	sl := security.NewSecurityLogger(logger, nil, true)

	event1 := &security.SecurityEvent{
		EventType:   security.EventCSRFFailure,
		Severity:    security.SeverityMedium,
		Description: "first event",
		Blocked:     true,
	}
	sl.LogEvent(event1)

	if event1.EntryHash == "" {
		t.Fatal("expected entry_hash to be set when tamper-proof is enabled")
	}

	event2 := &security.SecurityEvent{
		EventType:   security.EventXSSAttempt,
		Severity:    security.SeverityMedium,
		Description: "second event",
		Blocked:     true,
	}
	sl.LogEvent(event2)

	if event2.EntryHash == "" {
		t.Fatal("expected entry_hash on second event")
	}
	if event1.EntryHash == event2.EntryHash {
		t.Error("chained hashes should differ between events")
	}
}

func TestLogEvent_NoTamperProof(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	sl := security.NewSecurityLogger(logger, nil, false)

	event := &security.SecurityEvent{
		EventType:   security.EventRateLimited,
		Severity:    security.SeverityLow,
		Description: "rate limited",
	}
	sl.LogEvent(event)

	if event.EntryHash != "" {
		t.Error("expected empty entry_hash when tamper-proof is disabled")
	}
}

func TestLogEvent_MetricsIncremented(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	sl := security.NewSecurityLogger(logger, metrics, false)

	event := &security.SecurityEvent{
		EventType:   security.EventBOLAAttempt,
		Severity:    security.SeverityCritical,
		Description: "BOLA attempt",
		Blocked:     true,
		Category:    "ownership",
	}
	sl.LogEvent(event)

	// Verify metrics were collected (no panic, registry has data)
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if len(families) == 0 {
		t.Error("expected metrics to be registered")
	}
}

func TestLogFromRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	sl := security.NewSecurityLogger(logger, nil, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", nil)
	req.Header.Set("X-Request-ID", "req-abc")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	sl.LogFromRequest(req, security.EventSQLInjection, security.SeverityHigh, "SQL injection blocked", true)

	output := buf.String()
	for _, expected := range []string{"sql_injection_attempt", "/api/v1/assets", "POST", "192.168.1.100", "req-abc"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected log to contain %q, got: %s", expected, output)
		}
	}
}

func TestLogFromRequest_WithAuthContext(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	sl := security.NewSecurityLogger(logger, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	ctx := auth.WithUser(req.Context(), &auth.ContextUser{
		ID:       "user-42",
		TenantID: "tenant-99",
		Roles:    []string{"analyst"},
	})
	req = req.WithContext(ctx)

	sl.LogFromRequest(req, security.EventBFLAAttempt, security.SeverityHigh, "unauthorized function access", true)

	output := buf.String()
	if !strings.Contains(output, "user-42") {
		t.Errorf("expected user_id in log, got: %s", output)
	}
	if !strings.Contains(output, "tenant-99") {
		t.Errorf("expected tenant_id in log, got: %s", output)
	}
}

func TestLogFieldViolation(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	sl := security.NewSecurityLogger(logger, nil, false)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	sl.LogFieldViolation(req, security.EventXSSAttempt, "username", "script_tag")

	output := buf.String()
	if !strings.Contains(output, "username") {
		t.Errorf("expected field_name in log, got: %s", output)
	}
	if !strings.Contains(output, "script_tag") {
		t.Errorf("expected category in log, got: %s", output)
	}
}

func TestExtractClientIP_XForwardedFor(t *testing.T) {
	ip := security.ClientIPHash("10.0.0.1")
	if ip == "" {
		t.Error("expected non-empty IP hash")
	}
}

func TestIsPIIField(t *testing.T) {
	tests := []struct {
		field    string
		expected bool
	}{
		{"email", true},
		{"password", true},
		{"name", true},
		{"api_key", true},
		{"description", false},
		{"title", false},
		{"severity", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := security.IsPIIField(tt.field); got != tt.expected {
				t.Errorf("IsPIIField(%q) = %v, want %v", tt.field, got, tt.expected)
			}
		})
	}
}

func TestRedactPII(t *testing.T) {
	redacted := security.RedactPII("email", "user@example.com")
	if !strings.HasPrefix(redacted, "[REDACTED:") {
		t.Errorf("expected redacted PII, got: %s", redacted)
	}

	plain := security.RedactPII("description", "not sensitive")
	if plain != "not sensitive" {
		t.Errorf("expected non-PII field to pass through, got: %s", plain)
	}
}

func TestRequestContextFields(t *testing.T) {
	ctx := auth.WithUser(context.Background(), &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Roles:    []string{"admin"},
	})

	fields := security.RequestContextFields(ctx)
	if fields["user_id"] != "user-1" {
		t.Errorf("expected user_id=user-1, got %s", fields["user_id"])
	}
	if fields["tenant_id"] != "tenant-1" {
		t.Errorf("expected tenant_id=tenant-1, got %s", fields["tenant_id"])
	}
}

func TestRequestContextFields_Empty(t *testing.T) {
	fields := security.RequestContextFields(context.Background())
	if len(fields) != 0 {
		t.Errorf("expected empty fields for context without user, got %v", fields)
	}
}
