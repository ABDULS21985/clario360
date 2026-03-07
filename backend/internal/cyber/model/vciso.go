package model

import (
	"time"

	"github.com/google/uuid"
)

// ExecutiveBriefing is the structured output of the Virtual CISO briefing.
type ExecutiveBriefing struct {
	GeneratedAt       time.Time              `json:"generated_at"`
	Period            DateRange              `json:"period"`
	RiskPosture       RiskPostureSummary     `json:"risk_posture"`
	CriticalIssues    []CriticalIssue        `json:"critical_issues"`
	ThreatLandscape   ThreatLandscapeSummary `json:"threat_landscape"`
	RemediationStatus RemediationSummary     `json:"remediation_status"`
	KeyMetrics        BriefingMetrics        `json:"key_metrics"`
	Recommendations   []RiskRecommendation   `json:"recommendations"`
	ComplianceStatus  ComplianceSummary      `json:"compliance_status"`
	Comparison        *PeriodComparison      `json:"comparison,omitempty"`
}

// DateRange is a time period for a briefing.
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Days  int       `json:"days"`
}

// RiskPostureSummary describes the current risk posture and trend.
type RiskPostureSummary struct {
	CurrentScore  float64                `json:"current_score"`
	PreviousScore float64                `json:"previous_score"`
	Trend         string                 `json:"trend"`
	TrendDelta    float64                `json:"trend_delta"`
	Grade         string                 `json:"grade"`
	GradeChange   string                 `json:"grade_change"`
	Components    map[string]float64     `json:"components"`
	ComponentTrends map[string]string    `json:"component_trends"`
}

// CriticalIssue is a top-priority issue that requires executive attention.
type CriticalIssue struct {
	Rank        int    `json:"rank"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Impact      string `json:"impact"`
	Action      string `json:"action"`
	LinkID      string `json:"link_id,omitempty"`
	LinkType    string `json:"link_type,omitempty"`
	DaysOpen    int    `json:"days_open,omitempty"`
}

// ThreatLandscapeSummary describes the threat environment observed in the period.
type ThreatLandscapeSummary struct {
	AlertVolume        int      `json:"alert_volume"`
	AlertVolumeChange  float64  `json:"alert_volume_change_pct"`
	NewThreats         int      `json:"new_threats"`
	ContainmentRate    float64  `json:"containment_rate"`
	TopMITRETactics    []string `json:"top_mitre_tactics"`
	TopThreatTypes     []ThreatTypeSummary `json:"top_threat_types"`
}

// ThreatTypeSummary summarizes one threat type.
type ThreatTypeSummary struct {
	Type    string `json:"type"`
	Count   int    `json:"count"`
	Change  float64 `json:"change_pct"`
}

// RemediationSummary describes the remediation pipeline status.
type RemediationSummary struct {
	CompletedInPeriod    int     `json:"completed_in_period"`
	PendingApproval      int     `json:"pending_approval"`
	InProgress           int     `json:"in_progress"`
	FailedInPeriod       int     `json:"failed_in_period"`
	RollbackCount        int     `json:"rollback_count"`
	VerificationSuccessRate float64 `json:"verification_success_rate"`
	AvgTimeToExecuteHours float64 `json:"avg_time_to_execute_hours"`
}

// BriefingMetrics holds KPI measurements for the period.
type BriefingMetrics struct {
	MTTD                float64 `json:"mttd_hours"`
	MTTR                float64 `json:"mttr_hours"`
	MTTC                float64 `json:"mttc_hours"`
	AlertVolumeTotal    int     `json:"alert_volume_total"`
	AlertsBySeverity    map[string]int `json:"alerts_by_severity"`
	FalsePositiveRate   float64 `json:"false_positive_rate"`
	SLAComplianceRate   float64 `json:"sla_compliance_rate"`
	CTEMExposureScore   float64 `json:"ctem_exposure_score"`
	DSPMComplianceScore float64 `json:"dspm_compliance_score"`
}

// ComplianceSummary describes compliance and coverage status.
type ComplianceSummary struct {
	MITRECoveragePercent float64 `json:"mitre_coverage_percent"`
	DSPMPostureScore     float64 `json:"dspm_posture_score"`
	SLAComplianceRate    float64 `json:"sla_compliance_rate"`
	OpenAuditFindings    int     `json:"open_audit_findings"`
}

// PeriodComparison contrasts the current period with the previous period.
type PeriodComparison struct {
	RiskScoreDelta       float64 `json:"risk_score_delta"`
	AlertVolumeDelta     int     `json:"alert_volume_delta"`
	AlertVolumeChangePct float64 `json:"alert_volume_change_pct"`
	NewVulnerabilities   int     `json:"new_vulnerabilities"`
	ResolvedVulnerabilities int  `json:"resolved_vulnerabilities"`
	RemediationsCompleted int    `json:"remediations_completed"`
}

// VCISOBriefingRecord is a stored briefing in the database.
type VCISOBriefingRecord struct {
	ID               uuid.UUID         `json:"id"`
	TenantID         uuid.UUID         `json:"tenant_id"`
	Type             string            `json:"type"`
	PeriodStart      time.Time         `json:"period_start"`
	PeriodEnd        time.Time         `json:"period_end"`
	Content          *ExecutiveBriefing `json:"content"`
	RiskScoreAtTime  *float64          `json:"risk_score_at_time,omitempty"`
	GeneratedBy      uuid.UUID         `json:"generated_by"`
	CreatedAt        time.Time         `json:"created_at"`
}

// PostureSummary is a quick widget-friendly posture snapshot.
type PostureSummary struct {
	RiskScore       float64  `json:"risk_score"`
	Grade           string   `json:"grade"`
	Trend           string   `json:"trend"`
	TrendDelta      float64  `json:"trend_delta"`
	TopIssues       []string `json:"top_issues"`
	OpenCriticalAlerts int   `json:"open_critical_alerts"`
	UnpatchedCriticalVulns int `json:"unpatched_critical_vulns"`
	ActiveThreats   int      `json:"active_threats"`
	DSPMScore       float64  `json:"dspm_posture_score"`
}
