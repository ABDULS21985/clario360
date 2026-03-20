package dspm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

// PostureAssessment captures the evaluated security controls for a DSPM data asset.
type PostureAssessment struct {
	Score              float64
	Findings           []model.DSPMPostureFinding
	EncryptedAtRest    *bool
	EncryptedInTransit *bool
	AccessControlType  *string
	NetworkExposure    *string
	BackupConfigured   *bool
	AuditLogging       *bool
	LastAccessReview   *time.Time
	DatabaseType       *string
	SchemaInfo         map[string]interface{}
	RecordCount        *int64
}

// PostureAssessor evaluates the security posture of a data-bearing asset.
type PostureAssessor struct{}

// NewPostureAssessor creates a posture assessor.
func NewPostureAssessor() *PostureAssessor { return &PostureAssessor{} }

// Assess evaluates the asset against the DSPM control checklist.
func (p *PostureAssessor) Assess(_ context.Context, asset *model.Asset, classification *ClassificationResult) (*PostureAssessment, error) {
	metadata := decodeAssetMetadata(asset)
	assessment := &PostureAssessment{
		Findings:  make([]model.DSPMPostureFinding, 0),
		SchemaInfo: extractSchemaInfo(metadata),
	}

	if recordCount := extractInt64(metadata, "estimated_record_count", "record_count", "rows"); recordCount != nil {
		assessment.RecordCount = recordCount
	}

	assessment.DatabaseType = detectDatabaseType(asset, metadata)

	totalControls := 7.0
	passed := 0.0

	assessment.EncryptedAtRest = extractBool(metadata, "encrypted_at_rest", "encryption", "encryption_at_rest")
	if assessment.EncryptedAtRest != nil && *assessment.EncryptedAtRest {
		passed++
	} else {
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "encryption_at_rest",
			Severity:    severityForClassification(classification, "high"),
			Description: "Data asset does not have encryption at rest confirmed.",
			Guidance:    "Enable native disk, volume, or bucket encryption and verify key management ownership.",
		})
	}

	assessment.EncryptedInTransit = extractBool(metadata, "encrypted_in_transit", "ssl", "tls_enabled")
	if assessment.EncryptedInTransit != nil && *assessment.EncryptedInTransit {
		passed++
	} else {
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "encryption_in_transit",
			Severity:    severityForClassification(classification, "medium"),
			Description: "Encrypted transport was not detected for this data asset.",
			Guidance:    "Require TLS for application and database clients, and disable cleartext administrative access.",
		})
	}

	accessControl := deriveAccessControlType(metadata)
	assessment.AccessControlType = &accessControl
	if accessControl != "none" {
		passed++
	} else {
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "access_control",
			Severity:    severityForClassification(classification, "high"),
			Description: "Access control for this data asset is missing or too weak.",
			Guidance:    "Apply RBAC or ABAC with least-privilege access and periodic entitlement review.",
		})
	}

	networkExposure := deriveNetworkExposure(asset, metadata)
	assessment.NetworkExposure = &networkExposure
	if networkExposure == "internet_facing" {
		severity := "high"
		if asset.Type == model.AssetTypeDatabase || classification.Classification == "restricted" {
			severity = "critical"
		}
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "network_exposure",
			Severity:    severity,
			Description: fmt.Sprintf("Data asset is %s and directly reachable from untrusted networks.", networkExposure),
			Guidance:    "Move the asset behind internal-only or VPN-restricted network boundaries and segment consumer access paths.",
		})
	} else {
		passed++
	}

	assessment.BackupConfigured = extractBool(metadata, "backup_configured", "backups", "backup")
	if assessment.BackupConfigured != nil && *assessment.BackupConfigured {
		passed++
	} else {
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "backup",
			Severity:    severityForClassification(classification, "medium"),
			Description: "Backup configuration is not confirmed for this data asset.",
			Guidance:    "Enable immutable or versioned backups and validate restore coverage for sensitive datasets.",
		})
	}

	assessment.AuditLogging = extractBool(metadata, "audit_logging", "query_logging", "audit_log_enabled")
	if assessment.AuditLogging != nil && *assessment.AuditLogging {
		passed++
	} else {
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "audit_logging",
			Severity:    severityForClassification(classification, "medium"),
			Description: "Query or access audit logging is not enabled for this asset.",
			Guidance:    "Enable query or data-access auditing and forward the logs to central monitoring.",
		})
	}

	assessment.LastAccessReview = extractTime(metadata, "last_access_review")
	if assessment.LastAccessReview != nil && assessment.LastAccessReview.After(time.Now().UTC().AddDate(0, 0, -90)) {
		passed++
	} else {
		assessment.Findings = append(assessment.Findings, model.DSPMPostureFinding{
			Control:     "access_review",
			Severity:    severityForClassification(classification, "medium"),
			Description: "Access review is missing or older than 90 days.",
			Guidance:    "Perform and document a privileged access review for this dataset at least quarterly.",
		})
	}

	assessment.Score = round2((passed / totalControls) * 100)
	return assessment, nil
}

