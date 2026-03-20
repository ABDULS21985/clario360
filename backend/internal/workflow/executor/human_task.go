package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/workflow/expression"
	"github.com/clario360/platform/internal/workflow/model"
)

// TaskCreator is a narrow interface for creating human task records, avoiding
// a circular dependency with the full task repository.
type TaskCreator interface {
	Create(ctx context.Context, task *model.HumanTask) error
}

// HumanTaskExecutor creates a human task record and parks the workflow until
// a human completes the task. The task includes a form schema, assignee
// information, SLA deadline, and priority derived from the workflow context.
type HumanTaskExecutor struct {
	taskRepo TaskCreator
	producer *events.Producer
	resolver *expression.VariableResolver
	logger   zerolog.Logger
}

// NewHumanTaskExecutor creates a HumanTaskExecutor.
func NewHumanTaskExecutor(taskRepo TaskCreator, producer *events.Producer, logger zerolog.Logger) *HumanTaskExecutor {
	return &HumanTaskExecutor{
		taskRepo: taskRepo,
		producer: producer,
		resolver: expression.NewVariableResolver(),
		logger:   logger.With().Str("executor", "human_task").Logger(),
	}
}

// Execute creates a human task from the step configuration and parks the workflow.
//
// Expected step.Config keys:
//   - form_fields ([]interface{}, required): form field definitions
//   - assignee (string, optional): user ID or ${...} variable reference
//   - assignee_role (string, optional): role name for group assignment
//   - sla_hours (float64, optional): hours until SLA deadline, default 24
//   - escalation_role (string, optional): role to escalate to on SLA breach
//   - description (string, optional): task description
func (e *HumanTaskExecutor) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	dataCtx := buildDataContext(instance)

	// Extract and build form schema from config.
	formSchema, err := e.buildFormSchema(step.Config)
	if err != nil {
		return nil, fmt.Errorf("human_task %s: %w", step.ID, err)
	}

	// Resolve assignee: may be a ${...} variable reference.
	var assigneeID *string
	if assigneeRaw := configStringOptional(step.Config, "assignee"); assigneeRaw != "" {
		if strings.Contains(assigneeRaw, "${") {
			resolved, err := e.resolver.Resolve(assigneeRaw, dataCtx)
			if err != nil {
				return nil, fmt.Errorf("human_task %s: resolving assignee: %w", step.ID, err)
			}
			s := fmt.Sprintf("%v", resolved)
			assigneeID = &s
		} else {
			assigneeID = &assigneeRaw
		}
	}

	// Resolve assignee role.
	var assigneeRole *string
	if role := configStringOptional(step.Config, "assignee_role"); role != "" {
		assigneeRole = &role
	}

	// Parse SLA hours (default 24h).
	slaHours := 24.0
	if v, ok := step.Config["sla_hours"]; ok {
		if h := toFloat(v); h > 0 {
			slaHours = h
		}
	}
	slaDeadline := time.Now().UTC().Add(time.Duration(slaHours * float64(time.Hour)))

	// Resolve escalation role.
	var escalationRole *string
	if role := configStringOptional(step.Config, "escalation_role"); role != "" {
		escalationRole = &role
	}

	// Determine priority from instance variables.
	priority := determinePriority(instance.Variables)

	// Resolve task description.
	description := step.Name
	if desc := configStringOptional(step.Config, "description"); desc != "" {
		resolved, err := e.resolver.Resolve(desc, dataCtx)
		if err == nil {
			description = fmt.Sprintf("%v", resolved)
		}
	}

	// Build metadata for the task.
	metadata := map[string]interface{}{
		"workflow_instance_id": instance.ID,
		"workflow_definition":  instance.DefinitionID,
		"definition_version":   instance.DefinitionVer,
		"step_id":              step.ID,
		"step_execution_id":    exec.ID,
	}

	task := &model.HumanTask{
		ID:             events.GenerateUUID(),
		TenantID:       instance.TenantID,
		InstanceID:     instance.ID,
		StepID:         step.ID,
		StepExecID:     exec.ID,
		Name:           step.Name,
		Description:    description,
		Status:         model.TaskStatusPending,
		AssigneeID:     assigneeID,
		AssigneeRole:   assigneeRole,
		FormSchema:     formSchema,
		FormData:       make(map[string]interface{}),
		SLADeadline:    &slaDeadline,
		EscalationRole: escalationRole,
		Priority:       priority,
		Metadata:       metadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := e.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("human_task %s: creating task: %w", step.ID, err)
	}

	e.logger.Info().
		Str("task_id", task.ID).
		Str("step_id", step.ID).
		Str("instance_id", instance.ID).
		Int("priority", priority).
		Time("sla_deadline", slaDeadline).
		Msg("human task created, parking workflow")

	e.publishTaskCreated(ctx, instance, task)

	return &ExecutionResult{
		Output: map[string]interface{}{
			"task_id":      task.ID,
			"sla_deadline": slaDeadline.Format(time.RFC3339),
			"priority":     priority,
		},
		Parked: true,
	}, nil
}

