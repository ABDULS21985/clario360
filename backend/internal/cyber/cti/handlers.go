package cti

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	apperrors "github.com/clario360/platform/internal/errors"
	"github.com/clario360/platform/internal/middleware"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

type envelope map[string]any

type Handler struct {
	svc    *Service
	logger zerolog.Logger
}

func NewHandler(svc *Service, logger zerolog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ---------------------------------------------------------------------------
// Response helpers (match existing cyber handler pattern)
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(normalizePaginatedResponse(v))
}

func writeError(w http.ResponseWriter, status int, code, message string, details ...any) {
	var detailValue any
	if len(details) > 0 {
		detailValue = details[0]
	}
	if status >= http.StatusInternalServerError {
		code = "INTERNAL_ERROR"
		message = "internal server error"
		detailValue = nil
	}
	writeJSON(w, status, map[string]any{
		"code":       code,
		"message":    message,
		"details":    detailValue,
		"request_id": w.Header().Get(middleware.RequestIDHeader),
	})
}

func handleErr(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if ok := stderrors.As(err, &appErr); ok {
		writeError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Fields)
		return
	}
	status := apperrors.HTTPStatus(err)
	switch {
	case apperrors.IsNotFound(err):
		writeError(w, status, "NOT_FOUND", "resource not found")
	case apperrors.IsConflict(err):
		writeError(w, status, "CONFLICT", "resource already exists")
	case apperrors.IsValidation(err):
		writeError(w, status, "VALIDATION_ERROR", err.Error())
	default:
		writeError(w, status, "INTERNAL_ERROR", err.Error())
	}
}

func writeValidationError(w http.ResponseWriter, fieldErrs map[string]string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":       "VALIDATION_ERROR",
		"message":    "request validation failed",
		"details":    map[string]any{"fields": fieldErrs},
		"request_id": w.Header().Get(middleware.RequestIDHeader),
	})
}

func parseUUID(w http.ResponseWriter, s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("invalid UUID: %s", s), nil)
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "request body must be valid JSON", map[string]any{"cause": err.Error()})
		return false
	}
	return true
}

func normalizePaginatedResponse(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return v
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return v
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return v
	}
	dataField := rv.FieldByName("Data")
	if !dataField.IsValid() {
		return v
	}
	if dataField.Kind() == reflect.Slice && dataField.IsNil() {
		dataField.Set(reflect.MakeSlice(dataField.Type(), 0, 0))
	}
	return v
}

// ---------------------------------------------------------------------------
// Reference data
// ---------------------------------------------------------------------------

func (h *Handler) ListSeverityLevels(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListSeverityLevels(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListCategories(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *Handler) ListRegions(w http.ResponseWriter, r *http.Request) {
	var parentID *uuid.UUID
	if p := r.URL.Query().Get("parent_id"); p != "" {
		id, ok := parseUUID(w, p)
		if !ok {
			return
		}
		parentID = &id
	}
	items, err := h.svc.ListRegions(r.Context(), parentID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *Handler) ListSectors(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListSectors(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *Handler) ListDataSources(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListDataSources(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

// ---------------------------------------------------------------------------
// Threat events
// ---------------------------------------------------------------------------

func (h *Handler) CreateThreatEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateThreatEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.CreateThreatEvent(r.Context(), req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *Handler) ListThreatEvents(w http.ResponseWriter, r *http.Request) {
	f := ParseThreatEventFilters(r)
	if err := f.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.svc.ListThreatEvents(r.Context(), f)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetThreatEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	result, err := h.svc.GetThreatEvent(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) UpdateThreatEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	var req UpdateThreatEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.UpdateThreatEvent(r.Context(), id, req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) DeleteThreatEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	if err := h.svc.DeleteThreatEvent(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkEventFalsePositive(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	if err := h.svc.MarkEventFalsePositive(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"marked": true}})
}

func (h *Handler) ResolveEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	if err := h.svc.ResolveEvent(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"resolved": true}})
}

func (h *Handler) GetEventTags(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	tags, err := h.svc.GetEventTags(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	if tags == nil {
		tags = []string{}
	}
	writeJSON(w, http.StatusOK, envelope{"data": tags})
}

func (h *Handler) AddEventTags(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	var req AddTagsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.AddEventTags(r.Context(), id, req.Tags); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"added": len(req.Tags)}})
}

func (h *Handler) RemoveEventTag(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "tag is required")
		return
	}
	if err := h.svc.RemoveEventTag(r.Context(), id, tag); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Threat actors
// ---------------------------------------------------------------------------

