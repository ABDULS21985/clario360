package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

type CTEMHandler struct {
	svc    *service.CTEMService
	logger zerolog.Logger
}

func NewCTEMHandler(svc *service.CTEMService, logger zerolog.Logger) *CTEMHandler {
	return &CTEMHandler{svc: svc, logger: logger}
}

func (h *CTEMHandler) CreateAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateCTEMAssessmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	assessment, err := h.svc.CreateAssessment(r.Context(), tenantID, userID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": assessment})
}

func (h *CTEMHandler) ListAssessments(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.ListAssessments(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *CTEMHandler) GetAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	assessment, err := h.svc.GetAssessment(r.Context(), tenantID, assessmentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "assessment not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": assessment})
}

func (h *CTEMHandler) UpdateAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateCTEMAssessmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	assessment, err := h.svc.UpdateAssessment(r.Context(), tenantID, assessmentID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "assessment can only be updated while in created state", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": assessment})
}

func (h *CTEMHandler) StartAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.StartAssessment(r.Context(), tenantID, userID, assessmentID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": map[string]any{"status": "started"}})
}

func (h *CTEMHandler) CancelAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.CancelAssessment(r.Context(), tenantID, assessmentID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": map[string]any{"status": "cancelled"}})
}

func (h *CTEMHandler) DeleteAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteAssessment(r.Context(), tenantID, assessmentID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CTEMHandler) GetScope(w http.ResponseWriter, r *http.Request) {
	h.getPhaseResult(w, r, "scoping")
}

func (h *CTEMHandler) GetDiscovery(w http.ResponseWriter, r *http.Request) {
	h.getPhaseResult(w, r, "discovery")
}

func (h *CTEMHandler) GetPriorities(w http.ResponseWriter, r *http.Request) {
	h.getPhaseResult(w, r, "prioritizing")
}

func (h *CTEMHandler) GetValidation(w http.ResponseWriter, r *http.Request) {
	h.getPhaseResult(w, r, "validating")
}

func (h *CTEMHandler) GetMobilization(w http.ResponseWriter, r *http.Request) {
	h.getPhaseResult(w, r, "mobilizing")
}

func (h *CTEMHandler) ValidateAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ValidateAssessmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.ValidateAssessment(r.Context(), tenantID, assessmentID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": map[string]any{"status": "validation_started"}})
}

func (h *CTEMHandler) MobilizeAssessment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.MobilizeAssessment(r.Context(), tenantID, assessmentID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": map[string]any{"status": "mobilization_started"}})
}

func (h *CTEMHandler) ListFindings(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.ListFindings(r.Context(), tenantID, assessmentID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *CTEMHandler) GetFinding(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	findingID, ok := parseUUID(w, chi.URLParam(r, "findingId"))
	if !ok {
		return
	}
	finding, err := h.svc.GetFinding(r.Context(), tenantID, findingID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "finding not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": finding})
}

func (h *CTEMHandler) UpdateFindingStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	findingID, ok := parseUUID(w, chi.URLParam(r, "findingId"))
	if !ok {
		return
	}
	var req dto.UpdateCTEMFindingStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	finding, err := h.svc.UpdateFindingStatus(r.Context(), tenantID, findingID, userID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": finding})
}

func (h *CTEMHandler) ListRemediationGroups(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	groups, err := h.svc.ListRemediationGroups(r.Context(), tenantID, assessmentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": groups})
}

func (h *CTEMHandler) GetRemediationGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	groupID, ok := parseUUID(w, chi.URLParam(r, "groupId"))
	if !ok {
		return
	}
	group, findings, err := h.svc.GetRemediationGroup(r.Context(), tenantID, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"group": group, "findings": findings}})
}

func (h *CTEMHandler) UpdateRemediationGroupStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	groupID, ok := parseUUID(w, chi.URLParam(r, "groupId"))
	if !ok {
		return
	}
	var req dto.UpdateCTEMRemediationGroupStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	group, err := h.svc.UpdateRemediationGroupStatus(r.Context(), tenantID, groupID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": group})
}

func (h *CTEMHandler) ExecuteRemediationGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	groupID, ok := parseUUID(w, chi.URLParam(r, "groupId"))
	if !ok {
		return
	}
	group, err := h.svc.ExecuteRemediationGroup(r.Context(), tenantID, userID, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": group})
}

func (h *CTEMHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	dashboard, err := h.svc.Dashboard(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": dashboard})
}

func (h *CTEMHandler) GetExposureScore(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	score, err := h.svc.CurrentExposureScore(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": score})
}

func (h *CTEMHandler) GetExposureScoreHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	days := 90
	if v := r.URL.Query().Get("days"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "days must be an integer", nil)
			return
		}
		days = parsed
	}
	history, err := h.svc.ExposureHistory(r.Context(), tenantID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": history})
}

func (h *CTEMHandler) ForceCalculateExposureScore(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	score, err := h.svc.ForceCalculateExposureScore(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": score})
}

func (h *CTEMHandler) CompareAssessments(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	otherID, ok := parseUUID(w, chi.URLParam(r, "otherId"))
	if !ok {
		return
	}
	comparison, err := h.svc.CompareAssessments(r.Context(), tenantID, assessmentID, otherID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": comparison})
}

func (h *CTEMHandler) getPhaseResult(w http.ResponseWriter, r *http.Request, phase string) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assessmentID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	result, err := h.svc.GetPhaseResult(r.Context(), tenantID, assessmentID, phase)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var decoded any
	if err := json.Unmarshal(result, &decoded); err != nil {
		decoded = map[string]any{}
	}
	writeJSON(w, http.StatusOK, envelope{"data": decoded})
}

func parseCTEMAssessmentListParams(r *http.Request) (*dto.CTEMAssessmentListParams, error) {
	q := r.URL.Query()
	params := &dto.CTEMAssessmentListParams{}
	if v := q.Get("status"); v != "" {
		params.Status = &v
	}
	if v := q.Get("scheduled"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
		params.Scheduled = &parsed
	}
	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	if v := q.Get("tag"); v != "" {
		params.Tag = &v
	}
	if v := q.Get("page"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		params.Page = parsed
	}
	if v := q.Get("per_page"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		params.PerPage = parsed
	}
	params.Sort = q.Get("sort")
	params.Order = q.Get("order")
	return params, nil
}

func parseCTEMFindingListParams(r *http.Request) (*dto.CTEMFindingsListParams, error) {
	q := r.URL.Query()
	params := &dto.CTEMFindingsListParams{}
	if v := q.Get("severity"); v != "" {
		params.Severity = &v
	}
	if v := q.Get("type"); v != "" {
		params.Type = &v
	}
	if v := q.Get("status"); v != "" {
		params.Status = &v
	}
	if v := q.Get("priority_group"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		params.PriorityGroup = &parsed
	}
	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	if v := q.Get("page"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		params.Page = parsed
	}
	if v := q.Get("per_page"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		params.PerPage = parsed
	}
	params.Sort = q.Get("sort")
	params.Order = q.Get("order")
	return params, nil
}
