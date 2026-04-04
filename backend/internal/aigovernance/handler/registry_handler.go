package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/suiteapi"
)

type RegistryHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewRegistryHandler(services Services, logger zerolog.Logger) *RegistryHandler {
	return &RegistryHandler{services: services, logger: logger.With().Str("handler", "ai_registry").Logger()}
}

func (h *RegistryHandler) RegisterModel(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	userID := userID(r)
	if userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	var req aigovdto.RegisterModelRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Registry.RegisterModel(r.Context(), tenantID, *userID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *RegistryHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.services.Registry.ListModels(r.Context(), tenantID, repository.ListModelsParams{
		Suite:   r.URL.Query().Get("suite"),
		Type:    r.URL.Query().Get("type"),
		Status:  r.URL.Query().Get("status"),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *RegistryHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.services.Registry.GetModel(r.Context(), tenantID, modelID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *RegistryHandler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.UpdateModelRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Registry.UpdateModel(r.Context(), tenantID, modelID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *RegistryHandler) CreateVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	userID := userID(r)
	if userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.CreateVersionRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Registry.CreateVersion(r.Context(), tenantID, *userID, modelID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *RegistryHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	items, err := h.services.Registry.ListVersions(r.Context(), tenantID, modelID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *RegistryHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	versionID, err := suiteapi.UUIDParam(r, "vid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.services.Registry.GetVersion(r.Context(), tenantID, modelID, versionID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *RegistryHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.RegisterModel)
	r.Get("/", h.ListModels)
	r.Get("/{id}", h.GetModel)
	r.Put("/{id}", h.UpdateModel)
	r.Post("/{id}/versions", h.CreateVersion)
	r.Get("/{id}/versions", h.ListVersions)
	r.Get("/{id}/versions/{vid}", h.GetVersion)
	return r
}
