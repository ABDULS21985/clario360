package dto

import "time"

type DailyMetric struct {
	Day   time.Time `json:"day"`
	Value float64   `json:"value"`
}

type PipelineRunSummary struct {
	ID          string     `json:"id"`
	PipelineID  string     `json:"pipeline_id"`
	PipelineName string    `json:"pipeline_name"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMs  *int64     `json:"duration_ms,omitempty"`
}

type QualityScoreSummary struct {
	OverallScore float64 `json:"overall_score"`
	Grade        string  `json:"grade"`
	PassedRules  int     `json:"passed_rules"`
	FailedRules  int     `json:"failed_rules"`
	WarningRules int     `json:"warning_rules"`
	PassRate     float64 `json:"pass_rate"`
}

type ModelQualitySummary struct {
	ModelID        string  `json:"model_id"`
	ModelName      string  `json:"model_name"`
	Classification string  `json:"classification"`
	Score          float64 `json:"score"`
}

type QualityFailureSummary struct {
	RuleID        string `json:"rule_id"`
	RuleName      string `json:"rule_name"`
	ModelID       string `json:"model_id"`
	ModelName     string `json:"model_name"`
	Severity      string `json:"severity"`
	RecordsFailed int64  `json:"records_failed"`
}

type DataKPIs struct {
	TotalSources        int     `json:"total_sources"`
	ActivePipelines     int     `json:"active_pipelines"`
	QualityScore        float64 `json:"quality_score"`
	QualityGrade        string  `json:"quality_grade"`
	OpenContradictions  int     `json:"open_contradictions"`
	DarkDataAssets      int     `json:"dark_data_assets"`
	TotalModels         int     `json:"total_models"`
	FailedPipelines24h  int     `json:"failed_pipelines_24h"`
	SourcesDelta        int     `json:"sources_delta"`
	QualityDelta        float64 `json:"quality_delta"`
	ContradictionsDelta int     `json:"contradictions_delta"`
}

type DataSuiteDashboard struct {
	KPIs                     DataKPIs                  `json:"kpis"`
	SourcesByType            map[string]int            `json:"sources_by_type"`
	SourcesByStatus          map[string]int            `json:"sources_by_status"`
	PipelinesByStatus        map[string]int            `json:"pipelines_by_status"`
	RecentRuns               []PipelineRunSummary      `json:"recent_runs"`
	PipelineSuccessRate      float64                   `json:"pipeline_success_rate_30d"`
	PipelineTrend            []DailyMetric             `json:"pipeline_trend_30d"`
	QualityScore             QualityScoreSummary       `json:"quality_score"`
	QualityTrend             []DailyMetric             `json:"quality_trend_30d"`
	QualityByModel           []ModelQualitySummary     `json:"quality_by_model"`
	TopFailures              []QualityFailureSummary   `json:"top_quality_failures"`
	ContradictionsByType     map[string]int            `json:"contradictions_by_type"`
	ContradictionsBySeverity map[string]int            `json:"contradictions_by_severity"`
	OpenContradictions       int                       `json:"open_contradictions"`
	LineageStats             map[string]any            `json:"lineage_stats"`
	DarkDataStats            map[string]any            `json:"dark_data_stats"`
	CachedAt                 *time.Time                `json:"cached_at,omitempty"`
	CalculatedAt             time.Time                 `json:"calculated_at"`
	PartialFailures          []string                  `json:"partial_failures,omitempty"`
}
