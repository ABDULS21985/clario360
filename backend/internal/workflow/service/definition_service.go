package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/model"
)

// definitionRepo defines the persistence operations for workflow definitions.
type definitionRepo interface {
	Create(ctx context.Context, def *model.WorkflowDefinition) error
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error)
	GetActiveByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error)
	List(ctx context.Context, tenantID, status, nameFilter string, limit, offset int) ([]*model.WorkflowDefinition, int, error)
	ListVersions(ctx context.Context, tenantID, id string) ([]*model.WorkflowDefinition, error)
	Update(ctx context.Context, def *model.WorkflowDefinition) error
	SoftDelete(ctx context.Context, tenantID, id string) error
	GetMaxVersion(ctx context.Context, tenantID, name string) (int, error)
	GetActiveByTriggerTopic(ctx context.Context, topic string) ([]*model.WorkflowDefinition, error)
}

// DefinitionService manages the lifecycle of workflow definitions including
// creation, versioning, activation, validation, and soft-deletion.
type DefinitionService struct {
	repo   definitionRepo
	logger zerolog.Logger
}

// NewDefinitionService creates a new DefinitionService.
func NewDefinitionService(repo definitionRepo, logger zerolog.Logger) *DefinitionService {
	return &DefinitionService{
		repo:   repo,
		logger: logger.With().Str("service", "workflow-definition").Logger(),
	}
}

// Create creates a new workflow definition in draft status with version 1.
func (s *DefinitionService) Create(ctx context.Context, tenantID, userID string, req dto.CreateDefinitionRequest) (*model.WorkflowDefinition, error) {
	// Validate name constraints.
	if req.Name == "" {
		return nil, fmt.Errorf("workflow definition name is required")
	}
	if len(req.Name) > 200 {
		return nil, fmt.Errorf("workflow definition name must not exceed 200 characters")
	}

	now := time.Now().UTC()
	def := &model.WorkflowDefinition{
		ID:            generateUUID(),
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		Version:       1,
		Status:        model.DefinitionStatusDraft,
		TriggerConfig: req.TriggerConfig,
		Variables:     req.Variables,
		Steps:         req.Steps,
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if def.Variables == nil {
		def.Variables = make(map[string]model.VariableDef)
	}
	if def.Steps == nil {
		def.Steps = []model.StepDefinition{}
	}

	if err := s.repo.Create(ctx, def); err != nil {
		s.logger.Error().Err(err).
			Str("tenant_id", tenantID).
			Str("name", req.Name).
			Msg("failed to create workflow definition")
		return nil, fmt.Errorf("creating workflow definition: %w", err)
	}

	s.logger.Info().
		Str("id", def.ID).
		Str("tenant_id", tenantID).
		Str("name", def.Name).
		Int("version", def.Version).
		Msg("workflow definition created")

	return def, nil
}

// GetByID retrieves a workflow definition by tenant and definition ID.
func (s *DefinitionService) GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error) {
	def, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("getting workflow definition: %w", err)
	}
	return def, nil
}

// List returns a paginated list of workflow definitions for a tenant,
// optionally filtered by status and name substring.
func (s *DefinitionService) List(ctx context.Context, tenantID, status, nameFilter string, page, pageSize int) ([]*model.WorkflowDefinition, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	defs, total, err := s.repo.List(ctx, tenantID, status, nameFilter, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing workflow definitions: %w", err)
	}
	return defs, total, nil
}

