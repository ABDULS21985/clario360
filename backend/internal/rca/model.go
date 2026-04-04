package rca

import (
	"time"

	"github.com/google/uuid"
)

// AnalysisType identifies the kind of RCA being performed.
type AnalysisType string

const (
	AnalysisTypeSecurity AnalysisType = "security_alert"
	AnalysisTypePipeline AnalysisType = "pipeline_failure"
	AnalysisTypeQuality  AnalysisType = "quality_issue"
)

// RootCauseAnalysis is the complete result of an RCA investigation.
type RootCauseAnalysis struct {
	ID              uuid.UUID        `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	Type            AnalysisType     `json:"type"`
	IncidentID      uuid.UUID        `json:"incident_id"`
	Status          string           `json:"status"` // "pending", "analyzing", "completed", "failed"
	RootCause       *CausalStep      `json:"root_cause,omitempty"`
	CausalChain     []CausalStep     `json:"causal_chain"`
	Timeline        []TimelineEvent  `json:"timeline"`
	Impact          *ImpactAssessment `json:"impact,omitempty"`
	Recommendations []Recommendation `json:"recommendations"`
	Confidence      float64          `json:"confidence"` // 0.0-1.0
	Summary         string           `json:"summary"`
	AnalyzedAt      time.Time        `json:"analyzed_at"`
	DurationMs      int64            `json:"duration_ms"`
}

// CausalStep represents one step in the causal chain.
type CausalStep struct {
	Order       int                    `json:"order"`
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    string                 `json:"severity,omitempty"`
	MITREPhase  string                 `json:"mitre_phase,omitempty"`
	MITRETechID string                 `json:"mitre_technique_id,omitempty"`
	Evidence    []Evidence             `json:"evidence"`
	IsRootCause bool                   `json:"is_root_cause"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Evidence is a structured fact supporting a causal step.
type Evidence struct {
	Label       string      `json:"label"`
	Field       string      `json:"field"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// TimelineEvent is a chronologically ordered event in the investigation.
type TimelineEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"` // "alert", "ueba", "audit", "iam", "pipeline"
	Type        string                 `json:"type"`
	Summary     string                 `json:"summary"`
	Severity    string                 `json:"severity,omitempty"`
	SourceIP    string                 `json:"source_ip,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	AssetID     string                 `json:"asset_id,omitempty"`
	MITREPhase  string                 `json:"mitre_phase,omitempty"`
	MITRETechID string                 `json:"mitre_technique_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ImpactAssessment describes the blast radius of the incident.
type ImpactAssessment struct {
	DirectAssets     []AffectedAsset `json:"direct_assets"`
	TransitiveAssets []AffectedAsset `json:"transitive_assets"`
	TotalAffected    int             `json:"total_affected"`
	DataAtRisk       []DataRisk      `json:"data_at_risk"`
	UsersAtRisk      int             `json:"users_at_risk"`
	BusinessImpact   string          `json:"business_impact"` // "critical", "high", "medium", "low"
	Summary          string          `json:"summary"`
}

// AffectedAsset is an asset impacted by the incident.
type AffectedAsset struct {
	AssetID     uuid.UUID `json:"asset_id"`
	AssetName   string    `json:"asset_name"`
	AssetType   string    `json:"asset_type"`
	Criticality string    `json:"criticality"`
	ImpactType  string    `json:"impact_type"` // "direct", "transitive"
}

// DataRisk describes data classification at risk in the incident.
type DataRisk struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Classification string    `json:"classification"`
	ContainsPII    bool      `json:"contains_pii"`
	PIITypes       []string  `json:"pii_types,omitempty"`
}

// Recommendation is an actionable suggestion based on the root cause.
type Recommendation struct {
	Priority      int    `json:"priority"` // 1 = highest
	Category      string `json:"category"` // "immediate", "short_term", "long_term"
	Action        string `json:"action"`
	Rationale     string `json:"rationale"`
	RootCauseType string `json:"root_cause_type"`
}

// AnalyzeRequest is the API request to trigger RCA.
type AnalyzeRequest struct {
	Type       AnalysisType `json:"type"`
	IncidentID uuid.UUID    `json:"incident_id"`
}
