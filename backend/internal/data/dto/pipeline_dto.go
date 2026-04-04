package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreatePipelineRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	SourceID    uuid.UUID       `json:"source_id"`
	TargetID    *uuid.UUID      `json:"target_id,omitempty"`
	Config      json.RawMessage `json:"config"`
	Schedule    *string         `json:"schedule,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

type UpdatePipelineRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Type        *string         `json:"type,omitempty"`
	TargetID    *uuid.UUID      `json:"target_id,omitempty"`
	Config      json.RawMessage `json:"config,omitempty"`
	Schedule    *string         `json:"schedule,omitempty"`
	Status      *string         `json:"status,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

type RunPipelineRequest struct {
	TriggeredBy string `json:"triggered_by,omitempty"`
}

type ListPipelinesParams struct {
	Page     int
	PerPage  int
	Search   string
	Types    []string
	Statuses []string
	SourceID string
	Sort     string
	Order    string
}

type ListPipelineRunsParams struct {
	Page    int
	PerPage int
	Status  string
}

type PipelineRunResponse struct {
	ID                 uuid.UUID       `json:"id"`
	PipelineID         uuid.UUID       `json:"pipeline_id"`
	Status             string          `json:"status"`
	CurrentPhase       *string         `json:"current_phase,omitempty"`
	RecordsExtracted   int64           `json:"records_extracted"`
	RecordsTransformed int64           `json:"records_transformed"`
	RecordsLoaded      int64           `json:"records_loaded"`
	RecordsFailed      int64           `json:"records_failed"`
	DurationMs         *int64          `json:"duration_ms,omitempty"`
	StartedAt          time.Time       `json:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
	ErrorPhase         *string         `json:"error_phase,omitempty"`
	ErrorMessage       *string         `json:"error_message,omitempty"`
}