// Update creates a NEW version of an existing workflow definition.
// The old version is deprecated and a new record is created with version+1.
func (s *DefinitionService) Update(ctx context.Context, tenantID, id, userID string, req dto.UpdateDefinitionRequest) (*model.WorkflowDefinition, error) {
	// Fetch the current definition.
	current, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("getting current definition for update: %w", err)
	}

	// Validate name if provided.
	if req.Name != nil {
		if *req.Name == "" {
			return nil, fmt.Errorf("workflow definition name cannot be empty")
		}
		if len(*req.Name) > 200 {
			return nil, fmt.Errorf("workflow definition name must not exceed 200 characters")
		}
	}

	// Get the max version for this definition's name to ensure monotonic versioning.
	maxVersion, err := s.repo.GetMaxVersion(ctx, tenantID, current.Name)
	if err != nil {
		return nil, fmt.Errorf("getting max version: %w", err)
	}

	// Build the new version by cloning the current and applying updates.
	now := time.Now().UTC()
	newDef := &model.WorkflowDefinition{
		ID:            generateUUID(),
		TenantID:      tenantID,
		Name:          current.Name,
		Description:   current.Description,
		Version:       maxVersion + 1,
		Status:        model.DefinitionStatusDraft,
		TriggerConfig: current.TriggerConfig,
		Variables:     current.Variables,
		Steps:         current.Steps,
		CreatedBy:     current.CreatedBy,
		UpdatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Apply provided updates.
	if req.Name != nil {
		newDef.Name = *req.Name
	}
	if req.Description != nil {
		newDef.Description = *req.Description
	}
	if req.TriggerConfig != nil {
		newDef.TriggerConfig = *req.TriggerConfig
	}
	if req.Variables != nil {
		newDef.Variables = req.Variables
	}
	if req.Steps != nil {
		newDef.Steps = req.Steps
	}

	// Create the new version.
	if err := s.repo.Create(ctx, newDef); err != nil {
		return nil, fmt.Errorf("creating new definition version: %w", err)
	}

	// Deprecate the old version.
	current.Status = model.DefinitionStatusDeprecated
	current.UpdatedAt = now
	current.UpdatedBy = userID
	if err := s.repo.Update(ctx, current); err != nil {
		s.logger.Error().Err(err).
			Str("old_id", current.ID).
			Msg("failed to deprecate old definition version")
		// The new version was already created; log but do not fail.
	}

	s.logger.Info().
		Str("id", newDef.ID).
		Str("old_id", current.ID).
		Str("tenant_id", tenantID).
		Int("version", newDef.Version).
		Msg("workflow definition updated (new version created)")

	return newDef, nil
}

// Activate validates a definition and sets its status to active.
// Only draft definitions can be activated.
func (s *DefinitionService) Activate(ctx context.Context, tenantID, id string) error {
	def, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("getting definition for activation: %w", err)
	}

	if def.Status != model.DefinitionStatusDraft {
		return fmt.Errorf("only draft definitions can be activated, current status: %s", def.Status)
	}

	// Run full validation.
	validationErrors := s.ValidateDefinition(def)
	if len(validationErrors) > 0 {
		return fmt.Errorf("definition validation failed with %d errors: %s",
			len(validationErrors), formatValidationErrors(validationErrors))
	}

	def.Status = model.DefinitionStatusActive
	def.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, def); err != nil {
		return fmt.Errorf("activating definition: %w", err)
	}

	s.logger.Info().
		Str("id", def.ID).
		Str("tenant_id", tenantID).
		Str("name", def.Name).
		Msg("workflow definition activated")

	return nil
}

// Delete performs a soft-delete on a workflow definition.
func (s *DefinitionService) Delete(ctx context.Context, tenantID, id string) error {
	if err := s.repo.SoftDelete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("deleting workflow definition: %w", err)
	}

	s.logger.Info().
		Str("id", id).
		Str("tenant_id", tenantID).
		Msg("workflow definition soft-deleted")

	return nil
}

// ListVersions returns all versions of a workflow definition.
func (s *DefinitionService) ListVersions(ctx context.Context, tenantID, id string) ([]*model.WorkflowDefinition, error) {
	versions, err := s.repo.ListVersions(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("listing definition versions: %w", err)
	}
	return versions, nil
}

