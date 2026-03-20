package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/model"
)

// templateService defines operations for workflow template management.
type templateService interface {
	ListTemplates(ctx context.Context, category string) ([]*model.WorkflowTemplate, error)
	GetTemplate(ctx context.Context, id string) (*model.WorkflowTemplate, error)
	InstantiateTemplate(ctx context.Context, tenantID, userID, templateID, name, description string) (*model.WorkflowDefinition, error)
}

// TemplateHandler handles HTTP requests for workflow template operations.
type TemplateHandler struct {
	service templateService
	logger  zerolog.Logger
}

// NewTemplateHandler creates a new TemplateHandler with the given service and logger.
func NewTemplateHandler(service templateService, logger zerolog.Logger) *TemplateHandler {
	return &TemplateHandler{
		service: service,
		logger:  logger.With().Str("handler", "workflow_template").Logger(),
	}
}

// Routes returns a chi.Router with all template routes mounted.
func (h *TemplateHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListTemplates)
	r.Get("/{id}", h.GetTemplate)
	r.Post("/{id}/instantiate", h.InstantiateTemplate)

	return r
}

// ListTemplates handles GET / — lists available workflow templates, optionally filtered by category.
func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	category := r.URL.Query().Get("category")

	templates, err := h.service.ListTemplates(r.Context(), category)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("category", category).
			Msg("failed to list workflow templates")
		handleServiceError(w, err)
		return
	}

	items := make([]templateResponse, len(templates))
	for i, t := range templates {
		items[i] = toTemplateResponse(t)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"templates": items,
		"total":     len(items),
	})
}

// GetTemplate handles GET /{id} — retrieves a single workflow template by ID.
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "template id is required")
		return
	}

	tmpl, err := h.service.GetTemplate(r.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).
			Str("template_id", id).
			Msg("failed to get workflow template")
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toTemplateResponse(tmpl))
}

// InstantiateTemplate handles POST /{id}/instantiate — creates a new workflow definition from a template.
func (h *TemplateHandler) InstantiateTemplate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "template id is required")
		return
	}

	// Parse optional name/description overrides from request body.
	var overrides struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&overrides)
	}

	def, err := h.service.InstantiateTemplate(r.Context(), user.TenantID, user.ID, id, overrides.Name, overrides.Description)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("template_id", id).
			Str("user_id", user.ID).
			Msg("failed to instantiate workflow template")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("template_id", id).
		Str("definition_id", def.ID).
		Str("user_id", user.ID).
		Msg("workflow template instantiated")

	writeJSON(w, http.StatusCreated, dto.DefinitionToResponse(def))
}

// templateResponse is the API response shape for a workflow template.
type templateResponse struct {
	ID              string                       `json:"id"`
	Name            string                       `json:"name"`
	Description     string                       `json:"description"`
	Category        string                       `json:"category"`
	Icon            string                       `json:"icon"`
	PreviewImageURL *string                      `json:"preview_image_url"`
	Steps           []model.StepDefinition       `json:"steps"`
	Variables       map[string]model.VariableDef `json:"variables"`
	Tags            []string                     `json:"tags"`
	UsageCount      int                          `json:"usage_count"`
	CreatedAt       string                       `json:"created_at"`
}

// toTemplateResponse converts a WorkflowTemplate model to its API response form.
func toTemplateResponse(t *model.WorkflowTemplate) templateResponse {
	resp := templateResponse{
		ID:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		Category:        t.Category,
		Icon:            t.Icon,
		PreviewImageURL: t.PreviewImageURL,
		Tags:            t.Tags,
		UsageCount:      t.UsageCount,
		CreatedAt:       t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	// Parse definition JSON to extract steps and variables for the response.
	if len(t.DefinitionJSON) > 0 {
		var content model.TemplateDefinitionContent
		if err := json.Unmarshal(t.DefinitionJSON, &content); err == nil {
			resp.Steps = content.Steps
			resp.Variables = content.Variables
		}
	}

	// Ensure non-nil slices/maps for JSON serialization.
	if resp.Steps == nil {
		resp.Steps = []model.StepDefinition{}
	}
	if resp.Variables == nil {
		resp.Variables = make(map[string]model.VariableDef)
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}

	return resp
}
