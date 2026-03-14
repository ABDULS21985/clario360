package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// vcisoGovernanceService abstracts the governance service layer for testability.
type vcisoGovernanceService interface {
	// Risks
	ListRisks(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateRisk(ctx context.Context, tenantID uuid.UUID, req *dto.CreateRiskRequest) (*model.VCISORiskEntry, error)
	GetRisk(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISORiskEntry, error)
	UpdateRisk(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateRiskRequest) (*model.VCISORiskEntry, error)
	DeleteRisk(ctx context.Context, tenantID, id uuid.UUID) error
	RiskStats(ctx context.Context, tenantID uuid.UUID) (*model.VCISORiskStats, error)
	// Policies
	ListPolicies(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreatePolicy(ctx context.Context, tenantID uuid.UUID, req *dto.CreatePolicyRequest) (*model.VCISOPolicy, error)
	GetPolicy(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOPolicy, error)
	UpdatePolicy(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreatePolicyRequest) (*model.VCISOPolicy, error)
	DeletePolicy(ctx context.Context, tenantID, id uuid.UUID) error
	UpdatePolicyStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdatePolicyStatusRequest) (*model.VCISOPolicy, error)
	PolicyStats(ctx context.Context, tenantID uuid.UUID) (*dto.GovernanceListResponse, error)
	GeneratePolicy(ctx context.Context, tenantID uuid.UUID, domain string) (string, error)
	// Policy Exceptions
	ListPolicyExceptions(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreatePolicyException(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreatePolicyExceptionRequest, userName string) (*model.VCISOPolicyException, error)
	DecidePolicyException(ctx context.Context, tenantID, id, userID uuid.UUID, req *dto.DecidePolicyExceptionRequest, userName string) error
	// Vendors
	ListVendors(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateVendor(ctx context.Context, tenantID uuid.UUID, req *dto.CreateVendorRequest) (*model.VCISOVendor, error)
	GetVendor(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOVendor, error)
	UpdateVendor(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateVendorRequest) (*model.VCISOVendor, error)
	DeleteVendor(ctx context.Context, tenantID, id uuid.UUID) error
	UpdateVendorStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateVendorStatusRequest) (*model.VCISOVendor, error)
	VendorStats(ctx context.Context, tenantID uuid.UUID) (*dto.VendorStatsResponse, error)
	// Questionnaires
	ListQuestionnaires(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateQuestionnaire(ctx context.Context, tenantID uuid.UUID, req *dto.CreateQuestionnaireRequest) (*model.VCISOQuestionnaire, error)
	UpdateQuestionnaire(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateQuestionnaireRequest) error
	UpdateQuestionnaireStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateQuestionnaireStatusRequest) error
	// Evidence
	ListEvidence(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateEvidence(ctx context.Context, tenantID uuid.UUID, req *dto.CreateEvidenceRequest) (*model.VCISOEvidence, error)
	GetEvidence(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOEvidence, error)
	UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateEvidenceRequest) (*model.VCISOEvidence, error)
	DeleteEvidence(ctx context.Context, tenantID, id uuid.UUID) error
	VerifyEvidence(ctx context.Context, tenantID, id, userID uuid.UUID, status string) (*model.VCISOEvidence, error)
	EvidenceStats(ctx context.Context, tenantID uuid.UUID) (*model.VCISOEvidenceStats, error)
	// Maturity
	ListMaturityAssessments(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateMaturityAssessment(ctx context.Context, tenantID uuid.UUID, req *dto.CreateMaturityAssessmentRequest) (*model.VCISOMaturityAssessment, error)
	// Benchmarks
	ListBenchmarks(ctx context.Context, tenantID uuid.UUID, params *dto.BenchmarkListParams) ([]model.VCISOBenchmark, error)
	// Budget
	ListBudgetItems(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateBudgetItem(ctx context.Context, tenantID uuid.UUID, req *dto.CreateBudgetItemRequest) (*model.VCISOBudgetItem, error)
	UpdateBudgetItem(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateBudgetItemRequest) error
	DeleteBudgetItem(ctx context.Context, tenantID, id uuid.UUID) error
	BudgetSummary(ctx context.Context, tenantID uuid.UUID) (*dto.BudgetSummaryResponse, error)
	// Awareness
	ListAwarenessPrograms(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateAwarenessProgram(ctx context.Context, tenantID uuid.UUID, req *dto.CreateAwarenessProgramRequest) (*model.VCISOAwarenessProgram, error)
	UpdateAwarenessProgram(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateAwarenessProgramRequest) error
	// IAM Findings
	ListIAMFindings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	UpdateIAMFinding(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateIAMFindingRequest) error
	IAMFindingSummary(ctx context.Context, tenantID uuid.UUID) (*model.VCISOIAMSummary, error)
	// Escalation Rules
	ListEscalationRules(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateEscalationRule(ctx context.Context, tenantID uuid.UUID, req *dto.CreateEscalationRuleRequest) (*model.VCISOEscalationRule, error)
	UpdateEscalationRule(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateEscalationRuleRequest) error
	DeleteEscalationRule(ctx context.Context, tenantID, id uuid.UUID) error
	// Playbooks
	ListPlaybooks(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreatePlaybook(ctx context.Context, tenantID uuid.UUID, req *dto.CreatePlaybookRequest) (*model.VCISOPlaybook, error)
	UpdatePlaybook(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreatePlaybookRequest) error
	DeletePlaybook(ctx context.Context, tenantID, id uuid.UUID) error
	SimulatePlaybook(ctx context.Context, tenantID, id uuid.UUID, result string) error
	// Obligations
	ListObligations(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateObligation(ctx context.Context, tenantID uuid.UUID, req *dto.CreateObligationRequest) (*model.VCISORegulatoryObligation, error)
	UpdateObligation(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateObligationRequest) error
	DeleteObligation(ctx context.Context, tenantID, id uuid.UUID) error
	// Control Tests
	ListControlTests(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateControlTest(ctx context.Context, tenantID uuid.UUID, req *dto.CreateControlTestRequest) (*model.VCISOControlTest, error)
	// Control Dependencies
	ListControlDependencies(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	// Integrations
	ListIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*model.VCISOIntegration, error)
	CreateIntegration(ctx context.Context, tenantID uuid.UUID, req *dto.CreateIntegrationRequest) (*model.VCISOIntegration, error)
	UpdateIntegration(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateIntegrationRequest) error
	DeleteIntegration(ctx context.Context, tenantID, id uuid.UUID) error
	SyncIntegration(ctx context.Context, tenantID, id uuid.UUID) error
	// Control Ownership
	ListControlOwnerships(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	CreateControlOwnership(ctx context.Context, tenantID uuid.UUID, req *dto.CreateControlOwnershipRequest) (*model.VCISOControlOwnership, error)
	UpdateControlOwnership(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateControlOwnershipRequest) error
	MarkControlOwnershipReviewed(ctx context.Context, tenantID, id uuid.UUID) error
	// Approvals
	ListApprovals(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error)
	DecideApproval(ctx context.Context, tenantID, id, userID uuid.UUID, req *dto.UpdateApprovalRequest) error
}

// VCISOGovernanceHandler handles HTTP requests for vCISO governance.
type VCISOGovernanceHandler struct {
	svc    vcisoGovernanceService
	logger zerolog.Logger
}

// NewVCISOGovernanceHandler creates a new VCISOGovernanceHandler.
func NewVCISOGovernanceHandler(svc vcisoGovernanceService, logger zerolog.Logger) *VCISOGovernanceHandler {
	return &VCISOGovernanceHandler{svc: svc, logger: logger.With().Str("handler", "vciso-governance").Logger()}
}

func (h *VCISOGovernanceHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	default:
		h.logger.Error().Err(err).Msg("vciso governance handler error")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}

func parseGovernanceListParams(r *http.Request) *dto.VCISOGovernanceListParams {
	q := r.URL.Query()
	params := &dto.VCISOGovernanceListParams{
		Search:    q.Get("search"),
		Status:    q.Get("status"),
		Type:      q.Get("type"),
		Framework: q.Get("framework"),
		Category:  q.Get("category"),
		Sort:      q.Get("sort"),
		Order:     q.Get("order"),
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	return params
}

// ─── Risks ──────────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListRisks(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListRisks(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateRisk(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateRiskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateRisk(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) GetRisk(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetRisk(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateRisk(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateRiskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateRisk(r.Context(), tenantID, id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) DeleteRisk(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteRisk(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) RiskStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.RiskStats(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

// ─── Policies ───────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListPolicies(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreatePolicy(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetPolicy(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdatePolicy(r.Context(), tenantID, id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeletePolicy(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) UpdatePolicyStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdatePolicyStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdatePolicyStatus(r.Context(), tenantID, id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) PolicyStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.PolicyStats(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *VCISOGovernanceHandler) GeneratePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Domain string `json:"domain"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	content, err := h.svc.GeneratePolicy(r.Context(), tenantID, req.Domain)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]string{"content": content}})
}

// ─── Policy Exceptions ──────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListPolicyExceptions(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListPolicyExceptions(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreatePolicyException(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreatePolicyExceptionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userName := ""
	if user := auth.UserFromContext(r.Context()); user != nil {
		userName = user.Email
	}
	item, err := h.svc.CreatePolicyException(r.Context(), tenantID, userID, &req, userName)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) DecidePolicyException(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.DecidePolicyExceptionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userName := ""
	if user := auth.UserFromContext(r.Context()); user != nil {
		userName = user.Email
	}
	if err := h.svc.DecidePolicyException(r.Context(), tenantID, id, userID, &req, userName); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

// ─── Vendors ────────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListVendors(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListVendors(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateVendorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateVendor(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) GetVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetVendor(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateVendorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateVendor(r.Context(), tenantID, id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) DeleteVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteVendor(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) UpdateVendorStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateVendorStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateVendorStatus(r.Context(), tenantID, id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) VendorStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.VendorStats(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

// ─── Questionnaires ─────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListQuestionnaires(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListQuestionnaires(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateQuestionnaire(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateQuestionnaireRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateQuestionnaire(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateQuestionnaire(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateQuestionnaireRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateQuestionnaire(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) UpdateQuestionnaireStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateQuestionnaireStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateQuestionnaireStatus(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

// ─── Evidence ───────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListEvidence(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListEvidence(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateEvidence(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateEvidenceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateEvidence(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetEvidence(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateEvidence(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateEvidenceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateEvidence(r.Context(), tenantID, id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) DeleteEvidence(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteEvidence(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) VerifyEvidence(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.VerifyEvidenceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.VerifyEvidence(r.Context(), tenantID, id, userID, req.Status)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) EvidenceStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.EvidenceStats(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

// ─── Maturity ───────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListMaturityAssessments(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListMaturityAssessments(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateMaturityAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateMaturityAssessmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateMaturityAssessment(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

// ─── Benchmarks ─────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListBenchmarks(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := &dto.BenchmarkListParams{
		Framework: r.URL.Query().Get("framework"),
		Category:  r.URL.Query().Get("category"),
	}
	items, err := h.svc.ListBenchmarks(r.Context(), tenantID, params)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

// ─── Budget ─────────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListBudgetItems(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListBudgetItems(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateBudgetItem(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateBudgetItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateBudgetItem(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateBudgetItem(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateBudgetItemRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateBudgetItem(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) DeleteBudgetItem(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteBudgetItem(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) BudgetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.BudgetSummary(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": summary})
}

// ─── Awareness ──────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListAwarenessPrograms(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListAwarenessPrograms(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateAwarenessProgram(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateAwarenessProgramRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateAwarenessProgram(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateAwarenessProgram(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateAwarenessProgramRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateAwarenessProgram(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

// ─── IAM Findings ───────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListIAMFindings(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListIAMFindings(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) UpdateIAMFinding(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateIAMFindingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateIAMFinding(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) IAMFindingSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.IAMFindingSummary(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": summary})
}

// ─── Escalation Rules ───────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListEscalationRules(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListEscalationRules(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateEscalationRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateEscalationRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateEscalationRule(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateEscalationRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateEscalationRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateEscalationRule(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) DeleteEscalationRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteEscalationRule(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

// ─── Playbooks ──────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListPlaybooks(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListPlaybooks(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreatePlaybook(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreatePlaybookRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreatePlaybook(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdatePlaybook(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreatePlaybookRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdatePlaybook(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) DeletePlaybook(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeletePlaybook(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) SimulatePlaybook(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.SimulatePlaybookRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.SimulatePlaybook(r.Context(), tenantID, id, req.Result); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"simulated": true}})
}

// ─── Obligations ────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListObligations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListObligations(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateObligation(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateObligationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateObligation(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateObligation(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateObligationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateObligation(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) DeleteObligation(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteObligation(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

// ─── Control Tests ──────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListControlTests(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListControlTests(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateControlTest(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateControlTestRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateControlTest(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

// ─── Control Dependencies ───────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListControlDependencies(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListControlDependencies(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ─── Integrations ───────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListIntegrations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListIntegrations(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *VCISOGovernanceHandler) CreateIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateIntegrationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateIntegration(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateIntegrationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateIntegration(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) DeleteIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteIntegration(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *VCISOGovernanceHandler) SyncIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.SyncIntegration(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"synced": true}})
}

// ─── Control Ownership ──────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListControlOwnerships(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListControlOwnerships(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) CreateControlOwnership(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateControlOwnershipRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.CreateControlOwnership(r.Context(), tenantID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *VCISOGovernanceHandler) UpdateControlOwnership(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateControlOwnershipRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateControlOwnership(r.Context(), tenantID, id, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *VCISOGovernanceHandler) MarkControlOwnershipReviewed(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.MarkControlOwnershipReviewed(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"reviewed": true}})
}

// ─── Approvals ──────────────────────────────────────────────────────────────

func (h *VCISOGovernanceHandler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListApprovals(r.Context(), tenantID, parseGovernanceListParams(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOGovernanceHandler) DecideApproval(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateApprovalRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.svc.DecideApproval(r.Context(), tenantID, id, userID, &req); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}
