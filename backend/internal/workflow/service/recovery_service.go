package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/workflow/model"
)

// RecoveryService handles on-startup recovery of workflow instances that were
// in-progress when the engine was last shut down. It re-examines running instances
// and takes appropriate action: re-executes service tasks, skips waiting human
// tasks, re-registers timers, and advances completed steps.
type RecoveryService struct {
	instanceRepo instanceRepo
	defRepo      definitionRepo
	taskRepo     taskRepo
	rdb          *redis.Client
	engine       *EngineService
	logger       zerolog.Logger
	batchSize    int
}

// NewRecoveryService creates a new RecoveryService.
func NewRecoveryService(
	instanceRepo instanceRepo,
	defRepo definitionRepo,
	taskRepo taskRepo,
	rdb *redis.Client,
	engine *EngineService,
	logger zerolog.Logger,
	batchSize int,
) *RecoveryService {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &RecoveryService{
		instanceRepo: instanceRepo,
		defRepo:      defRepo,
		taskRepo:     taskRepo,
		rdb:          rdb,
		engine:       engine,
		logger:       logger.With().Str("service", "workflow-recovery").Logger(),
		batchSize:    batchSize,
	}
}

// Recover scans for running workflow instances in batches and attempts to
// resume each one. This should be called once during engine startup.
func (s *RecoveryService) Recover(ctx context.Context) error {
	s.logger.Info().
		Int("batch_size", s.batchSize).
		Msg("starting workflow recovery")

	totalRecovered := 0
	totalFailed := 0
	offset := 0

	for {
		select {
		case <-ctx.Done():
			s.logger.Warn().Msg("recovery cancelled by context")
			return ctx.Err()
		default:
		}

		instances, err := s.instanceRepo.ListRunning(ctx, s.batchSize, offset)
		if err != nil {
			return fmt.Errorf("listing running instances for recovery: %w", err)
		}

		if len(instances) == 0 {
			break
		}

		s.logger.Info().
			Int("batch_count", len(instances)).
			Int("offset", offset).
			Msg("processing recovery batch")

		for _, inst := range instances {
			if err := s.recoverInstance(ctx, inst); err != nil {
				s.logger.Error().Err(err).
					Str("instance_id", inst.ID).
					Str("tenant_id", inst.TenantID).
					Msg("failed to recover instance")
				totalFailed++
			} else {
				totalRecovered++
			}
		}

		if len(instances) < s.batchSize {
			break
		}
		offset += s.batchSize
	}

	// Also recover unfired timers from the database.
	if err := s.recoverTimers(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to recover timers")
	}

	s.logger.Info().
		Int("recovered", totalRecovered).
		Int("failed", totalFailed).
		Msg("workflow recovery completed")

	return nil
}

// recoverInstance examines a single running instance and takes the appropriate
// recovery action based on its current step's type and execution status.
func (s *RecoveryService) recoverInstance(ctx context.Context, inst *model.WorkflowInstance) error {
	if inst.CurrentStepID == nil || *inst.CurrentStepID == "" {
		s.logger.Warn().
			Str("instance_id", inst.ID).
			Msg("running instance has no current step, skipping")
		return nil
	}

	currentStepID := *inst.CurrentStepID

	// Load the definition to get step metadata.
	def, err := s.defRepo.GetByID(ctx, inst.TenantID, inst.DefinitionID)
	if err != nil {
		return fmt.Errorf("loading definition %s: %w", inst.DefinitionID, err)
	}

	step := findStep(def.Steps, currentStepID)
	if step == nil {
		return fmt.Errorf("current step %s not found in definition %s", currentStepID, def.ID)
	}

	// Get the latest step execution for the current step.
	executions, err := s.instanceRepo.GetStepExecutions(ctx, inst.ID)
	if err != nil {
		return fmt.Errorf("getting step executions: %w", err)
	}

	var latestExec *model.StepExecution
	for _, exec := range executions {
		if exec.StepID == currentStepID {
			if latestExec == nil || exec.CreatedAt.After(latestExec.CreatedAt) {
				latestExec = exec
			}
		}
	}

	s.logger.Debug().
		Str("instance_id", inst.ID).
		Str("step_id", currentStepID).
		Str("step_type", step.Type).
		Msg("recovering instance at step")

	switch step.Type {
	case model.StepTypeServiceTask:
		return s.recoverServiceTask(ctx, inst, def, step, latestExec)
	case model.StepTypeHumanTask:
		return s.recoverHumanTask(ctx, inst, step, latestExec)
	case model.StepTypeTimer:
		return s.recoverTimerStep(ctx, inst, step, latestExec)
	case model.StepTypeEventTask:
		return s.recoverEventTask(ctx, inst, step, latestExec)
	case model.StepTypeCondition:
		return s.recoverConditionStep(ctx, inst, step, latestExec)
	case model.StepTypeParallelGateway:
		return s.recoverParallelGateway(ctx, inst, step, latestExec)
	case model.StepTypeEnd:
		// If the current step is end but instance is still running, complete it.
		return s.engine.AdvanceWorkflow(ctx, inst.ID, currentStepID)
	default:
		s.logger.Warn().
			Str("instance_id", inst.ID).
			Str("step_type", step.Type).
			Msg("unknown step type during recovery, attempting advance")
		return s.engine.AdvanceWorkflow(ctx, inst.ID, currentStepID)
	}
}

