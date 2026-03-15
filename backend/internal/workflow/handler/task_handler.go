package handler

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/model"
)

// taskService defines operations available for human task management.
type taskService interface {
	ListTasks(ctx context.Context, tenantID, userID string, roles []string, statuses []string, page, pageSize int) ([]*model.HumanTask, int, error)
	GetTask(ctx context.Context, tenantID, taskID string) (*model.HumanTask, error)
	ClaimTask(ctx context.Context, tenantID, taskID, userID string) error
	CompleteTask(ctx context.Context, tenantID, taskID, userID string, formData map[string]interface{}) error
	DelegateTask(ctx context.Context, tenantID, taskID, fromUserID, toUserID string) error
	RejectTask(ctx context.Context, tenantID, taskID, userID, reason string) error
	CountTasks(ctx context.Context, tenantID, userID string, roles []string) (map[string]int, error)
	UpdateMetadata(ctx context.Context, tenantID, taskID string, metadata map[string]interface{}) error
}

// taskInstanceReader reads workflow instances for task enrichment.
type taskInstanceReader interface {
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowInstance, error)
}

// taskDefinitionReader reads workflow definitions for task enrichment.
type taskDefinitionReader interface {
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error)
}

// TaskHandler handles HTTP requests for human task operations within workflows.
type TaskHandler struct {
	service   taskService
	instRepo  taskInstanceReader
	defReader taskDefinitionReader
	logger    zerolog.Logger
}

// NewTaskHandler creates a new TaskHandler with the given service and logger.
func NewTaskHandler(service taskService, instRepo taskInstanceReader, defReader taskDefinitionReader, logger zerolog.Logger) *TaskHandler {
	return &TaskHandler{
		service:   service,
		instRepo:  instRepo,
		defReader: defReader,
		logger:    logger.With().Str("handler", "workflow_task").Logger(),
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
	r.Post("/{id}/assign", h.AssignTask) // alias: frontend uses "assign"
	r.Post("/{id}/reject", h.RejectTask)
	r.Post("/{id}/comment", h.AddComment)

	return r
}

// ListTasks handles GET / — lists human tasks assigned to or available for the current user.
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	// Accept comma-separated statuses (e.g. "pending,claimed").
	var statuses []string
	if raw := r.URL.Query().Get("status"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if !model.ValidTaskStatuses[s] {
				writeError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: pending, claimed, completed, rejected, escalated, cancelled")
				return
			}
			statuses = append(statuses, s)
		}
	}

	page, pageSize := parsePagination(r)

	tasks, total, err := h.service.ListTasks(r.Context(), user.TenantID, user.ID, user.Roles, statuses, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("user_id", user.ID).
			Msg("failed to list tasks")
		handleServiceError(w, err)
		return
	}

	items := make([]dto.TaskResponse, len(tasks))
	defCache := make(map[string]string) // definitionID -> name
	for i, t := range tasks {
		items[i] = dto.TaskToResponse(t)
		h.enrichTaskResponse(r.Context(), user.TenantID, t, &items[i], defCache)
	}

	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, dto.ListTasksResponse{
		Data: items,
		Meta: dto.TaskPaginationMeta{
			Page:       page,
			PerPage:    pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
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
		Completed:   counts["completed"],
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

	resp := dto.TaskToResponse(task)
	h.enrichTaskResponse(r.Context(), user.TenantID, task, &resp, nil)
	writeJSON(w, http.StatusOK, resp)
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

// AssignTask handles POST /{id}/assign — assigns a task to a specific user.
// Accepts { "user_id": "..." } and delegates to DelegateTask under the hood.
func (h *TaskHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	if err := h.service.DelegateTask(r.Context(), user.TenantID, id, user.ID, req.UserID); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Msg("failed to assign task")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("task_id", id).
		Str("to_user", req.UserID).
		Msg("task assigned")

	writeJSON(w, http.StatusOK, map[string]string{"message": "task assigned"})
}

// AddComment handles POST /{id}/comment — adds a comment to a task's metadata.
func (h *TaskHandler) AddComment(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Content string `json:"content"`
	}
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
		return
	}

	// Verify the task exists and belongs to this tenant.
	task, err := h.service.GetTask(r.Context(), user.TenantID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Build comment.
	comment := map[string]interface{}{
		"id":         generateCommentID(),
		"user_id":    user.ID,
		"user_name":  user.Email,
		"content":    req.Content,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Append to metadata.comments.
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	comments, _ := task.Metadata["comments"].([]interface{})
	comments = append(comments, comment)
	task.Metadata["comments"] = comments

	// Persist updated metadata to the database.
	if err := h.service.UpdateMetadata(r.Context(), user.TenantID, id, task.Metadata); err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", user.TenantID).
			Str("task_id", id).
			Msg("failed to persist task comment")
		handleServiceError(w, err)
		return
	}

	h.logger.Info().
		Str("tenant_id", user.TenantID).
		Str("task_id", id).
		Str("user_id", user.ID).
		Msg("comment added to task")

	writeJSON(w, http.StatusCreated, comment)
}

// generateCommentID returns a short random hex ID suitable for comments.
func generateCommentID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// enrichTaskResponse adds definition_name and workflow_name to a TaskResponse
// by looking up the instance → definition chain. Best-effort — errors are
// logged but not fatal. defCache is an optional map for batch enrichment.
func (h *TaskHandler) enrichTaskResponse(ctx context.Context, tenantID string, task *model.HumanTask, resp *dto.TaskResponse, defCache map[string]string) {
	if h.instRepo == nil || h.defReader == nil {
		return
	}

	inst, err := h.instRepo.GetByID(ctx, tenantID, task.InstanceID)
	if err != nil {
		return
	}

	// Use cache if available.
	if defCache != nil {
		if name, ok := defCache[inst.DefinitionID]; ok {
			resp.DefinitionName = name
			resp.WorkflowName = name
			return
		}
	}

	def, err := h.defReader.GetByID(ctx, tenantID, inst.DefinitionID)
	if err != nil {
		return
	}

	resp.DefinitionName = def.Name
	resp.WorkflowName = def.Name

	if defCache != nil {
		defCache[inst.DefinitionID] = def.Name
	}
}
