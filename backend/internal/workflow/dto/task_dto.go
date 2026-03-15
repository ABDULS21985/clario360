package dto

import (
	"time"

	"github.com/clario360/platform/internal/workflow/model"
)

// ---------- Requests ----------

// ListTasksRequest holds query parameters for listing human tasks.
type ListTasksRequest struct {
	TenantID string   `json:"tenant_id"`
	UserID   string   `json:"user_id"`
	Roles    []string `json:"roles"`
	Status   string   `json:"status" validate:"omitempty,oneof=pending claimed completed rejected escalated cancelled"`
	Page     int      `json:"page" validate:"min=1"`
	PageSize int      `json:"page_size" validate:"min=1,max=100"`
}

// ClaimTaskRequest is the payload for claiming a task.
type ClaimTaskRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

// CompleteTaskRequest is the payload for completing a human task.
type CompleteTaskRequest struct {
	FormData map[string]interface{} `json:"form_data" validate:"required"`
}

// DelegateTaskRequest is the payload for delegating a task to another user.
type DelegateTaskRequest struct {
	DelegateTo string `json:"delegate_to" validate:"required"`
}

// RejectTaskRequest is the payload for rejecting a human task.
type RejectTaskRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=2000"`
}

// ---------- Responses ----------

// TaskResponse wraps a HumanTask for API responses.
type TaskResponse struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	InstanceID     string                 `json:"instance_id"`
	StepID         string                 `json:"step_id"`
	StepExecID     string                 `json:"step_exec_id"`
	DefinitionName string                 `json:"definition_name,omitempty"`
	WorkflowName   string                 `json:"workflow_name,omitempty"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Status         string                 `json:"status"`
	AssigneeID     *string                `json:"assignee_id,omitempty"`
	AssigneeRole   *string                `json:"assignee_role,omitempty"`
	ClaimedBy      *string                `json:"claimed_by,omitempty"`
	ClaimedAt      *time.Time             `json:"claimed_at,omitempty"`
	FormSchema     []model.FormField      `json:"form_schema"`
	FormData       map[string]interface{} `json:"form_data,omitempty"`
	SLADeadline    *time.Time             `json:"sla_deadline,omitempty"`
	SLABreached    bool                   `json:"sla_breached"`
	EscalatedTo    *string                `json:"escalated_to,omitempty"`
	EscalationRole *string                `json:"escalation_role,omitempty"`
	DelegatedBy    *string                `json:"delegated_by,omitempty"`
	DelegatedAt    *time.Time             `json:"delegated_at,omitempty"`
	Priority       int                    `json:"priority"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// TaskPaginationMeta holds pagination metadata for task list responses.
type TaskPaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ListTasksResponse is the paginated response for listing tasks.
type ListTasksResponse struct {
	Data []TaskResponse     `json:"data"`
	Meta TaskPaginationMeta `json:"meta"`
}

// TaskCountResponse provides counts of tasks bucketed by status for a user dashboard.
type TaskCountResponse struct {
	Pending     int `json:"pending"`
	ClaimedByMe int `json:"claimed_by_me"`
	Completed   int `json:"completed"`
	Overdue     int `json:"overdue"`
	Escalated   int `json:"escalated"`
}

// ---------- Converters ----------

// TaskToResponse converts a HumanTask model to its API response form.
func TaskToResponse(t *model.HumanTask) TaskResponse {
	schema := t.FormSchema
	if schema == nil {
		schema = []model.FormField{}
	}
	metadata := t.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return TaskResponse{
		ID:             t.ID,
		TenantID:       t.TenantID,
		InstanceID:     t.InstanceID,
		StepID:         t.StepID,
		StepExecID:     t.StepExecID,
		Name:           t.Name,
		Description:    t.Description,
		Status:         t.Status,
		AssigneeID:     t.AssigneeID,
		AssigneeRole:   t.AssigneeRole,
		ClaimedBy:      t.ClaimedBy,
		ClaimedAt:      t.ClaimedAt,
		FormSchema:     schema,
		FormData:       t.FormData,
		SLADeadline:    t.SLADeadline,
		SLABreached:    t.SLABreached,
		EscalatedTo:    t.EscalatedTo,
		EscalationRole: t.EscalationRole,
		DelegatedBy:    t.DelegatedBy,
		DelegatedAt:    t.DelegatedAt,
		Priority:       t.Priority,
		Metadata:       metadata,
		CompletedAt:    t.CompletedAt,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

// TasksToResponse converts a slice of HumanTask models to their response form.
func TasksToResponse(tasks []model.HumanTask) []TaskResponse {
	resp := make([]TaskResponse, len(tasks))
	for i := range tasks {
		resp[i] = TaskToResponse(&tasks[i])
	}
	return resp
}

// DefaultListTasksRequest returns a ListTasksRequest with default pagination values.
func DefaultListTasksRequest() ListTasksRequest {
	return ListTasksRequest{
		Page:     1,
		PageSize: 20,
	}
}

// Offset computes the SQL OFFSET from Page and PageSize.
func (r *ListTasksRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}
