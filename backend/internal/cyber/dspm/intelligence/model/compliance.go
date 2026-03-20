package model

import (
	"time"

	"github.com/google/uuid"
)

// ControlStatus indicates the compliance state of a control.
type ControlStatus string

const (
	ControlCompliant     ControlStatus = "compliant"
	ControlPartial       ControlStatus = "partial"
	ControlNonCompliant  ControlStatus = "non_compliant"
	ControlNotApplicable ControlStatus = "not_applicable"
)

// TrendDirection describes the direction of a compliance score trend.
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendStable    TrendDirection = "stable"
	TrendDeclining TrendDirection = "declining"
)

// ComplianceFramework identifies a compliance standard.
type ComplianceFramework string

const (
	FrameworkGDPR      ComplianceFramework = "gdpr"
	FrameworkHIPAA     ComplianceFramework = "hipaa"
	FrameworkSOC2      ComplianceFramework = "soc2"
	FrameworkPCIDSS    ComplianceFramework = "pci_dss"
	FrameworkSaudiPDPL ComplianceFramework = "saudi_pdpl"
	FrameworkISO27001  ComplianceFramework = "iso27001"
)

// AllFrameworks returns all supported compliance frameworks.
func AllFrameworks() []ComplianceFramework {
	return []ComplianceFramework{
		FrameworkGDPR, FrameworkHIPAA, FrameworkSOC2,
		FrameworkPCIDSS, FrameworkSaudiPDPL, FrameworkISO27001,
	}
}

// ControlGap identifies a specific asset failing a control.
type ControlGap struct {
	AssetID       uuid.UUID  `json:"asset_id"`
	AssetName     string     `json:"asset_name"`
	Gap           string     `json:"gap"`
	RemediationID *uuid.UUID `json:"remediation_id,omitempty"`
}

// ControlDetail is a per-control compliance evaluation.
type ControlDetail struct {
	ControlID          string        `json:"control_id"`
	Name               string        `json:"name"`
	Description        string        `json:"description,omitempty"`
	Status             ControlStatus `json:"status"`
	Score              float64       `json:"score"`
	AssetsCompliant    int           `json:"assets_compliant"`
	AssetsNonCompliant int           `json:"assets_non_compliant"`
	AssetsTotal        int           `json:"assets_total"`
	Gaps               []ControlGap  `json:"gaps,omitempty"`
}

// CompliancePosture is the overall compliance status for one framework.
type CompliancePosture struct {
	ID                    uuid.UUID           `json:"id"`
	TenantID              uuid.UUID           `json:"tenant_id"`
	Framework             ComplianceFramework `json:"framework"`
	OverallScore          float64             `json:"overall_score"`
	ControlsTotal         int                 `json:"controls_total"`
	ControlsCompliant     int                 `json:"controls_compliant"`
	ControlsPartial       int                 `json:"controls_partial"`
	ControlsNonCompliant  int                 `json:"controls_non_compliant"`
	ControlsNotApplicable int                 `json:"controls_not_applicable"`
	ControlDetails        []ControlDetail     `json:"control_details"`
	Score7dAgo            *float64            `json:"score_7d_ago,omitempty"`
	Score30dAgo           *float64            `json:"score_30d_ago,omitempty"`
	Score90dAgo           *float64            `json:"score_90d_ago,omitempty"`
	TrendDirection        TrendDirection      `json:"trend_direction"`
	EstimatedFineExposure float64             `json:"estimated_fine_exposure"`
	FineCurrency          string              `json:"fine_currency"`
	EvaluatedAt           time.Time           `json:"evaluated_at"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

// ComplianceGap aggregates failing controls across frameworks.
type ComplianceGap struct {
	Framework   ComplianceFramework `json:"framework"`
	ControlID   string              `json:"control_id"`
	ControlName string              `json:"control_name"`
	Severity    string              `json:"severity"`
	AssetCount  int                 `json:"asset_count"`
	Gaps        []ControlGap        `json:"gaps"`
}

// ResidencyViolation flags a data residency infraction.
type ResidencyViolation struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Regulation     string    `json:"regulation"`
	RequiredRegion string    `json:"required_region"`
	ActualRegion   string    `json:"actual_region"`
	Severity       string    `json:"severity"`
	Description    string    `json:"description"`
}

// ControlDefinition is a single control in a compliance framework config.
type ControlDefinition struct {
	ControlID   string `json:"control_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	// Scope defines which assets are in-scope for this control.
	// "pii" = assets with contains_pii=true, "all" = all assets, "high_risk" = risk_score >= 75 AND contains_pii
	Scope string `json:"scope"`
	// CheckFn is set programmatically, not serialized.
}

// AuditReport is the audit evidence package for regulatory compliance.
type AuditReport struct {
	Framework         ComplianceFramework `json:"framework"`
	TenantID          uuid.UUID           `json:"tenant_id"`
	GeneratedAt       time.Time           `json:"generated_at"`
	ComplianceSummary CompliancePosture   `json:"compliance_summary"`
	AssetInventory    []AuditAssetEntry   `json:"asset_inventory"`
	GapAnalysis       []ComplianceGap     `json:"gap_analysis"`
	ExceptionLog      []AuditException    `json:"exception_log"`
	ScoreTrend        []AuditScorePoint   `json:"score_trend"`
}

// AuditAssetEntry is an asset inventory line in the audit report.
type AuditAssetEntry struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Classification string    `json:"classification"`
	PostureScore   float64   `json:"posture_score"`
	RiskScore      float64   `json:"risk_score"`
	Encrypted      bool      `json:"encrypted"`
	AccessControl  string    `json:"access_control"`
}

// AuditException is a risk exception entry in the audit report.
type AuditException struct {
	ExceptionID   uuid.UUID `json:"exception_id"`
	AssetName     string    `json:"asset_name"`
	Justification string    `json:"justification"`
	ApprovedBy    string    `json:"approved_by"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// AuditScorePoint is a compliance score at a point in time.
type AuditScorePoint struct {
	Date  string  `json:"date"`
	Score float64 `json:"score"`
}
