package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PolicyCategory enumerates data policy categories.
type PolicyCategory string

const (
	PolicyCategoryEncryption     PolicyCategory = "encryption"
	PolicyCategoryClassification PolicyCategory = "classification"
	PolicyCategoryRetention      PolicyCategory = "retention"
	PolicyCategoryExposure       PolicyCategory = "exposure"
	PolicyCategoryPIIProtection  PolicyCategory = "pii_protection"
	PolicyCategoryAccessReview   PolicyCategory = "access_review"
	PolicyCategoryBackup         PolicyCategory = "backup"
	PolicyCategoryAuditLogging   PolicyCategory = "audit_logging"
)

// ValidPolicyCategories returns all valid policy categories.
func ValidPolicyCategories() []PolicyCategory {
	return []PolicyCategory{
		PolicyCategoryEncryption, PolicyCategoryClassification,
		PolicyCategoryRetention, PolicyCategoryExposure,
		PolicyCategoryPIIProtection, PolicyCategoryAccessReview,
		PolicyCategoryBackup, PolicyCategoryAuditLogging,
	}
}

// IsValid returns true if the category is a known value.
func (c PolicyCategory) IsValid() bool {
	for _, v := range ValidPolicyCategories() {
		if c == v {
			return true
		}
	}
	return false
}

// PolicyEnforcement defines how a policy is enforced.
type PolicyEnforcement string

const (
	EnforcementAlert         PolicyEnforcement = "alert"
	EnforcementAutoRemediate PolicyEnforcement = "auto_remediate"
	EnforcementBlock         PolicyEnforcement = "block"
)

// IsValid returns true if the enforcement mode is a known value.
func (e PolicyEnforcement) IsValid() bool {
	switch e {
	case EnforcementAlert, EnforcementAutoRemediate, EnforcementBlock:
		return true
	}
	return false
}

// DataPolicy represents a policy-as-code definition for data governance.
type DataPolicy struct {
	ID                   uuid.UUID        `json:"id" db:"id"`
	TenantID             uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	Name                 string           `json:"name" db:"name"`
	Description          string           `json:"description,omitempty" db:"description"`
	Category             PolicyCategory   `json:"category" db:"category"`
	Rule                 json.RawMessage  `json:"rule" db:"rule"`
	Enforcement          PolicyEnforcement `json:"enforcement" db:"enforcement"`
	AutoPlaybookID       string           `json:"auto_playbook_id,omitempty" db:"auto_playbook_id"`
	Severity             string           `json:"severity" db:"severity"`
	ScopeClassification  []string         `json:"scope_classification,omitempty" db:"scope_classification"`
	ScopeAssetTypes      []string         `json:"scope_asset_types,omitempty" db:"scope_asset_types"`
	Enabled              bool             `json:"enabled" db:"enabled"`
	LastEvaluatedAt      *time.Time       `json:"last_evaluated_at,omitempty" db:"last_evaluated_at"`
	ViolationCount       int              `json:"violation_count" db:"violation_count"`
	ComplianceFrameworks []string         `json:"compliance_frameworks,omitempty" db:"compliance_frameworks"`
	CreatedBy            *uuid.UUID       `json:"created_by,omitempty" db:"created_by"`
	CreatedAt            time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at" db:"updated_at"`
}

// PolicyViolation captures a specific policy violation against a data asset.
type PolicyViolation struct {
	PolicyID      uuid.UUID `json:"policy_id"`
	PolicyName    string    `json:"policy_name"`
	Category      string    `json:"category"`
	AssetID       uuid.UUID `json:"asset_id"`
	AssetName     string    `json:"asset_name"`
	AssetType     string    `json:"asset_type"`
	Classification string  `json:"classification"`
	Severity      string    `json:"severity"`
	Description   string    `json:"description"`
	Enforcement   string    `json:"enforcement"`
	ComplianceFrameworks []string `json:"compliance_frameworks,omitempty"`
}

// PolicyImpact is the result of a dry-run policy evaluation.
type PolicyImpact struct {
	TotalAssetsEvaluated int               `json:"total_assets_evaluated"`
	ViolationsFound      int               `json:"violations_found"`
	AffectedAssets       []PolicyViolation `json:"affected_assets"`
}

// EncryptionRule is the typed rule for encryption policies.
type EncryptionRule struct {
	RequireAtRest      bool   `json:"require_at_rest"`
	RequireInTransit   bool   `json:"require_in_transit"`
	ClassificationMin  string `json:"classification_min,omitempty"`
}

// RetentionRule is the typed rule for retention policies.
type RetentionRule struct {
	MaxDays             int      `json:"max_days"`
	ClassificationScope []string `json:"classification_scope,omitempty"`
	Action              string   `json:"action"` // alert, archive, delete
}

// ExposureRule is the typed rule for exposure policies.
type ExposureRule struct {
	MaxExposure       string `json:"max_exposure"` // internal_only, vpn_accessible, internet_facing
	ClassificationMin string `json:"classification_min,omitempty"`
}

// PIIProtectionRule is the typed rule for PII protection policies.
type PIIProtectionRule struct {
	RequireEncryption    bool   `json:"require_encryption"`
	RequireAccessControl string `json:"require_access_control,omitempty"` // rbac, abac, etc.
	RequireAudit         bool   `json:"require_audit"`
}

// AccessReviewRule is the typed rule for access review policies.
type AccessReviewRule struct {
	ReviewIntervalDays int    `json:"review_interval_days"`
	ClassificationMin  string `json:"classification_min,omitempty"`
}

// BackupRule is the typed rule for backup policies.
type BackupRule struct {
	RequiredFor []string `json:"required_for,omitempty"` // classification levels
}

// AuditLoggingRule is the typed rule for audit logging policies.
type AuditLoggingRule struct {
	RequiredFor []string `json:"required_for,omitempty"` // classification levels
}

// ClassificationRule is the typed rule for classification policies.
type ClassificationRule struct {
	AutoEscalate   bool   `json:"auto_escalate"`
	PIIImpliesMin  string `json:"pii_implies_min,omitempty"`
}
