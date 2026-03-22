package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/model"
)

// engineService defines workflow engine operations for instance lifecycle management.
type engineService interface {
	StartInstance(ctx context.Context, tenantID, userID string, req dto.StartInstanceRequest) (*model.WorkflowInstance, error)
	CancelInstance(ctx context.Context, tenantID, instanceID string) error
	RetryInstance(ctx context.Context, tenantID, instanceID string) error
	SuspendInstance(ctx context.Context, tenantID, instanceID string) error
	ResumeInstance(ctx context.Context, tenantID, instanceID string) error
	GetHistory(ctx context.Context, tenantID, instanceID string) ([]*model.StepExecution, error)
}

// instanceReader defines read and variable-update operations for workflow instances.
type instanceReader interface {
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowInstance, error)
	List(ctx context.Context, tenantID, status, definitionID, startedBy string, dateFrom, dateTo *time.Time, sortBy, sortOrder string, limit, offset int) ([]*model.WorkflowInstance, int, error)
	GetStepExecutions(ctx context.Context, instanceID string) ([]*model.StepExecution, error)
	UpdateVariables(ctx context.Context, tenantID, instanceID string, variables map[string]interface{}) error
}

// definitionReader defines read operations for workflow definitions.
type definitionReader interface {
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error)
}

// InstanceHandler handles HTTP requests for workflow instance operations.
type InstanceHandler struct {
	engine       engineService
	instanceRepo instanceReader
	defReader    definitionReader
	userLookup   UserNameLookup
	logger       zerolog.Logger
}

// NewInstanceHandler creates a new InstanceHandler with the given engine, instance reader, definition reader, and logger.
func NewInstanceHandler(engine engineService, instanceRepo instanceReader, defReader definitionReader, logger zerolog.Logger) *InstanceHandler {
	return &InstanceHandler{
		engine:       engine,
		instanceRepo: instanceRepo,
		defReader:    defReader,
		logger:       logger.With().Str("handler", "workflow_instance").Logger(),
	}
}

// SetUserLookup sets the optional user name resolver for enriching display names.
func (h *InstanceHandler) SetUserLookup(lookup UserNameLookup) {
	h.userLookup = lookup
}

// Routes returns a chi.Router with all instance routes mounted.
func (h *InstanceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Start)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/cancel", h.Cancel)
	r.Post("/{id}/retry", h.Retry)
	r.Post("/{id}/suspend", h.Suspend)
	r.Post("/{id}/pause", h.Suspend) // alias: frontend uses "pause"
	r.Post("/{id}/resume", h.Resume)
	r.Get("/{id}/history", h.GetHistory)

	return r
}

// Start handles POST / — starts a new workflow instance.
func (h *InstanceHandler) Start(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req dto.StartInstanceRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.DefinitionID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "definition_id is required")
		return
	}

	instance, err := h.engine.StartInstance(r.Context(), user.TenantID, user.ID, req)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("definition_id", req.DefinitionID).
			Msg("failed to start workflow instance")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("instance_id", instance.ID).
		Str("definition_id", req.DefinitionID).
		Str("user_id", user.ID).
		Msg("workflow instance started")

	writeJSON(w, http.StatusCreated, dto.InstanceToResponse(instance))
}

