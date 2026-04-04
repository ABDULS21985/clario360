package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type QualityRuleType string

const (
	QualityRuleTypeNotNull     QualityRuleType = "not_null"
	QualityRuleTypeUnique      QualityRuleType = "unique"
	QualityRuleTypeRange       QualityRuleType = "range"
	QualityRuleTypeRegex       QualityRuleType = "regex"
	QualityRuleTypeReferential QualityRuleType = "referential"
	QualityRuleTypeEnum        QualityRuleType = "enum"
	QualityRuleTypeFreshness   QualityRuleType = "freshness"
	QualityRuleTypeRowCount    QualityRuleType = "row_count"
	QualityRuleTypeCustomSQL   QualityRuleType = "custom_sql"
	QualityRuleTypeStatistical QualityRuleType = "statistical"
)

func (t QualityRuleType) IsValid() bool {
	switch t {
	case QualityRuleTypeNotNull, QualityRuleTypeUnique, QualityRuleTypeRange, QualityRuleTypeRegex, QualityRuleTypeReferential,
		QualityRuleTypeEnum, QualityRuleTypeFreshness, QualityRuleTypeRowCount, QualityRuleTypeCustomSQL, QualityRuleTypeStatistical:
		return true
	default:
		return false
	}
}

type QualitySeverity string

const (
	QualitySeverityCritical QualitySeverity = "critical"
	QualitySeverityHigh     QualitySeverity = "high"
	QualitySeverityMedium   QualitySeverity = "medium"
	QualitySeverityLow      QualitySeverity = "low"
)

func (s QualitySeverity) IsValid() bool {
	switch s {
	case QualitySeverityCritical, QualitySeverityHigh, QualitySeverityMedium, QualitySeverityLow:
		return true
	default:
		return false
	}
}

type QualityResultStatus string

const (
	QualityResultPassed  QualityResultStatus = "passed"
	QualityResultFailed  QualityResultStatus = "failed"
	QualityResultWarning QualityResultStatus = "warning"
	QualityResultError   QualityResultStatus = "error"
)

type QualityRule struct {
	ID                  uuid.UUID           `json:"id"`
	TenantID            uuid.UUID           `json:"tenant_id"`
	ModelID             uuid.UUID           `json:"model_id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	RuleType            QualityRuleType     `json:"rule_type"`
	Severity            QualitySeverity     `json:"severity"`
	ColumnName          *string             `json:"column_name,omitempty"`
	Config              json.RawMessage     `json:"config"`
	Schedule            *string             `json:"schedule,omitempty"`
	Enabled             bool                `json:"enabled"`
	LastRunAt           *time.Time          `json:"last_run_at,omitempty"`
	LastStatus          *QualityResultStatus `json:"last_status,omitempty"`
	ConsecutiveFailures int                 `json:"consecutive_failures"`
	Tags                []string            `json:"tags"`
	CreatedBy           uuid.UUID           `json:"created_by"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
	DeletedAt           *time.Time          `json:"deleted_at,omitempty"`
}

type QualityResult struct {
	ID             uuid.UUID           `json:"id"`
	TenantID       uuid.UUID           `json:"tenant_id"`
	RuleID         uuid.UUID           `json:"rule_id"`
	ModelID        uuid.UUID           `json:"model_id"`
	PipelineRunID  *uuid.UUID          `json:"pipeline_run_id,omitempty"`
	Status         QualityResultStatus `json:"status"`
	RecordsChecked int64               `json:"records_checked"`
	RecordsPassed  int64               `json:"records_passed"`
	RecordsFailed  int64               `json:"records_failed"`
	PassRate       *float64            `json:"pass_rate,omitempty"`
	FailureSamples json.RawMessage     `json:"failure_samples"`
	FailureSummary *string             `json:"failure_summary,omitempty"`
	CheckedAt      time.Time           `json:"checked_at"`
	DurationMs     *int64              `json:"duration_ms,omitempty"`
	ErrorMessage   *string             `json:"error_message,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
}

type ModelQualityScore struct {
	ModelID             uuid.UUID `json:"model_id"`
	ModelName           string    `json:"model_name"`
	Classification      string    `json:"classification"`
	Score               float64   `json:"score"`
	TotalRules          int       `json:"total_rules"`
	PassedRules         int       `json:"passed_rules"`
	FailedRules         int       `json:"failed_rules"`
	WarningRules        int       `json:"warning_rules"`
	ClassificationWeight float64  `json:"classification_weight"`
}

type TopFailure struct {
	RuleID      uuid.UUID `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	ModelID     uuid.UUID `json:"model_id"`
	ModelName   string    `json:"model_name"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	RecordsFailed int64   `json:"records_failed"`
}

type QualityScore struct {
	OverallScore float64             `json:"overall_score"`
	Grade        string              `json:"grade"`
	ModelScores  []ModelQualityScore `json:"model_scores"`
	TotalRules   int                 `json:"total_rules"`
	PassedRules  int                 `json:"passed_rules"`
	FailedRules  int                 `json:"failed_rules"`
	WarningRules int                 `json:"warning_rules"`
	PassRate     float64             `json:"pass_rate"`
	TopFailures  []TopFailure        `json:"top_failures"`
	Trend        string              `json:"trend"`
	TrendDelta   float64             `json:"trend_delta"`
	CalculatedAt time.Time           `json:"calculated_at"`
	History      []float64           `json:"history,omitempty"`
}

type QualityTrendPoint struct {
	Day   time.Time `json:"day"`
	Score float64   `json:"score"`
}

type QualityDashboard struct {
	Score       *QualityScore      `json:"score"`
	RecentRules []QualityRule      `json:"recent_rules"`
	TopFailures []TopFailure       `json:"top_failures"`
	Trend       []QualityTrendPoint `json:"trend"`
}