func decodeAssetMetadata(asset *model.Asset) map[string]interface{} {
	if len(asset.Metadata) == 0 {
		return map[string]interface{}{}
	}
	decoded := map[string]interface{}{}
	if err := json.Unmarshal(asset.Metadata, &decoded); err != nil {
		return map[string]interface{}{}
	}
	return decoded
}

func extractSchemaInfo(metadata map[string]interface{}) map[string]interface{} {
	if schema, ok := metadata["schema_info"].(map[string]interface{}); ok {
		return schema
	}
	if columns, ok := metadata["columns"]; ok {
		return map[string]interface{}{"columns": columns}
	}
	return map[string]interface{}{}
}

func extractBool(metadata map[string]interface{}, keys ...string) *bool {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return &typed
		case string:
			lower := strings.ToLower(strings.TrimSpace(typed))
			if lower == "true" || lower == "yes" || lower == "enabled" {
				v := true
				return &v
			}
			if lower == "false" || lower == "no" || lower == "disabled" {
				v := false
				return &v
			}
		}
	}
	return nil
}

func extractString(metadata map[string]interface{}, keys ...string) *string {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			switch typed := value.(type) {
			case string:
				trimmed := strings.TrimSpace(typed)
				if trimmed != "" {
					return &trimmed
				}
			}
		}
	}
	return nil
}

func extractInt64(metadata map[string]interface{}, keys ...string) *int64 {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			v := int64(typed)
			return &v
		case int64:
			return &typed
		case int:
			v := int64(typed)
			return &v
		}
	}
	return nil
}

func extractTime(metadata map[string]interface{}, key string) *time.Time {
	value, ok := metadata[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case string:
		for _, layout := range []string{time.RFC3339, "2006-01-02", time.RFC3339Nano} {
			if parsed, err := time.Parse(layout, typed); err == nil {
				utc := parsed.UTC()
				return &utc
			}
		}
	case float64:
		t := time.Unix(int64(typed), 0).UTC()
		return &t
	}
	return nil
}

func deriveAccessControlType(metadata map[string]interface{}) string {
	if explicit := extractString(metadata, "access_control_type"); explicit != nil {
		return strings.ToLower(*explicit)
	}
	if flag := extractBool(metadata, "abac"); flag != nil && *flag {
		return "abac"
	}
	if flag := extractBool(metadata, "rbac"); flag != nil && *flag {
		return "rbac"
	}
	if flag := extractBool(metadata, "auth_enabled", "acl_enabled"); flag != nil && *flag {
		return "basic"
	}
	return "none"
}

func deriveNetworkExposure(asset *model.Asset, metadata map[string]interface{}) string {
	if explicit := extractString(metadata, "network_exposure"); explicit != nil {
		switch strings.ToLower(*explicit) {
		case "internet_facing", "vpn_accessible", "internal_only":
			return strings.ToLower(*explicit)
		}
	}
	for _, tag := range asset.Tags {
		switch strings.ToLower(tag) {
		case "internet-facing", "public", "dmz":
			return "internet_facing"
		case "vpn", "vpn-accessible":
			return "vpn_accessible"
		}
	}
	return "internal_only"
}

func detectDatabaseType(asset *model.Asset, metadata map[string]interface{}) *string {
	if explicit := extractString(metadata, "database_type", "engine"); explicit != nil {
		lower := strings.ToLower(*explicit)
		return &lower
	}
	for _, tag := range asset.Tags {
		switch strings.ToLower(tag) {
		case "postgres", "mysql", "mongodb", "redis", "s3", "gcs", "blob":
			value := strings.ToLower(tag)
			return &value
		}
	}
	if asset.OS != nil {
		lower := strings.ToLower(*asset.OS)
		if strings.Contains(lower, "postgres") || strings.Contains(lower, "mysql") {
			return &lower
		}
	}
	return nil
}

func severityForClassification(classification *ClassificationResult, fallback string) string {
	if classification == nil {
		return fallback
	}
	switch classification.Classification {
	case "restricted":
		return "critical"
	case "confidential":
		if fallback == "medium" {
			return "high"
		}
		return fallback
	default:
		return fallback
	}
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
