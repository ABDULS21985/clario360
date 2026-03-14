package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/service"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

type ThreatFeedHandler struct {
	svc *service.ThreatFeedService
}

func NewThreatFeedHandler(svc *service.ThreatFeedService) *ThreatFeedHandler {
	return &ThreatFeedHandler{svc: svc}
}

func (h *ThreatFeedHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	page := 1
	perPage := 25
	if v := r.URL.Query().Get("page"); v != "" {
		page, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		perPage, _ = strconv.Atoi(v)
	}
	result, err := h.svc.ListFeeds(r.Context(), tenantID, page, perPage, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ThreatFeedHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.ThreatFeedConfigRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.CreateFeed(r.Context(), tenantID, userID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *ThreatFeedHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	feedID, ok := parseUUID(w, chi.URLParam(r, "feedId"))
	if !ok {
		return
	}
	var req dto.ThreatFeedConfigRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.UpdateFeed(r.Context(), tenantID, feedID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatFeedHandler) Sync(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	feedID, ok := parseUUID(w, chi.URLParam(r, "feedId"))
	if !ok {
		return
	}
	result, err := h.svc.SyncFeed(r.Context(), tenantID, feedID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "SYNC_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *ThreatFeedHandler) History(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	feedID, ok := parseUUID(w, chi.URLParam(r, "feedId"))
	if !ok {
		return
	}
	items, err := h.svc.ListHistory(r.Context(), tenantID, feedID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "HISTORY_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}
