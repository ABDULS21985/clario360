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

// AuditStats holds aggregated statistics for audit queries.
type AuditStats struct {
	TotalRecords   int64            `json:"total_records"`
	ActionCounts   map[string]int64 `json:"action_counts"`
	SeverityCounts map[string]int64 `json:"severity_counts"`
	TopUsers       []UserActivity   `json:"top_users"`
	DailyVolume    []DailyCount     `json:"daily_volume"`
}

// UserActivity represents a user's audit activity count.
type UserActivity struct {
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	Count     int64  `json:"count"`
}

// DailyCount represents the count of audit entries for a specific day.
type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// PartitionInfo holds metadata about a database partition.
type PartitionInfo struct {
	Name        string    `json:"name"`
	RangeStart  time.Time `json:"range_start"`
	RangeEnd    time.Time `json:"range_end"`
	RecordCount int64     `json:"record_count"`
	SizeBytes   int64     `json:"size_bytes"`
}

// ChainState holds the last known hash chain state for a tenant.
type ChainState struct {
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	LastEntryID string    `json:"last_entry_id" db:"last_entry_id"`
	LastHash    string    `json:"last_hash" db:"last_hash"`
	LastCreated time.Time `json:"last_created_at" db:"last_created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ChainVerificationResult holds the result of a hash chain verification.
type ChainVerificationResult struct {
	OK       bool    `json:"ok"`
	BrokenAt *string `json:"broken_at,omitempty"`
	Checked  int64   `json:"checked"`
}

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
