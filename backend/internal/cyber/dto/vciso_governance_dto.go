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
	Title               string    `json:"title" validate:"required,min=2,max=255"`
	Description         string    `json:"description"`
	Category            string    `json:"category" validate:"required"`
	Department          string    `json:"department"`
	InherentScore       int       `json:"inherent_score" validate:"gte=0,lte=100"`
	ResidualScore       int       `json:"residual_score" validate:"gte=0,lte=100"`
	Likelihood          string    `json:"likelihood" validate:"omitempty,oneof=low medium high critical"`
	Impact              string    `json:"impact" validate:"omitempty,oneof=low medium high critical"`
	Status              string    `json:"status" validate:"omitempty,oneof=open mitigated accepted closed"`
	Treatment           string    `json:"treatment" validate:"omitempty,oneof=mitigate accept transfer avoid"`
	OwnerID             *string   `json:"owner_id,omitempty" validate:"omitempty,uuid"`
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
	Title     string   `json:"title" validate:"required,min=2,max=255"`
	Domain    string   `json:"domain" validate:"required"`
	Version   string   `json:"version" validate:"omitempty,max=50"`
	Status    string   `json:"status" validate:"omitempty,oneof=draft active under_review approved retired"`
	Content   string   `json:"content"`
	OwnerID   *string  `json:"owner_id,omitempty" validate:"omitempty,uuid"`
	OwnerName string   `json:"owner_name"`
	ReviewDue string   `json:"review_due" validate:"omitempty,datestr"`
	Tags      []string `json:"tags"`
}

// UpdatePolicyRequest is the JSON body for updating a policy.
type UpdatePolicyRequest = CreatePolicyRequest

// UpdatePolicyStatusRequest changes only the status of a policy.
type UpdatePolicyStatusRequest struct {
	Status         string  `json:"status" validate:"required,oneof=draft active under_review approved retired"`
	ReviewerID     *string `json:"reviewer_id,omitempty" validate:"omitempty,uuid"`
	ReviewerName   *string `json:"reviewer_name,omitempty"`
	ApprovedBy     *string `json:"approved_by,omitempty" validate:"omitempty,uuid"`
	ApprovedByName *string `json:"approved_by_name,omitempty"`
}

// ─── Policy Exceptions ──────────────────────────────────────────────────────

// CreatePolicyExceptionRequest is the JSON body for creating a policy exception.
type CreatePolicyExceptionRequest struct {
	PolicyID             string `json:"policy_id" validate:"required,uuid"`
	Title                string `json:"title" validate:"required,min=2,max=255"`
	Description          string `json:"description"`
	Justification        string `json:"justification"`
	CompensatingControls string `json:"compensating_controls"`
	ExpiresAt            string `json:"expires_at" validate:"required,datestr"`
}

// DecidePolicyExceptionRequest records an approval/rejection decision.
type DecidePolicyExceptionRequest struct {
	Status        string  `json:"status" validate:"required,oneof=approved rejected"`
	DecisionNotes *string `json:"decision_notes,omitempty"`
}

// ─── Vendors ────────────────────────────────────────────────────────────────

// CreateVendorRequest is the JSON body for creating a vendor.
type CreateVendorRequest struct {
	Name                 string   `json:"name" validate:"required,min=2,max=255"`
	Category             string   `json:"category" validate:"required"`
	RiskTier             string   `json:"risk_tier" validate:"required,oneof=critical high medium low"`
	Status               string   `json:"status" validate:"omitempty,oneof=active inactive under_review"`
	RiskScore            int      `json:"risk_score" validate:"gte=0,lte=100"`
	NextReviewDate       string   `json:"next_review_date" validate:"omitempty,datestr"`
	ContactName          *string  `json:"contact_name,omitempty"`
	ContactEmail         *string  `json:"contact_email,omitempty" validate:"omitempty,email"`
	ServicesProvided     []string `json:"services_provided"`
	DataShared           []string `json:"data_shared"`
	ComplianceFrameworks []string `json:"compliance_frameworks"`
	ControlsMet          int      `json:"controls_met" validate:"gte=0"`
	ControlsTotal        int      `json:"controls_total" validate:"gte=0"`
}