func (e *HumanTaskExecutor) publishTaskCreated(ctx context.Context, instance *model.WorkflowInstance, task *model.HumanTask) {
	if e.producer == nil {
		return
	}

	payload := map[string]interface{}{
		"task_id":      task.ID,
		"instance_id":  task.InstanceID,
		"step_id":      task.StepID,
		"task_name":    task.Name,
		"priority":     task.Priority,
		"sla_deadline": task.SLADeadline,
	}
	if task.AssigneeID != nil {
		payload["assignee_id"] = *task.AssigneeID
	}
	if task.AssigneeRole != nil {
		payload["assignee_role"] = *task.AssigneeRole
	}
	if task.EscalationRole != nil {
		payload["escalation_role"] = *task.EscalationRole
	}
	if instance.StartedBy != nil {
		payload["initiator_id"] = *instance.StartedBy
	}

	evt, err := events.NewEvent("workflow.task.created", "workflow-engine", task.TenantID, payload)
	if err != nil {
		e.logger.Warn().Err(err).Str("task_id", task.ID).Msg("failed to build workflow task created event")
		return
	}
	if err := e.producer.Publish(ctx, events.Topics.WorkflowEvents, evt); err != nil {
		e.logger.Warn().Err(err).Str("task_id", task.ID).Msg("failed to publish workflow task created event")
	}
}

// buildFormSchema converts the "form_fields" config into a []model.FormField slice.
func (e *HumanTaskExecutor) buildFormSchema(config map[string]interface{}) ([]model.FormField, error) {
	fieldsRaw, ok := config["form_fields"]
	if !ok {
		return nil, fmt.Errorf("missing required config key %q", "form_fields")
	}

	fieldsSlice, ok := fieldsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("config key %q must be an array", "form_fields")
	}

	fields := make([]model.FormField, 0, len(fieldsSlice))
	for i, raw := range fieldsSlice {
		fieldMap, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("form_fields[%d] must be an object", i)
		}

		field := model.FormField{}

		if name, ok := fieldMap["name"].(string); ok {
			field.Name = name
		} else {
			return nil, fmt.Errorf("form_fields[%d]: missing or invalid 'name'", i)
		}

		if typ, ok := fieldMap["type"].(string); ok {
			field.Type = typ
		} else {
			return nil, fmt.Errorf("form_fields[%d]: missing or invalid 'type'", i)
		}

		if label, ok := fieldMap["label"].(string); ok {
			field.Label = label
		} else {
			field.Label = field.Name
		}

		if req, ok := fieldMap["required"].(bool); ok {
			field.Required = req
		}

		if opts, ok := fieldMap["options"].([]interface{}); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok {
					field.Options = append(field.Options, s)
				}
			}
		}

		if def, ok := fieldMap["default"]; ok {
			field.Default = def
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// determinePriority returns a numeric priority based on the workflow's severity variable.
// critical -> 2, high -> 1, everything else -> 0
func determinePriority(variables map[string]interface{}) int {
	if variables == nil {
		return 0
	}

	severity, ok := variables["severity"]
	if !ok {
		return 0
	}

	// Handle both direct string and JSON-decoded values.
	var sevStr string
	switch v := severity.(type) {
	case string:
		sevStr = strings.ToLower(v)
	case json.Number:
		sevStr = v.String()
	default:
		sevStr = fmt.Sprintf("%v", v)
		sevStr = strings.ToLower(sevStr)
	}

	switch sevStr {
	case "critical":
		return 2
	case "high":
		return 1
	default:
		return 0
	}
}
