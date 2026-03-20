package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DataModelStatus string

const (
	DataModelStatusDraft      DataModelStatus = "draft"
	DataModelStatusActive     DataModelStatus = "active"
	DataModelStatusDeprecated DataModelStatus = "deprecated"
	DataModelStatusArchived   DataModelStatus = "archived"
)

func (s DataModelStatus) IsValid() bool {
	switch s {
	case DataModelStatusDraft, DataModelStatusActive, DataModelStatusDeprecated, DataModelStatusArchived:
		return true
	default:
		return false
	}
}

type DataModel struct {
	ID                 uuid.UUID          `json:"id"`
	TenantID           uuid.UUID          `json:"tenant_id"`
	Name               string             `json:"name"`
	DisplayName        string             `json:"display_name"`
	Description        string             `json:"description"`
	Status             DataModelStatus    `json:"status"`
	SchemaDefinition   []ModelField       `json:"schema_definition"`
	SourceID           *uuid.UUID         `json:"source_id,omitempty"`
	SourceTable        *string            `json:"source_table,omitempty"`
	QualityRules       []ValidationRule   `json:"quality_rules"`
	DataClassification DataClassification `json:"data_classification"`
	ContainsPII        bool               `json:"contains_pii"`
	PIIColumns         []string           `json:"pii_columns"`
	FieldCount         int                `json:"field_count"`
	Version            int                `json:"version"`
	PreviousVersionID  *uuid.UUID         `json:"previous_version_id,omitempty"`
	Tags               []string           `json:"tags"`
	Metadata           json.RawMessage    `json:"metadata"`
	CreatedBy          uuid.UUID          `json:"created_by"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	DeletedAt          *time.Time         `json:"deleted_at,omitempty"`
}

type ModelField struct {
	Name            string           `json:"name"`
	DisplayName     string           `json:"display_name"`
	DataType        string           `json:"data_type"`
	NativeType      string           `json:"native_type"`
	Nullable        bool             `json:"nullable"`
	IsPrimaryKey    bool             `json:"is_primary_key"`
	IsForeignKey    bool             `json:"is_foreign_key"`
	ForeignKeyRef   *ForeignKeyRef   `json:"foreign_key_ref,omitempty"`
	Description     string           `json:"description"`
	DefaultValue    *string          `json:"default_value,omitempty"`
	PIIType         string           `json:"pii_type,omitempty"`
	Classification  DataClassification `json:"classification"`
	SampleValues    []string         `json:"sample_values,omitempty"`
	ValidationRules []ValidationRule `json:"validation_rules"`
}

type ValidationRule struct {
	Type    string         `json:"type"`
	Field   string         `json:"field"`
	Params  map[string]any `json:"params,omitempty"`
	Message string         `json:"message,omitempty"`
}

type ModelValidationResult struct {
	Success bool               `json:"success"`
	Errors  []ModelValidationError `json:"errors"`
}

type ModelValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ModelLineage struct {
	Model          DataModel           `json:"model"`
	Source         *DataSource         `json:"source,omitempty"`
	SourceTable    *DiscoveredTable    `json:"source_table,omitempty"`
	UpstreamTables []ForeignKeyRef     `json:"upstream_tables,omitempty"`
	Consumers      []map[string]string `json:"consumers,omitempty"`
}
