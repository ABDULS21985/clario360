package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ContradictionType string

const (
	ContradictionTypeLogical    ContradictionType = "logical"
	ContradictionTypeSemantic   ContradictionType = "semantic"
	ContradictionTypeTemporal   ContradictionType = "temporal"
	ContradictionTypeAnalytical ContradictionType = "analytical"
)

type ContradictionStatus string

const (
	ContradictionStatusDetected      ContradictionStatus = "detected"
	ContradictionStatusInvestigating ContradictionStatus = "investigating"
	ContradictionStatusResolved      ContradictionStatus = "resolved"
	ContradictionStatusAccepted      ContradictionStatus = "accepted"
	ContradictionStatusFalsePositive ContradictionStatus = "false_positive"
)

type ContradictionResolutionAction string

const (
	ContradictionResolutionSourceACorrected ContradictionResolutionAction = "source_a_corrected"
	ContradictionResolutionSourceBCorrected ContradictionResolutionAction = "source_b_corrected"
	ContradictionResolutionBothCorrected    ContradictionResolutionAction = "both_corrected"
	ContradictionResolutionAcceptedAsIs     ContradictionResolutionAction = "accepted_as_is"
	ContradictionResolutionDataReconciled   ContradictionResolutionAction = "data_reconciled"
	ContradictionResolutionFalsePositive    ContradictionResolutionAction = "false_positive"
)

type ContradictionSource struct {
	SourceID      *uuid.UUID       `json:"source_id,omitempty"`
	SourceName    string           `json:"source_name"`
	ModelID       *uuid.UUID       `json:"model_id,omitempty"`
	ModelName     string           `json:"model_name,omitempty"`
	TableName     string           `json:"table_name,omitempty"`
	ColumnName    string           `json:"column_name,omitempty"`
	Value         any              `json:"value,omitempty"`
	LastSyncedAt  *time.Time       `json:"last_synced_at,omitempty"`
	Status        string           `json:"status,omitempty"`
	Metadata      json.RawMessage  `json:"metadata,omitempty"`
}

type Contradiction struct {
	ID                  uuid.UUID                      `json:"id"`
	TenantID            uuid.UUID                      `json:"tenant_id"`
	ScanID              *uuid.UUID                     `json:"scan_id,omitempty"`
	Type                ContradictionType              `json:"type"`
	Severity            QualitySeverity                `json:"severity"`
	ConfidenceScore     float64                        `json:"confidence_score"`
	Title               string                         `json:"title"`
	Description         string                         `json:"description"`
	SourceA             ContradictionSource            `json:"source_a"`
	SourceB             ContradictionSource            `json:"source_b"`
	EntityKeyColumn     *string                        `json:"entity_key_column,omitempty"`
	EntityKeyValue      *string                        `json:"entity_key_value,omitempty"`
	AffectedRecords     int                            `json:"affected_records"`
	SampleRecords       json.RawMessage                `json:"sample_records"`
	ResolutionGuidance  string                         `json:"resolution_guidance"`
	AuthoritativeSource *string                        `json:"authoritative_source,omitempty"`
	Status              ContradictionStatus            `json:"status"`
	ResolvedBy          *uuid.UUID                     `json:"resolved_by,omitempty"`
	ResolvedAt          *time.Time                     `json:"resolved_at,omitempty"`
	ResolutionNotes     *string                        `json:"resolution_notes,omitempty"`
	ResolutionAction    *ContradictionResolutionAction `json:"resolution_action,omitempty"`
	Metadata            json.RawMessage                `json:"metadata"`
	CreatedAt           time.Time                      `json:"created_at"`
	UpdatedAt           time.Time                      `json:"updated_at"`
}

type ContradictionScan struct {
	ID                  uuid.UUID       `json:"id"`
	TenantID            uuid.UUID       `json:"tenant_id"`
	Status              string          `json:"status"`
	ModelsScanned       int             `json:"models_scanned"`
	ModelPairsCompared  int             `json:"model_pairs_compared"`
	ContradictionsFound int             `json:"contradictions_found"`
	ByType              json.RawMessage `json:"by_type"`
	BySeverity          json.RawMessage `json:"by_severity"`
	StartedAt           time.Time       `json:"started_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	DurationMs          *int64          `json:"duration_ms,omitempty"`
	TriggeredBy         uuid.UUID       `json:"triggered_by"`
	CreatedAt           time.Time       `json:"created_at"`
}

type ContradictionStats struct {
	Total              int            `json:"total"`
	ByStatus           map[string]int `json:"by_status"`
	ByType             map[string]int `json:"by_type"`
	BySeverity         map[string]int `json:"by_severity"`
	AverageConfidence  float64        `json:"average_confidence"`
	OpenContradictions int            `json:"open_contradictions"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

