package model

import (
	"time"

	"github.com/google/uuid"
)

// AccessAuditEntry records a single data access event for usage tracking.
type AccessAuditEntry struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	IdentityType    string     `json:"identity_type"`
	IdentityID      string     `json:"identity_id"`
	DataAssetID     uuid.UUID  `json:"data_asset_id"`
	Action          string     `json:"action"`
	SourceIP        string     `json:"source_ip,omitempty"`
	QueryHash       string     `json:"query_hash,omitempty"`
	RowsAffected    *int64     `json:"rows_affected,omitempty"`
	DurationMs      *int       `json:"duration_ms,omitempty"`
	Success         bool       `json:"success"`
	AccessMappingID *uuid.UUID `json:"access_mapping_id,omitempty"`
	TableName       string     `json:"table_name,omitempty"`
	DatabaseName    string     `json:"database_name,omitempty"`
	EventTimestamp  time.Time  `json:"event_timestamp"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AccessDashboard aggregates DSPM access intelligence KPIs.
type AccessDashboard struct {
	TotalIdentities        int                `json:"total_identities"`
	HighRiskIdentities     int                `json:"high_risk_identities"`
	OverprivilegedMappings int                `json:"overprivileged_mappings"`
	StalePermissions       int                `json:"stale_permissions"`
	AvgBlastRadius         float64            `json:"avg_blast_radius"`
	PolicyViolations       int                `json:"policy_violations"`
	TotalMappings          int                `json:"total_mappings"`
	ActiveMappings         int                `json:"active_mappings"`
	RiskDistribution       map[string]int     `json:"risk_distribution"`
	ClassificationAccess   map[string]int     `json:"classification_access"`
	TopRiskyIdentities     []IdentityProfile  `json:"top_risky_identities"`
}
