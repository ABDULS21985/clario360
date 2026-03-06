package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/model"
)

// definitionService defines the operations available for workflow definitions.
type definitionService interface {
	Create(ctx context.Context, tenantID, userID string, req dto.CreateDefinitionRequest) (*model.WorkflowDefinition, error)
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error)
	List(ctx context.Context, tenantID, status, nameFilter string, page, pageSize int) ([]*model.WorkflowDefinition, int, error)
	Update(ctx context.Context, tenantID, id, userID string, req dto.UpdateDefinitionRequest) (*model.WorkflowDefinition, error)
	Activate(ctx context.Context, tenantID, id string) error
	Delete(ctx context.Context, tenantID, id string) error
	ListVersions(ctx context.Context, tenantID, id string) ([]*model.WorkflowDefinition, error)
}

// DefinitionHandler handles HTTP requests for workflow definition CRUD operations.
type DefinitionHandler struct {
	service definitionService
	logger  zerolog.Logger
}

// NewDefinitionHandler creates a new DefinitionHandler with the given service and logger.
func NewDefinitionHandler(service definitionService, logger zerolog.Logger) *DefinitionHandler {
	return &DefinitionHandler{
		service: service,
		logger:  logger.With().Str("handler", "workflow_definition").Logger(),
	}
}

// Routes returns a chi.Router with all definition routes mounted.
func (h *DefinitionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/activate", h.Activate)
	r.Get("/{id}/versions", h.ListVersions)

	return r
}

// Create handles POST / — creates a new workflow definition.
func (h *DefinitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req dto.CreateDefinitionRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Run domain-level validation.
	if validationErrs := req.Validate(); len(validationErrs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"code":    "VALIDATION_ERROR",
			"message": "definition validation failed",
			"errors":  validationErrs,
		})
		return
	}

	def, err := h.service.Create(r.Context(), user.TenantID, user.ID, req)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", user.TenantID).Msg("failed to create workflow definition")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("definition_id", def.ID).
		Str("user_id", user.ID).
		Msg("workflow definition created")

	writeJSON(w, http.StatusCreated, dto.DefinitionToResponse(def))
}

// List handles GET / — lists workflow definitions with filtering and pagination.
func (h *DefinitionHandler) List(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	status := r.URL.Query().Get("status")
	if status != "" && !model.ValidDefinitionStatuses[status] {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: draft, active, deprecated, archived")
		return
	}

	nameFilter := r.URL.Query().Get("name")
	page, pageSize := parsePagination(r)

	defs, total, err := h.service.List(r.Context(), user.TenantID, status, nameFilter, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", user.TenantID).Msg("failed to list workflow definitions")
		handleServiceError(w, err)
		return
	}

	// Convert model pointers to response DTOs.
	items := make([]dto.DefinitionResponse, len(defs))
	for i, d := range defs {
		items[i] = dto.DefinitionToResponse(d)
	}

	writeJSON(w, http.StatusOK, dto.ListDefinitionsResponse{
		Definitions: items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	})
}

// GetByID handles GET /{id} — retrieves a single workflow definition.
func (h *DefinitionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "definition id is required")
		return
	}

	def, err := h.service.GetByID(r.Context(), user.TenantID, id)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", user.TenantID).Str("definition_id", id).Msg("failed to get workflow definition")
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.DefinitionToResponse(def))
}

// Update handles PUT /{id} — updates a workflow definition (creates a new version).
func (h *DefinitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "definition id is required")
		return
	}

	var req dto.UpdateDefinitionRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	def, err := h.service.Update(r.Context(), user.TenantID, id, user.ID, req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", id).
			Msg("failed to update workflow definition")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("definition_id", def.ID).
		Int("version", def.Version).
		Str("user_id", user.ID).
		Msg("workflow definition updated")

	writeJSON(w, http.StatusOK, dto.DefinitionToResponse(def))
}

// Delete handles DELETE /{id} — soft-deletes a workflow definition.
func (h *DefinitionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "definition id is required")
		return
	}

	if err := h.service.Delete(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", id).
			Msg("failed to delete workflow definition")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("definition_id", id).
		Str("user_id", user.ID).
		Msg("workflow definition deleted")

	writeJSON(w, http.StatusOK, map[string]string{"message": "definition deleted"})
}

// Activate handles POST /{id}/activate — activates a draft workflow definition.
func (h *DefinitionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "definition id is required")
		return
	}

	if err := h.service.Activate(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", id).
			Msg("failed to activate workflow definition")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("definition_id", id).
		Str("user_id", user.ID).
		Msg("workflow definition activated")

	writeJSON(w, http.StatusOK, map[string]string{"message": "definition activated"})
}

// ListVersions handles GET /{id}/versions — lists all versions of a workflow definition.
func (h *DefinitionHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "definition id is required")
		return
	}

	versions, err := h.service.ListVersions(r.Context(), user.TenantID, id)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", id).
			Msg("failed to list workflow definition versions")
		handleServiceError(w, err)
		return
	}

	items := make([]dto.DefinitionResponse, len(versions))
	for i, v := range versions {
		items[i] = dto.DefinitionToResponse(v)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"versions": items,
	})
}