// List handles GET / — lists workflow instances with filtering and pagination.
func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	q := r.URL.Query()

	status := q.Get("status")
	if status != "" && !model.ValidInstanceStatuses[status] {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: running, completed, failed, cancelled, suspended")
		return
	}

	definitionID := q.Get("definition_id")
	startedBy := q.Get("started_by")

	var dateFrom, dateTo *time.Time
	if df := q.Get("date_from"); df != "" {
		t, err := time.Parse(time.RFC3339, df)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE", "date_from must be in RFC3339 format")
			return
		}
		dateFrom = &t
	}
	if dt := q.Get("date_to"); dt != "" {
		t, err := time.Parse(time.RFC3339, dt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE", "date_to must be in RFC3339 format")
			return
		}
		dateTo = &t
	}

	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize
	sortBy := q.Get("sort")
	sortOrder := q.Get("order")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = ""
	}

	instances, total, err := h.instanceRepo.List(
		r.Context(), user.TenantID, status, definitionID, startedBy,
		dateFrom, dateTo, sortBy, sortOrder, pageSize, offset,
	)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", user.TenantID).Msg("failed to list workflow instances")
		handleServiceError(w, err)
		return
	}

	// Cache definition lookups to avoid repeated DB calls within the same page.
	defCache := make(map[string]*model.WorkflowDefinition)

	items := make([]dto.InstanceResponse, len(instances))
	for i, inst := range instances {
		items[i] = dto.InstanceToResponse(inst)
		// Best-effort enrichment — don't fail the list if a definition lookup fails.
		if h.defReader != nil {
			def, ok := defCache[inst.DefinitionID]
			if !ok {
				def, _ = h.defReader.GetByID(r.Context(), user.TenantID, inst.DefinitionID)
				defCache[inst.DefinitionID] = def // may be nil
			}
			if def != nil {
				items[i].DefinitionName = def.Name
				items[i].TotalSteps = len(def.Steps)
				if inst.CurrentStepID != nil {
					for _, s := range def.Steps {
						if s.ID == *inst.CurrentStepID {
							items[i].CurrentStepName = &s.Name
							break
						}
					}
				}
			}
			// Compute completed steps from step executions.
			if execs, err := h.instanceRepo.GetStepExecutions(r.Context(), inst.ID); err == nil {
				completed := 0
				for _, e := range execs {
					if e.Status == "completed" {
						completed++
					}
				}
				items[i].CompletedSteps = completed
			}
		}
	}

	writeJSON(w, http.StatusOK, dto.ListInstancesResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(page, pageSize, total),
	})
}

// GetByID handles GET /{id} — retrieves a single workflow instance.
func (h *InstanceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	instance, err := h.instanceRepo.GetByID(r.Context(), user.TenantID, id)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to get workflow instance")
		handleServiceError(w, err)
		return
	}

	resp := dto.InstanceToResponse(instance)
	h.enrichInstanceResponse(r.Context(), user.TenantID, instance, &resp)

	writeJSON(w, http.StatusOK, resp)
}

