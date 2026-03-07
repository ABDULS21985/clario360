package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CTEMAssessmentStatus string

const (
	CTEMAssessmentStatusCreated      CTEMAssessmentStatus = "created"
	CTEMAssessmentStatusScoping      CTEMAssessmentStatus = "scoping"
	CTEMAssessmentStatusDiscovery    CTEMAssessmentStatus = "discovery"
	CTEMAssessmentStatusPrioritizing CTEMAssessmentStatus = "prioritizing"
	CTEMAssessmentStatusValidating   CTEMAssessmentStatus = "validating"
	CTEMAssessmentStatusMobilizing   CTEMAssessmentStatus = "mobilizing"
	CTEMAssessmentStatusCompleted    CTEMAssessmentStatus = "completed"
	CTEMAssessmentStatusFailed       CTEMAssessmentStatus = "failed"
	CTEMAssessmentStatusCancelled    CTEMAssessmentStatus = "cancelled"
)

var ValidCTEMAssessmentStatuses = []CTEMAssessmentStatus{
	CTEMAssessmentStatusCreated,
	CTEMAssessmentStatusScoping,
	CTEMAssessmentStatusDiscovery,
	CTEMAssessmentStatusPrioritizing,
	CTEMAssessmentStatusValidating,
	CTEMAssessmentStatusMobilizing,
	CTEMAssessmentStatusCompleted,
	CTEMAssessmentStatusFailed,
	CTEMAssessmentStatusCancelled,
}

func (s CTEMAssessmentStatus) IsValid() bool {
	for _, candidate := range ValidCTEMAssessmentStatuses {
		if candidate == s {
			return true
		}
	}
	return false
}

type CTEMPhaseStatus string

const (
	CTEMPhaseStatusPending   CTEMPhaseStatus = "pending"
	CTEMPhaseStatusRunning   CTEMPhaseStatus = "running"
	CTEMPhaseStatusCompleted CTEMPhaseStatus = "completed"
	CTEMPhaseStatusFailed    CTEMPhaseStatus = "failed"
	CTEMPhaseStatusSkipped   CTEMPhaseStatus = "skipped"
	CTEMPhaseStatusCancelled CTEMPhaseStatus = "cancelled"
)

type AssessmentScope struct {
	AssetTypes      []string    `json:"asset_types,omitempty"`
	AssetTags       []string    `json:"asset_tags,omitempty"`
	AssetIDs        []uuid.UUID `json:"asset_ids,omitempty"`
	Departments     []string    `json:"departments,omitempty"`
	CIDRRanges      []string    `json:"cidr_ranges,omitempty"`
	ExcludeAssetIDs []uuid.UUID `json:"exclude_asset_ids,omitempty"`
}

type PhaseProgress struct {
	Status         CTEMPhaseStatus `json:"status"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	ItemsProcessed int             `json:"items_processed"`
	ItemsTotal     int             `json:"items_total"`
	Errors         []string        `json:"errors,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
}

