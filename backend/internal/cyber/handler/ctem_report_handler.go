package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/service"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

type CTEMReportHandler struct {
	svc    *service.CTEMService
	logger zerolog.Logger
}

func NewCTEMReportHandler(svc *service.CTEMService, logger zerolog.Logger) *CTEMReportHandler {
	return &CTEMReportHandler{svc: svc, logger: logger}
}

func (h *CTEMReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	report, err := h.svc.BuildReport(r.Context(), tenantID, assessmentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": report})
}

func (h *CTEMReportHandler) GetExecutiveSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	summary, err := h.svc.BuildExecutiveSummary(r.Context(), tenantID, assessmentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": summary})
}

func (h *CTEMReportHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CTEMReportExportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	resp, err := h.svc.ExportReport(r.Context(), tenantID, assessmentID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": resp})
}
