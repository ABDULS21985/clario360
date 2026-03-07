package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PipelineType string

const (
	PipelineTypeETL       PipelineType = "etl"
	PipelineTypeELT       PipelineType = "elt"
	PipelineTypeBatch     PipelineType = "batch"
	PipelineTypeStreaming PipelineType = "streaming"
)

func (t PipelineType) IsValid() bool {
	switch t {
	case PipelineTypeETL, PipelineTypeELT, PipelineTypeBatch, PipelineTypeStreaming:
		return true
	default:
		return false
	}
}

type PipelineStatus string

const (
	PipelineStatusActive   PipelineStatus = "active"
	PipelineStatusPaused   PipelineStatus = "paused"
	PipelineStatusDisabled PipelineStatus = "disabled"
	PipelineStatusError    PipelineStatus = "error"
)

func (s PipelineStatus) IsValid() bool {
	switch s {
	case PipelineStatusActive, PipelineStatusPaused, PipelineStatusDisabled, PipelineStatusError:
		return true
	default:
		return false
	}
}

type PipelineRunStatus string

const (
	PipelineRunStatusRunning   PipelineRunStatus = "running"
	PipelineRunStatusCompleted PipelineRunStatus = "completed"
	PipelineRunStatusFailed    PipelineRunStatus = "failed"
	PipelineRunStatusCancelled PipelineRunStatus = "cancelled"
)

type PipelineTrigger string

const (
	PipelineTriggerManual   PipelineTrigger = "manual"
	PipelineTriggerSchedule PipelineTrigger = "schedule"
	PipelineTriggerEvent    PipelineTrigger = "event"
	PipelineTriggerAPI      PipelineTrigger = "api"
	PipelineTriggerRetry    PipelineTrigger = "retry"
)

type PipelinePhase string

const (
	PipelinePhaseExtracting   PipelinePhase = "extracting"
	PipelinePhaseTransforming PipelinePhase = "transforming"
	PipelinePhaseQualityGate  PipelinePhase = "quality_check"
	PipelinePhaseLoading      PipelinePhase = "loading"
)

type LoadStrategy string

const (
	LoadStrategyAppend      LoadStrategy = "append"
	LoadStrategyFullReplace LoadStrategy = "full_replace"
	LoadStrategyIncremental LoadStrategy = "incremental"
	LoadStrategyMerge       LoadStrategy = "merge"
)

type TransformationType string

const (
	TransformationRename      TransformationType = "rename"
	TransformationCast        TransformationType = "cast"
	TransformationFilter      TransformationType = "filter"
	TransformationMapValues   TransformationType = "map_values"
	TransformationDerive      TransformationType = "derive"
	TransformationDeduplicate TransformationType = "deduplicate"
	TransformationAggregate   TransformationType = "aggregate"
)

type QualityGateMetric string

const (
	QualityGateMetricNullPercentage   QualityGateMetric = "null_percentage"
	QualityGateMetricUniquePercentage QualityGateMetric = "unique_percentage"
	QualityGateMetricRowCountChange   QualityGateMetric = "row_count_change"
	QualityGateMetricMinRowCount      QualityGateMetric = "min_row_count"
	QualityGateMetricCustom           QualityGateMetric = "custom"
)

type Pipeline struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id"`
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	Type                  PipelineType    `json:"type"`
	SourceID              uuid.UUID       `json:"source_id"`
	TargetID              *uuid.UUID      `json:"target_id,omitempty"`
	Config                PipelineConfig  `json:"config"`
	Schedule              *string         `json:"schedule,omitempty"`
	Status                PipelineStatus  `json:"status"`
	LastRunID             *uuid.UUID      `json:"last_run_id,omitempty"`
	LastRunAt             *time.Time      `json:"last_run_at,omitempty"`
	LastRunStatus         *string         `json:"last_run_status,omitempty"`
	LastRunError          *string         `json:"last_run_error,omitempty"`
	NextRunAt             *time.Time      `json:"next_run_at,omitempty"`
	TotalRuns             int             `json:"total_runs"`
	SuccessfulRuns        int             `json:"successful_runs"`
	FailedRuns            int             `json:"failed_runs"`
	TotalRecordsProcessed int64           `json:"total_records_processed"`
	AvgDurationMs         *int64          `json:"avg_duration_ms,omitempty"`
	Tags                  []string        `json:"tags"`
	CreatedBy             uuid.UUID       `json:"created_by"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	DeletedAt             *time.Time      `json:"deleted_at,omitempty"`
}

type PipelineConfig struct {
	SourceTable       string           `json:"source_table,omitempty"`
	SourceQuery       string           `json:"source_query,omitempty"`
	TargetTable       string           `json:"target_table,omitempty"`
	TargetModelID     *uuid.UUID       `json:"target_model_id,omitempty"`
	BatchSize         int              `json:"batch_size,omitempty"`
	IncrementalField  string           `json:"incremental_field,omitempty"`
	IncrementalValue  *string          `json:"incremental_value,omitempty"`
	Transformations   []Transformation `json:"transformations,omitempty"`
	QualityGates      []QualityGate    `json:"quality_gates,omitempty"`
	FailOnQualityGate bool             `json:"fail_on_quality_gate,omitempty"`
	LoadStrategy      LoadStrategy     `json:"load_strategy,omitempty"`
	MergeKeys         []string         `json:"merge_keys,omitempty"`
	MaxRetries        int              `json:"max_retries,omitempty"`
	RetryBackoffSec   int              `json:"retry_backoff_sec,omitempty"`
	Metadata          json.RawMessage  `json:"metadata,omitempty"`
}

type Transformation struct {
	Type   TransformationType `json:"type"`
	Config json.RawMessage    `json:"config"`
}

type QualityGate struct {
	Name        string            `json:"name"`
	Metric      QualityGateMetric `json:"metric"`
	Column      string            `json:"column,omitempty"`
	Operator    string            `json:"operator,omitempty"`
	Threshold   *float64          `json:"threshold,omitempty"`
	MinValue    *float64          `json:"min_value,omitempty"`
	MaxValue    *float64          `json:"max_value,omitempty"`
	Expression  string            `json:"expression,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	Description string            `json:"description,omitempty"`
}

