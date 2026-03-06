package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/workflow/model"
)

// branchResult holds the output from a single completed branch.
type branchResult struct {
	index  int
	output map[string]interface{}
}

// StepLookup is a function type that resolves a step ID to its definition.
// The parallel gateway needs this to execute sub-steps within branches.
type StepLookup func(stepID string) (*model.StepDefinition, bool)

// ParallelGatewayExecutor implements fork/join parallel execution of workflow
// branches. Each branch is a sequence of step IDs executed sequentially, while
// branches run concurrently. A configurable completion policy determines when
// the gateway completes.
type ParallelGatewayExecutor struct {
	registry   *ExecutorRegistry
	stepLookup StepLookup
	logger     zerolog.Logger
}

// NewParallelGatewayExecutor creates a ParallelGatewayExecutor.
// stepLookup provides the ability to resolve step IDs to their definitions.
func NewParallelGatewayExecutor(registry *ExecutorRegistry, stepLookup StepLookup, logger zerolog.Logger) *ParallelGatewayExecutor {
	return &ParallelGatewayExecutor{
		registry:   registry,
		stepLookup: stepLookup,
		logger:     logger.With().Str("executor", "parallel_gateway").Logger(),
	}
}

// Execute runs branches in parallel according to the step configuration.
//
// Expected step.Config keys:
//   - branches ([]interface{}, required): each element is a []string of step IDs to execute sequentially
//   - completion_policy (string, optional): "all" (default), "any", or a numeric string/int N
//   - timeout_seconds (float64, optional): overall timeout for the gateway
//
// Returns merged output from all completed branches.
func (e *ParallelGatewayExecutor) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	// Parse branches from config.
	branches, err := e.parseBranches(step.Config)
	if err != nil {
		return nil, fmt.Errorf("parallel_gateway %s: %w", step.ID, err)
	}

	if len(branches) == 0 {
		return &ExecutionResult{
			Output: map[string]interface{}{"branches_completed": 0},
		}, nil
	}

	// Parse completion policy.
	policy, requiredN := e.parseCompletionPolicy(step.Config, len(branches))

	// Parse timeout.
	timeout := 5 * time.Minute // default timeout
	if v, ok := step.Config["timeout_seconds"]; ok {
		if seconds := toFloat(v); seconds > 0 {
			timeout = time.Duration(seconds * float64(time.Second))
		}
	}

	// Create a context with timeout.
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Track results from each branch.
	var (
		resultsMu sync.Mutex
		results   []branchResult
		completed int64
	)

	e.logger.Info().
		Str("step_id", step.ID).
		Str("instance_id", instance.ID).
		Int("branch_count", len(branches)).
		Str("policy", policy).
		Int("required_n", requiredN).
		Msg("starting parallel gateway execution")

	// Use errgroup for concurrent branch execution.
	g, gCtx := errgroup.WithContext(execCtx)

	for i, branch := range branches {
		branchIdx := i
		branchSteps := branch

		g.Go(func() error {
			output, err := e.executeBranch(gCtx, instance, branchIdx, branchSteps, exec)
			if err != nil {
				e.logger.Warn().
					Err(err).
					Str("step_id", step.ID).
					Int("branch", branchIdx).
					Msg("branch execution failed")
				return err
			}

			resultsMu.Lock()
			results = append(results, branchResult{index: branchIdx, output: output})
			resultsMu.Unlock()

			count := atomic.AddInt64(&completed, 1)

			e.logger.Debug().
				Str("step_id", step.ID).
				Int("branch", branchIdx).
				Int64("completed", count).
				Msg("branch completed")

			// For "any" policy or N-of-M, cancel remaining branches when threshold is met.
			if policy == "any" && count >= 1 {
				cancel()
			} else if policy == "n_of_m" && int(count) >= requiredN {
				cancel()
			}

			return nil
		})
	}

	// Wait for all goroutines. We may get context.Canceled errors from early
	// cancellation, which is expected for "any" and "n_of_m" policies.
	groupErr := g.Wait()

	// Check if we met the completion threshold.
	finalCompleted := int(atomic.LoadInt64(&completed))

	switch policy {
	case "all":
		if groupErr != nil && finalCompleted < len(branches) {
			return nil, fmt.Errorf("parallel_gateway %s: %d/%d branches failed: %w", step.ID, len(branches)-finalCompleted, len(branches), groupErr)
		}
	case "any":
		if finalCompleted < 1 {
			return nil, fmt.Errorf("parallel_gateway %s: no branches completed: %w", step.ID, groupErr)
		}
	case "n_of_m":
		if finalCompleted < requiredN {
			return nil, fmt.Errorf("parallel_gateway %s: only %d/%d required branches completed: %w", step.ID, finalCompleted, requiredN, groupErr)
		}
	}

	// Merge results from completed branches.
	mergedOutput := e.mergeResults(results)
	mergedOutput["branches_completed"] = finalCompleted
	mergedOutput["branches_total"] = len(branches)
	mergedOutput["completion_policy"] = policy

	e.logger.Info().
		Str("step_id", step.ID).
		Str("instance_id", instance.ID).
		Int("completed", finalCompleted).
		Int("total", len(branches)).
		Msg("parallel gateway completed")

	return &ExecutionResult{
		Output: mergedOutput,
	}, nil
}

