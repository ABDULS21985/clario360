package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateModelRequest struct {
	Name               string          `json:"name" validate:"required,min=2,max=255"`
	DisplayName        string          `json:"display_name" validate:"required,min=2,max=255"`
	Description        string          `json:"description" validate:"max=2000"`
	Status             string          `json:"status" validate:"omitempty,oneof=draft active deprecated archived"`
	SchemaDefinition   json.RawMessage `json:"schema_definition" validate:"required"`
	SourceID           *uuid.UUID      `json:"source_id,omitempty"`
	SourceTable        *string         `json:"source_table,omitempty"`
	QualityRules       json.RawMessage `json:"quality_rules,omitempty"`
	DataClassification string          `json:"data_classification" validate:"omitempty,oneof=public internal confidential restricted"`
	ContainsPII        bool            `json:"contains_pii"`
	PIIColumns         []string        `json:"pii_columns,omitempty"`
	Tags               []string        `json:"tags,omitempty" validate:"max=20,dive,max=64"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

type UpdateModelRequest struct {
	DisplayName        *string         `json:"display_name,omitempty" validate:"omitempty,min=2,max=255"`
	Description        *string         `json:"description,omitempty" validate:"omitempty,max=2000"`
	Status             *string         `json:"status,omitempty" validate:"omitempty,oneof=draft active deprecated archived"`
	SchemaDefinition   json.RawMessage `json:"schema_definition,omitempty"`
	QualityRules       json.RawMessage `json:"quality_rules,omitempty"`
	DataClassification *string         `json:"data_classification,omitempty" validate:"omitempty,oneof=public internal confidential restricted"`
	ContainsPII        *bool           `json:"contains_pii,omitempty"`
	PIIColumns         []string        `json:"pii_columns,omitempty"`
	Tags               []string        `json:"tags,omitempty" validate:"max=20,dive,max=64"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

type DeriveModelRequest struct {
	SourceID                 uuid.UUID `json:"source_id" validate:"required"`
	TableName                string    `json:"table_name" validate:"required"`
	Name                     string    `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	AutoGenerateQualityRules *bool     `json:"auto_generate_quality_rules,omitempty"`
}

type ValidateModelRequest struct {
	SampleLimit int `json:"sample_limit" validate:"omitempty,gte=1,lte=1000"`
}

type ListModelsParams struct {
	Page                int
	PerPage             int
	Search              string
	Statuses            []string
	SourceID            string
	DataClassifications []string
	ContainsPII         *bool
	Sort                string
	Order               string
}
