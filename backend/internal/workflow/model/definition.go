package model

import (
	"encoding/json"
	"time"
)

// WorkflowDefinition represents a versioned workflow blueprint that describes
// trigger conditions, variables, and the ordered steps to execute.
type WorkflowDefinition struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	Name          string                 `json:"name" db:"name"`
	Description   string                 `json:"description" db:"description"`
	Category      string                 `json:"category,omitempty" db:"category"` // approval, onboarding, review, escalation, notification, data_pipeline, compliance, custom
	Version       int                    `json:"version" db:"version"`
	Status        string                 `json:"status" db:"status"` // draft, active, deprecated, archived
	TriggerConfig TriggerConfig          `json:"trigger_config" db:"trigger_config"`
	Variables     map[string]VariableDef `json:"variables" db:"variables"`
	Steps         []StepDefinition       `json:"steps" db:"steps"`
	CreatedBy     string                 `json:"created_by" db:"created_by"`
	UpdatedBy     string                 `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Valid workflow definition categories.
const (
	CategoryApproval      = "approval"
	CategoryOnboarding    = "onboarding"
	CategoryReview        = "review"
	CategoryEscalation    = "escalation"
	CategoryNotification  = "notification"
	CategoryDataPipeline  = "data_pipeline"
	CategoryCompliance    = "compliance"
	CategoryCustom        = "custom"
)

// ValidCategories is the set of allowed category values.
var ValidCategories = map[string]bool{
	CategoryApproval:     true,
	CategoryOnboarding:   true,
	CategoryReview:       true,
	CategoryEscalation:   true,
	CategoryNotification: true,
	CategoryDataPipeline: true,
	CategoryCompliance:   true,
	CategoryCustom:       true,
}

// TriggerConfig describes how a workflow is initiated.
type TriggerConfig struct {
	Type   string                 `json:"type"` // "manual", "event", "schedule"
	Topic  string                 `json:"topic,omitempty"`
	Filter map[string]interface{} `json:"filter,omitempty"`
	Cron   string                 `json:"cron,omitempty"`
}

// VariableDef describes a workflow-level variable with its type and default value.
type VariableDef struct {
	Type    string      `json:"type"` // string, boolean, number, object, array
	Source  string      `json:"source,omitempty"`
	Default interface{} `json:"default,omitempty"`
}

// StepDefinition represents a single step within a workflow definition.
type StepDefinition struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // human_task, service_task, event_task, condition, parallel_gateway, timer, end
	Name        string                 `json:"name"`
	Config      map[string]interface{} `json:"config"`
	Transitions []Transition           `json:"transitions"`
}

// Transition defines the link from one step to another, optionally gated by a condition.
type Transition struct {
	Condition string `json:"condition,omitempty"`
	Target    string `json:"target"`
}

// MarshalJSON implements custom JSON marshalling for TriggerConfig.
func (tc TriggerConfig) MarshalJSON() ([]byte, error) {
	type Alias TriggerConfig
	return json.Marshal((*Alias)(&tc))
}

// Status constants for WorkflowDefinition.
const (
	DefinitionStatusDraft      = "draft"
	DefinitionStatusActive     = "active"
	DefinitionStatusDeprecated = "deprecated"
	DefinitionStatusArchived   = "archived"
)

// Step type constants.
const (
	StepTypeHumanTask       = "human_task"
	StepTypeServiceTask     = "service_task"
	StepTypeEventTask       = "event_task"
	StepTypeCondition       = "condition"
	StepTypeParallelGateway = "parallel_gateway"
	StepTypeTimer           = "timer"
	StepTypeEnd             = "end"
)

// Trigger type constants.
const (
	TriggerTypeManual   = "manual"
	TriggerTypeEvent    = "event"
	TriggerTypeSchedule = "schedule"
)

// ValidDefinitionStatuses is the set of allowed definition statuses.
var ValidDefinitionStatuses = map[string]bool{
	DefinitionStatusDraft:      true,
	DefinitionStatusActive:     true,
	DefinitionStatusDeprecated: true,
	DefinitionStatusArchived:   true,
}

// ValidStepTypes is the set of allowed step types.
var ValidStepTypes = map[string]bool{
	StepTypeHumanTask:       true,
	StepTypeServiceTask:     true,
	StepTypeEventTask:       true,
	StepTypeCondition:       true,
	StepTypeParallelGateway: true,
	StepTypeTimer:           true,
	StepTypeEnd:             true,
}

// ValidTriggerTypes is the set of allowed trigger types.
var ValidTriggerTypes = map[string]bool{
	TriggerTypeManual:   true,
	TriggerTypeEvent:    true,
	TriggerTypeSchedule: true,
}

// ValidVariableTypes is the set of allowed variable types.
var ValidVariableTypes = map[string]bool{
	"string":  true,
	"boolean": true,
	"number":  true,
	"object":  true,
	"array":   true,
}