type CTEMAssessment struct {
	ID                 uuid.UUID                `json:"id" db:"id"`
	TenantID           uuid.UUID                `json:"tenant_id" db:"tenant_id"`
	Name               string                   `json:"name" db:"name"`
	Description        string                   `json:"description" db:"description"`
	Status             CTEMAssessmentStatus     `json:"status" db:"status"`
	Scope              AssessmentScope          `json:"scope" db:"scope"`
	ResolvedAssetIDs   []uuid.UUID              `json:"resolved_asset_ids" db:"resolved_asset_ids"`
	ResolvedAssetCount int                      `json:"resolved_asset_count" db:"resolved_asset_count"`
	Phases             map[string]PhaseProgress `json:"phases" db:"phases"`
	CurrentPhase       *string                  `json:"current_phase,omitempty" db:"current_phase"`
	ExposureScore      *float64                 `json:"exposure_score,omitempty" db:"exposure_score"`
	ScoreBreakdown     json.RawMessage          `json:"score_breakdown,omitempty" db:"score_breakdown"`
	FindingsSummary    json.RawMessage          `json:"findings_summary,omitempty" db:"findings_summary"`
	StartedAt          *time.Time               `json:"started_at,omitempty" db:"started_at"`
	CompletedAt        *time.Time               `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs         *int64                   `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage       *string                  `json:"error_message,omitempty" db:"error_message"`
	ErrorPhase         *string                  `json:"error_phase,omitempty" db:"error_phase"`
	Scheduled          bool                     `json:"scheduled" db:"scheduled"`
	ScheduleCron       *string                  `json:"schedule_cron,omitempty" db:"schedule_cron"`
	ParentAssessmentID *uuid.UUID               `json:"parent_assessment_id,omitempty" db:"parent_assessment_id"`
	Tags               []string                 `json:"tags" db:"tags"`
	CreatedBy          *uuid.UUID               `json:"created_by,omitempty" db:"created_by"`
	CreatedAt          time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at" db:"updated_at"`
	DeletedAt          *time.Time               `json:"-" db:"deleted_at"`
}

type CTEMFindingType string

const (
	CTEMFindingTypeVulnerability      CTEMFindingType = "vulnerability"
	CTEMFindingTypeMisconfiguration   CTEMFindingType = "misconfiguration"
	CTEMFindingTypeAttackPath         CTEMFindingType = "attack_path"
	CTEMFindingTypeExposure           CTEMFindingType = "exposure"
	CTEMFindingTypeWeakCredential     CTEMFindingType = "weak_credential"
	CTEMFindingTypeMissingPatch       CTEMFindingType = "missing_patch"
	CTEMFindingTypeExpiredCertificate CTEMFindingType = "expired_certificate"
	CTEMFindingTypeInsecureProtocol   CTEMFindingType = "insecure_protocol"
)

var ValidCTEMFindingTypes = []CTEMFindingType{
	CTEMFindingTypeVulnerability,
	CTEMFindingTypeMisconfiguration,
	CTEMFindingTypeAttackPath,
	CTEMFindingTypeExposure,
	CTEMFindingTypeWeakCredential,
	CTEMFindingTypeMissingPatch,
	CTEMFindingTypeExpiredCertificate,
	CTEMFindingTypeInsecureProtocol,
}

func (t CTEMFindingType) IsValid() bool {
	for _, candidate := range ValidCTEMFindingTypes {
		if candidate == t {
			return true
		}
	}
	return false
}

type CTEMFindingCategory string

const (
	CTEMFindingCategoryTechnical     CTEMFindingCategory = "technical"
	CTEMFindingCategoryConfiguration CTEMFindingCategory = "configuration"
	CTEMFindingCategoryArchitectural CTEMFindingCategory = "architectural"
	CTEMFindingCategoryOperational   CTEMFindingCategory = "operational"
)

type CTEMValidationStatus string

const (
	CTEMValidationPending        CTEMValidationStatus = "pending"
	CTEMValidationValidated      CTEMValidationStatus = "validated"
	CTEMValidationCompensated    CTEMValidationStatus = "compensated"
	CTEMValidationNotExploitable CTEMValidationStatus = "not_exploitable"
	CTEMValidationRequiresManual CTEMValidationStatus = "requires_manual"
)

type CTEMRemediationType string

const (
	CTEMRemediationPatch         CTEMRemediationType = "patch"
	CTEMRemediationConfiguration CTEMRemediationType = "configuration"
	CTEMRemediationArchitecture  CTEMRemediationType = "architecture"
	CTEMRemediationUpgrade       CTEMRemediationType = "upgrade"
	CTEMRemediationDecommission  CTEMRemediationType = "decommission"
	CTEMRemediationAcceptRisk    CTEMRemediationType = "accept_risk"
)

type CTEMRemediationEffort string

