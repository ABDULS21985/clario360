package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

type DSPMHandler struct {
	svc *service.DSPMService
}

func NewDSPMHandler(svc *service.DSPMService) *DSPMHandler {
	return &DSPMHandler{svc: svc}
}

func (h *DSPMHandler) ListDataAssets(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.ListDataAssets(r.Context(), tenantID, params)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) GetDataAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	dataAssetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetDataAsset(r.Context(), tenantID, dataAssetID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *DSPMHandler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	scan, err := h.svc.TriggerScan(r.Context(), tenantID, userID, actorFromRequest(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": dto.DSPMScanTriggerResponse{Scan: scan}})
}

func (h *DSPMHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := &dto.DSPMScanListParams{}
	if v := r.URL.Query().Get("status"); v != "" {
		params.Status = &v
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
	result, err := h.svc.ListScans(r.Context(), tenantID, params)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) GetScan(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	scanID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	result, err := h.svc.GetScan(r.Context(), tenantID, scanID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) Classification(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ClassificationSummary(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) Exposure(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ExposureAnalysis(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) Dependencies(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.Dependencies(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.Dashboard(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *DSPMHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}

func parseDSPMAssetListParams(r *http.Request) (*dto.DSPMAssetListParams, error) {
	q := r.URL.Query()
	params := &dto.DSPMAssetListParams{
		Sort:  q.Get("sort"),
		Order: q.Get("order"),
	}
	if v := q.Get("classification"); v != "" {
		params.Classification = &v
	}
	if v := q.Get("contains_pii"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
		params.ContainsPII = &parsed
	}
	if v := q.Get("min_risk_score"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		params.MinRiskScore = &parsed
	}
	if v := q.Get("network_exposure"); v != "" {
		params.NetworkExposure = &v
	}
	if v := q.Get("asset_id"); v != "" {
		id, err := parseUUIDValue(v)
		if err != nil {
			return nil, err
		}
		params.AssetID = &id
	}
	if v := q.Get("search"); v != "" {
		params.Search = &v
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