// UpdateVendorRequest is the JSON body for updating a vendor.
type UpdateVendorRequest = CreateVendorRequest

// UpdateVendorStatusRequest changes vendor status.
type UpdateVendorStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

// ─── Questionnaires ─────────────────────────────────────────────────────────

// CreateQuestionnaireRequest is the JSON body for creating a questionnaire.
type CreateQuestionnaireRequest struct {
	Title          string  `json:"title" validate:"required,min=2,max=255"`
	Type           string  `json:"type" validate:"required"`
	Status         string  `json:"status"`
	VendorID       *string `json:"vendor_id,omitempty" validate:"omitempty,uuid"`
	VendorName     *string `json:"vendor_name,omitempty"`
	TotalQuestions int     `json:"total_questions" validate:"gte=0"`
	DueDate        string  `json:"due_date" validate:"omitempty,datestr"`
	AssignedTo     *string `json:"assigned_to,omitempty" validate:"omitempty,uuid"`
	AssignedToName *string `json:"assigned_to_name,omitempty"`
}

// UpdateQuestionnaireStatusRequest changes questionnaire status.
type UpdateQuestionnaireStatusRequest struct {
	Status            string  `json:"status" validate:"required"`
	AnsweredQuestions *int    `json:"answered_questions,omitempty"`
	Score             *int    `json:"score,omitempty"`
	CompletedAt       *string `json:"completed_at,omitempty"`
}

// ─── Evidence ───────────────────────────────────────────────────────────────

// CreateEvidenceRequest is the JSON body for creating evidence.
type CreateEvidenceRequest struct {
	Title         string   `json:"title" validate:"required,min=2,max=255"`
	Description   string   `json:"description"`
	Type          string   `json:"type" validate:"required"`
	Source        string   `json:"source" validate:"required"`
	Frameworks    []string `json:"frameworks"`
	ControlIDs    []string `json:"control_ids"`
	FileName      *string  `json:"file_name,omitempty"`
	FileSize      *int     `json:"file_size,omitempty"`
	FileURL       *string  `json:"file_url,omitempty"`
	CollectedAt   string   `json:"collected_at" validate:"required"`
	ExpiresAt     *string  `json:"expires_at,omitempty"`
	CollectorName *string  `json:"collector_name,omitempty"`
}

// VerifyEvidenceRequest records a verification of evidence.
type VerifyEvidenceRequest struct {
	Status string `json:"status" validate:"required,oneof=verified rejected pending active expired"`
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
	Framework    string                   `json:"framework" validate:"required"`
	Status       string                   `json:"status"`
	OverallScore float64                  `json:"overall_score" validate:"gte=0"`
	OverallLevel int                      `json:"overall_level" validate:"gte=0,lte=5"`
	Dimensions   []MaturityDimensionInput `json:"dimensions"`
	AssessorName *string                  `json:"assessor_name,omitempty"`
	AssessedAt   string                   `json:"assessed_at"`
}

// UpdateMaturityAssessmentRequest is the JSON body for updating a maturity assessment.
type UpdateMaturityAssessmentRequest = CreateMaturityAssessmentRequest

// ─── Budget ─────────────────────────────────────────────────────────────────

// CreateBudgetItemRequest is the JSON body for creating a budget item.
type CreateBudgetItemRequest struct {
	Title                   string   `json:"title" validate:"required,min=2,max=255"`
	Category                string   `json:"category" validate:"required"`
	Type                    string   `json:"type"`
	Amount                  float64  `json:"amount" validate:"gte=0"`
	Currency                string   `json:"currency" validate:"omitempty,len=3"`
	Status                  string   `json:"status"`
	RiskReductionEstimate   float64  `json:"risk_reduction_estimate"`
	Priority                int      `json:"priority" validate:"gte=0"`
	Justification           string   `json:"justification"`
	LinkedRiskIDs           []string `json:"linked_risk_ids"`
	LinkedRecommendationIDs []string `json:"linked_recommendation_ids"`
	FiscalYear              string   `json:"fiscal_year" validate:"required"`
	Quarter                 *string  `json:"quarter,omitempty"`
	OwnerName               *string  `json:"owner_name,omitempty"`
}

