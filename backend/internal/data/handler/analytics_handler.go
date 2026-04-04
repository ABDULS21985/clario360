package handler

import (
	"net"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/suiteapi"
)

type AnalyticsHandler struct {
	baseHandler
	service *service.AnalyticsService
}

func NewAnalyticsHandler(service *service.AnalyticsService, logger zerolog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		baseHandler: baseHandler{logger: logger},
		service:     service,
	}
}

func (h *AnalyticsHandler) Execute(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	var req dto.ExecuteAnalyticsQueryRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.Execute(r.Context(), tenantID, *userID, service.PermissionsFromContext(r.Context()), req, nil, clientIP(r), r.UserAgent())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AnalyticsHandler) Explore(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "modelId")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.ExploreAnalyticsRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.Explore(r.Context(), tenantID, *userID, modelID, service.PermissionsFromContext(r.Context()), req.Query, clientIP(r), r.UserAgent())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AnalyticsHandler) Explain(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	var req dto.ExplainAnalyticsQueryRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.Explain(r.Context(), tenantID, service.PermissionsFromContext(r.Context()), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AnalyticsHandler) ListSaved(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListSavedQueries(r.Context(), tenantID, *userID, dto.ListSavedQueriesParams{
		Page:       page,
		PerPage:    perPage,
		ModelID:    r.URL.Query().Get("model_id"),
		Visibility: r.URL.Query().Get("visibility"),
		Search:     r.URL.Query().Get("search"),
		Sort:       r.URL.Query().Get("sort"),
		Order:      r.URL.Query().Get("order"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *AnalyticsHandler) CreateSaved(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	var req dto.SaveQueryRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.CreateSavedQuery(r.Context(), tenantID, *userID, service.PermissionsFromContext(r.Context()), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *AnalyticsHandler) GetSaved(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GetSavedQuery(r.Context(), tenantID, *userID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AnalyticsHandler) UpdateSaved(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateSavedQueryRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.UpdateSavedQuery(r.Context(), tenantID, *userID, service.PermissionsFromContext(r.Context()), id, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AnalyticsHandler) DeleteSaved(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if err := h.service.DeleteSavedQuery(r.Context(), tenantID, *userID, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AnalyticsHandler) RunSaved(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := tenantAndUser(r)
	if !ok {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.RunSavedQuery(r.Context(), tenantID, *userID, service.PermissionsFromContext(r.Context()), id, clientIP(r), r.UserAgent())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AnalyticsHandler) Audit(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	params := dto.ListAnalyticsAuditParams{
		Page:           page,
		PerPage:        perPage,
		ModelID:        r.URL.Query().Get("model_id"),
		UserID:         r.URL.Query().Get("user_id"),
		Classification: r.URL.Query().Get("classification"),
		Sort:           r.URL.Query().Get("sort"),
		Order:          r.URL.Query().Get("order"),
	}
	if raw := r.URL.Query().Get("pii_accessed"); raw != "" {
		value := raw == "true"
		params.PIIAccessed = &value
	}
	items, total, err := h.service.ListAudit(r.Context(), tenantID, params)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
