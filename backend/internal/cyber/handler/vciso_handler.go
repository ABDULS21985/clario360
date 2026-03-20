package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
)

type VCISOHandler struct {
	svc    vcisoService
	logger zerolog.Logger
}

func NewVCISOHandler(svc vcisoService) *VCISOHandler {
	return &VCISOHandler{svc: svc, logger: log.Logger.With().Str("handler", "vciso").Logger()}
}

func (h *VCISOHandler) Briefing(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := &dto.VCISOBriefingParams{}
	if v := r.URL.Query().Get("period_days"); v != "" {
		params.PeriodDays, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	briefing, err := h.svc.GenerateBriefing(r.Context(), tenantID, userID, params.PeriodDays, actorFromRequest(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": briefing})
}

func (h *VCISOHandler) BriefingHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := &dto.VCISOBriefingHistoryParams{}
	if v := r.URL.Query().Get("type"); v != "" {
		params.Type = &v
	}
	if v := r.URL.Query().Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.ListBriefings(r.Context(), tenantID, params)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *VCISOHandler) Recommendations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.Recommendations(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *VCISOHandler) Report(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.VCISOReportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.GenerateReport(r.Context(), tenantID, userID, &req, actorFromRequest(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": result})
}

func (h *VCISOHandler) PostureSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.PostureSummary(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *VCISOHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	default:
		h.logger.Error().Err(err).Msg("vciso handler error")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