type QualityGateResult struct {
	Name         string    `json:"name"`
	Metric       string    `json:"metric"`
	Status       string    `json:"status"`
	MetricValue  float64   `json:"metric_value"`
	Threshold    *float64  `json:"threshold,omitempty"`
	MinValue     *float64  `json:"min_value,omitempty"`
	MaxValue     *float64  `json:"max_value,omitempty"`
	Operator     string    `json:"operator,omitempty"`
	Message      string    `json:"message,omitempty"`
	Severity     string    `json:"severity,omitempty"`
	EvaluatedAt  time.Time `json:"evaluated_at"`
}

type PipelineRun struct {
	ID                   uuid.UUID           `json:"id"`
	TenantID             uuid.UUID           `json:"tenant_id"`
	PipelineID           uuid.UUID           `json:"pipeline_id"`
	Status               PipelineRunStatus   `json:"status"`
	CurrentPhase         *string             `json:"current_phase,omitempty"`
	RecordsExtracted     int64               `json:"records_extracted"`
	RecordsTransformed   int64               `json:"records_transformed"`
	RecordsLoaded        int64               `json:"records_loaded"`
	RecordsFailed        int64               `json:"records_failed"`
	RecordsFiltered      int64               `json:"records_filtered"`
	RecordsDeduplicated  int64               `json:"records_deduplicated"`
	BytesRead            int64               `json:"bytes_read"`
	BytesWritten         int64               `json:"bytes_written"`
	QualityGateResults   []QualityGateResult `json:"quality_gate_results"`
	QualityGatesPassed   int                 `json:"quality_gates_passed"`
	QualityGatesFailed   int                 `json:"quality_gates_failed"`
	QualityGatesWarned   int                 `json:"quality_gates_warned"`
	StartedAt            time.Time           `json:"started_at"`
	ExtractStartedAt     *time.Time          `json:"extract_started_at,omitempty"`
	ExtractCompletedAt   *time.Time          `json:"extract_completed_at,omitempty"`
	TransformStartedAt   *time.Time          `json:"transform_started_at,omitempty"`
	TransformCompletedAt *time.Time          `json:"transform_completed_at,omitempty"`
	LoadStartedAt        *time.Time          `json:"load_started_at,omitempty"`
	LoadCompletedAt      *time.Time          `json:"load_completed_at,omitempty"`
	CompletedAt          *time.Time          `json:"completed_at,omitempty"`
	DurationMs           *int64              `json:"duration_ms,omitempty"`
	ErrorPhase           *string             `json:"error_phase,omitempty"`
	ErrorMessage         *string             `json:"error_message,omitempty"`
	ErrorDetails         json.RawMessage     `json:"error_details,omitempty"`
	TriggeredBy          PipelineTrigger     `json:"triggered_by"`
	TriggeredByUser      *uuid.UUID          `json:"triggered_by_user,omitempty"`
	RetryCount           int                 `json:"retry_count"`
	IncrementalFrom      *string             `json:"incremental_from,omitempty"`
	IncrementalTo        *string             `json:"incremental_to,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
}

type PipelineRunLog struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	RunID     uuid.UUID       `json:"run_id"`
	Level     string          `json:"level"`
	Phase     string          `json:"phase"`
	Message   string          `json:"message"`
	Details   json.RawMessage `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type PipelineStats struct {
	TotalPipelines   int              `json:"total_pipelines"`
	ActivePipelines  int              `json:"active_pipelines"`
	PausedPipelines  int              `json:"paused_pipelines"`
	ErrorPipelines   int              `json:"error_pipelines"`
	RunningPipelines int              `json:"running_pipelines"`
	CompletedRuns    int64            `json:"completed_runs"`
	FailedRuns       int64            `json:"failed_runs"`
	SuccessRate      float64          `json:"success_rate"`
	ByType           map[string]int   `json:"by_type"`
	ByStatus         map[string]int   `json:"by_status"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

