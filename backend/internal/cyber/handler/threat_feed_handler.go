package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

type ThreatFeedHandler struct {
	svc threatFeedService
}

func NewThreatFeedHandler(svc threatFeedService) *ThreatFeedHandler {
	return &ThreatFeedHandler{svc: svc}
}

func (h *ThreatFeedHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page := 1
	perPage := 25
	if v := q.Get("page"); v != "" {
		page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		perPage, _ = strconv.Atoi(v)
	}
	search := q.Get("search")
	sort := q.Get("sort")
	order := q.Get("order")
	result, err := h.svc.ListFeeds(r.Context(), tenantID, page, perPage, search, sort, order, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ThreatFeedHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	feedID, ok := parseUUID(w, chi.URLParam(r, "feedId"))
	if !ok {
		return
	}
	item, err := h.svc.GetFeed(r.Context(), tenantID, feedID, actorFromRequest(r))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "threat feed not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "GET_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
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
		var syncErr *service.SyncError
		if errors.As(err, &syncErr) {
			switch syncErr.Kind {
			case service.SyncErrNotFound:
				writeError(w, http.StatusNotFound, "NOT_FOUND", syncErr.Error(), nil)
			case service.SyncErrBadConfig:
				writeError(w, http.StatusUnprocessableEntity, "BAD_CONFIG", syncErr.Error(), nil)
			case service.SyncErrUpstream:
				writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", syncErr.Error(), nil)
			case service.SyncErrParse:
				writeError(w, http.StatusUnprocessableEntity, "PARSE_ERROR", syncErr.Error(), nil)
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
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

func (h *ThreatFeedHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	feedID, ok := parseUUID(w, chi.URLParam(r, "feedId"))
	if !ok {
		return
	}
	err := h.svc.DeleteFeed(r.Context(), tenantID, feedID, actorFromRequest(r))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "threat feed not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
