package executor

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/workflow/expression"
	"github.com/clario360/platform/internal/workflow/model"
)

// ConditionExecutor evaluates a boolean expression against the workflow's current
// data context (variables, step outputs, trigger data). The result is used by the
// workflow engine to determine which transition to follow.
type ConditionExecutor struct {
	evaluator *expression.Evaluator
}

// NewConditionExecutor creates a ConditionExecutor.
func NewConditionExecutor() *ConditionExecutor {
	return &ConditionExecutor{
		evaluator: expression.NewEvaluator(),
	}
}

// Execute evaluates the condition expression from the step configuration.
//
// Expected step.Config keys:
//   - expression (string, required): boolean expression to evaluate
//     (e.g., "variables.severity == 'critical' && steps.triage.output.is_valid == true")
//
// Returns an ExecutionResult with Output {"result": true/false}.
func (e *ConditionExecutor) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	expr, err := configString(step.Config, "expression")
	if err != nil {
		return nil, fmt.Errorf("condition %s: %w", step.ID, err)
	}

	// Build the data context the expression evaluator expects:
	// {"variables": {...}, "steps": {"stepId": {"output": {...}}}, "trigger": {"data": {...}}}
	dataCtx := buildDataContext(instance)

	result, err := e.evaluator.Evaluate(expr, dataCtx)
	if err != nil {
		return nil, fmt.Errorf("condition %s: evaluating expression %q: %w", step.ID, expr, err)
	}

	return &ExecutionResult{
		Output: map[string]interface{}{
			"result": result,
		},
	}, nil
}