func (h *Handler) CreateThreatActor(w http.ResponseWriter, r *http.Request) {
	var req CreateThreatActorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.CreateThreatActor(r.Context(), req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *Handler) ListThreatActors(w http.ResponseWriter, r *http.Request) {
	f := ParseThreatActorFilters(r)
	result, err := h.svc.ListThreatActors(r.Context(), f)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetThreatActor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "actorID"))
	if !ok {
		return
	}
	result, err := h.svc.GetThreatActor(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) UpdateThreatActor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "actorID"))
	if !ok {
		return
	}
	var req UpdateThreatActorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.UpdateThreatActor(r.Context(), id, req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) DeleteThreatActor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "actorID"))
	if !ok {
		return
	}
	if err := h.svc.DeleteThreatActor(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

func (h *Handler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.CreateCampaign(r.Context(), req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *Handler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	f := ParseCampaignFilters(r)
	result, err := h.svc.ListCampaigns(r.Context(), f)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	result, err := h.svc.GetCampaign(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	var req UpdateCampaignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.UpdateCampaign(r.Context(), id, req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	if err := h.svc.DeleteCampaign(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateCampaignStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	var req UpdateStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.UpdateCampaignStatus(r.Context(), id, req.Status); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"status": req.Status}})
}

func (h *Handler) ListCampaignEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	p := ParseListParams(r)
	result, err := h.svc.ListCampaignEvents(r.Context(), id, p)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) LinkEventToCampaign(w http.ResponseWriter, r *http.Request) {
	cid, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	eid, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	if err := h.svc.LinkEventToCampaign(r.Context(), cid, eid); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": map[string]any{"linked": true}})
}

func (h *Handler) UnlinkEventFromCampaign(w http.ResponseWriter, r *http.Request) {
	cid, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	eid, ok := parseUUID(w, chi.URLParam(r, "eventID"))
	if !ok {
		return
	}
	if err := h.svc.UnlinkEventFromCampaign(r.Context(), cid, eid); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListCampaignIOCs(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	p := ParseListParams(r)
	result, err := h.svc.ListCampaignIOCs(r.Context(), id, p)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateCampaignIOC(w http.ResponseWriter, r *http.Request) {
	cid, ok := parseUUID(w, chi.URLParam(r, "campaignID"))
	if !ok {
		return
	}
	var req CreateCampaignIOCRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.CreateCampaignIOC(r.Context(), cid, req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *Handler) DeleteCampaignIOC(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "iocID"))
	if !ok {
		return
	}
	if err := h.svc.DeleteCampaignIOC(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Brand abuse
// ---------------------------------------------------------------------------

func (h *Handler) CreateMonitoredBrand(w http.ResponseWriter, r *http.Request) {
	var req CreateMonitoredBrandRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.CreateMonitoredBrand(r.Context(), req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *Handler) ListMonitoredBrands(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListMonitoredBrands(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *Handler) UpdateMonitoredBrand(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "brandID"))
	if !ok {
		return
	}
	var req UpdateMonitoredBrandRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.UpdateMonitoredBrand(r.Context(), id, req); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *Handler) DeleteMonitoredBrand(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "brandID"))
	if !ok {
		return
	}
	if err := h.svc.DeleteMonitoredBrand(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateBrandAbuseIncident(w http.ResponseWriter, r *http.Request) {
	var req CreateBrandAbuseIncidentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.CreateBrandAbuseIncident(r.Context(), req)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *Handler) ListBrandAbuseIncidents(w http.ResponseWriter, r *http.Request) {
	f := ParseBrandAbuseFilters(r)
	result, err := h.svc.ListBrandAbuseIncidents(r.Context(), f)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetBrandAbuseIncident(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "incidentID"))
	if !ok {
		return
	}
	result, err := h.svc.GetBrandAbuseIncident(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) UpdateBrandAbuseIncident(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "incidentID"))
	if !ok {
		return
	}
	var raw map[string]interface{}
	if !decodeJSON(w, r, &raw) {
		return
	}
	if err := h.svc.UpdateBrandAbuseIncident(r.Context(), id, raw); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"updated": true}})
}

func (h *Handler) UpdateTakedownStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "incidentID"))
	if !ok {
		return
	}
	var req UpdateTakedownStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.UpdateTakedownStatus(r.Context(), id, req.Status); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"takedown_status": req.Status}})
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

func (h *Handler) GetGlobalThreatMap(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7d"
	}
	result, err := h.svc.GetGlobalThreatMap(r.Context(), period)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) GetSectorThreatOverview(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7d"
	}
	result, err := h.svc.GetSectorThreatOverview(r.Context(), period)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) GetExecutiveDashboard(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetExecutiveDashboard(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *Handler) RefreshAggregations(w http.ResponseWriter, r *http.Request) {
	scope, err := ParseAggregationRefreshScope(r.URL.Query().Get("scope"))
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := h.svc.RefreshAggregations(r.Context(), scope); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"refreshed": true, "scope": scope}})
}
