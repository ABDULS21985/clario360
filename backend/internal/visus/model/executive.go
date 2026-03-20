package model

import (
	"time"
)

type ExecutiveView struct {
	CyberSecurity    *CyberSecuritySummary     `json:"cyber_security"`
	DataIntelligence *DataIntelligenceSummary  `json:"data_intelligence"`
	Governance       *GovernanceSummary        `json:"governance"`
	Legal            *LegalSummary             `json:"legal"`
	KPIs             []KPISnapshot             `json:"kpis"`
	Alerts           []ExecutiveAlert          `json:"alerts"`
	SuiteHealth      map[string]SuiteStatus    `json:"suite_health"`
	GeneratedAt      time.Time                 `json:"generated_at"`
	CacheStatus      map[string]string         `json:"cache_status"`
}

type SuiteStatus struct {
	Available   bool      `json:"available"`
	LastSuccess time.Time `json:"last_success"`
	LatencyMS   int       `json:"latency_ms"`
	Error       string    `json:"error,omitempty"`
}

type CyberSecuritySummary struct {
	RiskScore       float64 `json:"risk_score"`
	RiskGrade       string  `json:"risk_grade"`
	OpenAlerts      int     `json:"open_alerts"`
	CriticalAlerts  int     `json:"critical_alerts"`
	MTTRHours       float64 `json:"mttr_hours"`
	MITRECoverage   float64 `json:"mitre_coverage_percent"`
	AssetsMonitored int     `json:"assets_monitored"`
	Trend           string  `json:"trend"`
}

type DataIntelligenceSummary struct {
	QualityScore       float64 `json:"quality_score"`
	QualityGrade       string  `json:"quality_grade"`
	ActivePipelines    int     `json:"active_pipelines"`
	FailedPipelines24h int     `json:"failed_pipelines_24h"`
	PipelineSuccessRate float64 `json:"pipeline_success_rate"`
	OpenContradictions int     `json:"open_contradictions"`
	DarkDataAssets     int     `json:"dark_data_assets"`
	Trend              string  `json:"trend"`
}

type GovernanceSummary struct {
	UpcomingMeetings   int     `json:"upcoming_meetings_30d"`
	OverdueActionItems int     `json:"overdue_action_items"`
	ComplianceScore    float64 `json:"compliance_score"`
	ComplianceGrade    string  `json:"compliance_grade"`
	OpenActionItems    int     `json:"open_action_items"`
	MinutesPending     int     `json:"minutes_pending_approval"`
	Trend              string  `json:"trend"`
}

type LegalSummary struct {
	ActiveContracts      int     `json:"active_contracts"`
	TotalContractValue   float64 `json:"total_contract_value"`
	ExpiringIn30Days     int     `json:"expiring_in_30_days"`
	HighRiskContracts    int     `json:"high_risk_contracts"`
	AvgRiskScore         float64 `json:"avg_risk_score"`
	OpenComplianceAlerts int     `json:"open_compliance_alerts"`
	PendingReview        int     `json:"pending_review"`
	Trend                string  `json:"trend"`
}