// UpdateBudgetItemRequest is the JSON body for updating a budget item.
type UpdateBudgetItemRequest = CreateBudgetItemRequest

// ─── Awareness Programs ─────────────────────────────────────────────────────

// CreateAwarenessProgramRequest is the JSON body for creating an awareness program.
type CreateAwarenessProgramRequest struct {
	Name           string `json:"name" validate:"required,min=2,max=255"`
	Type           string `json:"type" validate:"required"`
	Status         string `json:"status"`
	TotalUsers     int    `json:"total_users" validate:"gte=0"`
	CompletedUsers int    `json:"completed_users" validate:"gte=0"`
	PassedUsers    int    `json:"passed_users" validate:"gte=0"`
	FailedUsers    int    `json:"failed_users" validate:"gte=0"`
	StartDate      string `json:"start_date" validate:"required,datestr"`
	EndDate        string `json:"end_date" validate:"required,datestr"`
}

// UpdateAwarenessProgramRequest is the JSON body for updating an awareness program.
type UpdateAwarenessProgramRequest = CreateAwarenessProgramRequest

// ─── IAM Findings ───────────────────────────────────────────────────────────

// UpdateIAMFindingRequest is the JSON body for updating an IAM finding.
type UpdateIAMFindingRequest struct {
	Status      string  `json:"status" validate:"required"`
	Remediation *string `json:"remediation,omitempty"`
}

// ─── Escalation Rules ───────────────────────────────────────────────────────

// CreateEscalationRuleRequest is the JSON body for creating an escalation rule.
type CreateEscalationRuleRequest struct {
	Name                 string   `json:"name" validate:"required,min=2,max=255"`
	Description          string   `json:"description"`
	TriggerType          string   `json:"trigger_type" validate:"required,oneof=severity time count custom"`
	TriggerCondition     string   `json:"trigger_condition" validate:"required"`
	EscalationTarget     string   `json:"escalation_target" validate:"required,oneof=management legal regulator board custom"`
	TargetContacts       []string `json:"target_contacts"`
	NotificationChannels []string `json:"notification_channels" validate:"required,min=1"`
	Enabled              bool     `json:"enabled"`
}

// UpdateEscalationRuleRequest is the JSON body for updating an escalation rule.
type UpdateEscalationRuleRequest = CreateEscalationRuleRequest

// ─── Playbooks ──────────────────────────────────────────────────────────────

// CreatePlaybookRequest is the JSON body for creating a playbook.
type CreatePlaybookRequest struct {
	Name         string   `json:"name" validate:"required,min=2,max=255"`
	Scenario     string   `json:"scenario" validate:"required"`
	Status       string   `json:"status" validate:"omitempty,oneof=draft approved tested retired"`
	NextTestDate string   `json:"next_test_date" validate:"required,datestr"`
	OwnerID      *string  `json:"owner_id,omitempty" validate:"omitempty,uuid"`
	OwnerName    string   `json:"owner_name"`
	StepsCount   int      `json:"steps_count" validate:"gte=0"`
	Dependencies []string `json:"dependencies"`
	RTOHours     *float64 `json:"rto_hours,omitempty"`
	RPOHours     *float64 `json:"rpo_hours,omitempty"`
}

// UpdatePlaybookRequest is the JSON body for updating a playbook.
type UpdatePlaybookRequest = CreatePlaybookRequest

// SimulatePlaybookRequest triggers a playbook simulation.
type SimulatePlaybookRequest struct {
	Result string `json:"result" validate:"required,oneof=pass partial fail"`
}

// ─── Obligations ────────────────────────────────────────────────────────────

// CreateObligationRequest is the JSON body for creating an obligation.
type CreateObligationRequest struct {
	Name              string   `json:"name" validate:"required,min=2,max=255"`
	Type              string   `json:"type" validate:"required"`
	Jurisdiction      string   `json:"jurisdiction" validate:"required"`
	Description       string   `json:"description"`
	Requirements      []string `json:"requirements"`
	Status            string   `json:"status"`
	MappedControls    int      `json:"mapped_controls" validate:"gte=0"`
	TotalRequirements int      `json:"total_requirements" validate:"gte=0"`
	MetRequirements   int      `json:"met_requirements" validate:"gte=0"`
	OwnerID           *string  `json:"owner_id,omitempty" validate:"omitempty,uuid"`
	OwnerName         *string  `json:"owner_name,omitempty"`
	EffectiveDate     string   `json:"effective_date" validate:"omitempty,datestr"`
	ReviewDate        string   `json:"review_date" validate:"omitempty,datestr"`
}

