package model

import (
	"encoding/json"
	"time"
)

// WorkflowInstance represents a single execution of a workflow definition.
// LockVersion is used for optimistic concurrency control on updates.
type WorkflowInstance struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	DefinitionID  string                 `json:"definition_id" db:"definition_id"`
	DefinitionVer int                    `json:"definition_ver" db:"definition_ver"`
	Status        string                 `json:"status" db:"status"`
	CurrentStepID *string                `json:"current_step_id,omitempty" db:"current_step_id"`
	Variables     map[string]interface{} `json:"variables" db:"variables"`
	StepOutputs   map[string]interface{} `json:"step_outputs" db:"step_outputs"`
	TriggerData   json.RawMessage        `json:"trigger_data,omitempty" db:"trigger_data"`
	ErrorMessage  *string                `json:"error_message,omitempty" db:"error_message"`
	StartedBy     *string                `json:"started_by,omitempty" db:"started_by"`
	StartedAt     time.Time              `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	LockVersion   int                    `json:"lock_version" db:"lock_version"`
}

// StepExecution records the execution of a single step within a workflow instance.
// Attempt tracks retry count for the same step.
type StepExecution struct {
	ID           string          `json:"id" db:"id"`
	InstanceID   string          `json:"instance_id" db:"instance_id"`
	StepID       string          `json:"step_id" db:"step_id"`
	StepType     string          `json:"step_type" db:"step_type"`
	Status       string          `json:"status" db:"status"`
	InputData    json.RawMessage `json:"input_data,omitempty" db:"input_data"`
	OutputData   json.RawMessage `json:"output_data,omitempty" db:"output_data"`
	ErrorMessage *string         `json:"error_message,omitempty" db:"error_message"`
	Attempt      int             `json:"attempt" db:"attempt"`
	StartedAt    *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// Instance status constants.
const (
	InstanceStatusRunning   = "running"
	InstanceStatusCompleted = "completed"
	InstanceStatusFailed    = "failed"
	InstanceStatusCancelled = "cancelled"
	InstanceStatusSuspended = "suspended"
)

// Step execution status constants.
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
	StepStatusCancelled = "cancelled"
)

// ValidInstanceStatuses is the set of allowed instance statuses.
var ValidInstanceStatuses = map[string]bool{
	InstanceStatusRunning:   true,
	InstanceStatusCompleted: true,
	InstanceStatusFailed:    true,
	InstanceStatusCancelled: true,
	InstanceStatusSuspended: true,
}

// ValidStepStatuses is the set of allowed step execution statuses.
var ValidStepStatuses = map[string]bool{
	StepStatusPending:   true,
	StepStatusRunning:   true,
	StepStatusCompleted: true,
	StepStatusFailed:    true,
	StepStatusSkipped:   true,
	StepStatusCancelled: true,
}

// IsTerminal returns true if the instance is in a terminal state.
func (wi *WorkflowInstance) IsTerminal() bool {
	return wi.Status == InstanceStatusCompleted ||
		wi.Status == InstanceStatusFailed ||
		wi.Status == InstanceStatusCancelled
}

// IsRunnable returns true if the instance can accept new step transitions.
func (wi *WorkflowInstance) IsRunnable() bool {
	return wi.Status == InstanceStatusRunning
}
