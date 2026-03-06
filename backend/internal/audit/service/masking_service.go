package service

import (
	"encoding/json"
	"strings"

	"github.com/clario360/platform/internal/audit/model"
)

// MaskingService applies role-based PII masking to audit entries.
type MaskingService struct{}

// NewMaskingService creates a new MaskingService.
func NewMaskingService() *MaskingService {
	return &MaskingService{}
}

// MaskEntry applies field-level masking based on the caller's role.
// Returns a COPY — never mutates the original.
//
// Role-based masking rules:
//
//	super_admin, compliance_officer → no masking
//	auditor                        → mask ip_address, user_agent, metadata.session_id
//	tenant_admin                   → mask ip_address, user_agent
//	all other roles                → mask ip_address, user_agent, user_email (show domain only)
func (s *MaskingService) MaskEntry(entry *model.AuditEntry, roles []string) model.AuditEntry {
	// Copy the entry
	masked := *entry

	// Determine masking level
	level := s.maskingLevel(roles)
	if level == maskNone {
		return masked
	}

	// All masked roles get IP and UA masked
	masked.IPAddress = MaskIP(entry.IPAddress)
	masked.UserAgent = MaskUserAgent(entry.UserAgent)

	if level == maskAuditor {
		// Also mask metadata.session_id
		masked.Metadata = maskMetadataField(entry.Metadata, "session_id")
	}

	if level == maskFull {
		// Also mask email
		masked.UserEmail = MaskEmail(entry.UserEmail)
	}

	return masked
}

// MaskEntries applies masking to a slice of entries.
func (s *MaskingService) MaskEntries(entries []model.AuditEntry, roles []string) []model.AuditEntry {
	result := make([]model.AuditEntry, len(entries))
	for i := range entries {
		result[i] = s.MaskEntry(&entries[i], roles)
	}
	return result
}

type maskLevel int

const (
	maskNone    maskLevel = iota
	maskTenant            // tenant_admin: IP, UA
	maskAuditor           // auditor: IP, UA, metadata.session_id
	maskFull              // others: IP, UA, email
)

func (s *MaskingService) maskingLevel(roles []string) maskLevel {
	for _, role := range roles {
		switch role {
		case "super_admin", "compliance_officer":
			return maskNone
		}
	}
	for _, role := range roles {
		if role == "auditor" {
			return maskAuditor
		}
	}
	for _, role := range roles {
		if role == "tenant_admin" {
			return maskTenant
		}
	}
	return maskFull
}

// MaskIP masks an IP address, showing only the first octet.
// "192.168.1.100" → "192.*.*.*"
func MaskIP(ip string) string {
	if ip == "" {
		return ""
	}
	parts := strings.SplitN(ip, ".", 2)
	if len(parts) < 2 {
		return ip
	}
	return parts[0] + ".*.*.*"
}

// MaskEmail masks an email address, showing only the domain.
// "john@acme.com" → "****@acme.com"
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) < 2 {
		return "****"
	}
	return "****@" + parts[1]
}

// MaskUserAgent masks a user agent string, showing only the first 20 chars.
func MaskUserAgent(ua string) string {
	if ua == "" {
		return ""
	}
	if len(ua) <= 20 {
		return ua
	}
	return ua[:20] + "..."
}

// maskMetadataField redacts a specific field in the metadata JSON.
func maskMetadataField(metadata json.RawMessage, field string) json.RawMessage {
	if len(metadata) == 0 {
		return metadata
	}

	var m map[string]interface{}
	if err := json.Unmarshal(metadata, &m); err != nil {
		return metadata
	}

	if _, ok := m[field]; ok {
		m[field] = "***"
		result, err := json.Marshal(m)
		if err != nil {
			return metadata
		}
		return result
	}

	return metadata
}
