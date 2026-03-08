package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DataAccessEvent struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	TenantID         uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EntityType       EntityType      `json:"entity_type" db:"entity_type"`
	EntityID         string          `json:"entity_id" db:"entity_id"`
	SourceType       string          `json:"source_type" db:"source_type"`
	SourceID         *uuid.UUID      `json:"source_id,omitempty" db:"source_id"`
	Action           string          `json:"action" db:"action"`
	DatabaseName     string          `json:"database_name,omitempty" db:"database_name"`
	SchemaName       string          `json:"schema_name,omitempty" db:"schema_name"`
	TableName        string          `json:"table_name,omitempty" db:"table_name"`
	QueryHash        string          `json:"query_hash,omitempty" db:"query_hash"`
	RowsAccessed     int64           `json:"rows_accessed,omitempty" db:"rows_accessed"`
	BytesAccessed    int64           `json:"bytes_accessed,omitempty" db:"bytes_accessed"`
	DurationMS       int             `json:"duration_ms,omitempty" db:"duration_ms"`
	SourceIP         string          `json:"source_ip,omitempty" db:"source_ip"`
	UserAgent        string          `json:"user_agent,omitempty" db:"user_agent"`
	Success          bool            `json:"success" db:"success"`
	ErrorMessage     string          `json:"error_message,omitempty" db:"error_message"`
	TableSensitivity string          `json:"table_sensitivity,omitempty" db:"table_sensitivity"`
	ContainsPII      bool            `json:"contains_pii,omitempty" db:"contains_pii"`
	AnomalySignals   []AnomalySignal `json:"anomaly_signals" db:"anomaly_signals"`
	AnomalyCount     int             `json:"anomaly_count" db:"anomaly_count"`
	EventTimestamp   time.Time       `json:"event_timestamp" db:"event_timestamp"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`

	// QueryPreview is intentionally not persisted. It is used only during the
	// current cycle to support detectors such as WHERE-clause bulk reads while
	// keeping the raw event table schema bounded.
	QueryPreview string `json:"query_preview,omitempty" db:"-"`

	Metadata json.RawMessage `json:"metadata,omitempty" db:"-"`
}
