package model

import (
	"time"

	"github.com/google/uuid"
)

// AccessMapping represents a single identity-to-data-asset permission mapping.
type AccessMapping struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	IdentityType       string     `json:"identity_type"`
	IdentityID         string     `json:"identity_id"`
	IdentityName       string     `json:"identity_name,omitempty"`
	IdentitySource     string     `json:"identity_source"`
	DataAssetID        uuid.UUID  `json:"data_asset_id"`
	DataAssetName      string     `json:"data_asset_name,omitempty"`
	DataClassification string     `json:"data_classification,omitempty"`
	PermissionType     string     `json:"permission_type"`
	PermissionSource   string     `json:"permission_source"`
	PermissionPath     []string   `json:"permission_path,omitempty"`
	IsWildcard         bool       `json:"is_wildcard"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	UsageCount30d      int        `json:"usage_count_30d"`
	UsageCount90d      int        `json:"usage_count_90d"`
	IsStale            bool       `json:"is_stale"`
	SensitivityWeight  float64    `json:"sensitivity_weight"`
	AccessRiskScore    float64    `json:"access_risk_score"`
	Status             string     `json:"status"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	DiscoveredAt       time.Time  `json:"discovered_at"`
	LastVerifiedAt     time.Time  `json:"last_verified_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// IdentityProfile is an aggregated risk profile for an identity across all its access mappings.
type IdentityProfile struct {
	ID                    uuid.UUID              `json:"id"`
	TenantID              uuid.UUID              `json:"tenant_id"`
	IdentityType          string                 `json:"identity_type"`
	IdentityID            string                 `json:"identity_id"`
	IdentityName          string                 `json:"identity_name,omitempty"`
	IdentityEmail         string                 `json:"identity_email,omitempty"`
	IdentitySource        string                 `json:"identity_source"`
	TotalAssetsAccessible int                    `json:"total_assets_accessible"`
	SensitiveAssetsCount  int                    `json:"sensitive_assets_count"`
	PermissionCount       int                    `json:"permission_count"`
	OverprivilegedCount   int                    `json:"overprivileged_count"`
	StalePermissionCount  int                    `json:"stale_permission_count"`
	BlastRadiusScore      float64                `json:"blast_radius_score"`
	BlastRadiusLevel      string                 `json:"blast_radius_level"`
	AccessRiskScore       float64                `json:"access_risk_score"`
	AccessRiskLevel       string                 `json:"access_risk_level"`
	RiskFactors           []IdentityRiskFactor   `json:"risk_factors"`
	LastActivityAt        *time.Time             `json:"last_activity_at,omitempty"`
	AvgDailyAccessCount   float64                `json:"avg_daily_access_count"`
	AccessPatternSummary  map[string]interface{} `json:"access_pattern_summary"`
	Recommendations       []Recommendation       `json:"recommendations"`
	Status                string                 `json:"status"`
	LastReviewAt          *time.Time             `json:"last_review_at,omitempty"`
	NextReviewDue         *time.Time             `json:"next_review_due,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// IdentityRiskFactor describes one factor contributing to identity risk.
type IdentityRiskFactor struct {
	Factor      string  `json:"factor"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Value       float64 `json:"value"`
}

// EffectiveAccess is the union of all permissions for one identity.
type EffectiveAccess struct {
	IdentityType string          `json:"identity_type"`
	IdentityID   string          `json:"identity_id"`
	IdentityName string          `json:"identity_name"`
	Assets       []AssetAccess   `json:"assets"`
	TotalAssets  int             `json:"total_assets"`
	MaxLevel     string          `json:"max_level"`
}

// AssetAccess describes the effective permission on one data asset.
type AssetAccess struct {
	DataAssetID        uuid.UUID `json:"data_asset_id"`
	DataAssetName      string    `json:"data_asset_name"`
	DataClassification string    `json:"data_classification"`
	MaxPermissionLevel string    `json:"max_permission_level"`
	PermissionCount    int       `json:"permission_count"`
	SensitivityWeight  float64   `json:"sensitivity_weight"`
	IsStale            bool      `json:"is_stale"`
}

// PermissionNode is a node in the permission inheritance graph.
type PermissionNode struct {
	Type     string            `json:"type"`
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Children []*PermissionNode `json:"children,omitempty"`
}

// Sensitivity weights based on data classification.
// restricted=10x, confidential=5x, internal=2x, public=1x.
// Weights reflect approximate cost ratios from IBM Cost of a Data Breach Report.
func SensitivityWeight(classification string) float64 {
	switch classification {
	case "restricted":
		return 10.0
	case "confidential":
		return 5.0
	case "internal":
		return 2.0
	case "public":
		return 1.0
	default:
		return 1.0
	}
}

// PermissionBreadth returns a numeric factor representing the breadth of a permission type.
func PermissionBreadth(permissionType string) float64 {
	switch permissionType {
	case "full_control":
		return 5.0
	case "admin":
		return 4.0
	case "alter":
		return 3.0
	case "write", "delete", "create":
		return 2.0
	case "execute":
		return 1.5
	case "read":
		return 1.0
	default:
		return 1.0
	}
}

// PermissionLevel returns a numeric rank for permission comparison (higher = more powerful).
func PermissionLevel(permissionType string) int {
	switch permissionType {
	case "full_control":
		return 8
	case "admin":
		return 7
	case "alter":
		return 6
	case "delete":
		return 5
	case "create":
		return 4
	case "write":
		return 3
	case "execute":
		return 2
	case "read":
		return 1
	default:
		return 0
	}
}

// RiskLevel returns risk level from score using 25/50/75 boundaries.
func RiskLevel(score float64) string {
	switch {
	case score >= 75:
		return "critical"
	case score >= 50:
		return "high"
	case score >= 25:
		return "medium"
	default:
		return "low"
	}
}
