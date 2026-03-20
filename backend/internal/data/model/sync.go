package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SyncStatus string

const (
	SyncStatusRunning   SyncStatus = "running"
	SyncStatusSuccess   SyncStatus = "success"
	SyncStatusPartial   SyncStatus = "partial"
	SyncStatusFailed    SyncStatus = "failed"
	SyncStatusCancelled SyncStatus = "cancelled"
)

type SyncType string

const (
	SyncTypeFull        SyncType = "full"
	SyncTypeIncremental SyncType = "incremental"
	SyncTypeSchemaOnly  SyncType = "schema_only"
)

type SyncTrigger string

const (
	SyncTriggerManual   SyncTrigger = "manual"
	SyncTriggerSchedule SyncTrigger = "schedule"
	SyncTriggerEvent    SyncTrigger = "event"
	SyncTriggerAPI      SyncTrigger = "api"
)

type SyncHistory struct {
	ID               uuid.UUID    `json:"id"`
	TenantID         uuid.UUID    `json:"tenant_id"`
	SourceID         uuid.UUID    `json:"source_id"`
	Status           SyncStatus   `json:"status"`
	SyncType         SyncType     `json:"sync_type"`
	TablesSynced     int          `json:"tables_synced"`
	RowsRead         int64        `json:"rows_read"`
	RowsWritten      int64        `json:"rows_written"`
	BytesTransferred int64        `json:"bytes_transferred"`
	Errors           json.RawMessage `json:"errors"`
	ErrorCount       int          `json:"error_count"`
	StartedAt        time.Time    `json:"started_at"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	DurationMs       *int64       `json:"duration_ms,omitempty"`
	TriggeredBy      SyncTrigger  `json:"triggered_by"`
	TriggeredByUser  *uuid.UUID   `json:"triggered_by_user,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
}