const (
	CTEMRemediationEffortLow    CTEMRemediationEffort = "low"
	CTEMRemediationEffortMedium CTEMRemediationEffort = "medium"
	CTEMRemediationEffortHigh   CTEMRemediationEffort = "high"
)

type CTEMFindingStatus string

const (
	CTEMFindingStatusOpen          CTEMFindingStatus = "open"
	CTEMFindingStatusInRemediation CTEMFindingStatus = "in_remediation"
	CTEMFindingStatusRemediated    CTEMFindingStatus = "remediated"
	CTEMFindingStatusAcceptedRisk  CTEMFindingStatus = "accepted_risk"
	CTEMFindingStatusFalsePositive CTEMFindingStatus = "false_positive"
	CTEMFindingStatusDeferred      CTEMFindingStatus = "deferred"
)

type CTEMFinding struct {
	ID                     uuid.UUID              `json:"id" db:"id"`
	TenantID               uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	AssessmentID           uuid.UUID              `json:"assessment_id" db:"assessment_id"`
	Type                   CTEMFindingType        `json:"type" db:"type"`
	Category               CTEMFindingCategory    `json:"category" db:"category"`
	Severity               string                 `json:"severity" db:"severity"`
	Title                  string                 `json:"title" db:"title"`
	Description            string                 `json:"description" db:"description"`
	Evidence               json.RawMessage        `json:"evidence" db:"evidence"`
	AffectedAssetIDs       []uuid.UUID            `json:"affected_asset_ids" db:"affected_asset_ids"`
	AffectedAssetCount     int                    `json:"affected_asset_count" db:"affected_asset_count"`
	PrimaryAssetID         *uuid.UUID             `json:"primary_asset_id,omitempty" db:"primary_asset_id"`
	VulnerabilityIDs       []uuid.UUID            `json:"vulnerability_ids" db:"vulnerability_ids"`
	CVEIDs                 []string               `json:"cve_ids" db:"cve_ids"`
	BusinessImpactScore    float64                `json:"business_impact_score" db:"business_impact_score"`
	BusinessImpactFactors  json.RawMessage        `json:"business_impact_factors" db:"business_impact_factors"`
	ExploitabilityScore    float64                `json:"exploitability_score" db:"exploitability_score"`
	ExploitabilityFactors  json.RawMessage        `json:"exploitability_factors" db:"exploitability_factors"`
	PriorityScore          float64                `json:"priority_score" db:"priority_score"`
	PriorityGroup          int                    `json:"priority_group" db:"priority_group"`
	PriorityRank           *int                   `json:"priority_rank,omitempty" db:"priority_rank"`
	ValidationStatus       CTEMValidationStatus   `json:"validation_status" db:"validation_status"`
	CompensatingControls   []string               `json:"compensating_controls" db:"compensating_controls"`
	ValidationNotes        *string                `json:"validation_notes,omitempty" db:"validation_notes"`
	ValidatedAt            *time.Time             `json:"validated_at,omitempty" db:"validated_at"`
	RemediationType        *CTEMRemediationType   `json:"remediation_type,omitempty" db:"remediation_type"`
	RemediationDescription *string                `json:"remediation_description,omitempty" db:"remediation_description"`
	RemediationEffort      *CTEMRemediationEffort `json:"remediation_effort,omitempty" db:"remediation_effort"`
	RemediationGroupID     *uuid.UUID             `json:"remediation_group_id,omitempty" db:"remediation_group_id"`
	EstimatedDays          *int                   `json:"estimated_days,omitempty" db:"estimated_days"`
	Status                 CTEMFindingStatus      `json:"status" db:"status"`
	StatusChangedBy        *uuid.UUID             `json:"status_changed_by,omitempty" db:"status_changed_by"`
	StatusChangedAt        *time.Time             `json:"status_changed_at,omitempty" db:"status_changed_at"`
	StatusNotes            *string                `json:"status_notes,omitempty" db:"status_notes"`
	AttackPath             json.RawMessage        `json:"attack_path,omitempty" db:"attack_path"`
	AttackPathLength       *int                   `json:"attack_path_length,omitempty" db:"attack_path_length"`
	Metadata               json.RawMessage        `json:"metadata" db:"metadata"`
	CreatedAt              time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at" db:"updated_at"`
}