// executeBranch runs a sequence of steps within a single branch.
func (e *ParallelGatewayExecutor) executeBranch(ctx context.Context, instance *model.WorkflowInstance, branchIdx int, stepIDs []string, parentExec *model.StepExecution) (map[string]interface{}, error) {
	branchOutput := make(map[string]interface{})

	for _, stepID := range stepIDs {
		// Check context before each step.
		select {
		case <-ctx.Done():
			return branchOutput, ctx.Err()
		default:
		}

		// Look up the step definition.
		stepDef, ok := e.stepLookup(stepID)
		if !ok {
			return nil, fmt.Errorf("branch %d: step %q not found in workflow definition", branchIdx, stepID)
		}

		// Create a synthetic step execution for the sub-step.
		subExec := &model.StepExecution{
			ID:         events.GenerateUUID(),
			InstanceID: instance.ID,
			StepID:     stepID,
			StepType:   stepDef.Type,
			Status:     model.StepStatusRunning,
			Attempt:    1,
			CreatedAt:  time.Now().UTC(),
		}

		// Execute the step through the registry.
		result, err := e.registry.Execute(ctx, instance, stepDef, subExec)
		if err != nil {
			return nil, fmt.Errorf("branch %d: step %q failed: %w", branchIdx, stepID, err)
		}

		// If a sub-step parks, the branch cannot continue.
		if result.Parked {
			return nil, fmt.Errorf("branch %d: step %q parked; parallel branches cannot contain parking steps", branchIdx, stepID)
		}

		// Accumulate step outputs.
		if result.Output != nil {
			branchOutput[stepID] = result.Output
		}
	}

	return branchOutput, nil
}

// parseBranches extracts the branches configuration. Each branch is a list of step IDs.
func (e *ParallelGatewayExecutor) parseBranches(config map[string]interface{}) ([][]string, error) {
	branchesRaw, ok := config["branches"]
	if !ok {
		return nil, fmt.Errorf("missing required config key %q", "branches")
	}

	branchesSlice, ok := branchesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("config key %q must be an array of arrays", "branches")
	}

	branches := make([][]string, 0, len(branchesSlice))
	for i, branchRaw := range branchesSlice {
		branchSlice, ok := branchRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("branches[%d] must be an array of step IDs", i)
		}

		stepIDs := make([]string, 0, len(branchSlice))
		for j, idRaw := range branchSlice {
			id, ok := idRaw.(string)
			if !ok {
				return nil, fmt.Errorf("branches[%d][%d] must be a string step ID", i, j)
			}
			stepIDs = append(stepIDs, id)
		}
		branches = append(branches, stepIDs)
	}

	return branches, nil
}

// parseCompletionPolicy parses the completion policy from config.
// Returns the policy type ("all", "any", or "n_of_m") and the N value for n_of_m.
func (e *ParallelGatewayExecutor) parseCompletionPolicy(config map[string]interface{}, totalBranches int) (string, int) {
	policyRaw, ok := config["completion_policy"]
	if !ok {
		return "all", totalBranches
	}

	switch v := policyRaw.(type) {
	case string:
		switch v {
		case "any":
			return "any", 1
		case "all":
			return "all", totalBranches
		default:
			// Try to parse as a number string.
			n := toInt(v)
			if n > 0 && n <= totalBranches {
				return "n_of_m", n
			}
			return "all", totalBranches
		}
	case float64:
		n := int(v)
		if n > 0 && n <= totalBranches {
			return "n_of_m", n
		}
		return "all", totalBranches
	case int:
		if v > 0 && v <= totalBranches {
			return "n_of_m", v
		}
		return "all", totalBranches
	case json.Number:
		n := toInt(v)
		if n > 0 && n <= totalBranches {
			return "n_of_m", n
		}
		return "all", totalBranches
	default:
		return "all", totalBranches
	}
}

// mergeResults combines outputs from multiple branch results into a single map.
// Each branch's output is stored under "branch_{index}".
func (e *ParallelGatewayExecutor) mergeResults(results []branchResult) map[string]interface{} {
	merged := make(map[string]interface{})
	branchOutputs := make(map[string]interface{})

	for _, r := range results {
		key := fmt.Sprintf("branch_%d", r.index)
		branchOutputs[key] = r.output
	}

	merged["branch_outputs"] = branchOutputs
	return merged
}
