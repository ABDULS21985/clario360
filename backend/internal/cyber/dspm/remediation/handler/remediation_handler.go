package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/engine"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/repository"
)

// DSPMRemediationHandler handles all DSPM remediation, policy, and exception endpoints.
type DSPMRemediationHandler struct {
	engine        *engine.RemediationEngine
	policyRepo    *repository.PolicyRepository
	exceptionRepo *repository.ExceptionRepository
}

// NewDSPMRemediationHandler creates a new handler with all dependencies.
func NewDSPMRemediationHandler(
	eng *engine.RemediationEngine,
	policyRepo *repository.PolicyRepository,
	exceptionRepo *repository.ExceptionRepository,
) *DSPMRemediationHandler {
	return &DSPMRemediationHandler{
		engine:        eng,
		policyRepo:    policyRepo,
		exceptionRepo: exceptionRepo,
	}
}

// ── Remediations ────────────────────────────────────────────────────────────────

// ListRemediations handles GET /api/v1/cyber/dspm/remediations
func (h *DSPMRemediationHandler) ListRemediations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, perPage := parsePageParams(r, 25)
	params := &dto.RemediationListParams{
		Status:      parseCSV(q.Get("status")),
		Severity:    parseCSV(q.Get("severity")),
		FindingType: parseCSV(q.Get("finding_type")),
		AssignedTo:  uuidPtr(q.Get("assigned_to")),
		AssetID:     uuidPtr(q.Get("asset_id")),
		SLABreached: boolPtr(q.Get("sla_breached")),
		Search:      q.Get("search"),
		Sort:        q.Get("sort"),
		Order:       q.Get("order"),
		Page:        page,
		PerPage:     perPage,
	}
	params.SetDefaults()
	data, total, err := h.engine.ListRemediations(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": data, "total": total, "page": page, "limit": perPage})
}

