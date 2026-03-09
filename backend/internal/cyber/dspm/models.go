package dspm

import (
	"time"

	"github.com/google/uuid"
)

// ComplianceTag maps a PII type to a specific regulation article.
type ComplianceTag struct {
	Framework   string `json:"framework"`   // "gdpr", "hipaa", "soc2", "pci_dss", "saudi_pdpl"
	Article     string `json:"article"`     // "Art. 4(1)", "§164.514(b)", "CC6.1", "Req 3.4", "Art. 5"
	Category    string `json:"category"`    // "personal_data", "special_category", "phi", "cardholder_data"
	Requirement string `json:"requirement"` // Human-readable requirement description
	Impact      string `json:"impact"`      // Impact description for non-compliance
	Severity    string `json:"severity"`    // "high", "medium", "low"
}

// ShadowCopy represents a detected unauthorized data duplicate.
type ShadowCopy struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	SourceAssetID   uuid.UUID `json:"source_asset_id"`
	SourceAssetName string    `json:"source_asset_name"`
	SourceTable     string    `json:"source_table"`
	TargetAssetID   uuid.UUID `json:"target_asset_id"`
	TargetAssetName string    `json:"target_asset_name"`
	TargetTable     string    `json:"target_table"`
	Fingerprint     string    `json:"fingerprint"`
	Similarity      float64   `json:"similarity"` // 0.0–1.0
	HasLineage      bool      `json:"has_lineage"`
	Status          string    `json:"status"` // "detected", "reviewed", "legitimate", "unauthorized"
	DetectedAt      time.Time `json:"detected_at"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy      *uuid.UUID `json:"reviewed_by,omitempty"`
}

// ScanEvent records a continuous DSPM scan trigger.
type ScanEvent struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	WatcherType  string    `json:"watcher_type"` // "pipeline", "transit", "at_rest", "shadow"
	TriggerEvent string    `json:"trigger_event"`
	AssetIDs     []uuid.UUID `json:"asset_ids,omitempty"`
	Status       string    `json:"status"` // "triggered", "scanning", "completed", "failed"
	AlertsRaised int       `json:"alerts_raised"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DurationMs   *int64    `json:"duration_ms,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// TransitCheckResult holds the result of a data-in-transit security check.
type TransitCheckResult struct {
	PipelineID      uuid.UUID `json:"pipeline_id"`
	PipelineName    string    `json:"pipeline_name"`
	SourceEncrypted bool      `json:"source_encrypted"`
	TargetEncrypted bool      `json:"target_encrypted"`
	HasApproval     bool      `json:"has_approval"`
	AlertsRaised    []string  `json:"alerts_raised"`
}

// DriftResult describes classification drift between consecutive scans.
type DriftResult struct {
	AssetID           uuid.UUID `json:"asset_id"`
	AssetName         string    `json:"asset_name"`
	ColumnName        string    `json:"column_name"`
	PreviousClass     string    `json:"previous_classification"`
	CurrentClass      string    `json:"current_classification"`
	PreviousPIIType   string    `json:"previous_pii_type,omitempty"`
	CurrentPIIType    string    `json:"current_pii_type,omitempty"`
	DriftDirection    string    `json:"drift_direction"` // "escalated", "deescalated"
}

// SchemaFingerprint is the hash of a table's structural definition.
type SchemaFingerprint struct {
	SourceID    uuid.UUID `json:"source_id"`
	SourceName  string    `json:"source_name"`
	TableName   string    `json:"table_name"`
	Fingerprint string    `json:"fingerprint"`
	ColumnCount int       `json:"column_count"`
	Columns     []string  `json:"columns"`
	Types       []string  `json:"types"`
}
