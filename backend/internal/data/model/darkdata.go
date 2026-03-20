package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DarkDataAssetType string

const (
	DarkDataAssetDatabaseTable DarkDataAssetType = "database_table"
	DarkDataAssetDatabaseView  DarkDataAssetType = "database_view"
	DarkDataAssetFile          DarkDataAssetType = "file"
	DarkDataAssetAPIEndpoint   DarkDataAssetType = "api_endpoint"
	DarkDataAssetStreamTopic   DarkDataAssetType = "stream_topic"
)

type DarkDataReason string

const (
	DarkDataReasonUnmodeled    DarkDataReason = "unmodeled"
	DarkDataReasonOrphanedFile DarkDataReason = "orphaned_file"
	DarkDataReasonStale        DarkDataReason = "stale"
	DarkDataReasonUngoverned   DarkDataReason = "ungoverned"
	DarkDataReasonUnclassified DarkDataReason = "unclassified"
)

type DarkDataGovernanceStatus string

const (
	DarkDataGovernanceUnmanaged         DarkDataGovernanceStatus = "unmanaged"
	DarkDataGovernanceUnderReview       DarkDataGovernanceStatus = "under_review"
	DarkDataGovernanceGoverned          DarkDataGovernanceStatus = "governed"
	DarkDataGovernanceArchived          DarkDataGovernanceStatus = "archived"
	DarkDataGovernanceScheduledDeletion DarkDataGovernanceStatus = "scheduled_deletion"
)

type DarkDataScanStatus string

const (
	DarkDataScanRunning   DarkDataScanStatus = "running"
	DarkDataScanCompleted DarkDataScanStatus = "completed"
	DarkDataScanFailed    DarkDataScanStatus = "failed"
)

type RiskFactor struct {
	Factor      string  `json:"factor"`
	Value       float64 `json:"value"`
	Description string  `json:"description,omitempty"`
}

type DarkDataAsset struct {
	ID                     uuid.UUID                `json:"id"`
	TenantID               uuid.UUID                `json:"tenant_id"`
	ScanID                 *uuid.UUID               `json:"scan_id,omitempty"`
	Name                   string                   `json:"name"`
	AssetType              DarkDataAssetType        `json:"asset_type"`
	SourceID               *uuid.UUID               `json:"source_id,omitempty"`
	SourceName             *string                  `json:"source_name,omitempty"`
	SchemaName             *string                  `json:"schema_name,omitempty"`
	TableName              *string                  `json:"table_name,omitempty"`
	FilePath               *string                  `json:"file_path,omitempty"`
	Reason                 DarkDataReason           `json:"reason"`
	EstimatedRowCount      *int64                   `json:"estimated_row_count,omitempty"`
	EstimatedSizeBytes     *int64                   `json:"estimated_size_bytes,omitempty"`
	ColumnCount            *int                     `json:"column_count,omitempty"`
	ContainsPII            bool                     `json:"contains_pii"`
	PIITypes               []string                 `json:"pii_types"`
	InferredClassification *DataClassification      `json:"inferred_classification,omitempty"`
	LastAccessedAt         *time.Time               `json:"last_accessed_at,omitempty"`
	LastModifiedAt         *time.Time               `json:"last_modified_at,omitempty"`
	DaysSinceAccess        *int                     `json:"days_since_access,omitempty"`
	RiskScore              float64                  `json:"risk_score"`
	RiskFactors            []RiskFactor             `json:"risk_factors"`
	GovernanceStatus       DarkDataGovernanceStatus `json:"governance_status"`
	GovernanceNotes        *string                  `json:"governance_notes,omitempty"`
	ReviewedBy             *uuid.UUID               `json:"reviewed_by,omitempty"`
	ReviewedAt             *time.Time               `json:"reviewed_at,omitempty"`
	LinkedModelID          *uuid.UUID               `json:"linked_model_id,omitempty"`
	Metadata               json.RawMessage          `json:"metadata"`
	DiscoveredAt           time.Time                `json:"discovered_at"`
	CreatedAt              time.Time                `json:"created_at"`
	UpdatedAt              time.Time                `json:"updated_at"`
}

type DarkDataScan struct {
	ID               uuid.UUID          `json:"id"`
	TenantID         uuid.UUID          `json:"tenant_id"`
	Status           DarkDataScanStatus `json:"status"`
	SourcesScanned   int                `json:"sources_scanned"`
	StorageScanned   bool               `json:"storage_scanned"`
	AssetsDiscovered int                `json:"assets_discovered"`
	ByReason         json.RawMessage    `json:"by_reason"`
	ByType           json.RawMessage    `json:"by_type"`
	PIIAssetsFound   int                `json:"pii_assets_found"`
	HighRiskFound    int                `json:"high_risk_found"`
	TotalSizeBytes   int64              `json:"total_size_bytes"`
	StartedAt        time.Time          `json:"started_at"`
	CompletedAt      *time.Time         `json:"completed_at,omitempty"`
	DurationMs       *int64             `json:"duration_ms,omitempty"`
	TriggeredBy      uuid.UUID          `json:"triggered_by"`
	CreatedAt        time.Time          `json:"created_at"`
}

type DarkDataStatsSummary struct {
	TotalAssets            int            `json:"total_assets"`
	ByReason               map[string]int `json:"by_reason"`
	ByType                 map[string]int `json:"by_type"`
	ByGovernanceStatus     map[string]int `json:"by_governance_status"`
	PIIAssets              int            `json:"pii_assets"`
	HighRiskAssets         int            `json:"high_risk_assets"`
	TotalSizeBytes         int64          `json:"total_size_bytes"`
	AverageRiskScore       float64        `json:"average_risk_score"`
	GovernedAssets         int            `json:"governed_assets"`
	ScheduledDeletionCount int            `json:"scheduled_deletion_count"`
}
