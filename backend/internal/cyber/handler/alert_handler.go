package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

// AlertHandler handles alert lifecycle endpoints.
type AlertHandler struct {
	svc alertService
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(svc alertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseAlertListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.ListAlerts(r.Context(), tenantID, params, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AlertHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	alert, err := h.svc.GetAlert(r.Context(), tenantID, alertID, actorFromRequest(r))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "alert not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "GET_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *AlertHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertStatusUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	alert, err := h.svc.UpdateStatus(r.Context(), tenantID, alertID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "STATUS_UPDATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *AlertHandler) Assign(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertAssignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	alert, err := h.svc.Assign(r.Context(), tenantID, alertID, actorFromRequest(r), req.AssignedTo)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ASSIGN_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *AlertHandler) Escalate(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertEscalateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	alert, err := h.svc.Escalate(r.Context(), tenantID, alertID, actorFromRequest(r), req.EscalatedTo, req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ESCALATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *AlertHandler) MarkFalsePositive(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertFalsePositiveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	alert, err := h.svc.MarkFalsePositive(r.Context(), tenantID, alertID, actorFromRequest(r), req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, "FALSE_POSITIVE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *AlertHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertCommentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	comment, err := h.svc.AddComment(r.Context(), tenantID, alertID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "COMMENT_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": comment})
}

func (h *AlertHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.ListComments(r.Context(), tenantID, alertID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "COMMENTS_LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *AlertHandler) ListTimeline(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.ListTimeline(r.Context(), tenantID, alertID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TIMELINE_LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *AlertHandler) Merge(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertMergeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	alert, err := h.svc.Merge(r.Context(), tenantID, alertID, req.MergeIDs, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "MERGE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *AlertHandler) Related(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.Related(r.Context(), tenantID, alertID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "RELATED_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *AlertHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.Stats(r.Context(), tenantID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

func (h *AlertHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.BulkAlertStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.BulkUpdateStatus(r.Context(), tenantID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "BULK_STATUS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *AlertHandler) BulkAssign(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.BulkAlertAssignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.BulkAssign(r.Context(), tenantID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "BULK_ASSIGN_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *AlertHandler) BulkMarkFalsePositive(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.BulkAlertFalsePositiveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.BulkMarkFalsePositive(r.Context(), tenantID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "BULK_FALSE_POSITIVE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *AlertHandler) Count(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseAlertListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	resp, err := h.svc.CountWithHistory(r.Context(), tenantID, params, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "COUNT_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": resp})
}
