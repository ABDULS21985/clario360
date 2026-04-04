package model

import (
	"encoding/json"
	"time"
)

// AuditEntry represents a single immutable audit log record.
type AuditEntry struct {
	ID            string          `json:"id" db:"id"`
	TenantID      string          `json:"tenant_id" db:"tenant_id"`
	UserID        *string         `json:"user_id,omitempty" db:"user_id"`
	UserEmail     string          `json:"user_email" db:"user_email"`
	Service       string          `json:"service" db:"service"`
	Action        string          `json:"action" db:"action"`
	Severity      string          `json:"severity" db:"severity"`
	ResourceType  string          `json:"resource_type" db:"resource_type"`
	ResourceID    string          `json:"resource_id" db:"resource_id"`
	OldValue      json.RawMessage `json:"old_value,omitempty" db:"old_value"`
	NewValue      json.RawMessage `json:"new_value,omitempty" db:"new_value"`
	IPAddress     string          `json:"ip_address" db:"ip_address"`
	UserAgent     string          `json:"user_agent" db:"user_agent"`
	Metadata      json.RawMessage `json:"metadata" db:"metadata"`
	EventID       string          `json:"event_id" db:"event_id"`
	CorrelationID string          `json:"correlation_id" db:"correlation_id"`
	PreviousHash  string          `json:"previous_hash" db:"previous_hash"`
	EntryHash     string          `json:"entry_hash" db:"entry_hash"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// Severity constants for audit entries.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// ValidSeverities is the set of allowed severity values.
var ValidSeverities = map[string]bool{
	SeverityInfo:     true,
	SeverityWarning:  true,
	SeverityHigh:     true,
	SeverityCritical: true,
}

// ──────────────────────────────────────────────────────────────────────────────
// Statistics types — field names match frontend AuditLogStats interface exactly
// ──────────────────────────────────────────────────────────────────────────────

// AuditGroupStat holds an aggregated count for a single group key.
type AuditGroupStat struct {
	Key        string  `json:"key"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// AuditTimeseriesStat holds an event count for a time bucket.
type AuditTimeseriesStat struct {
	Timestamp string `json:"timestamp"`
	Count     int64  `json:"count"`
}

// AuditUserStat holds top-user activity metrics.
type AuditUserStat struct {
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	UserEmail   string `json:"user_email"`
	EventCount  int64  `json:"event_count"`
	LastEventAt string `json:"last_event_at"`
}

// AuditResourceStat holds top-resource activity metrics.
type AuditResourceStat struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	EventCount   int64  `json:"event_count"`
}

// AuditStats holds aggregated statistics for the audit dashboard.
// Field names match the frontend AuditLogStats interface exactly.
type AuditStats struct {
	TotalEvents     int64                 `json:"total_events"`
	EventsToday     int64                 `json:"events_today"`
	EventsThisWeek  int64                 `json:"events_this_week"`
	EventsThisMonth int64                 `json:"events_this_month"`
	UniqueUsers     int64                 `json:"unique_users"`
	UniqueServices  int64                 `json:"unique_services"`
	ByService       []AuditGroupStat      `json:"by_service"`
	ByAction        []AuditGroupStat      `json:"by_action"`
	BySeverity      []AuditGroupStat      `json:"by_severity"`
	ByHour          []AuditTimeseriesStat `json:"by_hour"`
	ByDay           []AuditTimeseriesStat `json:"by_day"`
	TopUsers        []AuditUserStat       `json:"top_users"`
	TopResources    []AuditResourceStat   `json:"top_resources"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Timeline types — field names match frontend AuditTimeline interface exactly
// ──────────────────────────────────────────────────────────────────────────────

// AuditChangeRecord represents a single field-level change within an audit event.
type AuditChangeRecord struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// AuditTimelineEvent is a single event in a resource's activity timeline.
type AuditTimelineEvent struct {
	ID        string              `json:"id"`
	Action    string              `json:"action"`
	UserName  string              `json:"user_name"`
	Timestamp string              `json:"timestamp"`
	Changes   []AuditChangeRecord `json:"changes"`
	Summary   string              `json:"summary"`
}

// AuditTimeline is the full timeline response for a resource.
type AuditTimeline struct {
	ResourceID   string               `json:"resource_id"`
	ResourceType string               `json:"resource_type"`
	ResourceName string               `json:"resource_name"`
	Events       []AuditTimelineEvent `json:"events"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Partition types — field names match frontend AuditPartition interface exactly
// ──────────────────────────────────────────────────────────────────────────────

// PartitionInfo holds metadata about a database partition, including derived fields.
type PartitionInfo struct {
	ID             string    `json:"id"`               // partition table name used as stable ID
	Name           string    `json:"name"`             // human-readable name (same as table name)
	DateRangeStart time.Time `json:"date_range_start"` // inclusive lower bound
	DateRangeEnd   time.Time `json:"date_range_end"`   // exclusive upper bound
	RecordCount    int64     `json:"record_count"`
	SizeBytes      int64     `json:"size_bytes"`
	Status         string    `json:"status"`     // "active" | "archived" | "pending"
	CreatedAt      time.Time `json:"created_at"` // derived from date_range_start
}

// ──────────────────────────────────────────────────────────────────────────────
// Chain state types (internal — not serialized to client)
// ──────────────────────────────────────────────────────────────────────────────

// ChainState holds the last known hash chain state for a tenant.
type ChainState struct {
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	LastEntryID string    `json:"last_entry_id" db:"last_entry_id"`
	LastHash    string    `json:"last_hash" db:"last_hash"`
	LastCreated time.Time `json:"last_created_at" db:"last_created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ChainVerificationResult holds the result of a hash chain verification.
// Field names match the frontend AuditVerificationResult interface exactly.
type ChainVerificationResult struct {
	Verified         bool    `json:"verified"`
	TotalRecords     int64   `json:"total_records"`
	VerifiedRecords  int64   `json:"verified_records"`
	BrokenChainAt    *string `json:"broken_chain_at"`
	FirstRecord      string  `json:"first_record"`
	LastRecord       string  `json:"last_record"`
	VerificationHash string  `json:"verification_hash"`
	VerifiedAt       string  `json:"verified_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Export job types
// ──────────────────────────────────────────────────────────────────────────────

// ExportJob tracks the status of an async export.
type ExportJob struct {
	JobID       string    `json:"job_id"`
	TenantID    string    `json:"tenant_id"`
	Status      string    `json:"status"` // processing, completed, failed
	Format      string    `json:"format"` // csv, ndjson
	RecordCount int64     `json:"record_count"`
	DownloadURL string    `json:"download_url,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}
