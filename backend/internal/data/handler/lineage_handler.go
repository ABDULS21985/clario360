package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/suiteapi"
)

type LineageHandler struct {
	baseHandler
	service *service.LineageService
}

func NewLineageHandler(service *service.LineageService, logger zerolog.Logger) *LineageHandler {
	return &LineageHandler{
		baseHandler: baseHandler{logger: logger},
		service:     service,
	}
}

func (h *LineageHandler) FullGraph(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	graph, err := h.service.FullGraph(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, graph)
}

func (h *LineageHandler) EntityGraph(w http.ResponseWriter, r *http.Request) {
	h.writeDirectionalGraph(w, r, "")
}

func (h *LineageHandler) Upstream(w http.ResponseWriter, r *http.Request) {
	h.writeDirectionalGraph(w, r, "upstream")
}

func (h *LineageHandler) Downstream(w http.ResponseWriter, r *http.Request) {
	h.writeDirectionalGraph(w, r, "downstream")
}

func (h *LineageHandler) Impact(w http.ResponseWriter, r *http.Request) {
	tenantID, entityType, entityID, ok := lineageEntityRequest(w, r)
	if !ok {
		return
	}
	item, err := h.service.Impact(r.Context(), tenantID, entityType, entityID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *LineageHandler) Record(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	var req dto.RecordLineageEdgeRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.Record(r.Context(), tenantID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *LineageHandler) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if err := h.service.DeleteEdge(r.Context(), tenantID, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *LineageHandler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	limit := 25
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.service.Search(r.Context(), tenantID, dto.SearchLineageParams{
		Query: r.URL.Query().Get("q"),
		Type:  r.URL.Query().Get("type"),
		Limit: limit,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *LineageHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	stats, err := h.service.Stats(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, stats)
}

func (h *LineageHandler) writeDirectionalGraph(w http.ResponseWriter, r *http.Request, direction string) {
	tenantID, entityType, entityID, ok := lineageEntityRequest(w, r)
	if !ok {
		return
	}
	depth := 3
	if raw := r.URL.Query().Get("depth"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			depth = parsed
		}
	}
	var (
		graph *model.LineageGraph
		err   error
	)
	switch direction {
	case "upstream":
		graph, err = h.service.Upstream(r.Context(), tenantID, entityType, entityID, depth)
	case "downstream":
		graph, err = h.service.Downstream(r.Context(), tenantID, entityType, entityID, depth)
	default:
		graph, err = h.service.EntityGraph(r.Context(), tenantID, entityType, entityID, depth)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, graph)
}

func lineageEntityRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, model.LineageEntityType, uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, "", uuid.Nil, false
	}
	entityType := model.LineageEntityType(chi.URLParam(r, "entityType"))
	if !entityType.IsValid() {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid entityType", nil)
		return uuid.Nil, "", uuid.Nil, false
	}
	entityID, err := uuid.Parse(chi.URLParam(r, "entityId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid entityId", nil)
		return uuid.Nil, "", uuid.Nil, false
	}
	return tenantID, entityType, entityID, true
}
