package dto

import (
	"encoding/json"
	"time"

	"github.com/clario360/platform/internal/workflow/model"
)

// ---------- Requests ----------

// StartInstanceRequest is the payload for starting a new workflow instance.
type StartInstanceRequest struct {
	DefinitionID   string                 `json:"definition_id" validate:"required"`
	InputVariables map[string]interface{} `json:"input_variables"`
	TriggerData    json.RawMessage        `json:"trigger_data,omitempty"`
}

// ListInstancesRequest holds query parameters for listing workflow instances.
type ListInstancesRequest struct {
	TenantID     string     `json:"tenant_id"`
	Status       string     `json:"status" validate:"omitempty,oneof=running completed failed cancelled suspended"`
	DefinitionID string     `json:"definition_id"`
	StartedBy    string     `json:"started_by"`
	DateFrom     *time.Time `json:"date_from"`
	DateTo       *time.Time `json:"date_to"`
	Page         int        `json:"page" validate:"min=1"`
	PageSize     int        `json:"page_size" validate:"min=1,max=100"`
}

// ---------- Responses ----------

// InstanceResponse wraps a WorkflowInstance for API responses.
type InstanceResponse struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	DefinitionID    string                 `json:"definition_id"`
	DefinitionVer   int                    `json:"definition_ver"`
	DefinitionName  string                 `json:"definition_name,omitempty"`
	Status          string                 `json:"status"`
	CurrentStepID   *string                `json:"current_step_id,omitempty"`
	CurrentStepName *string                `json:"current_step_name,omitempty"`
	CompletedSteps  int                    `json:"completed_steps"`
	TotalSteps      int                    `json:"total_steps"`
	Variables       map[string]interface{} `json:"variables"`
	StepOutputs     map[string]interface{} `json:"step_outputs"`
	TriggerData     json.RawMessage        `json:"trigger_data,omitempty"`
	ErrorMessage    *string                `json:"error_message,omitempty"`
	StartedBy       *string                `json:"started_by,omitempty"`
	StartedByName   *string                `json:"started_by_name,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DurationMs      *int64                 `json:"duration_ms,omitempty"`
	DefinitionSteps []model.StepDefinition `json:"definition_steps,omitempty"`
}

// ListInstancesResponse is the paginated response for listing instances.
type ListInstancesResponse struct {
	Data []InstanceResponse `json:"data"`
	Meta PaginationMeta     `json:"meta"`
}

// StepExecutionResponse wraps a StepExecution for API responses.
type StepExecutionResponse struct {
	ID           string          `json:"id"`
	InstanceID   string          `json:"instance_id"`
	StepID       string          `json:"step_id"`
	StepName     string          `json:"step_name,omitempty"`
	StepType     string          `json:"step_type"`
	Status       string          `json:"status"`
	InputData    json.RawMessage `json:"input_data,omitempty"`
	OutputData   json.RawMessage `json:"output_data,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	Attempt      int             `json:"attempt"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	DurationMs   *int64          `json:"duration_ms,omitempty"`
}

// InstanceHistoryResponse contains the ordered step executions for an instance.
type InstanceHistoryResponse struct {
	InstanceID     string                  `json:"instance_id"`
	StepExecutions []StepExecutionResponse `json:"step_executions"`
}

// ---------- Converters ----------

// InstanceToResponse converts a WorkflowInstance model to its API response form.
func InstanceToResponse(inst *model.WorkflowInstance) InstanceResponse {
	vars := inst.Variables
	if vars == nil {
		vars = make(map[string]interface{})
	}
	outputs := inst.StepOutputs
	if outputs == nil {
		outputs = make(map[string]interface{})
	}

	resp := InstanceResponse{
		ID:            inst.ID,
		TenantID:      inst.TenantID,
		DefinitionID:  inst.DefinitionID,
		DefinitionVer: inst.DefinitionVer,
		Status:        inst.Status,
		CurrentStepID: inst.CurrentStepID,
		Variables:     vars,
		StepOutputs:   outputs,
		TriggerData:   inst.TriggerData,
		ErrorMessage:  inst.ErrorMessage,
		StartedBy:     inst.StartedBy,
		StartedAt:     inst.StartedAt,
		CompletedAt:   inst.CompletedAt,
		UpdatedAt:     inst.UpdatedAt,
	}

	if inst.CompletedAt != nil {
		dur := inst.CompletedAt.Sub(inst.StartedAt).Milliseconds()
		resp.DurationMs = &dur
	}

	return resp
}

// InstancesToResponse converts a slice of instances to their response form.
func InstancesToResponse(instances []model.WorkflowInstance) []InstanceResponse {
	resp := make([]InstanceResponse, len(instances))
	for i := range instances {
		resp[i] = InstanceToResponse(&instances[i])
	}
	return resp
}

// StepExecutionToResponse converts a StepExecution model to its API response form.
func StepExecutionToResponse(se *model.StepExecution) StepExecutionResponse {
	resp := StepExecutionResponse{
		ID:           se.ID,
		InstanceID:   se.InstanceID,
		StepID:       se.StepID,
		StepType:     se.StepType,
		Status:       se.Status,
		InputData:    se.InputData,
		OutputData:   se.OutputData,
		ErrorMessage: se.ErrorMessage,
		Attempt:      se.Attempt,
		StartedAt:    se.StartedAt,
		CompletedAt:  se.CompletedAt,
		CreatedAt:    se.CreatedAt,
	}

	if se.StartedAt != nil && se.CompletedAt != nil {
		dur := se.CompletedAt.Sub(*se.StartedAt).Milliseconds()
		resp.DurationMs = &dur
	}

	return resp
}

// StepExecutionsToResponse converts a slice of step executions to their response form.
func StepExecutionsToResponse(execs []model.StepExecution) []StepExecutionResponse {
	resp := make([]StepExecutionResponse, len(execs))
	for i := range execs {
		resp[i] = StepExecutionToResponse(&execs[i])
	}
	return resp
}

// DefaultListInstancesRequest returns a ListInstancesRequest with default pagination values.
func DefaultListInstancesRequest() ListInstancesRequest {
	return ListInstancesRequest{
		Page:     1,
		PageSize: 20,
	}
}

// Offset computes the SQL OFFSET from Page and PageSize.
func (r *ListInstancesRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}
