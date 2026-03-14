package dto

import "github.com/google/uuid"

// VCISOGovernanceListParams is a generic set of pagination/filter/sort
// parameters reused by all governance resource list endpoints.
type VCISOGovernanceListParams struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Sort    string `json:"sort"`
	Order   string `json:"order"`
	Search  string `json:"search"`
	// Resource-specific filters
	Status    string `json:"status,omitempty"`
	Type      string `json:"type,omitempty"`
	Framework string `json:"framework,omitempty"`
	Category  string `json:"category,omitempty"`
}

// SetDefaults applies safe defaults to unset pagination values.
func (p *VCISOGovernanceListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 100 {
		p.PerPage = 25
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

// Offset returns the SQL OFFSET value.
func (p *VCISOGovernanceListParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// ─── Risks ──────────────────────────────────────────────────────────────────

// CreateRiskRequest is the JSON body for creating a risk entry.
type CreateRiskRequest struct {
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	Category            string    `json:"category"`
	Department          string    `json:"department"`
	InherentScore       int       `json:"inherent_score"`
	ResidualScore       int       `json:"residual_score"`
	Likelihood          string    `json:"likelihood"`
	Impact              string    `json:"impact"`
	Status              string    `json:"status"`
	Treatment           string    `json:"treatment"`
	OwnerID             *string   `json:"owner_id,omitempty"`
	OwnerName           string    `json:"owner_name"`
	ReviewDate          *string   `json:"review_date,omitempty"`
	BusinessServices    []string  `json:"business_services"`
	Controls            []string  `json:"controls"`
	Tags                []string  `json:"tags"`
	TreatmentPlan       string    `json:"treatment_plan"`
	AcceptanceRationale *string   `json:"acceptance_rationale,omitempty"`
	AcceptanceExpiry    *string   `json:"acceptance_expiry,omitempty"`
}

// UpdateRiskRequest is the JSON body for updating a risk entry.
type UpdateRiskRequest = CreateRiskRequest

// ─── Policies ───────────────────────────────────────────────────────────────

// CreatePolicyRequest is the JSON body for creating a policy.
type CreatePolicyRequest struct {
	Title      string   `json:"title"`
	Domain     string   `json:"domain"`
	Version    string   `json:"version"`
	Status     string   `json:"status"`
	Content    string   `json:"content"`
	OwnerID    string   `json:"owner_id"`
	OwnerName  string   `json:"owner_name"`
	ReviewDue  string   `json:"review_due"`
	Tags       []string `json:"tags"`
}

// UpdatePolicyRequest is the JSON body for updating a policy.
type UpdatePolicyRequest = CreatePolicyRequest

// UpdatePolicyStatusRequest changes only the status of a policy.
type UpdatePolicyStatusRequest struct {
	Status         string  `json:"status"`
	ReviewerID     *string `json:"reviewer_id,omitempty"`
	ReviewerName   *string `json:"reviewer_name,omitempty"`
	ApprovedBy     *string `json:"approved_by,omitempty"`
	ApprovedByName *string `json:"approved_by_name,omitempty"`
}

// ─── Policy Exceptions ──────────────────────────────────────────────────────

// CreatePolicyExceptionRequest is the JSON body for creating a policy exception.
type CreatePolicyExceptionRequest struct {
	PolicyID              string `json:"policy_id"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	Justification         string `json:"justification"`
	CompensatingControls  string `json:"compensating_controls"`
	ExpiresAt             string `json:"expires_at"`
}

// DecidePolicyExceptionRequest records an approval/rejection decision.
type DecidePolicyExceptionRequest struct {
	Status        string  `json:"status"`
	DecisionNotes *string `json:"decision_notes,omitempty"`
}

// ─── Vendors ────────────────────────────────────────────────────────────────

// CreateVendorRequest is the JSON body for creating a vendor.
type CreateVendorRequest struct {
	Name                 string   `json:"name"`
	Category             string   `json:"category"`
	RiskTier             string   `json:"risk_tier"`
	Status               string   `json:"status"`
	RiskScore            int      `json:"risk_score"`
	NextReviewDate       string   `json:"next_review_date"`
	ContactName          *string  `json:"contact_name,omitempty"`
	ContactEmail         *string  `json:"contact_email,omitempty"`
	ServicesProvided     []string `json:"services_provided"`
	DataShared           []string `json:"data_shared"`
	ComplianceFrameworks []string `json:"compliance_frameworks"`
	ControlsMet          int      `json:"controls_met"`
	ControlsTotal        int      `json:"controls_total"`
}

// UpdateVendorRequest is the JSON body for updating a vendor.
type UpdateVendorRequest = CreateVendorRequest

// UpdateVendorStatusRequest changes vendor status.
type UpdateVendorStatusRequest struct {
	Status string `json:"status"`
}

// ─── Questionnaires ─────────────────────────────────────────────────────────

// CreateQuestionnaireRequest is the JSON body for creating a questionnaire.
type CreateQuestionnaireRequest struct {
	Title          string  `json:"title"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	VendorID       *string `json:"vendor_id,omitempty"`
	VendorName     *string `json:"vendor_name,omitempty"`
	TotalQuestions int     `json:"total_questions"`
	DueDate        string  `json:"due_date"`
	AssignedTo     *string `json:"assigned_to,omitempty"`
	AssignedToName *string `json:"assigned_to_name,omitempty"`
}

// UpdateQuestionnaireStatusRequest changes questionnaire status.
type UpdateQuestionnaireStatusRequest struct {
	Status            string  `json:"status"`
	AnsweredQuestions *int    `json:"answered_questions,omitempty"`
	Score             *int    `json:"score,omitempty"`
	CompletedAt       *string `json:"completed_at,omitempty"`
}

// ─── Evidence ───────────────────────────────────────────────────────────────

// CreateEvidenceRequest is the JSON body for creating evidence.
type CreateEvidenceRequest struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Type          string   `json:"type"`
	Source        string   `json:"source"`
	Frameworks    []string `json:"frameworks"`
	ControlIDs    []string `json:"control_ids"`
	FileName      *string  `json:"file_name,omitempty"`
	FileSize      *int     `json:"file_size,omitempty"`
	FileURL       *string  `json:"file_url,omitempty"`
	CollectedAt   string   `json:"collected_at"`
	ExpiresAt     *string  `json:"expires_at,omitempty"`
	CollectorName *string  `json:"collector_name,omitempty"`
}

// VerifyEvidenceRequest records a verification of evidence.
type VerifyEvidenceRequest struct {
	Status string `json:"status"`
}

// ─── Maturity Assessments ───────────────────────────────────────────────────

// MaturityDimensionInput represents one dimension in a maturity assessment.
type MaturityDimensionInput struct {
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	CurrentLevel    int      `json:"current_level"`
	TargetLevel     int      `json:"target_level"`
	Score           float64  `json:"score"`
	Findings        []string `json:"findings"`
	Recommendations []string `json:"recommendations"`
}

// CreateMaturityAssessmentRequest is the JSON body for creating a maturity assessment.
type CreateMaturityAssessmentRequest struct {
	Framework    string                   `json:"framework"`
	Status       string                   `json:"status"`
	OverallScore float64                  `json:"overall_score"`
	OverallLevel int                      `json:"overall_level"`
	Dimensions   []MaturityDimensionInput `json:"dimensions"`
	AssessorName *string                  `json:"assessor_name,omitempty"`
	AssessedAt   string                   `json:"assessed_at"`
}

// UpdateMaturityAssessmentRequest is the JSON body for updating a maturity assessment.
type UpdateMaturityAssessmentRequest = CreateMaturityAssessmentRequest

// ─── Budget ─────────────────────────────────────────────────────────────────

// CreateBudgetItemRequest is the JSON body for creating a budget item.
type CreateBudgetItemRequest struct {
	Title                   string   `json:"title"`
	Category                string   `json:"category"`
	Type                    string   `json:"type"`
	Amount                  float64  `json:"amount"`
	Currency                string   `json:"currency"`
	Status                  string   `json:"status"`
	RiskReductionEstimate   float64  `json:"risk_reduction_estimate"`
	Priority                int      `json:"priority"`
	Justification           string   `json:"justification"`
	LinkedRiskIDs           []string `json:"linked_risk_ids"`
	LinkedRecommendationIDs []string `json:"linked_recommendation_ids"`
	FiscalYear              string   `json:"fiscal_year"`
	Quarter                 *string  `json:"quarter,omitempty"`
	OwnerName               *string  `json:"owner_name,omitempty"`
}

// UpdateBudgetItemRequest is the JSON body for updating a budget item.
type UpdateBudgetItemRequest = CreateBudgetItemRequest

// ─── Awareness Programs ─────────────────────────────────────────────────────

// CreateAwarenessProgramRequest is the JSON body for creating an awareness program.
type CreateAwarenessProgramRequest struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	TotalUsers     int    `json:"total_users"`
	CompletedUsers int    `json:"completed_users"`
	PassedUsers    int    `json:"passed_users"`
	FailedUsers    int    `json:"failed_users"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
}

// UpdateAwarenessProgramRequest is the JSON body for updating an awareness program.
type UpdateAwarenessProgramRequest = CreateAwarenessProgramRequest

// ─── IAM Findings ───────────────────────────────────────────────────────────

// UpdateIAMFindingRequest is the JSON body for updating an IAM finding.
type UpdateIAMFindingRequest struct {
	Status      string  `json:"status"`
	Remediation *string `json:"remediation,omitempty"`
}

// ─── Escalation Rules ───────────────────────────────────────────────────────

// CreateEscalationRuleRequest is the JSON body for creating an escalation rule.
type CreateEscalationRuleRequest struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	TriggerType          string   `json:"trigger_type"`
	TriggerCondition     string   `json:"trigger_condition"`
	EscalationTarget     string   `json:"escalation_target"`
	TargetContacts       []string `json:"target_contacts"`
	NotificationChannels []string `json:"notification_channels"`
	Enabled              bool     `json:"enabled"`
}

// UpdateEscalationRuleRequest is the JSON body for updating an escalation rule.
type UpdateEscalationRuleRequest = CreateEscalationRuleRequest

// ─── Playbooks ──────────────────────────────────────────────────────────────

// CreatePlaybookRequest is the JSON body for creating a playbook.
type CreatePlaybookRequest struct {
	Name         string   `json:"name"`
	Scenario     string   `json:"scenario"`
	Status       string   `json:"status"`
	NextTestDate string   `json:"next_test_date"`
	OwnerID      string   `json:"owner_id"`
	OwnerName    string   `json:"owner_name"`
	StepsCount   int      `json:"steps_count"`
	Dependencies []string `json:"dependencies"`
	RTOHours     *float64 `json:"rto_hours,omitempty"`
	RPOHours     *float64 `json:"rpo_hours,omitempty"`
}

// UpdatePlaybookRequest is the JSON body for updating a playbook.
type UpdatePlaybookRequest = CreatePlaybookRequest

// SimulatePlaybookRequest triggers a playbook simulation.
type SimulatePlaybookRequest struct {
	Result string `json:"result"` // pass | partial | fail
}

// ─── Obligations ────────────────────────────────────────────────────────────

// CreateObligationRequest is the JSON body for creating an obligation.
type CreateObligationRequest struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	Jurisdiction      string   `json:"jurisdiction"`
	Description       string   `json:"description"`
	Requirements      []string `json:"requirements"`
	Status            string   `json:"status"`
	MappedControls    int      `json:"mapped_controls"`
	TotalRequirements int      `json:"total_requirements"`
	MetRequirements   int      `json:"met_requirements"`
	OwnerID           *string  `json:"owner_id,omitempty"`
	OwnerName         *string  `json:"owner_name,omitempty"`
	EffectiveDate     string   `json:"effective_date"`
	ReviewDate        string   `json:"review_date"`
}

// UpdateObligationRequest is the JSON body for updating an obligation.
type UpdateObligationRequest = CreateObligationRequest

// ─── Control Tests ──────────────────────────────────────────────────────────

// CreateControlTestRequest is the JSON body for creating a control test.
type CreateControlTestRequest struct {
	ControlID    string   `json:"control_id"`
	ControlName  string   `json:"control_name"`
	Framework    string   `json:"framework"`
	TestType     string   `json:"test_type"`
	Result       string   `json:"result"`
	TesterName   string   `json:"tester_name"`
	TestDate     string   `json:"test_date"`
	NextTestDate string   `json:"next_test_date"`
	Findings     string   `json:"findings"`
	EvidenceIDs  []string `json:"evidence_ids"`
}

// ─── Integrations ───────────────────────────────────────────────────────────

// CreateIntegrationRequest is the JSON body for creating an integration.
type CreateIntegrationRequest struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Provider      string                 `json:"provider"`
	Status        string                 `json:"status"`
	SyncFrequency string                 `json:"sync_frequency"`
	Config        map[string]interface{} `json:"config"`
}

// UpdateIntegrationRequest is the JSON body for updating an integration.
type UpdateIntegrationRequest = CreateIntegrationRequest

// ─── Control Ownership ──────────────────────────────────────────────────────

// CreateControlOwnershipRequest is the JSON body for creating control ownership.
type CreateControlOwnershipRequest struct {
	ControlID      string  `json:"control_id"`
	ControlName    string  `json:"control_name"`
	Framework      string  `json:"framework"`
	OwnerID        string  `json:"owner_id"`
	OwnerName      string  `json:"owner_name"`
	DelegateID     *string `json:"delegate_id,omitempty"`
	DelegateName   *string `json:"delegate_name,omitempty"`
	Status         string  `json:"status"`
	NextReviewDate string  `json:"next_review_date"`
}

// UpdateControlOwnershipRequest is the JSON body for updating control ownership.
type UpdateControlOwnershipRequest = CreateControlOwnershipRequest

// ─── Approvals ──────────────────────────────────────────────────────────────

// UpdateApprovalRequest records an approval decision.
type UpdateApprovalRequest struct {
	Status        string  `json:"status"`
	DecisionNotes *string `json:"decision_notes,omitempty"`
}

// ─── Response helpers ───────────────────────────────────────────────────────

// GovernanceListResponse wraps a paginated governance list for the handler layer.
type GovernanceListResponse struct {
	Data       interface{}    `json:"data"`
	Meta       PaginationMeta `json:"meta"`
	Total      int            `json:"total"`
}

// NewGovernanceListResponse builds a paginated response.
func NewGovernanceListResponse(data interface{}, page, perPage, total int) *GovernanceListResponse {
	return &GovernanceListResponse{
		Data:  data,
		Meta:  NewPaginationMeta(page, perPage, total),
		Total: total,
	}
}

// ─── Vendor stats (no separate model needed) ────────────────────────────────

// VendorStatsResponse carries vendor summary stats.
type VendorStatsResponse struct {
	Total           int            `json:"total"`
	ByRiskTier      map[string]int `json:"by_risk_tier"`
	ByStatus        map[string]int `json:"by_status"`
	OverdueReviews  int            `json:"overdue_reviews"`
	AvgRiskScore    float64        `json:"avg_risk_score"`
}

// ─── Budget summary ─────────────────────────────────────────────────────────

// BudgetSummaryResponse carries budget summary stats. Mirrors VCISOBudgetSummary
// on the frontend.
type BudgetSummaryResponse struct {
	TotalProposed      float64            `json:"total_proposed"`
	TotalApproved      float64            `json:"total_approved"`
	TotalSpent         float64            `json:"total_spent"`
	TotalRiskReduction float64            `json:"total_risk_reduction"`
	ByCategory         map[string]float64 `json:"by_category"`
	ByStatus           map[string]float64 `json:"by_status"`
	Currency           string             `json:"currency"`
}

// ─── Create IAM Finding (internally used by scanners) ───────────────────────

// CreateIAMFindingRequest is used by automated scanners to persist findings.
type CreateIAMFindingRequest struct {
	Type          string `json:"type"`
	Severity      string `json:"severity"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	AffectedUsers int    `json:"affected_users"`
	Status        string `json:"status"`
	DiscoveredAt  string `json:"discovered_at"`
}

// ─── Create Approval ────────────────────────────────────────────────────────

// CreateApprovalRequest creates a new approval request.
type CreateApprovalRequest struct {
	Type             string `json:"type"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	ApproverID       string `json:"approver_id"`
	ApproverName     string `json:"approver_name"`
	Priority         string `json:"priority"`
	Deadline         string `json:"deadline"`
	LinkedEntityType string `json:"linked_entity_type"`
	LinkedEntityID   string `json:"linked_entity_id"`
}

// ─── Benchmark list params ──────────────────────────────────────────────────

// BenchmarkListParams specializes for benchmark queries which are read-only.
type BenchmarkListParams struct {
	Framework string `json:"framework,omitempty"`
	Category  string `json:"category,omitempty"`
}

// ─── IAM Finding bulk create ────────────────────────────────────────────────

// Placeholder UUID parser helper used in handler layer.
func ParseOptionalUUID(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}
