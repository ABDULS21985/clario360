package handler

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/suiteapi"
)

type ComplianceHandler struct {
	baseHandler
	service *service.ComplianceService
}

func NewComplianceHandler(service *service.ComplianceService, logger zerolog.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		baseHandler: baseHandler{logger: logger},
		service:     service,
	}
}

func (h *ComplianceHandler) Run(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	report, err := h.service.RunChecks(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, report)
}

func (h *ComplianceHandler) Results(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	filters := model.ComplianceFilters{
		Page:    page,
		PerPage: perPage,
	}
	committeeID, err := parseOptionalUUID(r.URL.Query().Get("committee_id"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid committee_id", nil)
		return
	}
	dateFrom, err := parseOptionalDateTime(r.URL.Query().Get("date_from"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid date_from", nil)
		return
	}
	dateTo, err := parseOptionalDateTime(r.URL.Query().Get("date_to"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid date_to", nil)
		return
	}
	filters.CommitteeID = committeeID
	filters.DateFrom = dateFrom
	filters.DateTo = dateTo
	if raw := r.URL.Query().Get("check_type"); raw != "" {
		checkType := model.ComplianceCheckType(raw)
		filters.CheckType = &checkType
	}
	rawStatuses := suiteapi.ParseCSVParam(r, "status")
	if len(rawStatuses) > 0 {
		filters.Statuses = make([]model.ComplianceStatus, 0, len(rawStatuses))
		for _, status := range rawStatuses {
			filters.Statuses = append(filters.Statuses, model.ComplianceStatus(status))
		}
	}
	items, total, err := h.service.ListResults(r.Context(), tenantID, filters)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *ComplianceHandler) Report(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	report, err := h.service.RunChecks(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, report)
}

func (h *ComplianceHandler) Score(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	score, err := h.service.Score(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]float64{"score": score})
}
