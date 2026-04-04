package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	policymodel "github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// SIEMFormat defines the output format for SIEM event export.
type SIEMFormat string

const (
	SIEMFormatCEF    SIEMFormat = "CEF"
	SIEMFormatJSON   SIEMFormat = "JSON"
	SIEMFormatSyslog SIEMFormat = "Syslog"
)

// SIEMEvent is a structured security event suitable for ingestion by a SIEM
// platform. Each event maps a policy violation to a normalised event record.
type SIEMEvent struct {
	TenantID             uuid.UUID  `json:"tenant_id"`
	AssetID              uuid.UUID  `json:"asset_id"`
	Classification       string     `json:"classification"`
	Severity             string     `json:"severity"`
	FindingType          string     `json:"finding_type"`
	ComplianceFrameworks []string   `json:"compliance_frameworks,omitempty"`
	RecommendedAction    string     `json:"recommended_action"`
	Timestamp            time.Time  `json:"timestamp"`
	Format               SIEMFormat `json:"format"`
}

// cefSeverityMap maps textual severity levels to CEF integer severity (0-10).
var cefSeverityMap = map[string]int{
	"low":      3,
	"medium":   5,
	"high":     7,
	"critical": 10,
}

// SIEMExporter converts DSPM policy violations into structured SIEM events
// for downstream ingestion by security information and event management platforms.
type SIEMExporter struct {
	logger zerolog.Logger
}

// NewSIEMExporter constructs a SIEMExporter.
func NewSIEMExporter(logger zerolog.Logger) *SIEMExporter {
	return &SIEMExporter{
		logger: logger.With().Str("component", "siem_exporter").Logger(),
	}
}

// ExportFindings converts a slice of policy violations into SIEM events.
// Each violation produces one event with a recommended remediation action
// derived from the violation category and severity.
func (se *SIEMExporter) ExportFindings(ctx context.Context, tenantID uuid.UUID, violations []policymodel.PolicyViolation) ([]SIEMEvent, error) {
	if len(violations) == 0 {
		return nil, nil
	}

	se.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("violation_count", len(violations)).
		Msg("exporting findings to SIEM events")

	now := time.Now().UTC()
	events := make([]SIEMEvent, 0, len(violations))

	for i := range violations {
		v := &violations[i]

		event := SIEMEvent{
			TenantID:             tenantID,
			AssetID:              v.AssetID,
			Classification:       v.Classification,
			Severity:             v.Severity,
			FindingType:          v.Category,
			ComplianceFrameworks: v.ComplianceFrameworks,
			RecommendedAction:    recommendedAction(v.Category, v.Severity),
			Timestamp:            now,
			Format:               SIEMFormatJSON,
		}

		events = append(events, event)
	}

	se.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("events_exported", len(events)).
		Msg("SIEM event export complete")

	return events, nil
}

// FormatCEF renders a SIEMEvent as a Common Event Format (CEF) string.
// CEF format: CEF:0|Vendor|Product|Version|SignatureID|Name|Severity|Extension
func (se *SIEMExporter) FormatCEF(event SIEMEvent) string {
	severity := cefSeverityMap[strings.ToLower(event.Severity)]
	if severity == 0 {
		severity = 5 // default to medium
	}

	// Escape CEF special characters in fields.
	name := cefEscape(fmt.Sprintf("DSPM %s violation on %s", event.FindingType, event.AssetID.String()))

	extensions := []string{
		fmt.Sprintf("tenantId=%s", event.TenantID.String()),
		fmt.Sprintf("assetId=%s", event.AssetID.String()),
		fmt.Sprintf("classification=%s", cefEscape(event.Classification)),
		fmt.Sprintf("findingType=%s", cefEscape(event.FindingType)),
		fmt.Sprintf("recommendedAction=%s", cefEscape(event.RecommendedAction)),
		fmt.Sprintf("rt=%d", event.Timestamp.UnixMilli()),
	}

	if len(event.ComplianceFrameworks) > 0 {
		extensions = append(extensions, fmt.Sprintf("complianceFrameworks=%s", cefEscape(strings.Join(event.ComplianceFrameworks, ","))))
	}

	return fmt.Sprintf("CEF:0|Clario360|DSPM|1.0|%s|%s|%d|%s",
		cefEscape(event.FindingType),
		name,
		severity,
		strings.Join(extensions, " "),
	)
}

// FormatJSON serialises a SIEMEvent as an indented JSON byte slice.
func (se *SIEMExporter) FormatJSON(event SIEMEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("siem exporter: marshal json: %w", err)
	}
	return data, nil
}

// recommendedAction derives a human-readable remediation action from the
// violation category and severity.
func recommendedAction(category, severity string) string {
	cat := strings.ToLower(category)
	sev := strings.ToLower(severity)

	switch cat {
	case "encryption":
		if sev == "critical" || sev == "high" {
			return "Immediately enable encryption at rest and in transit for the affected data asset."
		}
		return "Enable encryption for the affected data asset during the next maintenance window."

	case "retention":
		if sev == "critical" {
			return "Initiate emergency data disposition review. Asset significantly exceeds retention policy."
		}
		return "Review data retention compliance and archive or delete data per policy."

	case "exposure":
		if sev == "critical" || sev == "high" {
			return "Immediately restrict network access. Isolate internet-facing sensitive data asset."
		}
		return "Review and tighten network exposure controls for the affected data asset."

	case "pii_protection":
		if sev == "critical" || sev == "high" {
			return "Immediately apply PII protection controls: encryption, access restrictions, and audit logging."
		}
		return "Apply PII protection controls including encryption and access logging."

	case "access_review":
		return "Conduct access review for the affected data asset. Revoke stale or overprivileged permissions."

	case "classification":
		return "Re-classify data asset and apply controls appropriate to the updated classification level."

	case "backup":
		return "Configure backup for the affected data asset per organizational backup policy."

	case "audit_logging":
		return "Enable audit logging for the affected data asset to ensure compliance traceability."

	default:
		return fmt.Sprintf("Investigate and remediate %s violation on the affected data asset.", category)
	}
}

// cefEscape escapes characters that have special meaning in CEF format:
// backslash, pipe, and equals sign.
func cefEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `|`, `\|`)
	s = strings.ReplaceAll(s, `=`, `\=`)
	return s
}