// GetRemediation handles GET /api/v1/cyber/dspm/remediations/{id}
func (h *DSPMRemediationHandler) GetRemediation(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rem, err := h.engine.GetRemediation(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	history, _, _ := h.engine.GetHistory(r.Context(), tenantID, id, 1, 50)
	writeJSON(w, http.StatusOK, envelope{"data": rem, "history": history})
}

// ExecuteStep handles POST /api/v1/cyber/dspm/remediations/{id}/execute
func (h *DSPMRemediationHandler) ExecuteStep(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	result, err := h.engine.ExecuteStep(r.Context(), tenantID, id, &userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		if strings.Contains(err.Error(), "cannot execute") || strings.Contains(err.Error(), "all steps") {
			writeError(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

// ApproveRemediation handles POST /api/v1/cyber/dspm/remediations/{id}/approve
func (h *DSPMRemediationHandler) ApproveRemediation(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.engine.ApproveRemediation(r.Context(), tenantID, id, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not awaiting") {
			writeError(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"message": "remediation approved"})
}

// CancelRemediation handles POST /api/v1/cyber/dspm/remediations/{id}/cancel
func (h *DSPMRemediationHandler) CancelRemediation(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.engine.CancelRemediation(r.Context(), tenantID, id, &userID, body.Reason); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"message": "remediation cancelled"})
}

// RollbackRemediation handles POST /api/v1/cyber/dspm/remediations/{id}/rollback
func (h *DSPMRemediationHandler) RollbackRemediation(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.engine.RollbackRemediation(r.Context(), tenantID, id, &userID, body.Reason); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		if strings.Contains(err.Error(), "rollback not available") {
			writeError(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"message": "remediation rolled back"})
}

// AssignRemediation handles PUT /api/v1/cyber/dspm/remediations/{id}/assign
func (h *DSPMRemediationHandler) AssignRemediation(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AssignRemediationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.engine.AssignRemediation(r.Context(), tenantID, id, &req); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"message": "remediation assigned"})
}

// GetHistory handles GET /api/v1/cyber/dspm/remediations/{id}/history
func (h *DSPMRemediationHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	page, perPage := parsePageParams(r, 50)
	data, total, err := h.engine.GetHistory(r.Context(), tenantID, id, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": data, "total": total})
}

// GetStats handles GET /api/v1/cyber/dspm/remediations/stats
func (h *DSPMRemediationHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.engine.GetStats(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

// GetDashboard handles GET /api/v1/cyber/dspm/remediations/dashboard
func (h *DSPMRemediationHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	dashboard, err := h.engine.GetDashboard(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": dashboard})
}

// ── Policies ────────────────────────────────────────────────────────────────────

// ListPolicies handles GET /api/v1/cyber/dspm/policies
func (h *DSPMRemediationHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, perPage := parsePageParams(r, 25)
	params := &dto.PolicyListParams{
		Category:    q.Get("category"),
		Enforcement: q.Get("enforcement"),
		Enabled:     boolPtr(q.Get("enabled")),
		Search:      q.Get("search"),
		Page:        page,
		PerPage:     perPage,
	}
	params.SetDefaults()
	data, total, err := h.policyRepo.List(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": data, "total": total})
}

// CreatePolicy handles POST /api/v1/cyber/dspm/policies
func (h *DSPMRemediationHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	policy := &model.DataPolicy{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             model.PolicyCategory(req.Category),
		Rule:                 req.Rule,
		Enforcement:          model.PolicyEnforcement(req.Enforcement),
		AutoPlaybookID:       req.AutoPlaybookID,
		Severity:             req.Severity,
		ScopeClassification:  req.ScopeClassification,
		ScopeAssetTypes:      req.ScopeAssetTypes,
		Enabled:              true,
		ComplianceFrameworks: req.ComplianceFrameworks,
		CreatedBy:            &userID,
	}
	created, err := h.policyRepo.Create(r.Context(), policy)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "CONFLICT", "a policy with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": created})
}

// GetPolicy handles GET /api/v1/cyber/dspm/policies/{id}
func (h *DSPMRemediationHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	policy, err := h.policyRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": policy})
}

// UpdatePolicy handles PUT /api/v1/cyber/dspm/policies/{id}
func (h *DSPMRemediationHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	existing, err := h.policyRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if len(req.Rule) > 0 {
		existing.Rule = req.Rule
	}
	if req.Enforcement != nil {
		existing.Enforcement = model.PolicyEnforcement(*req.Enforcement)
	}
	if req.AutoPlaybookID != nil {
		existing.AutoPlaybookID = *req.AutoPlaybookID
	}
	if req.Severity != nil {
		existing.Severity = *req.Severity
	}
	if req.ScopeClassification != nil {
		existing.ScopeClassification = req.ScopeClassification
	}
	if req.ScopeAssetTypes != nil {
		existing.ScopeAssetTypes = req.ScopeAssetTypes
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.ComplianceFrameworks != nil {
		existing.ComplianceFrameworks = req.ComplianceFrameworks
	}
	if err := h.policyRepo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": existing})
}

// DeletePolicy handles DELETE /api/v1/cyber/dspm/policies/{id}
func (h *DSPMRemediationHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.policyRepo.Delete(r.Context(), tenantID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"message": "policy deleted"})
}

// DryRunPolicy handles POST /api/v1/cyber/dspm/policies/{id}/dry-run
func (h *DSPMRemediationHandler) DryRunPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	policy, err := h.policyRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	impact, err := h.engine.DryRun(r.Context(), tenantID, policy.AutoPlaybookID, nil, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": impact})
}

// EvaluatePolicy handles POST /api/v1/cyber/dspm/policies/{id}/evaluate
func (h *DSPMRemediationHandler) EvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	violations, err := h.engine.EvaluatePolicies(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": violations, "total": len(violations)})
}

// GetViolations handles GET /api/v1/cyber/dspm/policies/violations
func (h *DSPMRemediationHandler) GetViolations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	violations, err := h.engine.EvaluatePolicies(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": violations, "total": len(violations)})
}

// ── Risk Exceptions ─────────────────────────────────────────────────────────────

// ListExceptions handles GET /api/v1/cyber/dspm/exceptions
func (h *DSPMRemediationHandler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, perPage := parsePageParams(r, 25)
	params := &dto.ExceptionListParams{
		Status:         q.Get("status"),
		ApprovalStatus: q.Get("approval_status"),
		ExceptionType:  q.Get("exception_type"),
		AssetID:        uuidPtr(q.Get("asset_id")),
		Search:         q.Get("search"),
		Page:           page,
		PerPage:        perPage,
	}
	params.SetDefaults()
	data, total, err := h.exceptionRepo.List(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": data, "total": total})
}

// CreateException handles POST /api/v1/cyber/dspm/exceptions
func (h *DSPMRemediationHandler) CreateException(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateExceptionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	exception := &model.RiskException{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		ExceptionType:        model.ExceptionType(req.ExceptionType),
		RemediationID:        req.RemediationID,
		DataAssetID:          req.DataAssetID,
		PolicyID:             req.PolicyID,
		Justification:        req.Justification,
		BusinessReason:       req.BusinessReason,
		CompensatingControls: req.CompensatingControls,
		RiskScore:            req.RiskScore,
		RiskLevel:            req.RiskLevel,
		RequestedBy:          userID,
		ApprovalStatus:       model.ApprovalPending,
		ExpiresAt:            req.ExpiresAt,
		ReviewIntervalDays:   req.ReviewIntervalDays,
		Status:               model.ExceptionStatusActive,
	}
	created, err := h.exceptionRepo.Create(r.Context(), exception)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": created})
}

// GetException handles GET /api/v1/cyber/dspm/exceptions/{id}
func (h *DSPMRemediationHandler) GetException(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	exc, err := h.exceptionRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": exc})
}

// ApproveException handles POST /api/v1/cyber/dspm/exceptions/{id}/approve
func (h *DSPMRemediationHandler) ApproveException(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	exc, err := h.exceptionRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if exc.ApprovalStatus != model.ApprovalPending {
		writeError(w, http.StatusConflict, "CONFLICT", "exception is not pending approval")
		return
	}
	if exc.RequestedBy == userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "approver cannot be the same as requester")
		return
	}
	exc.ApprovalStatus = model.ApprovalApproved
	exc.ApprovedBy = &userID
	now := r.Context().Value("now")
	if now == nil {
		t := uuid.Nil // just use time.Now
		_ = t
	}
	approvedAt := timeNow()
	exc.ApprovedAt = &approvedAt
	nextReview := approvedAt.AddDate(0, 0, exc.ReviewIntervalDays)
	exc.NextReviewAt = &nextReview
	if err := h.exceptionRepo.Update(r.Context(), exc); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": exc})
}

// RejectException handles POST /api/v1/cyber/dspm/exceptions/{id}/reject
func (h *DSPMRemediationHandler) RejectException(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.RejectExceptionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	exc, err := h.exceptionRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if exc.ApprovalStatus != model.ApprovalPending {
		writeError(w, http.StatusConflict, "CONFLICT", "exception is not pending approval")
		return
	}
	exc.ApprovalStatus = model.ApprovalRejected
	exc.ApprovedBy = &userID
	exc.RejectionReason = req.Reason
	if err := h.exceptionRepo.Update(r.Context(), exc); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": exc})
}

// ReviewException handles POST /api/v1/cyber/dspm/exceptions/{id}/review
func (h *DSPMRemediationHandler) ReviewException(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	exc, err := h.exceptionRepo.GetByID(r.Context(), tenantID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	now := timeNow()
	exc.LastReviewedAt = &now
	exc.ReviewCount++
	nextReview := now.AddDate(0, 0, exc.ReviewIntervalDays)
	exc.NextReviewAt = &nextReview
	if err := h.exceptionRepo.Update(r.Context(), exc); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": exc})
}