// ValidateDefinition performs comprehensive validation of a workflow definition.
// Returns a slice of validation errors; an empty slice means the definition is valid.
func (s *DefinitionService) ValidateDefinition(def *model.WorkflowDefinition) []dto.ValidationError {
	var errs []dto.ValidationError

	// 1. Name: non-empty, max 200 chars.
	if def.Name == "" {
		errs = append(errs, dto.ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	} else if len(def.Name) > 200 {
		errs = append(errs, dto.ValidationError{
			Field:   "name",
			Message: "name must not exceed 200 characters",
		})
	}

	// 2. At least 2 steps.
	if len(def.Steps) < 2 {
		errs = append(errs, dto.ValidationError{
			Field:   "steps",
			Message: "workflow must contain at least 2 steps",
		})
	}

	// Build a map of step IDs for reference checking.
	stepIDs := make(map[string]bool, len(def.Steps))
	stepMap := make(map[string]*model.StepDefinition, len(def.Steps))
	endStepCount := 0

	for i := range def.Steps {
		step := &def.Steps[i]

		// 4. All step IDs must be unique.
		if step.ID == "" {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.id",
				StepID:  step.ID,
				Message: "step id is required",
			})
			continue
		}
		if stepIDs[step.ID] {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.id",
				StepID:  step.ID,
				Message: fmt.Sprintf("duplicate step id: %s", step.ID),
			})
		}
		stepIDs[step.ID] = true
		stepMap[step.ID] = step

		if step.Type == model.StepTypeEnd {
			endStepCount++
		}

		// Validate step type.
		if !model.ValidStepTypes[step.Type] {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.type",
				StepID:  step.ID,
				Message: fmt.Sprintf("invalid step type: %s", step.Type),
			})
		}
	}

	// 3. Exactly one "end" step.
	if endStepCount == 0 {
		errs = append(errs, dto.ValidationError{
			Field:   "steps",
			Message: "workflow must contain exactly one end step",
		})
	} else if endStepCount > 1 {
		errs = append(errs, dto.ValidationError{
			Field:   "steps",
			Message: fmt.Sprintf("workflow must contain exactly one end step, found %d", endStepCount),
		})
	}

	// 5. All transition targets reference existing step IDs.
	reachable := make(map[string]bool)
	for _, step := range def.Steps {
		for _, t := range step.Transitions {
			if t.Target != "" && !stepIDs[t.Target] {
				errs = append(errs, dto.ValidationError{
					Field:   "steps.transitions.target",
					StepID:  step.ID,
					Message: fmt.Sprintf("transition target '%s' does not reference a valid step", t.Target),
				})
			}
			if t.Target != "" {
				reachable[t.Target] = true
			}
		}
	}

	// 6. No orphan steps: every non-first step must be reachable via transitions.
	if len(def.Steps) > 1 {
		for i, step := range def.Steps {
			if i == 0 {
				continue // First step is the entry point, always reachable.
			}
			if !reachable[step.ID] {
				errs = append(errs, dto.ValidationError{
					Field:   "steps",
					StepID:  step.ID,
					Message: fmt.Sprintf("step '%s' is not reachable from any transition (orphan step)", step.ID),
				})
			}
		}
	}

	// 7. Step-specific validation.
	for _, step := range def.Steps {
		errs = append(errs, s.validateStepConfig(step)...)
	}

	// 8. Trigger validation.
	errs = append(errs, s.validateTrigger(def.TriggerConfig)...)

	// 9. Variable validation.
	errs = append(errs, s.validateVariables(def.Variables)...)

	return errs
}

