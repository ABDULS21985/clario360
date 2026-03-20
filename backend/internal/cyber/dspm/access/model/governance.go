package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AccessPolicy defines a governance policy for data access.
type AccessPolicy struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	PolicyType  string          `json:"policy_type"`
	RuleConfig  json.RawMessage `json:"rule_config"`
	Enforcement string          `json:"enforcement"`
	Severity    string          `json:"severity"`
	Enabled     bool            `json:"enabled"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// PolicyViolation describes a single violation of an access policy.
type PolicyViolation struct {
	PolicyID       uuid.UUID  `json:"policy_id"`
	PolicyName     string     `json:"policy_name"`
	PolicyType     string     `json:"policy_type"`
	Enforcement    string     `json:"enforcement"`
	Severity       string     `json:"severity"`
	IdentityType   string     `json:"identity_type"`
	IdentityID     string     `json:"identity_id"`
	IdentityName   string     `json:"identity_name"`
	MappingID      *uuid.UUID `json:"mapping_id,omitempty"`
	DataAssetName  string     `json:"data_asset_name,omitempty"`
	ViolationType  string     `json:"violation_type"`
	Description    string     `json:"description"`
	DetectedAt     time.Time  `json:"detected_at"`
}

// Recommendation is an actionable suggestion for reducing access risk.
type Recommendation struct {
	Type           string    `json:"type"`
	MappingID      uuid.UUID `json:"mapping_id"`
	IdentityID     string    `json:"identity_id"`
	IdentityName   string    `json:"identity_name"`
	DataAssetName  string    `json:"data_asset_name"`
	PermissionType string    `json:"permission_type"`
	Reason         string    `json:"reason"`
	Impact         string    `json:"impact"`
	RiskReduction  float64   `json:"risk_reduction"`
}

// Campaign represents an access certification campaign.
type Campaign struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	Name            string         `json:"name"`
	Status          string         `json:"status"`
	Scope           CampaignScope  `json:"scope"`
	TotalItems      int            `json:"total_items"`
	ReviewedItems   int            `json:"reviewed_items"`
	ApprovedItems   int            `json:"approved_items"`
	RevokedItems    int            `json:"revoked_items"`
	Deadline        time.Time      `json:"deadline"`
	CreatedBy       uuid.UUID      `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
}

// CampaignScope defines the scope of a certification campaign.
type CampaignScope struct {
	MinClassification string `json:"min_classification"`
	IdentityTypes     []string `json:"identity_types,omitempty"`
}

// CampaignParams are the parameters for creating a new campaign.
type CampaignParams struct {
	Name              string   `json:"name"`
	MinClassification string   `json:"min_classification"`
	IdentityTypes     []string `json:"identity_types,omitempty"`
	DeadlineDays      int      `json:"deadline_days"`
}

// CampaignReviewItem is a single item in a certification campaign.
type CampaignReviewItem struct {
	ID             uuid.UUID  `json:"id"`
	CampaignID     uuid.UUID  `json:"campaign_id"`
	MappingID      uuid.UUID  `json:"mapping_id"`
	IdentityID     string     `json:"identity_id"`
	IdentityName   string     `json:"identity_name"`
	DataAssetName  string     `json:"data_asset_name"`
	PermissionType string     `json:"permission_type"`
	Decision       string     `json:"decision"`
	ReviewerID     *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
}

// MaxIdleDaysConfig holds the configuration for max_idle_days policy.
type MaxIdleDaysConfig struct {
	MaxDays           int    `json:"max_days"`
	ClassificationMin string `json:"classification_min"`
	AutoRevoke        bool   `json:"auto_revoke"`
}

// ClassificationRestrictConfig holds the configuration for classification_restrict policy.
type ClassificationRestrictConfig struct {
	Classification       string   `json:"classification"`
	AllowedIdentityTypes []string `json:"allowed_identity_types"`
	RequireApproval      bool     `json:"require_approval"`
}

// SeparationOfDutiesConfig holds the configuration for separation_of_duties policy.
type SeparationOfDutiesConfig struct {
	ConflictingPermissions [][]string `json:"conflicting_permissions"`
	ClassificationMin      string     `json:"classification_min"`
}

// TimeBoundAccessConfig holds the configuration for time_bound_access policy.
type TimeBoundAccessConfig struct {
	ClassificationMin string `json:"classification_min"`
	MaxGrantDays      int    `json:"max_grant_days"`
}

// BlastRadiusLimitConfig holds the configuration for blast_radius_limit policy.
type BlastRadiusLimitConfig struct {
	MaxScore      float64 `json:"max_score"`
	AlertSeverity string  `json:"alert_severity"`
}

// PeriodicReviewConfig holds the configuration for periodic_review policy.
type PeriodicReviewConfig struct {
	ReviewIntervalDays int    `json:"review_interval_days"`
	ClassificationMin  string `json:"classification_min"`
}

// ClassificationRank maps classification strings to a sortable rank.
func ClassificationRank(classification string) int {
	switch classification {
	case "restricted":
		return 4
	case "confidential":
		return 3
	case "internal":
		return 2
	case "public":
		return 1
	default:
		return 0
	}
}
