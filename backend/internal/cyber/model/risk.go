package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ComponentScore struct {
	Score       float64                `json:"score"`
	Weight      float64                `json:"weight"`
	Weighted    float64                `json:"weighted"`
	Trend       string                 `json:"trend"`
	TrendDelta  float64                `json:"trend_delta"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type RiskComponents struct {
	VulnerabilityRisk ComponentScore `json:"vulnerability_risk"`
	ThreatExposure    ComponentScore `json:"threat_exposure"`
	ConfigurationRisk ComponentScore `json:"configuration_risk"`
	AttackSurfaceRisk ComponentScore `json:"attack_surface_risk"`
	ComplianceGapRisk ComponentScore `json:"compliance_gap_risk"`
}

type RiskContext struct {
	TotalAssets          int `json:"total_assets"`
	TotalOpenVulns       int `json:"total_open_vulnerabilities"`
	TotalOpenAlerts      int `json:"total_open_alerts"`
	TotalActiveThreats   int `json:"total_active_threats"`
	InternetFacingAssets int `json:"internet_facing_assets"`
	CriticalAssets       int `json:"critical_assets"`
}

type RiskContributor struct {
	Type        string     `json:"type"`
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Score       float64    `json:"score"`
	Impact      float64    `json:"impact_percent"`
	Severity    string     `json:"severity"`
	AssetID     *uuid.UUID `json:"asset_id,omitempty"`
	AssetName   string     `json:"asset_name,omitempty"`
	Remediation string     `json:"remediation"`
	Link        string     `json:"link"`
}

type RiskRecommendation struct {
	Priority        int      `json:"priority"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Component       string   `json:"component"`
	EstimatedImpact float64  `json:"estimated_score_reduction"`
	Effort          string   `json:"effort"`
	Category        string   `json:"category"`
	Actions         []string `json:"actions"`
}

type OrganizationRiskScore struct {
	TenantID        uuid.UUID             `json:"tenant_id"`
	OverallScore    float64               `json:"overall_score"`
	Grade           string                `json:"grade"`
	Trend           string                `json:"trend"`
	TrendDelta      float64               `json:"trend_delta"`
	Components      RiskComponents        `json:"components"`
	TopContributors []RiskContributor     `json:"top_contributors"`
	Recommendations []RiskRecommendation  `json:"recommendations"`
	Context         RiskContext           `json:"context"`
	CalculatedAt    time.Time             `json:"calculated_at"`
}

type RiskScoreHistory struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	OverallScore       float64         `json:"overall_score"`
	Grade              string          `json:"grade"`
	VulnerabilityScore float64         `json:"vulnerability_score"`
	ThreatScore        float64         `json:"threat_score"`
	ConfigScore        float64         `json:"config_score"`
	SurfaceScore       float64         `json:"surface_score"`
	ComplianceScore    float64         `json:"compliance_score"`
	TotalAssets        int             `json:"total_assets"`
	TotalOpenVulns     int             `json:"total_open_vulns"`
	TotalOpenAlerts    int             `json:"total_open_alerts"`
	TotalActiveThreats int             `json:"total_active_threats"`
	Components         json.RawMessage `json:"components"`
	TopContributors    json.RawMessage `json:"top_contributors"`
	Recommendations    json.RawMessage `json:"recommendations"`
	SnapshotType       string          `json:"snapshot_type"`
	TriggerEvent       *string         `json:"trigger_event,omitempty"`
	CalculatedAt       time.Time       `json:"calculated_at"`
}

type RiskTrendPoint struct {
	Time              time.Time `json:"time"`
	OverallScore      float64   `json:"overall_score"`
	Grade             string    `json:"grade"`
	VulnerabilityRisk float64   `json:"vulnerability_risk"`
	ThreatRisk        float64   `json:"threat_risk"`
	ConfigRisk        float64   `json:"config_risk"`
	SurfaceRisk       float64   `json:"surface_risk"`
	ComplianceRisk    float64   `json:"compliance_risk"`
}