// validateStepConfig validates the configuration of individual step types.
func (s *DefinitionService) validateStepConfig(step model.StepDefinition) []dto.ValidationError {
	var errs []dto.ValidationError

	switch step.Type {
	case model.StepTypeHumanTask:
		// human_task: needs form_fields, assignee_role or assignee.
		if _, ok := step.Config["form_fields"]; !ok {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.form_fields",
				StepID:  step.ID,
				Message: "human_task step requires form_fields configuration",
			})
		}
		hasAssigneeRole := configStringNotEmpty(step.Config, "assignee_role")
		hasAssignee := configStringNotEmpty(step.Config, "assignee")
		if !hasAssigneeRole && !hasAssignee {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config",
				StepID:  step.ID,
				Message: "human_task step requires either assignee_role or assignee",
			})
		}

	case model.StepTypeServiceTask:
		// service_task: needs service, method, url.
		if !configStringNotEmpty(step.Config, "service") {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.service",
				StepID:  step.ID,
				Message: "service_task step requires service configuration",
			})
		}
		if !configStringNotEmpty(step.Config, "method") {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.method",
				StepID:  step.ID,
				Message: "service_task step requires method configuration",
			})
		}
		if !configStringNotEmpty(step.Config, "url") {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.url",
				StepID:  step.ID,
				Message: "service_task step requires url configuration",
			})
		}

	case model.StepTypeEventTask:
		// event_task: needs topic; if mode=wait, needs correlation_field.
		if !configStringNotEmpty(step.Config, "topic") {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.topic",
				StepID:  step.ID,
				Message: "event_task step requires topic configuration",
			})
		}
		mode, _ := step.Config["mode"].(string)
		if mode == "wait" {
			if !configStringNotEmpty(step.Config, "correlation_field") {
				errs = append(errs, dto.ValidationError{
					Field:   "steps.config.correlation_field",
					StepID:  step.ID,
					Message: "event_task step with mode=wait requires correlation_field configuration",
				})
			}
		}

	case model.StepTypeCondition:
		// condition: needs expression.
		if !configStringNotEmpty(step.Config, "expression") {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.expression",
				StepID:  step.ID,
				Message: "condition step requires expression configuration",
			})
		}

	case model.StepTypeParallelGateway:
		// parallel_gateway: needs branches.
		if _, ok := step.Config["branches"]; !ok {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config.branches",
				StepID:  step.ID,
				Message: "parallel_gateway step requires branches configuration",
			})
		}

	case model.StepTypeTimer:
		// timer: needs duration or fire_at.
		hasDuration := configStringNotEmpty(step.Config, "duration")
		hasFireAt := configStringNotEmpty(step.Config, "fire_at")
		if !hasDuration && !hasFireAt {
			errs = append(errs, dto.ValidationError{
				Field:   "steps.config",
				StepID:  step.ID,
				Message: "timer step requires either duration or fire_at configuration",
			})
		}

	case model.StepTypeEnd:
		// End steps do not need additional config validation.
	}

	return errs
}

// validateTrigger validates the trigger configuration.
func (s *DefinitionService) validateTrigger(tc model.TriggerConfig) []dto.ValidationError {
	var errs []dto.ValidationError

	if !model.ValidTriggerTypes[tc.Type] {
		errs = append(errs, dto.ValidationError{
			Field:   "trigger_config.type",
			Message: fmt.Sprintf("invalid trigger type: %s; must be one of: manual, event, schedule", tc.Type),
		})
		return errs
	}

	switch tc.Type {
	case model.TriggerTypeEvent:
		if tc.Topic == "" {
			errs = append(errs, dto.ValidationError{
				Field:   "trigger_config.topic",
				Message: "topic is required for event triggers",
			})
		}
	case model.TriggerTypeSchedule:
		if tc.Cron == "" {
			errs = append(errs, dto.ValidationError{
				Field:   "trigger_config.cron",
				Message: "cron expression is required for schedule triggers",
			})
		}
	}

	return errs
}

// validateVariables validates the variable definitions.
func (s *DefinitionService) validateVariables(vars map[string]model.VariableDef) []dto.ValidationError {
	var errs []dto.ValidationError

	for name, v := range vars {
		if name == "" {
			errs = append(errs, dto.ValidationError{
				Field:   "variables",
				Message: "variable name cannot be empty",
			})
			continue
		}
		if !model.ValidVariableTypes[v.Type] {
			errs = append(errs, dto.ValidationError{
				Field:   fmt.Sprintf("variables.%s.type", name),
				Message: fmt.Sprintf("invalid variable type '%s'", v.Type),
			})
		}
	}

	return errs
}

// configStringNotEmpty checks whether a config map contains a non-empty string for the given key.
func configStringNotEmpty(cfg map[string]interface{}, key string) bool {
	if cfg == nil {
		return false
	}
	v, ok := cfg[key]
	if !ok {
		return false
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	return s != ""
}

// formatValidationErrors produces a human-readable summary of validation errors.
func formatValidationErrors(errs []dto.ValidationError) string {
	if len(errs) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msg := e.Field + ": " + e.Message
		if e.StepID != "" {
			msg = "[step:" + e.StepID + "] " + msg
		}
		msgs = append(msgs, msg)
	}
	result := msgs[0]
	for i := 1; i < len(msgs); i++ {
		result += "; " + msgs[i]
	}
	return result
}

// generateUUID returns a UUID v4 string using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