type CTEMRemediationGroupStatus string

const (
	CTEMRemediationGroupPlanned    CTEMRemediationGroupStatus = "planned"
	CTEMRemediationGroupInProgress CTEMRemediationGroupStatus = "in_progress"
	CTEMRemediationGroupCompleted  CTEMRemediationGroupStatus = "completed"
	CTEMRemediationGroupDeferred   CTEMRemediationGroupStatus = "deferred"
	CTEMRemediationGroupAccepted   CTEMRemediationGroupStatus = "accepted"
)

type CTEMRemediationGroup struct {
	ID                 uuid.UUID                  `json:"id" db:"id"`
	TenantID           uuid.UUID                  `json:"tenant_id" db:"tenant_id"`
	AssessmentID       uuid.UUID                  `json:"assessment_id" db:"assessment_id"`
	Title              string                     `json:"title" db:"title"`
	Description        string                     `json:"description" db:"description"`
	Type               CTEMRemediationType        `json:"type" db:"type"`
	FindingCount       int                        `json:"finding_count" db:"finding_count"`
	AffectedAssetCount int                        `json:"affected_asset_count" db:"affected_asset_count"`
	CVEIDs             []string                   `json:"cve_ids" db:"cve_ids"`
	MaxPriorityScore   float64                    `json:"max_priority_score" db:"max_priority_score"`
	PriorityGroup      int                        `json:"priority_group" db:"priority_group"`
	Effort             CTEMRemediationEffort      `json:"effort" db:"effort"`
	EstimatedDays      *int                       `json:"estimated_days,omitempty" db:"estimated_days"`
	ScoreReduction     *float64                   `json:"score_reduction,omitempty" db:"score_reduction"`
	Status             CTEMRemediationGroupStatus `json:"status" db:"status"`
	WorkflowInstanceID *string                    `json:"workflow_instance_id,omitempty" db:"workflow_instance_id"`
	TargetDate         *time.Time                 `json:"target_date,omitempty" db:"target_date"`
	StartedAt          *time.Time                 `json:"started_at,omitempty" db:"started_at"`
	CompletedAt        *time.Time                 `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt          time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at" db:"updated_at"`
}

type ExposureScore struct {
	Score        float64        `json:"score"`
	Grade        string         `json:"grade"`
	Breakdown    ScoreBreakdown `json:"breakdown"`
	Trend        string         `json:"trend"`
	TrendDelta   float64        `json:"trend_delta"`
	CalculatedAt time.Time      `json:"calculated_at"`
}

type ScoreBreakdown struct {
	VulnerabilityScore  float64 `json:"vulnerability_score"`
	AttackSurfaceScore  float64 `json:"attack_surface_score"`
	ThreatExposureScore float64 `json:"threat_exposure_score"`
	RemediationVelocity float64 `json:"remediation_velocity_score"`
}

