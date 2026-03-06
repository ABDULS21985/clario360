package model

import "time"

// HumanTask represents a task that requires human interaction within a workflow.
// Tasks can be assigned to a specific user or a role, claimed, delegated, and escalated.
type HumanTask struct {
	ID             string                 `json:"id" db:"id"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	InstanceID     string                 `json:"instance_id" db:"instance_id"`
	StepID         string                 `json:"step_id" db:"step_id"`
	StepExecID     string                 `json:"step_exec_id" db:"step_exec_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	Status         string                 `json:"status" db:"status"`
	AssigneeID     *string                `json:"assignee_id,omitempty" db:"assignee_id"`
	AssigneeRole   *string                `json:"assignee_role,omitempty" db:"assignee_role"`
	ClaimedBy      *string                `json:"claimed_by,omitempty" db:"claimed_by"`
	ClaimedAt      *time.Time             `json:"claimed_at,omitempty" db:"claimed_at"`
	FormSchema     []FormField            `json:"form_schema" db:"form_schema"`
	FormData       map[string]interface{} `json:"form_data,omitempty" db:"form_data"`
	SLADeadline    *time.Time             `json:"sla_deadline,omitempty" db:"sla_deadline"`
	SLABreached    bool                   `json:"sla_breached" db:"sla_breached"`
	EscalatedTo    *string                `json:"escalated_to,omitempty" db:"escalated_to"`
	EscalationRole *string                `json:"escalation_role,omitempty" db:"escalation_role"`
	DelegatedBy    *string                `json:"delegated_by,omitempty" db:"delegated_by"`
	DelegatedAt    *time.Time             `json:"delegated_at,omitempty" db:"delegated_at"`
	Priority       int                    `json:"priority" db:"priority"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// FormField describes a single field in a human task form.
type FormField struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // boolean, text, textarea, select, number, date
	Label    string      `json:"label"`
	Required bool        `json:"required"`
	Options  []string    `json:"options,omitempty"`
	Default  interface{} `json:"default,omitempty"`
}

// SLAConfig defines the SLA parameters for a human task step.
type SLAConfig struct {
	Hours          int    `json:"sla_hours"`
	EscalationRole string `json:"escalation_role,omitempty"`
}

// Task status constants.
const (
	TaskStatusPending   = "pending"
	TaskStatusClaimed   = "claimed"
	TaskStatusCompleted = "completed"
	TaskStatusRejected  = "rejected"
	TaskStatusEscalated = "escalated"
	TaskStatusCancelled = "cancelled"
)

// ValidTaskStatuses is the set of allowed task statuses.
var ValidTaskStatuses = map[string]bool{
	TaskStatusPending:   true,
	TaskStatusClaimed:   true,
	TaskStatusCompleted: true,
	TaskStatusRejected:  true,
	TaskStatusEscalated: true,
	TaskStatusCancelled: true,
}

// ValidFormFieldTypes is the set of allowed form field types.
var ValidFormFieldTypes = map[string]bool{
	"boolean":  true,
	"text":     true,
	"textarea": true,
	"select":   true,
	"number":   true,
	"date":     true,
}

// IsClaimable returns true if the task can be claimed by a user.
func (ht *HumanTask) IsClaimable() bool {
	return ht.Status == TaskStatusPending
}

// IsCompletable returns true if the task can be completed.
func (ht *HumanTask) IsCompletable() bool {
	return ht.Status == TaskStatusClaimed
}

// IsCancellable returns true if the task can be cancelled.
func (ht *HumanTask) IsCancellable() bool {
	return ht.Status == TaskStatusPending || ht.Status == TaskStatusClaimed
}
