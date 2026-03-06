package dto

import (
	"time"

	"github.com/clario360/platform/internal/workflow/model"
)

// ---------- Requests ----------

// CreateDefinitionRequest is the payload for creating a new workflow definition.
type CreateDefinitionRequest struct {
	Name          string                        `json:"name" validate:"required,min=1,max=255"`
	Description   string                        `json:"description" validate:"max=2000"`
	TriggerConfig model.TriggerConfig           `json:"trigger_config" validate:"required"`
	Variables     map[string]model.VariableDef  `json:"variables"`
	Steps         []model.StepDefinition        `json:"steps" validate:"required,min=1"`
}

// UpdateDefinitionRequest is the payload for updating an existing workflow definition.
type UpdateDefinitionRequest struct {
	Name          *string                       `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description   *string                       `json:"description,omitempty" validate:"omitempty,max=2000"`
	TriggerConfig *model.TriggerConfig          `json:"trigger_config,omitempty"`
	Variables     map[string]model.VariableDef  `json:"variables,omitempty"`
	Steps         []model.StepDefinition        `json:"steps,omitempty" validate:"omitempty,min=1"`
}

// ListDefinitionsRequest holds query parameters for listing workflow definitions.
type ListDefinitionsRequest struct {
	TenantID string `json:"tenant_id"`
	Status   string `json:"status" validate:"omitempty,oneof=draft active deprecated archived"`
	Name     string `json:"name"`
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"page_size" validate:"min=1,max=100"`
}

// ---------- Responses ----------

// DefinitionResponse wraps a WorkflowDefinition with computed fields for API responses.
type DefinitionResponse struct {
	ID            string                        `json:"id"`
	TenantID      string                        `json:"tenant_id"`
	Name          string                        `json:"name"`
	Description   string                        `json:"description"`
	Version       int                           `json:"version"`
	Status        string                        `json:"status"`
	TriggerConfig model.TriggerConfig           `json:"trigger_config"`
	Variables     map[string]model.VariableDef  `json:"variables"`
	Steps         []model.StepDefinition        `json:"steps"`
	StepCount     int                           `json:"step_count"`
	CreatedBy     string                        `json:"created_by"`
	UpdatedBy     string                        `json:"updated_by,omitempty"`
	CreatedAt     time.Time                     `json:"created_at"`
	UpdatedAt     time.Time                     `json:"updated_at"`
}

// ListDefinitionsResponse is the paginated response for listing definitions.
type ListDefinitionsResponse struct {
	Definitions []DefinitionResponse `json:"definitions"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"page_size"`
}

// ValidationError describes a single validation problem, optionally scoped to a step.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	StepID  string `json:"step_id,omitempty"`
}

// ---------- Converters ----------

// DefinitionToResponse converts a WorkflowDefinition model to its API response form.
func DefinitionToResponse(d *model.WorkflowDefinition) DefinitionResponse {
	vars := d.Variables
	if vars == nil {
		vars = make(map[string]model.VariableDef)
	}
	steps := d.Steps
	if steps == nil {
		steps = []model.StepDefinition{}
	}
	return DefinitionResponse{
		ID:            d.ID,
		TenantID:      d.TenantID,
		Name:          d.Name,
		Description:   d.Description,
		Version:       d.Version,
		Status:        d.Status,
		TriggerConfig: d.TriggerConfig,
		Variables:     vars,
		Steps:         steps,
		StepCount:     len(steps),
		CreatedBy:     d.CreatedBy,
		UpdatedBy:     d.UpdatedBy,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// DefinitionsToResponse converts a slice of definitions to their response form.
func DefinitionsToResponse(defs []model.WorkflowDefinition) []DefinitionResponse {
	resp := make([]DefinitionResponse, len(defs))
	for i := range defs {
		resp[i] = DefinitionToResponse(&defs[i])
	}
	return resp
}

// DefaultListDefinitionsRequest returns a ListDefinitionsRequest with default pagination values.
func DefaultListDefinitionsRequest() ListDefinitionsRequest {
	return ListDefinitionsRequest{
		Page:     1,
		PageSize: 20,
	}
}

// Offset computes the SQL OFFSET from Page and PageSize.
func (r *ListDefinitionsRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

// Validate checks for workflow definition validation errors in the definition request.
// Returns a slice of ValidationError; empty means valid.
func (r *CreateDefinitionRequest) Validate() []ValidationError {
	var errs []ValidationError

	if r.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	}

	if !model.ValidTriggerTypes[r.TriggerConfig.Type] {
		errs = append(errs, ValidationError{
			Field:   "trigger_config.type",
			Message: "must be one of: manual, event, schedule",
		})
	}

	if r.TriggerConfig.Type == model.TriggerTypeEvent && r.TriggerConfig.Topic == "" {
		errs = append(errs, ValidationError{
			Field:   "trigger_config.topic",
			Message: "topic is required for event triggers",
		})
	}

	if r.TriggerConfig.Type == model.TriggerTypeSchedule && r.TriggerConfig.Cron == "" {
		errs = append(errs, ValidationError{
			Field:   "trigger_config.cron",
			Message: "cron expression is required for schedule triggers",
		})
	}

	if len(r.Steps) == 0 {
		errs = append(errs, ValidationError{Field: "steps", Message: "at least one step is required"})
	}

	stepIDs := make(map[string]bool)
	hasEnd := false
	for _, step := range r.Steps {
		if step.ID == "" {
			errs = append(errs, ValidationError{
				Field:  "steps.id",
				StepID: step.ID,
				Message: "step id is required",
			})
			continue
		}
		if stepIDs[step.ID] {
			errs = append(errs, ValidationError{
				Field:  "steps.id",
				StepID: step.ID,
				Message: "duplicate step id",
			})
		}
		stepIDs[step.ID] = true

		if !model.ValidStepTypes[step.Type] {
			errs = append(errs, ValidationError{
				Field:   "steps.type",
				StepID:  step.ID,
				Message: "invalid step type",
			})
		}
		if step.Type == model.StepTypeEnd {
			hasEnd = true
		}
	}

	if !hasEnd && len(r.Steps) > 0 {
		errs = append(errs, ValidationError{
			Field:   "steps",
			Message: "workflow must contain at least one end step",
		})
	}

	// Validate transition targets reference existing step IDs.
	for _, step := range r.Steps {
		for _, t := range step.Transitions {
			if t.Target != "" && !stepIDs[t.Target] {
				errs = append(errs, ValidationError{
					Field:   "steps.transitions.target",
					StepID:  step.ID,
					Message: "transition target '" + t.Target + "' does not reference a valid step",
				})
			}
		}
	}

	// Validate variable definitions.
	for name, v := range r.Variables {
		if !model.ValidVariableTypes[v.Type] {
			errs = append(errs, ValidationError{
				Field:   "variables." + name + ".type",
				Message: "invalid variable type '" + v.Type + "'",
			})
		}
	}

	return errs
}