// recoverServiceTask re-executes a service task that was in progress.
// If the step execution was already completed, it advances instead.
func (s *RecoveryService) recoverServiceTask(ctx context.Context, inst *model.WorkflowInstance, def *model.WorkflowDefinition, step *model.StepDefinition, exec *model.StepExecution) error {
	if exec != nil && exec.Status == model.StepStatusCompleted {
		// Step was completed; just advance to next.
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Msg("service task already completed, advancing workflow")
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Re-execute the service task.
	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("step_id", step.ID).
		Msg("re-executing service task during recovery")

	return s.engine.executeStep(ctx, inst, def, step.ID)
}

// recoverHumanTask checks if the associated human task still exists and is active.
// If the task is still pending or claimed, the instance remains waiting.
// If the task was completed, it advances the workflow.
func (s *RecoveryService) recoverHumanTask(ctx context.Context, inst *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) error {
	if exec != nil && exec.Status == model.StepStatusCompleted {
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Msg("human task step already completed, advancing workflow")
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Human tasks that are still pending or claimed are correctly waiting.
	// No action needed - the task service will resume the workflow when completed.
	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("step_id", step.ID).
		Msg("human task still waiting for completion, no recovery action needed")

	return nil
}

// recoverTimerStep re-registers the timer in Redis if it hasn't fired yet.
func (s *RecoveryService) recoverTimerStep(ctx context.Context, inst *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) error {
	if exec != nil && exec.Status == model.StepStatusCompleted {
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Msg("timer step already completed, advancing workflow")
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Determine fire time from step config.
	fireAt, err := s.resolveTimerFireAt(step, inst)
	if err != nil {
		return fmt.Errorf("resolving timer fire_at for recovery: %w", err)
	}

	// If the timer has already passed, fire it immediately.
	if fireAt.Before(time.Now().UTC()) {
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Msg("timer already expired during downtime, advancing workflow")
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Re-register the timer in Redis.
	member := inst.ID + ":" + step.ID
	score := float64(fireAt.UnixMilli())
	if err := s.rdb.ZAdd(ctx, redisTimerKey, redis.Z{
		Score:  score,
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("re-registering timer in Redis: %w", err)
	}

	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("step_id", step.ID).
		Time("fire_at", fireAt).
		Msg("timer re-registered in Redis during recovery")

	return nil
}

// recoverEventTask handles recovery for event wait steps.
// If the event was already received, it advances. Otherwise, the step
// remains parked waiting for the event.
func (s *RecoveryService) recoverEventTask(ctx context.Context, inst *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) error {
	if exec != nil && exec.Status == model.StepStatusCompleted {
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Msg("event task already completed, advancing workflow")
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Event wait steps remain parked; the consumer will resume them.
	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("step_id", step.ID).
		Msg("event task still waiting for event, no recovery action needed")

	return nil
}

// recoverConditionStep re-evaluates a condition step and advances.
func (s *RecoveryService) recoverConditionStep(ctx context.Context, inst *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) error {
	if exec != nil && exec.Status == model.StepStatusCompleted {
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Re-evaluate the condition by re-running the step.
	def, err := s.defRepo.GetByID(ctx, inst.TenantID, inst.DefinitionID)
	if err != nil {
		return fmt.Errorf("loading definition for condition recovery: %w", err)
	}
	return s.engine.executeStep(ctx, inst, def, step.ID)
}

// recoverParallelGateway handles recovery for parallel gateway steps.
func (s *RecoveryService) recoverParallelGateway(ctx context.Context, inst *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) error {
	if exec != nil && exec.Status == model.StepStatusCompleted {
		return s.engine.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// Re-execute the parallel gateway.
	def, err := s.defRepo.GetByID(ctx, inst.TenantID, inst.DefinitionID)
	if err != nil {
		return fmt.Errorf("loading definition for parallel gateway recovery: %w", err)
	}
	return s.engine.executeStep(ctx, inst, def, step.ID)
}

// recoverTimers scans all running instances for timer steps and ensures
// they are registered in Redis. This handles cases where Redis data was lost.
func (s *RecoveryService) recoverTimers(ctx context.Context) error {
	s.logger.Info().Msg("recovering unfired timers from running instances")

	offset := 0
	registered := 0

	for {
		instances, err := s.instanceRepo.ListRunning(ctx, s.batchSize, offset)
		if err != nil {
			return fmt.Errorf("listing running instances for timer recovery: %w", err)
		}

		if len(instances) == 0 {
			break
		}

		for _, inst := range instances {
			if inst.CurrentStepID == nil || *inst.CurrentStepID == "" {
				continue
			}

			def, err := s.defRepo.GetByID(ctx, inst.TenantID, inst.DefinitionID)
			if err != nil {
				s.logger.Error().Err(err).
					Str("instance_id", inst.ID).
					Msg("failed to load definition for timer recovery")
				continue
			}

			step := findStep(def.Steps, *inst.CurrentStepID)
			if step == nil || step.Type != model.StepTypeTimer {
				continue
			}

			// Check if timer already exists in Redis.
			member := inst.ID + ":" + step.ID
			score, err := s.rdb.ZScore(ctx, redisTimerKey, member).Result()
			if err == nil && score > 0 {
				continue // Timer already registered.
			}

			// Resolve fire time and register.
			fireAt, err := s.resolveTimerFireAt(step, inst)
			if err != nil {
				s.logger.Error().Err(err).
					Str("instance_id", inst.ID).
					Str("step_id", step.ID).
					Msg("failed to resolve timer fire_at during recovery")
				continue
			}

			timerScore := float64(fireAt.UnixMilli())
			if err := s.rdb.ZAdd(ctx, redisTimerKey, redis.Z{
				Score:  timerScore,
				Member: member,
			}).Err(); err != nil {
				s.logger.Error().Err(err).
					Str("instance_id", inst.ID).
					Str("step_id", step.ID).
					Msg("failed to re-register timer in Redis")
				continue
			}

			registered++
		}

		if len(instances) < s.batchSize {
			break
		}
		offset += s.batchSize
	}

	s.logger.Info().
		Int("timers_registered", registered).
		Msg("timer recovery completed")

	return nil
}

// resolveTimerFireAt computes the fire time for a timer step from its config.
// It supports two config formats:
//   - "fire_at": an RFC3339 timestamp string
//   - "duration": a Go duration string (e.g., "4h", "30m") relative to now
//     or relative to the step execution start time if available
func (s *RecoveryService) resolveTimerFireAt(step *model.StepDefinition, inst *model.WorkflowInstance) (time.Time, error) {
	// Try "fire_at" first (absolute time).
	if fireAtStr, ok := step.Config["fire_at"].(string); ok && fireAtStr != "" {
		// Check if it's a variable reference.
		if len(fireAtStr) > 3 && fireAtStr[:2] == "${" && fireAtStr[len(fireAtStr)-1] == '}' {
			// Resolve from instance context.
			path := fireAtStr[2 : len(fireAtStr)-1]
			resolved := resolveFromInstance(path, inst)
			if resolvedStr, ok := resolved.(string); ok {
				t, err := time.Parse(time.RFC3339, resolvedStr)
				if err != nil {
					return time.Time{}, fmt.Errorf("parsing resolved fire_at time %q: %w", resolvedStr, err)
				}
				return t, nil
			}
		}

		t, err := time.Parse(time.RFC3339, fireAtStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing fire_at time %q: %w", fireAtStr, err)
		}
		return t, nil
	}

	// Try "duration" (relative time).
	if durationStr, ok := step.Config["duration"].(string); ok && durationStr != "" {
		dur, err := time.ParseDuration(durationStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing timer duration %q: %w", durationStr, err)
		}
		// Calculate from instance start time as a stable reference.
		return inst.StartedAt.Add(dur), nil
	}

	return time.Time{}, fmt.Errorf("timer step %s has neither fire_at nor duration configured", step.ID)
}

// resolveFromInstance resolves a simple dotted path against instance data.
func resolveFromInstance(path string, inst *model.WorkflowInstance) interface{} {
	ctx := map[string]interface{}{
		"variables": inst.Variables,
		"steps":     inst.StepOutputs,
	}

	var triggerData map[string]interface{}
	if inst.TriggerData != nil {
		_ = json.Unmarshal(inst.TriggerData, &triggerData)
	}
	ctx["trigger"] = map[string]interface{}{
		"data": triggerData,
	}

	// Walk the path.
	segments := splitPath(path)
	var current interface{} = ctx
	for _, seg := range segments {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := m[seg]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}

// splitPath splits a dotted path into segments.
func splitPath(path string) []string {
	var segments []string
	current := ""
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		segments = append(segments, current)
	}
	return segments
}