type ExposureScoreSnapshot struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	TenantID     uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Score        float64         `json:"score" db:"score"`
	Breakdown    json.RawMessage `json:"breakdown" db:"breakdown"`
	AssetCount   int             `json:"asset_count" db:"asset_count"`
	VulnCount    int             `json:"vuln_count" db:"vuln_count"`
	FindingCount int             `json:"finding_count" db:"finding_count"`
	AssessmentID *uuid.UUID      `json:"assessment_id,omitempty" db:"assessment_id"`
	SnapshotType string          `json:"snapshot_type" db:"snapshot_type"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

type TimeSeriesPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type AssetExposureSummary struct {
	AssetID      uuid.UUID `json:"asset_id"`
	AssetName    string    `json:"asset_name"`
	AssetType    string    `json:"asset_type"`
	Criticality  string    `json:"criticality"`
	FindingCount int       `json:"finding_count"`
	HighestScore float64   `json:"highest_priority_score"`
}

type AttackPathSummary struct {
	FindingID   uuid.UUID `json:"finding_id"`
	Title       string    `json:"title"`
	Score       float64   `json:"score"`
	PathLength  int       `json:"path_length"`
	EntryAsset  string    `json:"entry_asset"`
	TargetAsset string    `json:"target_asset"`
}

type RemediationGroupStats struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByType     map[string]int `json:"by_type"`
	InProgress int            `json:"in_progress"`
	Completed  int            `json:"completed"`
}

type ComplianceSummary struct {
	ValidatedPercent      float64 `json:"validated_percent"`
	AcceptedRiskFindings  int     `json:"accepted_risk_findings"`
	ImmediateOpenFindings int     `json:"immediate_open_findings"`
	OverdueGroups         int     `json:"overdue_groups"`
}

type CTEMDashboard struct {
	ExposureScore           ExposureScore          `json:"exposure_score"`
	ExposureScoreTrend      []TimeSeriesPoint      `json:"exposure_score_trend"`
	FindingsByPriorityGroup map[int]int            `json:"findings_by_priority_group"`
	FindingsBySeverity      map[string]int         `json:"findings_by_severity"`
	FindingsByType          map[string]int         `json:"findings_by_type"`
	FindingsByStatus        map[string]int         `json:"findings_by_status"`
	RemediationRate         float64                `json:"remediation_rate"`
	MeanTimeToRemediate     map[string]float64     `json:"mttr_by_severity"`
	TopExposedAssets        []AssetExposureSummary `json:"top_exposed_assets"`
	TopAttackPaths          []AttackPathSummary    `json:"top_attack_paths"`
	ActiveAssessments       int                    `json:"active_assessments"`
	LastAssessmentDate      *time.Time             `json:"last_assessment_date"`
	RemediationGroupStats   RemediationGroupStats  `json:"remediation_groups"`
	ComplianceSummary       ComplianceSummary      `json:"compliance_summary"`
}

type CTEMAssessmentComparison struct {
	Current  CTEMAssessmentComparisonSide `json:"current"`
	Previous CTEMAssessmentComparisonSide `json:"previous"`
	Delta    CTEMAssessmentDelta          `json:"delta"`
}

type CTEMAssessmentComparisonSide struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name"`
	ExposureScore *float64       `json:"exposure_score,omitempty"`
	Findings      map[string]int `json:"findings"`
}

type FindingSummary struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Type          string    `json:"type"`
	Severity      string    `json:"severity"`
	PriorityScore float64   `json:"priority_score"`
}

type CTEMAssessmentDelta struct {
	ScoreChange       float64          `json:"score_change"`
	ScoreDirection    string           `json:"score_direction"`
	FindingsNew       int              `json:"findings_new"`
	FindingsResolved  int              `json:"findings_resolved"`
	FindingsUnchanged int              `json:"findings_unchanged"`
	FindingsWorsened  int              `json:"findings_worsened"`
	NewFindings       []FindingSummary `json:"new_findings"`
	ResolvedFindings  []FindingSummary `json:"resolved_findings"`
}

type CTEMReport struct {
	Assessment        *CTEMAssessment         `json:"assessment"`
	Scoping           json.RawMessage         `json:"scoping"`
	Discovery         map[string]int          `json:"discovery"`
	Findings          []*CTEMFinding          `json:"findings"`
	RemediationGroups []*CTEMRemediationGroup `json:"remediation_groups"`
	ExposureScore     ExposureScore           `json:"exposure_score"`
	ExecutiveSummary  string                  `json:"executive_summary"`
	GeneratedAt       time.Time               `json:"generated_at"`
}
