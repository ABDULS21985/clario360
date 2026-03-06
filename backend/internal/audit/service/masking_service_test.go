package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/clario360/platform/internal/audit/model"
)

func makeMaskTestEntry() *model.AuditEntry {
	userID := "user-123"
	return &model.AuditEntry{
		ID:           "entry-1",
		TenantID:     "tenant-1",
		UserID:       &userID,
		UserEmail:    "john.doe@acme.com",
		Service:      "iam-service",
		Action:       "user.login.success",
		Severity:     "info",
		ResourceType: "user",
		ResourceID:   "user-123",
		IPAddress:    "192.168.1.100",
		UserAgent:    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		Metadata:     json.RawMessage(`{"session_id": "sess-abc", "other": "value"}`),
		EventID:      "evt-1",
		CreatedAt:    time.Now(),
	}
}

func TestMaskIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.100", "192.*.*.*"},
		{"10.0.0.1", "10.*.*.*"},
		{"", ""},
		{"localhost", "localhost"},
	}

	for _, tt := range tests {
		result := MaskIP(tt.input)
		if result != tt.expected {
			t.Errorf("MaskIP(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"john@acme.com", "****@acme.com"},
		{"admin@company.org", "****@company.org"},
		{"", ""},
		{"noemail", "****"},
	}

	for _, tt := range tests {
		result := MaskEmail(tt.input)
		if result != tt.expected {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskUserAgent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36", "Mozilla/5.0 (X11; Li..."},
		{"short", "short"},
		{"", ""},
		{"exactly20characters!", "exactly20characters!"},
	}

	for _, tt := range tests {
		result := MaskUserAgent(tt.input)
		if result != tt.expected {
			t.Errorf("MaskUserAgent(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskEntry_SuperAdmin_NoMasking(t *testing.T) {
	svc := NewMaskingService()
	entry := makeMaskTestEntry()

	masked := svc.MaskEntry(entry, []string{"super_admin"})

	if masked.IPAddress != entry.IPAddress {
		t.Errorf("super_admin should see full IP, got %s", masked.IPAddress)
	}
	if masked.UserAgent != entry.UserAgent {
		t.Errorf("super_admin should see full user agent, got %s", masked.UserAgent)
	}
	if masked.UserEmail != entry.UserEmail {
		t.Errorf("super_admin should see full email, got %s", masked.UserEmail)
	}
}

func TestMaskEntry_ComplianceOfficer_NoMasking(t *testing.T) {
	svc := NewMaskingService()
	entry := makeMaskTestEntry()

	masked := svc.MaskEntry(entry, []string{"compliance_officer"})

	if masked.IPAddress != entry.IPAddress {
		t.Errorf("compliance_officer should see full IP, got %s", masked.IPAddress)
	}
}

func TestMaskEntry_TenantAdmin_MasksIPAndUA(t *testing.T) {
	svc := NewMaskingService()
	entry := makeMaskTestEntry()

	masked := svc.MaskEntry(entry, []string{"tenant_admin"})

	if masked.IPAddress != "192.*.*.*" {
		t.Errorf("tenant_admin should see masked IP, got %s", masked.IPAddress)
	}
	if masked.UserAgent == entry.UserAgent {
		t.Error("tenant_admin should see masked user agent")
	}
	if masked.UserEmail != entry.UserEmail {
		t.Error("tenant_admin should see full email")
	}
}

func TestMaskEntry_Auditor_MasksIPUAAndSessionID(t *testing.T) {
	svc := NewMaskingService()
	entry := makeMaskTestEntry()

	masked := svc.MaskEntry(entry, []string{"auditor"})

	if masked.IPAddress != "192.*.*.*" {
		t.Errorf("auditor should see masked IP, got %s", masked.IPAddress)
	}
	if masked.UserAgent == entry.UserAgent {
		t.Error("auditor should see masked user agent")
	}

	// Verify session_id is masked in metadata
	var meta map[string]interface{}
	if err := json.Unmarshal(masked.Metadata, &meta); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}
	if meta["session_id"] != "***" {
		t.Errorf("auditor should see masked session_id, got %v", meta["session_id"])
	}
	if meta["other"] != "value" {
		t.Error("other metadata fields should not be masked")
	}
}

func TestMaskEntry_ViewerRole_MasksAll(t *testing.T) {
	svc := NewMaskingService()
	entry := makeMaskTestEntry()

	masked := svc.MaskEntry(entry, []string{"viewer"})

	if masked.IPAddress != "192.*.*.*" {
		t.Errorf("viewer should see masked IP, got %s", masked.IPAddress)
	}
	if masked.UserAgent == entry.UserAgent {
		t.Error("viewer should see masked user agent")
	}
	if masked.UserEmail != "****@acme.com" {
		t.Errorf("viewer should see masked email, got %s", masked.UserEmail)
	}
}

func TestMaskEntry_ReturnsCopy(t *testing.T) {
	svc := NewMaskingService()
	entry := makeMaskTestEntry()
	originalIP := entry.IPAddress

	_ = svc.MaskEntry(entry, []string{"viewer"})

	// Original should not be modified
	if entry.IPAddress != originalIP {
		t.Error("MaskEntry should return a copy — original entry was modified")
	}
}

func TestMaskEntries_AppliesMaskingToAll(t *testing.T) {
	svc := NewMaskingService()
	entries := []model.AuditEntry{
		*makeMaskTestEntry(),
		*makeMaskTestEntry(),
	}
	entries[1].IPAddress = "10.0.0.1"

	masked := svc.MaskEntries(entries, []string{"viewer"})

	if len(masked) != 2 {
		t.Fatalf("expected 2 masked entries, got %d", len(masked))
	}
	if masked[0].IPAddress != "192.*.*.*" {
		t.Errorf("expected masked IP for entry 0, got %s", masked[0].IPAddress)
	}
	if masked[1].IPAddress != "10.*.*.*" {
		t.Errorf("expected masked IP for entry 1, got %s", masked[1].IPAddress)
	}
}
