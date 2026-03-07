package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
)

type CreateSourceRequest struct {
	Name             string          `json:"name" validate:"required,min=2,max=255"`
	Description      string          `json:"description" validate:"max=2000"`
	Type             string          `json:"type" validate:"required,oneof=postgresql mysql mssql api csv s3 stream"`
	ConnectionConfig json.RawMessage `json:"connection_config" validate:"required"`
	SyncFrequency    *string         `json:"sync_frequency,omitempty"`
	Tags             []string        `json:"tags,omitempty" validate:"max=20,dive,max=64"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

type TestSourceConfigRequest struct {
	Type             string          `json:"type" validate:"required,oneof=postgresql mysql mssql api csv s3 stream"`
	ConnectionConfig json.RawMessage `json:"connection_config" validate:"required"`
}

type UpdateSourceRequest struct {
	Name             *string         `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description      *string         `json:"description,omitempty" validate:"omitempty,max=2000"`
	ConnectionConfig json.RawMessage `json:"connection_config,omitempty"`
	SyncFrequency    *string         `json:"sync_frequency,omitempty"`
	Tags             []string        `json:"tags,omitempty" validate:"max=20,dive,max=64"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

type ChangeStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active inactive"`
}

type ListSourcesParams struct {
	Page       int
	PerPage    int
	Search     string
	Type       string
	Status     string
	HasSchema  *bool
	Sort       string
	Order      string
}

type DataSourceResponse struct {
	ID                 uuid.UUID        `json:"id"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	Type               model.DataSourceType `json:"type"`
	ConnectionConfig   json.RawMessage  `json:"connection_config"`
	Status             model.DataSourceStatus `json:"status"`
	LastError          *string          `json:"last_error,omitempty"`
	SchemaMetadata     *model.DiscoveredSchema `json:"schema_metadata,omitempty"`
	SchemaDiscoveredAt *time.Time       `json:"schema_discovered_at,omitempty"`
	LastSyncedAt       *time.Time       `json:"last_synced_at,omitempty"`
	LastSyncStatus     *string          `json:"last_sync_status,omitempty"`
	LastSyncError      *string          `json:"last_sync_error,omitempty"`
	LastSyncDurationMs *int64           `json:"last_sync_duration_ms,omitempty"`
	SyncFrequency      *string          `json:"sync_frequency,omitempty"`
	NextSyncAt         *time.Time       `json:"next_sync_at,omitempty"`
	TableCount         *int             `json:"table_count,omitempty"`
	TotalRowCount      *int64           `json:"total_row_count,omitempty"`
	TotalSizeBytes     *int64           `json:"total_size_bytes,omitempty"`
	Tags               []string         `json:"tags"`
	Metadata           json.RawMessage  `json:"metadata"`
	CreatedBy          uuid.UUID        `json:"created_by"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

type TestConnectionResponse struct {
	Success     bool     `json:"success"`
	LatencyMs   int64    `json:"latency_ms"`
	Version     string   `json:"version,omitempty"`
	Message     string   `json:"message"`
	Permissions []string `json:"permissions,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type SourceStatsResponse struct {
	TableCount         int        `json:"table_count"`
	TotalRowCount      int64      `json:"total_row_count"`
	TotalSizeBytes     int64      `json:"total_size_bytes"`
	SchemaDiscoveredAt *time.Time `json:"schema_discovered_at,omitempty"`
	LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`
	LastSyncStatus     *string    `json:"last_sync_status,omitempty"`
}

type AggregateSourceStatsResponse struct {
	TotalSources       int            `json:"total_sources"`
	ByType             map[string]int `json:"by_type"`
	ByStatus           map[string]int `json:"by_status"`
	SourcesWithSchema  int            `json:"sources_with_schema"`
	TotalRows          int64          `json:"total_rows"`
	TotalSizeBytes     int64          `json:"total_size_bytes"`
}

type TriggerSyncRequest struct {
	SyncType string `json:"sync_type" validate:"required,oneof=full incremental schema_only"`
}
