package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/service"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

// RuleHandler handles detection rule endpoints.
type RuleHandler struct {
	svc *service.RuleService
}

// NewRuleHandler creates a new RuleHandler.
func NewRuleHandler(svc *service.RuleService) *RuleHandler {
	return &RuleHandler{svc: svc}
}

func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListRules(r.Context(), tenantID, parseRuleListParams(r), actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.CreateRule(r.Context(), tenantID, userID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	ruleID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetRule(r.Context(), tenantID, ruleID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "GET_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	ruleID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateRule(r.Context(), tenantID, ruleID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	ruleID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteRule(r.Context(), tenantID, ruleID, actorFromRequest(r)); err != nil {
		writeError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RuleHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	ruleID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.RuleToggleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.Toggle(r.Context(), tenantID, ruleID, actorFromRequest(r), req.Enabled)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TOGGLE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RuleHandler) TestRule(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	ruleID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.RuleTestRequest
	_ = decodeJSON(w, r, &req)
	result, err := h.svc.TestRule(r.Context(), tenantID, ruleID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TEST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *RuleHandler) Feedback(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.RuleFeedbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.SubmitFeedback(r.Context(), tenantID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "FEEDBACK_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RuleHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListTemplates(r.Context(), tenantID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TEMPLATES_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}