type HeatmapCell struct {
	VulnCount      int `json:"vuln_count"`
	AffectedAssets int `json:"affected_assets"`
}

type HeatmapRow struct {
	AssetType  string                 `json:"asset_type"`
	AssetCount int                    `json:"asset_count"`
	Cells      map[string]HeatmapCell `json:"cells"`
	TotalVulns int                    `json:"total_vulns"`
}

type RiskHeatmap struct {
	Rows     []HeatmapRow `json:"rows"`
	MaxValue int          `json:"max_value"`
}

type AgingBucketCell struct {
	Count int `json:"count"`
}

type AgingBucket struct {
	Label      string                    `json:"label"`
	MinDays    int                       `json:"min_days"`
	MaxDays    *int                      `json:"max_days,omitempty"`
	BySeverity map[string]AgingBucketCell `json:"by_severity"`
	Total      int                       `json:"total"`
	AvgAgeDays float64                   `json:"avg_age_days"`
}

type VulnerabilityAgingReport struct {
	Buckets    []AgingBucket `json:"buckets"`
	TotalOpen  int           `json:"total_open"`
	AvgAgeDays float64       `json:"avg_age_days"`
}

type VulnerabilityOverview struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	AssetID          uuid.UUID       `json:"asset_id"`
	AssetName        string          `json:"asset_name"`
	AssetType        string          `json:"asset_type"`
	AssetCriticality string          `json:"asset_criticality"`
	CVEID            *string         `json:"cve_id,omitempty"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Severity         string          `json:"severity"`
	CVSSScore        *float64        `json:"cvss_score,omitempty"`
	CVSSVector       *string         `json:"cvss_vector,omitempty"`
	Status           string          `json:"status"`
	DetectedAt       time.Time       `json:"detected_at"`
	ResolvedAt       *time.Time      `json:"resolved_at,omitempty"`
	Source           string          `json:"source"`
	Remediation      *string         `json:"remediation,omitempty"`
	Proof            *string         `json:"proof,omitempty"`
	Metadata         json.RawMessage `json:"metadata"`
	AgeDays          float64         `json:"age_days"`
	HasExploit       bool            `json:"has_exploit"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type VulnerabilityDetail struct {
	VulnerabilityOverview
	Asset     *Asset                 `json:"asset,omitempty"`
	Findings  []CTEMFindingReference `json:"ctem_findings,omitempty"`
}

type CTEMFindingReference struct {
	ID            uuid.UUID `json:"id"`
	AssessmentID  uuid.UUID `json:"assessment_id"`
	AssessmentName string   `json:"assessment_name"`
	Title         string    `json:"title"`
	PriorityScore float64   `json:"priority_score"`
	Status        string    `json:"status"`
}

type VulnerabilityStats struct {
	BySeverity map[string]int `json:"by_severity"`
	ByStatus   map[string]int `json:"by_status"`
	BySource   map[string]int `json:"by_source"`
	ByAge      map[string]int `json:"by_age_bucket"`
	TotalOpen  int            `json:"total_open"`
	AvgAgeDays float64        `json:"avg_age_days"`
}

type TopCVEEntry struct {
	CVEID           string  `json:"cve_id"`
	Title           string  `json:"title"`
	Severity        string  `json:"severity"`
	AffectedAssets  int     `json:"affected_assets"`
	OpenCount       int     `json:"open_count"`
	AvgCVSS         float64 `json:"avg_cvss"`
}

type KPICards struct {
	OpenAlerts              int     `json:"open_alerts"`
	CriticalAlerts          int     `json:"critical_alerts"`
	OpenVulnerabilities     int     `json:"open_vulnerabilities"`
	CriticalVulnerabilities int     `json:"critical_vulnerabilities"`
	ActiveThreats           int     `json:"active_threats"`
	MeanTimeToRespond       float64 `json:"mttr_hours"`
	MeanTimeToResolve       float64 `json:"mean_resolve_hours"`
	RiskScore               float64 `json:"risk_score"`
	RiskGrade               string  `json:"risk_grade"`
	AlertsDelta             int     `json:"alerts_delta"`
	VulnsDelta              int     `json:"vulns_delta"`
}

