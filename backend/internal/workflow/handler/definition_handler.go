package handler

import (
	"context"
	"net/http"
	"strings"

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
	List(ctx context.Context, tenantID, status, nameFilter, category, sortBy, sortOrder string, page, pageSize int) ([]*model.WorkflowDefinition, int, error)
	Update(ctx context.Context, tenantID, id, userID string, req dto.UpdateDefinitionRequest) (*model.WorkflowDefinition, error)
	Activate(ctx context.Context, tenantID, id string) error
	Archive(ctx context.Context, tenantID, id string) error
	Clone(ctx context.Context, tenantID, id, userID string) (*model.WorkflowDefinition, error)
	Delete(ctx context.Context, tenantID, id string) error
	ListVersions(ctx context.Context, tenantID, id string) ([]*model.WorkflowDefinition, error)
}

// definitionTemplateService is an optional service for creating definitions from templates.
type definitionTemplateService interface {
	InstantiateTemplate(ctx context.Context, tenantID, userID, templateID, name, description string) (*model.WorkflowDefinition, error)
}

// DefinitionHandler handles HTTP requests for workflow definition CRUD operations.
type DefinitionHandler struct {
	service    definitionService
	tmplSvc    definitionTemplateService
	logger     zerolog.Logger
}

// NewDefinitionHandler creates a new DefinitionHandler with the given service and logger.
func NewDefinitionHandler(service definitionService, logger zerolog.Logger) *DefinitionHandler {
	return &DefinitionHandler{
		service: service,
		logger:  logger.With().Str("handler", "workflow_definition").Logger(),
	}
}

// SetTemplateService sets the optional template service for from-template creation.
func (h *DefinitionHandler) SetTemplateService(svc definitionTemplateService) {
	h.tmplSvc = svc
}

// Routes returns a chi.Router with all definition routes mounted.
func (h *DefinitionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/from-template", h.CreateFromTemplate)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/activate", h.Activate)
	r.Post("/{id}/publish", h.Activate)   // alias: frontend uses "publish"
	r.Post("/{id}/archive", h.Archive)
	r.Post("/{id}/clone", h.Clone)
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
	if status != "" {
		// Validate each comma-separated status value.
		for _, s := range strings.Split(status, ",") {
			s = strings.TrimSpace(s)
			if s != "" && !model.ValidDefinitionStatuses[s] {
				writeError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: draft, active, deprecated, archived")
				return
			}
		}
	}

	nameFilter := r.URL.Query().Get("name")
	if nameFilter == "" {
		nameFilter = r.URL.Query().Get("search")
	}
	category := r.URL.Query().Get("category")
	if category != "" {
		// Validate each comma-separated category value.
		for _, c := range strings.Split(category, ",") {
			c = strings.TrimSpace(c)
			if c != "" && !model.ValidCategories[c] {
				writeError(w, http.StatusBadRequest, "INVALID_CATEGORY",
					"category must be one of: approval, onboarding, review, escalation, notification, data_pipeline, compliance, custom")
				return
			}
		}
	}
	sortBy := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")
	page, pageSize := parsePagination(r)

	defs, total, err := h.service.List(r.Context(), user.TenantID, status, nameFilter, category, sortBy, sortOrder, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", user.TenantID).Msg("failed to list workflow definitions")
		handleServiceError(w, err)
		return
	}

	items := make([]dto.DefinitionResponse, 0, len(defs))
	for _, d := range defs {
		items = append(items, dto.DefinitionToResponse(d))
	}

	writeJSON(w, http.StatusOK, dto.ListDefinitionsResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(page, pageSize, total),
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

	// Fetch before deleting so we can return the definition in the response.
	def, err := h.service.GetByID(r.Context(), user.TenantID, id)
	if err != nil {
		handleServiceError(w, err)
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

	writeJSON(w, http.StatusOK, dto.DefinitionToResponse(def))
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

	// Return the updated definition so the frontend can update its cache.
	updated, err := h.service.GetByID(r.Context(), user.TenantID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.DefinitionToResponse(updated))
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

// Archive handles POST /{id}/archive — archives an active workflow definition.
func (h *DefinitionHandler) Archive(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.Archive(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", id).
			Msg("failed to archive workflow definition")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("definition_id", id).
		Str("user_id", user.ID).
		Msg("workflow definition archived")

	// Return the updated definition so the frontend can update its cache.
	updated, err := h.service.GetByID(r.Context(), user.TenantID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.DefinitionToResponse(updated))
}

// Clone handles POST /{id}/clone — creates a copy of a workflow definition in draft status.
func (h *DefinitionHandler) Clone(w http.ResponseWriter, r *http.Request) {
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

	def, err := h.service.Clone(r.Context(), user.TenantID, id, user.ID)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", id).
			Msg("failed to clone workflow definition")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("definition_id", def.ID).
		Str("user_id", user.ID).
		Msg("workflow definition cloned")

	writeJSON(w, http.StatusCreated, dto.DefinitionToResponse(def))
}

// CreateFromTemplate handles POST /from-template — creates a new definition from a template.
// Expects JSON body: { "template_id": "...", "name": "...", "description": "..." }
func (h *DefinitionHandler) CreateFromTemplate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	if h.tmplSvc == nil {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "template service not configured")
		return
	}

	var req struct {
		TemplateID  string `json:"template_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if req.TemplateID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "template_id is required")
		return
	}

	def, err := h.tmplSvc.InstantiateTemplate(r.Context(), user.TenantID, user.ID, req.TemplateID, req.Name, req.Description)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("template_id", req.TemplateID).
			Msg("failed to create definition from template")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("template_id", req.TemplateID).
		Str("definition_id", def.ID).
		Str("user_id", user.ID).
		Msg("workflow definition created from template")

	writeJSON(w, http.StatusCreated, dto.DefinitionToResponse(def))
}
