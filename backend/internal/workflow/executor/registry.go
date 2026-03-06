package executor

import (
	"context"
	"errors"
	"fmt"

	"github.com/clario360/platform/internal/workflow/model"
)

// ErrParked indicates the step has been parked and is waiting for external
// completion (e.g., a human task approval, a timer firing, or an external event).
var ErrParked = errors.New("step parked: waiting for external completion")

// ExecutionResult holds the output of a step execution.
type ExecutionResult struct {
	// Output contains the step's output data, keyed by field name.
	Output map[string]interface{}
	// Parked indicates the step is waiting for an external signal to continue.
	// When true, the workflow engine should suspend the instance at this step.
	Parked bool
}

// StepExecutor is implemented by every step type to provide its execution logic.
type StepExecutor interface {
	Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error)
}

// ExecutorRegistry maps step types (e.g., "service_task", "human_task") to their
// corresponding StepExecutor implementations. The workflow engine uses this registry
// to dispatch step execution.
type ExecutorRegistry struct {
	executors map[string]StepExecutor
}

// NewExecutorRegistry creates an empty registry. Callers should register executors
// for each step type before the engine begins processing workflow instances.
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[string]StepExecutor),
	}
}

// Register adds a StepExecutor for the given step type. If an executor is already
// registered for the type, it is replaced.
func (r *ExecutorRegistry) Register(stepType string, executor StepExecutor) {
	r.executors[stepType] = executor
}

// Execute dispatches execution to the registered executor for the step's type.
// Returns an error if no executor is registered for the step type.
func (r *ExecutorRegistry) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	executor, ok := r.executors[step.Type]
	if !ok {
		return nil, fmt.Errorf("no executor registered for step type %q", step.Type)
	}
	return executor.Execute(ctx, instance, step, exec)
}

// Get returns the executor for the given step type, or nil if not registered.
func (r *ExecutorRegistry) Get(stepType string) StepExecutor {
	return r.executors[stepType]
}

// Has returns true if an executor is registered for the given step type.
func (r *ExecutorRegistry) Has(stepType string) bool {
	_, ok := r.executors[stepType]
	return ok
}