// UpdateObligationRequest is the JSON body for updating an obligation.
type UpdateObligationRequest = CreateObligationRequest

// ─── Control Tests ──────────────────────────────────────────────────────────

// CreateControlTestRequest is the JSON body for creating a control test.
type CreateControlTestRequest struct {
	ControlID    string   `json:"control_id" validate:"required"`
	ControlName  string   `json:"control_name" validate:"required"`
	Framework    string   `json:"framework" validate:"required"`
	TestType     string   `json:"test_type" validate:"required,oneof=design operating_effectiveness"`
	Result       string   `json:"result" validate:"required,oneof=effective partially_effective ineffective not_tested"`
	TesterName   string   `json:"tester_name" validate:"required"`
	TestDate     string   `json:"test_date" validate:"required,datestr"`
	NextTestDate string   `json:"next_test_date"`
	Findings     string   `json:"findings"`
	EvidenceIDs  []string `json:"evidence_ids"`
}

// ─── Integrations ───────────────────────────────────────────────────────────

// CreateIntegrationRequest is the JSON body for creating an integration.
type CreateIntegrationRequest struct {
	Name          string                 `json:"name" validate:"required,min=2,max=255"`
	Type          string                 `json:"type" validate:"required,oneof=asset_management ticketing cloud_security data_protection siem iam"`
	Provider      string                 `json:"provider" validate:"required"`
	Status        string                 `json:"status"`
	SyncFrequency string                 `json:"sync_frequency" validate:"omitempty,oneof=every_5m every_15m every_hour every_6h daily"`
	Config        map[string]interface{} `json:"config"`
}

// UpdateIntegrationRequest is the JSON body for updating an integration.
type UpdateIntegrationRequest = CreateIntegrationRequest

// ─── Control Ownership ──────────────────────────────────────────────────────

// CreateControlOwnershipRequest is the JSON body for creating control ownership.
type CreateControlOwnershipRequest struct {
	ControlID      string  `json:"control_id" validate:"required"`
	ControlName    string  `json:"control_name" validate:"required"`
	Framework      string  `json:"framework" validate:"required"`
	OwnerID        string  `json:"owner_id" validate:"required"`
	OwnerName      string  `json:"owner_name" validate:"required"`
	DelegateID     *string `json:"delegate_id,omitempty" validate:"omitempty,uuid"`
	DelegateName   *string `json:"delegate_name,omitempty"`
	Status         string  `json:"status"`
	NextReviewDate string  `json:"next_review_date" validate:"required,datestr"`
}

// UpdateControlOwnershipRequest is the JSON body for updating control ownership.
type UpdateControlOwnershipRequest = CreateControlOwnershipRequest

// ─── Approvals ──────────────────────────────────────────────────────────────

// UpdateApprovalRequest records an approval decision.
type UpdateApprovalRequest struct {
	Status        string  `json:"status" validate:"required,oneof=approved rejected"`
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
	Type          string `json:"type" validate:"required"`
	Severity      string `json:"severity" validate:"required"`
	Title         string `json:"title" validate:"required"`
	Description   string `json:"description"`
	AffectedUsers int    `json:"affected_users"`
	Status        string `json:"status" validate:"required"`
	DiscoveredAt  string `json:"discovered_at"`
}

// ─── Create Approval ────────────────────────────────────────────────────────

// CreateApprovalRequest creates a new approval request.
type CreateApprovalRequest struct {
	Type             string `json:"type" validate:"required"`
	Title            string `json:"title" validate:"required,min=2,max=255"`
	Description      string `json:"description"`
	ApproverID       string `json:"approver_id" validate:"required"`
	ApproverName     string `json:"approver_name" validate:"required"`
	Priority         string `json:"priority" validate:"required"`
	Deadline         string `json:"deadline" validate:"required,datestr"`
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