type AlertTimelinePoint struct {
	Bucket time.Time `json:"bucket"`
	Count  int       `json:"count"`
}

type AlertTimelineData struct {
	Granularity string               `json:"granularity"`
	Points      []AlertTimelinePoint `json:"points"`
}

type SeverityDistribution struct {
	Counts map[string]int `json:"counts"`
	Total  int            `json:"total"`
}

type DailyMetric struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

type AlertSummary struct {
	ID                uuid.UUID `json:"id"`
	Title             string    `json:"title"`
	Severity          string    `json:"severity"`
	Status            string    `json:"status"`
	AssetID           *uuid.UUID `json:"asset_id,omitempty"`
	AssetName         *string   `json:"asset_name,omitempty"`
	AssignedTo        *uuid.UUID `json:"assigned_to,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	MITRETechniqueID  *string   `json:"mitre_technique_id,omitempty"`
	MITRETechniqueName *string  `json:"mitre_technique_name,omitempty"`
}

type AssetAlertSummary struct {
	AssetID      uuid.UUID `json:"asset_id"`
	AssetName    string    `json:"asset_name"`
	AssetType    string    `json:"asset_type"`
	Criticality  string    `json:"criticality"`
	AlertCount   int       `json:"alert_count"`
	CriticalOpen int       `json:"critical_open"`
}

type AnalystWorkloadEntry struct {
	UserID           uuid.UUID `json:"user_id"`
	Name             string    `json:"name"`
	OpenAssigned     int       `json:"open_assigned"`
	CriticalOpen     int       `json:"critical_open"`
	ResolvedThisWeek int       `json:"resolved_this_week"`
	AvgResolveHours  *float64  `json:"avg_resolve_hours"`
}

type MTTREntry struct {
	AvgResponseHours    float64  `json:"avg_response_hours"`
	MedianResponseHours float64  `json:"median_response_hours"`
	P95ResponseHours    float64  `json:"p95_response_hours"`
	AvgResolveHours     *float64 `json:"avg_resolve_hours,omitempty"`
	SampleSize          int      `json:"sample_size"`
	SLACompliance       float64  `json:"sla_compliance"`
}

type MTTRReport struct {
	BySeverity map[string]MTTREntry `json:"by_severity"`
	Overall    MTTREntry            `json:"overall"`
	Period     string               `json:"period"`
}

type MITREHeatmapCell struct {
	TacticID      string    `json:"tactic_id"`
	TacticName    string    `json:"tactic_name"`
	TechniqueID   string    `json:"technique_id"`
	TechniqueName string    `json:"technique_name"`
	AlertCount    int       `json:"alert_count"`
	CriticalCount int       `json:"critical_count"`
	LastSeen      time.Time `json:"last_seen"`
	HasDetection  bool      `json:"has_detection"`
}

type MITREHeatmapData struct {
	Cells    []MITREHeatmapCell `json:"cells"`
	MaxCount int                `json:"max_count"`
}

type SOCDashboard struct {
	KPIs                 KPICards              `json:"kpis"`
	AlertTimeline        AlertTimelineData     `json:"alert_timeline"`
	SeverityDistribution SeverityDistribution  `json:"severity_distribution"`
	AlertTrend           []DailyMetric         `json:"alert_trend"`
	VulnerabilityTrend   []DailyMetric         `json:"vulnerability_trend"`
	RecentAlerts         []AlertSummary        `json:"recent_alerts"`
	TopAttackedAssets    []AssetAlertSummary   `json:"top_attacked_assets"`
	AnalystWorkload      []AnalystWorkloadEntry `json:"analyst_workload"`
	MITREHeatmap         MITREHeatmapData      `json:"mitre_heatmap"`
	RiskScore            *OrganizationRiskScore `json:"risk_score"`
	CachedAt             *time.Time            `json:"cached_at,omitempty"`
	CalculatedAt         time.Time             `json:"calculated_at"`
	PartialFailures      []string              `json:"partial_failures,omitempty"`
}
