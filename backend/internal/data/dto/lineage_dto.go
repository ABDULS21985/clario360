package dto

import "github.com/google/uuid"

type RecordLineageEdgeRequest struct {
	SourceType         string    `json:"source_type"`
	SourceID           uuid.UUID `json:"source_id"`
	SourceName         string    `json:"source_name"`
	TargetType         string    `json:"target_type"`
	TargetID           uuid.UUID `json:"target_id"`
	TargetName         string    `json:"target_name"`
	Relationship       string    `json:"relationship"`
	TransformationDesc *string   `json:"transformation_desc,omitempty"`
	TransformationType *string   `json:"transformation_type,omitempty"`
	ColumnsAffected    []string  `json:"columns_affected,omitempty"`
	PipelineID         *uuid.UUID `json:"pipeline_id,omitempty"`
	PipelineRunID      *uuid.UUID `json:"pipeline_run_id,omitempty"`
	RecordedBy         string    `json:"recorded_by,omitempty"`
}

type SearchLineageParams struct {
	Query string
	Type  string
	Limit int
}
