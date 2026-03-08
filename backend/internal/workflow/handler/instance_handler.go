package handler

import (
	"context"
	"net/http"
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

// instanceReader defines read operations for workflow instances.
type instanceReader interface {
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowInstance, error)
	List(ctx context.Context, tenantID, status, definitionID, startedBy string, dateFrom, dateTo *time.Time, limit, offset int) ([]*model.WorkflowInstance, int, error)
}

// InstanceHandler handles HTTP requests for workflow instance operations.
type InstanceHandler struct {
	engine       engineService
	instanceRepo instanceReader
	logger       zerolog.Logger
}

// NewInstanceHandler creates a new InstanceHandler with the given engine, instance reader, and logger.
func NewInstanceHandler(engine engineService, instanceRepo instanceReader, logger zerolog.Logger) *InstanceHandler {
	return &InstanceHandler{
		engine:       engine,
		instanceRepo: instanceRepo,
		logger:       logger.With().Str("handler", "workflow_instance").Logger(),
	}
}

// Routes returns a chi.Router with all instance routes mounted.
func (h *InstanceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Start)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Post("/{id}/cancel", h.Cancel)
	r.Post("/{id}/retry", h.Retry)
	r.Post("/{id}/suspend", h.Suspend)
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

	instances, total, err := h.instanceRepo.List(
		r.Context(), user.TenantID, status, definitionID, startedBy,
		dateFrom, dateTo, pageSize, offset,
	)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", user.TenantID).Msg("failed to list workflow instances")
		handleServiceError(w, err)
		return
	}

	items := make([]dto.InstanceResponse, len(instances))
	for i, inst := range instances {
		items[i] = dto.InstanceToResponse(inst)
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

	writeJSON(w, http.StatusOK, dto.InstanceToResponse(instance))
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

	items := make([]dto.StepExecutionResponse, len(executions))
	for i, se := range executions {
		items[i] = dto.StepExecutionToResponse(se)
	}

	writeJSON(w, http.StatusOK, dto.InstanceHistoryResponse{
		InstanceID:     id,
		StepExecutions: items,
	})
}
