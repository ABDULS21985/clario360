package model

import (
	"time"

	"github.com/google/uuid"
)

// ─── Risk ───────────────────────────────────────────────────────────────────

// VCISORiskEntry represents a single entry in the risk register.
type VCISORiskEntry struct {
	ID                       uuid.UUID  `json:"id"`
	TenantID                 uuid.UUID  `json:"tenant_id"`
	Title                    string     `json:"title"`
	Description              string     `json:"description"`
	Category                 string     `json:"category"`
	Department               string     `json:"department"`
	InherentScore            int        `json:"inherent_score"`
	ResidualScore            int        `json:"residual_score"`
	Likelihood               string     `json:"likelihood"`
	Impact                   string     `json:"impact"`
	Status                   string     `json:"status"`
	Treatment                string     `json:"treatment"`
	OwnerID                  *uuid.UUID `json:"owner_id,omitempty"`
	OwnerName                string     `json:"owner_name"`
	ReviewDate               *string    `json:"review_date,omitempty"`
	BusinessServices         []string   `json:"business_services"`
	Controls                 []string   `json:"controls"`
	Tags                     []string   `json:"tags"`
	TreatmentPlan            string     `json:"treatment_plan"`
	AcceptanceRationale      *string    `json:"acceptance_rationale,omitempty"`
	AcceptanceApprovedBy     *uuid.UUID `json:"acceptance_approved_by,omitempty"`
	AcceptanceApprovedByName *string    `json:"acceptance_approved_by_name,omitempty"`
	AcceptanceExpiry         *string    `json:"acceptance_expiry,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// VCISORiskStats carries aggregate risk statistics.
type VCISORiskStats struct {
	Total            int            `json:"total"`
	ByStatus         map[string]int `json:"by_status"`
	ByTreatment      map[string]int `json:"by_treatment"`
	ByLikelihood     map[string]int `json:"by_likelihood"`
	ByImpact         map[string]int `json:"by_impact"`
	AvgInherentScore float64        `json:"avg_inherent_score"`
	AvgResidualScore float64        `json:"avg_residual_score"`
	OverdueReviews   int            `json:"overdue_reviews"`
	AcceptedCount    int            `json:"accepted_count"`
}

// ─── Policy ─────────────────────────────────────────────────────────────────

// VCISOPolicy represents a governance policy document.
type VCISOPolicy struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Title           string     `json:"title"`
	Domain          string     `json:"domain"`
	Version         string     `json:"version"`
	Status          string     `json:"status"`
	Content         string     `json:"content"`
	OwnerID         uuid.UUID  `json:"owner_id"`
	OwnerName       string     `json:"owner_name"`
	ReviewerID      *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewerName    *string    `json:"reviewer_name,omitempty"`
	ApprovedBy      *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedByName  *string    `json:"approved_by_name,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ReviewDue       string     `json:"review_due"`
	LastReviewedAt  *time.Time `json:"last_reviewed_at,omitempty"`
	Tags            []string   `json:"tags"`
	ExceptionsCount int        `json:"exceptions_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ─── Policy Exception ──────────────────────────────────────────────────────

// VCISOPolicyException represents an exception request against a policy.
type VCISOPolicyException struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	PolicyID             uuid.UUID  `json:"policy_id"`
	PolicyTitle          string     `json:"policy_title"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Justification        string     `json:"justification"`
	CompensatingControls string     `json:"compensating_controls"`
	Status               string     `json:"status"`
	RequestedBy          uuid.UUID  `json:"requested_by"`
	RequestedByName      string     `json:"requested_by_name"`
	ApprovedBy           *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedByName       *string    `json:"approved_by_name,omitempty"`
	DecisionNotes        *string    `json:"decision_notes,omitempty"`
	ExpiresAt            string     `json:"expires_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ─── Vendor ─────────────────────────────────────────────────────────────────

// VCISOVendor represents a third-party vendor record.
type VCISOVendor struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Name                 string     `json:"name"`
	Category             string     `json:"category"`
	RiskTier             string     `json:"risk_tier"`
	Status               string     `json:"status"`
	RiskScore            int        `json:"risk_score"`
	LastAssessmentDate   *time.Time `json:"last_assessment_date,omitempty"`
	NextReviewDate       string     `json:"next_review_date"`
	ContactName          *string    `json:"contact_name,omitempty"`
	ContactEmail         *string    `json:"contact_email,omitempty"`
	ServicesProvided     []string   `json:"services_provided"`
	DataShared           []string   `json:"data_shared"`
	ComplianceFrameworks []string   `json:"compliance_frameworks"`
	ControlsMet          int        `json:"controls_met"`
	ControlsTotal        int        `json:"controls_total"`
	OpenFindings         int        `json:"open_findings"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ─── Questionnaire ──────────────────────────────────────────────────────────

// VCISOQuestionnaire represents a vendor/audit questionnaire.
type VCISOQuestionnaire struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	Title             string     `json:"title"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	VendorID          *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName        *string    `json:"vendor_name,omitempty"`
	TotalQuestions    int        `json:"total_questions"`
	AnsweredQuestions int        `json:"answered_questions"`
	DueDate           string     `json:"due_date"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	Score             *int       `json:"score,omitempty"`
	AssignedTo        *uuid.UUID `json:"assigned_to,omitempty"`
	AssignedToName    *string    `json:"assigned_to_name,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ─── Evidence ───────────────────────────────────────────────────────────────

// VCISOEvidence represents a piece of compliance evidence.
type VCISOEvidence struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Type           string     `json:"type"`
	Source         string     `json:"source"`
	Status         string     `json:"status"`
	Frameworks     []string   `json:"frameworks"`
	ControlIDs     []string   `json:"control_ids"`
	FileName       *string    `json:"file_name,omitempty"`
	FileSize       *int       `json:"file_size,omitempty"`
	FileURL        *string    `json:"file_url,omitempty"`
	CollectedAt    time.Time  `json:"collected_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CollectorName  *string    `json:"collector_name,omitempty"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// VCISOEvidenceStats carries aggregate evidence statistics.
type VCISOEvidenceStats struct {
	Total                   int            `json:"total"`
	ByStatus                map[string]int `json:"by_status"`
	ByType                  map[string]int `json:"by_type"`
	BySource                map[string]int `json:"by_source"`
	StaleCount              int            `json:"stale_count"`
	ExpiredCount            int            `json:"expired_count"`
	FrameworksCovered       int            `json:"frameworks_covered"`
	ControlsWithEvidence    int            `json:"controls_with_evidence"`
	ControlsWithoutEvidence int            `json:"controls_without_evidence"`
}

// ─── Maturity ───────────────────────────────────────────────────────────────

// VCISOMaturityDimension represents a single dimension in a maturity assessment.
type VCISOMaturityDimension struct {
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	CurrentLevel    int      `json:"current_level"`
	TargetLevel     int      `json:"target_level"`
	Score           float64  `json:"score"`
	Findings        []string `json:"findings"`
	Recommendations []string `json:"recommendations"`
}

// VCISOMaturityAssessment represents a maturity assessment record.
type VCISOMaturityAssessment struct {
	ID           uuid.UUID                `json:"id"`
	TenantID     uuid.UUID                `json:"tenant_id"`
	Framework    string                   `json:"framework"`
	Status       string                   `json:"status"`
	OverallScore float64                  `json:"overall_score"`
	OverallLevel int                      `json:"overall_level"`
	Dimensions   []VCISOMaturityDimension `json:"dimensions"`
	AssessorName *string                  `json:"assessor_name,omitempty"`
	AssessedAt   time.Time                `json:"assessed_at"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// VCISOBenchmark represents industry benchmark data for a single dimension.
type VCISOBenchmark struct {
	Dimension          string  `json:"dimension"`
	Category           string  `json:"category"`
	OrganizationScore  float64 `json:"organization_score"`
	IndustryAverage    float64 `json:"industry_average"`
	IndustryTopQuartile float64 `json:"industry_top_quartile"`
	PeerAverage        float64 `json:"peer_average"`
	Gap                float64 `json:"gap"`
}

// ─── Budget ─────────────────────────────────────────────────────────────────

// VCISOBudgetItem represents a security budget line item.
type VCISOBudgetItem struct {
	ID                      uuid.UUID `json:"id"`
	TenantID                uuid.UUID `json:"tenant_id"`
	Title                   string    `json:"title"`
	Category                string    `json:"category"`
	Type                    string    `json:"type"`
	Amount                  float64   `json:"amount"`
	Currency                string    `json:"currency"`
	Status                  string    `json:"status"`
	RiskReductionEstimate   float64   `json:"risk_reduction_estimate"`
	Priority                int       `json:"priority"`
	Justification           string    `json:"justification"`
	LinkedRiskIDs           []string  `json:"linked_risk_ids"`
	LinkedRecommendationIDs []string  `json:"linked_recommendation_ids"`
	FiscalYear              string    `json:"fiscal_year"`
	Quarter                 *string   `json:"quarter,omitempty"`
	OwnerName               *string   `json:"owner_name,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// ─── Awareness ──────────────────────────────────────────────────────────────

// VCISOAwarenessProgram represents a security awareness training program.
type VCISOAwarenessProgram struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	TotalUsers     int       `json:"total_users"`
	CompletedUsers int       `json:"completed_users"`
	PassedUsers    int       `json:"passed_users"`
	FailedUsers    int       `json:"failed_users"`
	CompletionRate float64   `json:"completion_rate"`
	PassRate       float64   `json:"pass_rate"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ─── IAM Finding ────────────────────────────────────────────────────────────

// VCISOIAMFinding represents an IAM-related security finding.
type VCISOIAMFinding struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	Type          string     `json:"type"`
	Severity      string     `json:"severity"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	AffectedUsers int        `json:"affected_users"`
	Status        string     `json:"status"`
	Remediation   *string    `json:"remediation,omitempty"`
	DiscoveredAt  time.Time  `json:"discovered_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// VCISOIAMSummary carries aggregate IAM finding statistics.
type VCISOIAMSummary struct {
	TotalFindings      int            `json:"total_findings"`
	ByType             map[string]int `json:"by_type"`
	BySeverity         map[string]int `json:"by_severity"`
	MFACoveragePercent float64        `json:"mfa_coverage_percent"`
	PrivilegedAccounts int            `json:"privileged_accounts"`
	OrphanedAccounts   int            `json:"orphaned_accounts"`
	StaleAccessCount   int            `json:"stale_access_count"`
}

// ─── Escalation Rule ────────────────────────────────────────────────────────

// VCISOEscalationRule represents an escalation rule for incident management.
type VCISOEscalationRule struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	TriggerType          string     `json:"trigger_type"`
	TriggerCondition     string     `json:"trigger_condition"`
	EscalationTarget     string     `json:"escalation_target"`
	TargetContacts       []string   `json:"target_contacts"`
	NotificationChannels []string   `json:"notification_channels"`
	Enabled              bool       `json:"enabled"`
	LastTriggeredAt      *time.Time `json:"last_triggered_at,omitempty"`
	TriggerCount         int        `json:"trigger_count"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ─── Playbook ───────────────────────────────────────────────────────────────

// VCISOPlaybook represents an incident response / BCP playbook.
type VCISOPlaybook struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Name                 string     `json:"name"`
	Scenario             string     `json:"scenario"`
	Status               string     `json:"status"`
	LastTestedAt         *time.Time `json:"last_tested_at,omitempty"`
	NextTestDate         string     `json:"next_test_date"`
	OwnerID              uuid.UUID  `json:"owner_id"`
	OwnerName            string     `json:"owner_name"`
	StepsCount           int        `json:"steps_count"`
	Dependencies         []string   `json:"dependencies"`
	RTOHours             *float64   `json:"rto_hours,omitempty"`
	RPOHours             *float64   `json:"rpo_hours,omitempty"`
	LastSimulationResult *string    `json:"last_simulation_result,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ─── Obligation ─────────────────────────────────────────────────────────────

// VCISORegulatoryObligation represents a regulatory or contractual obligation.
type VCISORegulatoryObligation struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	Name              string     `json:"name"`
	Type              string     `json:"type"`
	Jurisdiction      string     `json:"jurisdiction"`
	Description       string     `json:"description"`
	Requirements      []string   `json:"requirements"`
	Status            string     `json:"status"`
	MappedControls    int        `json:"mapped_controls"`
	TotalRequirements int        `json:"total_requirements"`
	MetRequirements   int        `json:"met_requirements"`
	OwnerID           *uuid.UUID `json:"owner_id,omitempty"`
	OwnerName         *string    `json:"owner_name,omitempty"`
	EffectiveDate     string     `json:"effective_date"`
	ReviewDate        string     `json:"review_date"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ─── Control Test ───────────────────────────────────────────────────────────

// VCISOControlTest represents a control effectiveness test record.
type VCISOControlTest struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	ControlID    string    `json:"control_id"`
	ControlName  string    `json:"control_name"`
	Framework    string    `json:"framework"`
	TestType     string    `json:"test_type"`
	Result       string    `json:"result"`
	TesterName   string    `json:"tester_name"`
	TestDate     string    `json:"test_date"`
	NextTestDate string    `json:"next_test_date"`
	Findings     string    `json:"findings"`
	EvidenceIDs  []string  `json:"evidence_ids"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ─── Control Dependency ─────────────────────────────────────────────────────

// VCISOControlDependency represents a control's dependency graph.
type VCISOControlDependency struct {
	ControlID         string   `json:"control_id"`
	ControlName       string   `json:"control_name"`
	Framework         string   `json:"framework"`
	DependsOn         []string `json:"depends_on"`
	DependedBy        []string `json:"depended_by"`
	RiskDomains       []string `json:"risk_domains"`
	ComplianceDomains []string `json:"compliance_domains"`
	FailureImpact     string   `json:"failure_impact"`
}

// ─── Integration ────────────────────────────────────────────────────────────

// VCISOIntegration represents an external tool integration.
type VCISOIntegration struct {
	ID            uuid.UUID              `json:"id"`
	TenantID      uuid.UUID              `json:"tenant_id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Provider      string                 `json:"provider"`
	Status        string                 `json:"status"`
	LastSyncAt    *time.Time             `json:"last_sync_at,omitempty"`
	SyncFrequency string                 `json:"sync_frequency"`
	ItemsSynced   int                    `json:"items_synced"`
	Config        map[string]interface{} `json:"config"`
	HealthStatus  string                 `json:"health_status"`
	ErrorMessage  *string                `json:"error_message,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ─── Control Ownership ──────────────────────────────────────────────────────

// VCISOControlOwnership represents control ownership assignment.
type VCISOControlOwnership struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ControlID      string     `json:"control_id"`
	ControlName    string     `json:"control_name"`
	Framework      string     `json:"framework"`
	OwnerID        uuid.UUID  `json:"owner_id"`
	OwnerName      string     `json:"owner_name"`
	DelegateID     *uuid.UUID `json:"delegate_id,omitempty"`
	DelegateName   *string    `json:"delegate_name,omitempty"`
	Status         string     `json:"status"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	NextReviewDate string     `json:"next_review_date"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ─── Approval Request ───────────────────────────────────────────────────────

// VCISOApprovalRequest represents a governance approval request.
type VCISOApprovalRequest struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	RequestedBy      uuid.UUID  `json:"requested_by"`
	RequestedByName  string     `json:"requested_by_name"`
	ApproverID       uuid.UUID  `json:"approver_id"`
	ApproverName     string     `json:"approver_name"`
	Priority         string     `json:"priority"`
	DecisionNotes    *string    `json:"decision_notes,omitempty"`
	DecidedAt        *time.Time `json:"decided_at,omitempty"`
	Deadline         string     `json:"deadline"`
	LinkedEntityType string     `json:"linked_entity_type"`
	LinkedEntityID   string     `json:"linked_entity_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
