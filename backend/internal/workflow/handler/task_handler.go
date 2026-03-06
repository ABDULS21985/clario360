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

// taskService defines operations available for human task management.
type taskService interface {
	ListTasks(ctx context.Context, tenantID, userID string, roles []string, status string, page, pageSize int) ([]*model.HumanTask, int, error)
	GetTask(ctx context.Context, tenantID, taskID string) (*model.HumanTask, error)
	ClaimTask(ctx context.Context, tenantID, taskID, userID string) error
	CompleteTask(ctx context.Context, tenantID, taskID, userID string, formData map[string]interface{}) error
	DelegateTask(ctx context.Context, tenantID, taskID, fromUserID, toUserID string) error
	RejectTask(ctx context.Context, tenantID, taskID, userID, reason string) error
	CountTasks(ctx context.Context, tenantID, userID string, roles []string) (map[string]int, error)
}

// TaskHandler handles HTTP requests for human task operations within workflows.
type TaskHandler struct {
	service taskService
	logger  zerolog.Logger
}

// NewTaskHandler creates a new TaskHandler with the given service and logger.
func NewTaskHandler(service taskService, logger zerolog.Logger) *TaskHandler {
	return &TaskHandler{
		service: service,
		logger:  logger.With().Str("handler", "workflow_task").Logger(),
	}
}

// Routes returns a chi.Router with all task routes mounted.
func (h *TaskHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListTasks)
	r.Get("/count", h.CountTasks)
	r.Get("/{id}", h.GetTask)
	r.Post("/{id}/claim", h.ClaimTask)
	r.Post("/{id}/complete", h.CompleteTask)
	r.Post("/{id}/delegate", h.DelegateTask)
	r.Post("/{id}/reject", h.RejectTask)

	return r
}

// ListTasks handles GET / — lists human tasks assigned to or available for the current user.
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	status := r.URL.Query().Get("status")
	if status != "" && !model.ValidTaskStatuses[status] {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: pending, claimed, completed, rejected, escalated, cancelled")
		return
	}

	page, pageSize := parsePagination(r)

	tasks, total, err := h.service.ListTasks(r.Context(), user.TenantID, user.ID, user.Roles, status, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("user_id", user.ID).
			Msg("failed to list tasks")
		handleServiceError(w, err)
		return
	}

	items := make([]dto.TaskResponse, len(tasks))
	for i, t := range tasks {
		items[i] = dto.TaskToResponse(t)
	}

	writeJSON(w, http.StatusOK, dto.ListTasksResponse{
		Tasks:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CountTasks handles GET /count — returns task counts grouped by status for the current user.
func (h *TaskHandler) CountTasks(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	counts, err := h.service.CountTasks(r.Context(), user.TenantID, user.ID, user.Roles)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("user_id", user.ID).
			Msg("failed to count tasks")
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TaskCountResponse{
		Pending:     counts["pending"],
		ClaimedByMe: counts["claimed_by_me"],
		Overdue:     counts["overdue"],
		Escalated:   counts["escalated"],
	})
}

// GetTask handles GET /{id} — retrieves a single human task by ID.
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "task id is required")
		return
	}

	task, err := h.service.GetTask(r.Context(), user.TenantID, id)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Msg("failed to get task")
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TaskToResponse(task))
}

// ClaimTask handles POST /{id}/claim — claims an available task for the current user.
func (h *TaskHandler) ClaimTask(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "task id is required")
		return
	}

	if err := h.service.ClaimTask(r.Context(), user.TenantID, id, user.ID); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Str("user_id", user.ID).
			Msg("failed to claim task")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("task_id", id).
		Str("user_id", user.ID).
		Msg("task claimed")

	writeJSON(w, http.StatusOK, map[string]string{"message": "task claimed"})
}

// CompleteTask handles POST /{id}/complete — completes a claimed task with form data.
func (h *TaskHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "task id is required")
		return
	}

	var req dto.CompleteTaskRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.FormData == nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "form_data is required")
		return
	}

	if err := h.service.CompleteTask(r.Context(), user.TenantID, id, user.ID, req.FormData); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Str("user_id", user.ID).
			Msg("failed to complete task")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("task_id", id).
		Str("user_id", user.ID).
		Msg("task completed")

	writeJSON(w, http.StatusOK, map[string]string{"message": "task completed"})
}

// DelegateTask handles POST /{id}/delegate — delegates a task to another user.
func (h *TaskHandler) DelegateTask(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "task id is required")
		return
	}

	var req dto.DelegateTaskRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.DelegateTo == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "delegate_to is required")
		return
	}

	if err := h.service.DelegateTask(r.Context(), user.TenantID, id, user.ID, req.DelegateTo); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Str("from_user", user.ID).
			Str("to_user", req.DelegateTo).
			Msg("failed to delegate task")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("task_id", id).
		Str("from_user", user.ID).
		Str("to_user", req.DelegateTo).
		Msg("task delegated")

	writeJSON(w, http.StatusOK, map[string]string{"message": "task delegated"})
}

// RejectTask handles POST /{id}/reject — rejects a task with a reason.
func (h *TaskHandler) RejectTask(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	id := urlParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "task id is required")
		return
	}

	var req dto.RejectTaskRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "reason is required")
		return
	}

	if err := h.service.RejectTask(r.Context(), user.TenantID, id, user.ID, req.Reason); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Str("user_id", user.ID).
			Msg("failed to reject task")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("task_id", id).
		Str("user_id", user.ID).
		Msg("task rejected")

	writeJSON(w, http.StatusOK, map[string]string{"message": "task rejected"})
}
