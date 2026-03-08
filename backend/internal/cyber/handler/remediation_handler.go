package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/remediation"
	"github.com/clario360/platform/internal/cyber/repository"
)

type RemediationHandler struct {
	svc remediationService
}

func NewRemediationHandler(svc remediationService) *RemediationHandler {
	return &RemediationHandler{svc: svc}
}

func (h *RemediationHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateRemediationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.Create(r.Context(), tenantID, userID, actorFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *RemediationHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseRemediationListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.List(r.Context(), tenantID, params)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *RemediationHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.Get(r.Context(), tenantID, remediationID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateRemediationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.Update(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), tenantID, remediationID, actorFromRequest(r)); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *RemediationHandler) Submit(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.Submit(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Approve(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ApproveRemediationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.Approve(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Reject(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.RejectRemediationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.Reject(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) RequestRevision(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.RequestRevisionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.RequestRevision(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) DryRun(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	result, err := h.svc.DryRun(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *RemediationHandler) GetDryRun(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	result, err := h.svc.GetDryRun(r.Context(), tenantID, remediationID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *RemediationHandler) Execute(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ExecuteRemediationRequest
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	item, err := h.svc.Execute(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Verify(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.VerifyRemediationRequest
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	item, err := h.svc.Verify(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.RollbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.Rollback(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) Close(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.Close(r.Context(), tenantID, remediationID, userID, auth.UserFromContext(r.Context()).Email, remediationRoleFromRequest(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *RemediationHandler) AuditTrail(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	remediationID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.AuditTrail(r.Context(), tenantID, remediationID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *RemediationHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.Stats(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

func (h *RemediationHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "remediation not found", nil)
	case errors.Is(err, remediation.ErrInsufficientPermission):
		writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, remediation.ErrPreConditionFailed), errors.Is(err, remediation.ErrInvalidTransition):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}

func remediationRoleFromRequest(r *http.Request) string {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		return "viewer"
	}
	order := []string{"admin", "ciso", "tenant_admin", "security_manager", "analyst", "security_analyst", "viewer"}
	for _, role := range order {
		for _, candidate := range user.Roles {
			if candidate == role {
				return role
			}
		}
	}
	return "viewer"
}

func parseRemediationListParams(r *http.Request) (*dto.RemediationListParams, error) {
	q := r.URL.Query()
	params := &dto.RemediationListParams{
		Statuses:   splitQueryValues(q, "status"),
		Types:      splitQueryValues(q, "type"),
		Severities: splitQueryValues(q, "severity"),
		Tags:       splitQueryValues(q, "tag"),
		Sort:       q.Get("sort"),
		Order:      q.Get("order"),
	}
	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	if v := q.Get("asset_id"); v != "" {
		id, err := parseUUIDValue(v)
		if err != nil {
			return nil, err
		}
		params.AssetID = &id
	}
	if v := q.Get("alert_id"); v != "" {
		id, err := parseUUIDValue(v)
		if err != nil {
			return nil, err
		}
		params.AlertID = &id
	}
	if v := q.Get("vulnerability_id"); v != "" {
		id, err := parseUUIDValue(v)
		if err != nil {
			return nil, err
		}
		params.VulnID = &id
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	return params, params.Validate()
}

func parseUUIDValue(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}
