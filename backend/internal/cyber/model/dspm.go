package model

import (
	"time"

	"github.com/google/uuid"
)

// DSPMDataAsset is the security posture view of a data-bearing asset.
type DSPMDataAsset struct {
	ID                   uuid.UUID              `json:"id"`
	TenantID             uuid.UUID              `json:"tenant_id"`
	AssetID              uuid.UUID              `json:"asset_id"`
	AssetName            string                 `json:"asset_name,omitempty"`
	AssetType            string                 `json:"asset_type,omitempty"`
	ScanID               *uuid.UUID             `json:"scan_id,omitempty"`
	DataClassification   string                 `json:"data_classification"`
	SensitivityScore     float64                `json:"sensitivity_score"`
	ContainsPII          bool                   `json:"contains_pii"`
	PIITypes             []string               `json:"pii_types"`
	PIIColumnCount       int                    `json:"pii_column_count"`
	EstimatedRecordCount *int64                 `json:"estimated_record_count,omitempty"`
	EncryptedAtRest      *bool                  `json:"encrypted_at_rest,omitempty"`
	EncryptedInTransit   *bool                  `json:"encrypted_in_transit,omitempty"`
	AccessControlType    *string                `json:"access_control_type,omitempty"`
	NetworkExposure      *string                `json:"network_exposure,omitempty"`
	BackupConfigured     *bool                  `json:"backup_configured,omitempty"`
	AuditLogging         *bool                  `json:"audit_logging,omitempty"`
	LastAccessReview     *time.Time             `json:"last_access_review,omitempty"`
	RiskScore            float64                `json:"risk_score"`
	RiskFactors          []DSPMRiskFactor       `json:"risk_factors"`
	PostureScore         float64                `json:"posture_score"`
	PostureFindings      []DSPMPostureFinding   `json:"posture_findings"`
	ConsumerCount        int                    `json:"consumer_count"`
	ProducerCount        int                    `json:"producer_count"`
	DatabaseType         *string                `json:"database_type,omitempty"`
	SchemaInfo           map[string]interface{} `json:"schema_info,omitempty"`
	Metadata             map[string]interface{} `json:"metadata"`
	LastScannedAt        *time.Time             `json:"last_scanned_at,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// DSPMRiskFactor describes a contributing factor to the DSPM risk score.
type DSPMRiskFactor struct {
	Factor      string  `json:"factor"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Value       float64 `json:"value"`
}

// DSPMPostureFinding is a security gap found during posture assessment.
type DSPMPostureFinding struct {
	Control     string `json:"control"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Guidance    string `json:"guidance"`
}

// DSPMScan records the outcome of a DSPM scan session.
type DSPMScan struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	Status         string     `json:"status"`
	AssetsScanned  int        `json:"assets_scanned"`
	PIIAssetsFound int        `json:"pii_assets_found"`
	HighRiskFound  int        `json:"high_risk_found"`
	FindingsCount  int        `json:"findings_count"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	DurationMs     *int64     `json:"duration_ms,omitempty"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
}

// DSPMScanResult summarizes one completed scan run.
type DSPMScanResult struct {
	Scan           *DSPMScan `json:"scan"`
	AssetsScanned  int       `json:"assets_scanned"`
	PIIAssetsFound int       `json:"pii_assets_found"`
	HighRiskFound  int       `json:"high_risk_found"`
	FindingsCount  int       `json:"findings_count"`
}

// DSPMDashboard aggregates DSPM metrics for the dashboard widget.
type DSPMDashboard struct {
	TotalDataAssets         int             `json:"total_data_assets"`
	PIIAssetsCount          int             `json:"pii_assets_count"`
	HighRiskAssetsCount     int             `json:"high_risk_assets_count"`
	AvgPostureScore         float64         `json:"avg_posture_score"`
	AvgRiskScore            float64         `json:"avg_risk_score"`
	UnencryptedCount        int             `json:"unencrypted_count"`
	NoAccessControlCount    int             `json:"no_access_control_count"`
	InternetFacingCount     int             `json:"internet_facing_count"`
	ClassificationBreakdown map[string]int  `json:"classification_breakdown"`
	ExposureBreakdown       map[string]int  `json:"exposure_breakdown"`
	TopRiskyAssets          []DSPMDataAsset `json:"top_risky_assets"`
	RecentScans             []DSPMScan      `json:"recent_scans"`
	PIITypeFrequency        map[string]int  `json:"pii_type_frequency"`
}

// DSPMClassificationSummary shows counts per classification level.
type DSPMClassificationSummary struct {
	Public       int `json:"public"`
	Internal     int `json:"internal"`
	Confidential int `json:"confidential"`
	Restricted   int `json:"restricted"`
	Total        int `json:"total"`
}

// DSPMExposureAnalysis describes internet-facing data exposure.
type DSPMExposureAnalysis struct {
	InternalOnly      int             `json:"internal_only"`
	VPNAccessible     int             `json:"vpn_accessible"`
	InternetFacing    int             `json:"internet_facing"`
	Unknown           int             `json:"unknown"`
	CriticalExposures []DSPMDataAsset `json:"critical_exposures"`
}

// DSPMDependencyNode is one node in the data flow dependency graph.
type DSPMDependencyNode struct {
	AssetID        uuid.UUID            `json:"asset_id"`
	AssetName      string               `json:"asset_name"`
	AssetType      string               `json:"asset_type"`
	Classification string               `json:"classification"`
	RiskScore      float64              `json:"risk_score"`
	ConsumerCount  int                  `json:"consumer_count"`
	ProducerCount  int                  `json:"producer_count"`
	Dependencies   []DSPMDependencyEdge `json:"dependencies"`
}

// DSPMDependencyEdge is one relationship in the dependency map.
type DSPMDependencyEdge struct {
	FromAssetID  uuid.UUID `json:"from_asset_id"`
	ToAssetID    uuid.UUID `json:"to_asset_id"`
	Relationship string    `json:"relationship"`
}
