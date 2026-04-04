package model

import (
	"time"

	"github.com/google/uuid"
)

// BlastRadius quantifies the impact if an identity's credentials are compromised.
type BlastRadius struct {
	IdentityID              string                 `json:"identity_id"`
	IdentityName            string                 `json:"identity_name"`
	IdentityType            string                 `json:"identity_type"`
	TotalAssetsExposed      int                    `json:"total_assets_exposed"`
	SensitiveAssets         int                    `json:"sensitive_assets"`
	WeightedScore           float64                `json:"weighted_score"`
	Level                   string                 `json:"level"`
	ExposedClassifications  map[string]int         `json:"exposed_classifications"`
	TopRiskyAssets          []AssetExposure        `json:"top_risky_assets"`
	PrivilegeEscalationPaths []EscalationPath      `json:"privilege_escalation_paths,omitempty"`
	RecommendedActions      []string               `json:"recommended_actions"`
}

// AssetExposure details a single asset's exposure to an identity.
type AssetExposure struct {
	DataAssetID        uuid.UUID `json:"data_asset_id"`
	DataAssetName      string    `json:"data_asset_name"`
	DataClassification string    `json:"data_classification"`
	MaxPermission      string    `json:"max_permission"`
	SensitivityWeight  float64   `json:"sensitivity_weight"`
	RiskContribution   float64   `json:"risk_contribution"`
}

// OverprivilegeResult describes an overprivileged access finding.
type OverprivilegeResult struct {
	MappingID          uuid.UUID  `json:"mapping_id"`
	IdentityType       string     `json:"identity_type"`
	IdentityID         string     `json:"identity_id"`
	IdentityName       string     `json:"identity_name"`
	DataAssetID        uuid.UUID  `json:"data_asset_id"`
	DataAssetName      string     `json:"data_asset_name"`
	DataClassification string     `json:"data_classification"`
	PermissionType     string     `json:"permission_type"`
	PermissionSource   string     `json:"permission_source"`
	UsageCount90d      int        `json:"usage_count_90d"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	Severity           string     `json:"severity"`
	Confidence         float64    `json:"confidence"`
	Recommendation     string     `json:"recommendation"`
	DaysUnused         int        `json:"days_unused"`
}

// StaleAccessResult describes a stale (unused) permission.
type StaleAccessResult struct {
	MappingID          uuid.UUID  `json:"mapping_id"`
	IdentityType       string     `json:"identity_type"`
	IdentityID         string     `json:"identity_id"`
	IdentityName       string     `json:"identity_name"`
	DataAssetID        uuid.UUID  `json:"data_asset_id"`
	DataAssetName      string     `json:"data_asset_name"`
	DataClassification string     `json:"data_classification"`
	PermissionType     string     `json:"permission_type"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	DaysStale          int        `json:"days_stale"`
	SensitivityWeight  float64    `json:"sensitivity_weight"`
}

// EscalationPath describes a path from one permission to higher privileges.
type EscalationPath struct {
	SourcePermission string `json:"source_permission"`
	IntermediateStep string `json:"intermediate_step"`
	TargetEscalated  string `json:"target_escalated"`
	RiskLevel        string `json:"risk_level"`
}

// CrossAssetResult identifies an identity with access spanning multiple sensitive data domains.
type CrossAssetResult struct {
	IdentityType           string   `json:"identity_type"`
	IdentityID             string   `json:"identity_id"`
	IdentityName           string   `json:"identity_name"`
	DistinctClassifications int     `json:"distinct_classifications"`
	DistinctAssetTypes     int      `json:"distinct_asset_types"`
	BreadthScore           int      `json:"breadth_score"`
	SensitiveAssetCount    int      `json:"sensitive_asset_count"`
	Recommendation         string   `json:"recommendation"`
}

// AccessAnomaly describes an anomalous data access pattern.
type AccessAnomaly struct {
	IdentityType string    `json:"identity_type"`
	IdentityID   string    `json:"identity_id"`
	IdentityName string    `json:"identity_name"`
	AnomalyType  string    `json:"anomaly_type"`
	Description  string    `json:"description"`
	Severity     string    `json:"severity"`
	Confidence   float64   `json:"confidence"`
	DetectedAt   time.Time `json:"detected_at"`
}

// AccessRiskRanking is used for the risk ranking endpoint.
type AccessRiskRanking struct {
	Identities []IdentityProfile `json:"identities"`
	Total      int               `json:"total"`
}

// BlastRadiusRanking is used for the blast radius ranking endpoint.
type BlastRadiusRanking struct {
	Identities []IdentityProfile `json:"identities"`
	Total      int               `json:"total"`
}