// Update handles PUT /{id} — updates mutable fields (variables) on a workflow instance.
func (h *InstanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	var req struct {
		Variables      map[string]interface{} `json:"variables"`
		InputVariables map[string]interface{} `json:"input_variables"`
	}
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Merge all provided variable sources into a single update map.
	merged := make(map[string]interface{})
	for k, v := range req.Variables {
		merged[k] = v
	}
	for k, v := range req.InputVariables {
		merged[k] = v
	}

	if len(merged) == 0 {
		// Nothing to update — fetch and return current state.
		instance, err := h.instanceRepo.GetByID(r.Context(), user.TenantID, id)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		resp := dto.InstanceToResponse(instance)
		h.enrichInstanceResponse(r.Context(), user.TenantID, instance, &resp)
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Persist the merged variables to the database.
	if err := h.instanceRepo.UpdateVariables(r.Context(), user.TenantID, id, merged); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to update workflow instance variables")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("instance_id", id).
		Str("user_id", user.ID).
		Int("variable_count", len(merged)).
		Msg("workflow instance variables updated")

	// Re-fetch to return the persisted state.
	instance, err := h.instanceRepo.GetByID(r.Context(), user.TenantID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	resp := dto.InstanceToResponse(instance)
	h.enrichInstanceResponse(r.Context(), user.TenantID, instance, &resp)

	writeJSON(w, http.StatusOK, resp)
}

// Delete handles DELETE /{id} — cancels a running instance (soft delete for audit trail).
func (h *InstanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	// Cancel the instance if it is still running — workflow instances are not hard-deleted.
	if err := h.engine.CancelInstance(r.Context(), user.TenantID, id); err != nil {
		// If already in a terminal state, treat as success.
		if !isTerminalStatusError(err) {
			h.logger.Error().Err(err).
				Str("tenant_id", user.TenantID).
				Str("instance_id", id).
				Msg("failed to delete workflow instance")
			handleServiceError(w, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// isTerminalStatusError returns true if the error indicates the instance is
// already in a terminal state (completed, cancelled, failed).
func isTerminalStatusError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "already completed") ||
		strings.Contains(msg, "already cancelled") ||
		strings.Contains(msg, "already failed") ||
		strings.Contains(msg, "terminal state")
}

// Cancel handles POST /{id}/cancel — cancels a running workflow instance.
func (h *InstanceHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	if err := h.engine.CancelInstance(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to cancel workflow instance")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("instance_id", id).
		Str("user_id", user.ID).
		Msg("workflow instance cancelled")

	writeJSON(w, http.StatusOK, map[string]string{"message": "instance cancelled"})
}

// Retry handles POST /{id}/retry — retries a failed workflow instance.
func (h *InstanceHandler) Retry(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	if err := h.engine.RetryInstance(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to retry workflow instance")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("instance_id", id).
		Str("user_id", user.ID).
		Msg("workflow instance retried")

	writeJSON(w, http.StatusOK, map[string]string{"message": "instance retry initiated"})
}

// Suspend handles POST /{id}/suspend — suspends a running workflow instance.
func (h *InstanceHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	if err := h.engine.SuspendInstance(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to suspend workflow instance")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("instance_id", id).
		Str("user_id", user.ID).
		Msg("workflow instance suspended")

	writeJSON(w, http.StatusOK, map[string]string{"message": "instance suspended"})
}

// Resume handles POST /{id}/resume — resumes a suspended workflow instance.
func (h *InstanceHandler) Resume(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	if err := h.engine.ResumeInstance(r.Context(), user.TenantID, id); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to resume workflow instance")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("instance_id", id).
		Str("user_id", user.ID).
		Msg("workflow instance resumed")

	writeJSON(w, http.StatusOK, map[string]string{"message": "instance resumed"})
}

// GetHistory handles GET /{id}/history — returns the step execution history for an instance.
func (h *InstanceHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "instance id is required")
		return
	}

	executions, err := h.engine.GetHistory(r.Context(), user.TenantID, id)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("instance_id", id).
			Msg("failed to get workflow instance history")
		handleServiceError(w, err)
		return
	}

	// Build a step-ID → name map from the definition for enrichment.
	stepNames := make(map[string]string)
	if h.defReader != nil {
		inst, err := h.instanceRepo.GetByID(r.Context(), user.TenantID, id)
		if err == nil {
			if def, err := h.defReader.GetByID(r.Context(), user.TenantID, inst.DefinitionID); err == nil {
				for _, s := range def.Steps {
					stepNames[s.ID] = s.Name
				}
			}
		}
	}

	items := make([]dto.StepExecutionResponse, len(executions))
	for i, se := range executions {
		items[i] = dto.StepExecutionToResponse(se)
		if name, ok := stepNames[se.StepID]; ok {
			items[i].StepName = name
		}
	}

	writeJSON(w, http.StatusOK, dto.InstanceHistoryResponse{
		InstanceID:     id,
		StepExecutions: items,
	})
}

// enrichInstanceResponse adds definition_name, current_step_name, completed_steps,
// and total_steps to a single InstanceResponse. Best-effort — errors are logged but not fatal.
func (h *InstanceHandler) enrichInstanceResponse(ctx context.Context, tenantID string, inst *model.WorkflowInstance, resp *dto.InstanceResponse) {
	if h.defReader == nil {
		return
	}

	def, err := h.defReader.GetByID(ctx, tenantID, inst.DefinitionID)
	if err != nil {
		h.logger.Warn().Err(err).Str("definition_id", inst.DefinitionID).Msg("failed to enrich instance with definition name")
		return
	}

	resp.DefinitionName = def.Name
	resp.TotalSteps = len(def.Steps)
	resp.DefinitionSteps = def.Steps

	if inst.CurrentStepID != nil {
		for _, s := range def.Steps {
			if s.ID == *inst.CurrentStepID {
				resp.CurrentStepName = &s.Name
				break
			}
		}
	}

	// Count completed step executions.
	if execs, err := h.instanceRepo.GetStepExecutions(ctx, inst.ID); err == nil {
		completed := 0
		for _, e := range execs {
			if e.Status == "completed" {
				completed++
			}
		}
		resp.CompletedSteps = completed
	}

	// Resolve started_by to a display name.
	resp.StartedByName = resolveUserName(ctx, h.userLookup, resp.StartedBy)
}
