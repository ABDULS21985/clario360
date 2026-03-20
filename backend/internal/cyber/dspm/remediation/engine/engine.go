package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/exception"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/integration"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/lifecycle"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/playbook"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/policy"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/repository"
	"github.com/clario360/platform/internal/events"
)

// RemediationEngine is the main orchestrator for the DSPM remediation module.
// It coordinates playbook execution, policy enforcement, SLA tracking, exception
// management, and integration with external systems (SIEM, ITSM, DLP).
type RemediationEngine struct {
	cfg Config

	// Repositories
	remRepo       *repository.RemediationRepository
	histRepo      *repository.HistoryRepository
	policyRepo    *repository.PolicyRepository
	exceptionRepo *repository.ExceptionRepository

	// Playbook components
	playbookRegistry  *playbook.Registry
	playbookExecutor  *playbook.PlaybookExecutor
	playbookValidator *playbook.Validator

	// Policy components
	policyEngine   *policy.PolicyEngine
	policyEnforcer *policy.Enforcer

	// Lifecycle components
	retentionEnforcer *lifecycle.RetentionEnforcer
	staleDetector     *lifecycle.StaleDataDetector

	// Exception components
	exceptionMgr  *exception.ExceptionManager
	expiryChecker *exception.ExpiryChecker

	// Integration components
	siemExporter  *integration.SIEMExporter
	itsmConnector *integration.ITSMConnector
	dlpGenerator  *integration.DLPPolicyGenerator

	// Event publishing
	producer *events.Producer

	logger zerolog.Logger
}

// NewRemediationEngine constructs a fully-wired RemediationEngine with all dependencies.
func NewRemediationEngine(
	cfg Config,
	remRepo *repository.RemediationRepository,
	histRepo *repository.HistoryRepository,
	policyRepo *repository.PolicyRepository,
	exceptionRepo *repository.ExceptionRepository,
	playbookRegistry *playbook.Registry,
	playbookExecutor *playbook.PlaybookExecutor,
	playbookValidator *playbook.Validator,
	policyEngine *policy.PolicyEngine,
	policyEnforcer *policy.Enforcer,
	retentionEnforcer *lifecycle.RetentionEnforcer,
	staleDetector *lifecycle.StaleDataDetector,
	exceptionMgr *exception.ExceptionManager,
	expiryChecker *exception.ExpiryChecker,
	siemExporter *integration.SIEMExporter,
	itsmConnector *integration.ITSMConnector,
	dlpGenerator *integration.DLPPolicyGenerator,
	producer *events.Producer,
	logger zerolog.Logger,
) *RemediationEngine {
	return &RemediationEngine{
		cfg:               cfg,
		remRepo:           remRepo,
		histRepo:          histRepo,
		policyRepo:        policyRepo,
		exceptionRepo:     exceptionRepo,
		playbookRegistry:  playbookRegistry,
		playbookExecutor:  playbookExecutor,
		playbookValidator: playbookValidator,
		policyEngine:      policyEngine,
		policyEnforcer:    policyEnforcer,
		retentionEnforcer: retentionEnforcer,
		staleDetector:     staleDetector,
		exceptionMgr:      exceptionMgr,
		expiryChecker:     expiryChecker,
		siemExporter:      siemExporter,
		itsmConnector:     itsmConnector,
		dlpGenerator:      dlpGenerator,
		producer:          producer,
		logger:            logger.With().Str("component", "remediation_engine").Logger(),
	}
}

// CreateRemediation validates the request, resolves the playbook from the registry,
// builds the remediation record with steps derived from the playbook, computes the
// SLA deadline, persists the record, records a "created" history entry, and publishes
// a creation event.
func (e *RemediationEngine) CreateRemediation(ctx context.Context, tenantID uuid.UUID, createdBy *uuid.UUID, req *dto.CreateRemediationRequest) (*model.Remediation, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("create remediation: validation failed: %w", err)
	}

	pb, ok := e.playbookRegistry.Get(req.PlaybookID)
	if !ok {
		return nil, fmt.Errorf("create remediation: playbook %q not found in registry", req.PlaybookID)
	}

	// Marshal playbook steps into JSON for the remediation's steps field.
	remSteps := make([]model.RemediationStep, len(pb.Steps))
	for i, step := range pb.Steps {
		remSteps[i] = model.RemediationStep{
			StepID:      step.ID,
			Order:       step.Order,
			Action:      string(step.Action),
			Description: step.Description,
			Params:      step.Params,
			Status:      "pending",
		}
	}

	stepsJSON, err := json.Marshal(remSteps)
	if err != nil {
		return nil, fmt.Errorf("create remediation: marshal steps: %w", err)
	}

	now := time.Now().UTC()
	slaDue := dto.SLADueAt(now, req.Severity)

	// Marshal compliance tags.
	var complianceJSON json.RawMessage
	if len(req.ComplianceTags) > 0 {
		complianceJSON, err = json.Marshal(req.ComplianceTags)
		if err != nil {
			complianceJSON = json.RawMessage("[]")
		}
	} else {
		complianceJSON = json.RawMessage("[]")
	}

	// Determine initial status based on playbook approval requirements.
	initialStatus := model.StatusOpen
	if pb.RequiresApproval {
		initialStatus = model.StatusAwaitingApproval
	}

	rem := &model.Remediation{
		ID:                uuid.New(),
		TenantID:          tenantID,
		FindingType:       model.FindingType(req.FindingType),
		FindingID:         req.FindingID,
		DataAssetID:       req.DataAssetID,
		DataAssetName:     req.DataAssetName,
		IdentityID:        req.IdentityID,
		PlaybookID:        req.PlaybookID,
		Title:             req.Title,
		Description:       req.Description,
		Severity:          req.Severity,
		Steps:             stepsJSON,
		CurrentStep:       0,
		TotalSteps:        len(pb.Steps),
		AssignedTo:        req.AssignedTo,
		AssignedTeam:      req.AssignedTeam,
		SLADueAt:          &slaDue,
		SLABreached:       false,
		RollbackAvailable: pb.AutoRollback,
		RolledBack:        false,
		Status:            initialStatus,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
		ComplianceTags:    complianceJSON,
	}

	created, err := e.remRepo.Create(ctx, rem)
	if err != nil {
		return nil, fmt.Errorf("create remediation: insert: %w", err)
	}
	rem = created

	// Record the creation in the tamper-evident history chain.
	_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionCreated, createdBy, model.ActorTypeUser, map[string]interface{}{
		"playbook_id":    req.PlaybookID,
		"finding_type":   req.FindingType,
		"severity":       req.Severity,
		"total_steps":    len(pb.Steps),
		"initial_status": string(initialStatus),
	})

	e.publishRemediationEvent(tenantID, "dspm.remediation.created", map[string]interface{}{
		"remediation_id": rem.ID.String(),
		"playbook_id":    rem.PlaybookID,
		"finding_type":   string(rem.FindingType),
		"severity":       rem.Severity,
		"status":         string(rem.Status),
	})

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("remediation_id", rem.ID.String()).
		Str("playbook_id", rem.PlaybookID).
		Str("status", string(rem.Status)).
		Int("total_steps", rem.TotalSteps).
		Msg("remediation created")

	return rem, nil
}

// ExecuteStep executes the current step of a remediation via the playbook executor.
// It validates that the remediation is in an actionable state (open or in_progress),
// runs the step, updates the remediation's progress, records history, and publishes
// events. If all steps complete successfully, the remediation is marked completed.
func (e *RemediationEngine) ExecuteStep(ctx context.Context, tenantID, remediationID uuid.UUID, actorID *uuid.UUID) (*model.StepResult, error) {
	rem, err := e.remRepo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, fmt.Errorf("execute step: get remediation: %w", err)
	}

	// Validate the remediation is in an actionable state.
	if rem.Status != model.StatusOpen && rem.Status != model.StatusInProgress {
		return nil, fmt.Errorf("execute step: remediation status is %q; must be open or in_progress", rem.Status)
	}

	if rem.CurrentStep >= rem.TotalSteps {
		return nil, fmt.Errorf("execute step: all %d steps already completed", rem.TotalSteps)
	}

	// Resolve the playbook to get step definitions.
	pb, ok := e.playbookRegistry.Get(rem.PlaybookID)
	if !ok {
		return nil, fmt.Errorf("execute step: playbook %q not found", rem.PlaybookID)
	}

	// Execute the current step.
	stepResult, err := e.playbookExecutor.Execute(ctx, pb, rem.CurrentStep)
	if err != nil {
		return nil, fmt.Errorf("execute step: executor error: %w", err)
	}

	// Unmarshal existing steps, update the current step, and re-marshal.
	var remSteps []model.RemediationStep
	if err := json.Unmarshal(rem.Steps, &remSteps); err != nil {
		return nil, fmt.Errorf("execute step: unmarshal steps: %w", err)
	}

	if rem.CurrentStep < len(remSteps) {
		remSteps[rem.CurrentStep].Status = string(stepResult.Status)
		remSteps[rem.CurrentStep].StartedAt = &stepResult.StartedAt
		remSteps[rem.CurrentStep].CompletedAt = stepResult.CompletedAt
		remSteps[rem.CurrentStep].Result = stepResult.Result
		if stepResult.Error != "" {
			remSteps[rem.CurrentStep].Error = stepResult.Error
		}
	}

	updatedStepsJSON, err := json.Marshal(remSteps)
	if err != nil {
		return nil, fmt.Errorf("execute step: marshal updated steps: %w", err)
	}

	now := time.Now().UTC()
	rem.Steps = updatedStepsJSON
	rem.UpdatedAt = now

	// Record step history.
	histAction := model.HistoryActionStepCompleted
	if stepResult.Status == model.StepStatusFailed {
		histAction = model.HistoryActionStepFailed
	}

	_ = e.recordHistory(ctx, tenantID, rem.ID, histAction, actorID, model.ActorTypeUser, map[string]interface{}{
		"step_id":     stepResult.StepID,
		"step_index":  rem.CurrentStep,
		"action":      stepResult.Action,
		"status":      string(stepResult.Status),
		"duration_ms": stepResult.DurationMs,
		"error":       stepResult.Error,
	})

	// Advance to the next step on success.
	if stepResult.Status == model.StepStatusCompleted {
		rem.CurrentStep++
	}

	// If the step failed with abort handling, mark the remediation as failed.
	if stepResult.Status == model.StepStatusFailed && rem.CurrentStep < len(pb.Steps) {
		step := pb.Steps[rem.CurrentStep]
		if step.FailureHandling == model.FailureHandlingAbort {
			rem.Status = model.StatusFailed
			rem.UpdatedAt = now

			_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionStatusChanged, actorID, model.ActorTypeSystem, map[string]interface{}{
				"from_status": string(model.StatusInProgress),
				"to_status":   string(model.StatusFailed),
				"reason":      fmt.Sprintf("step %q failed with abort handling", step.ID),
			})

			e.publishRemediationEvent(tenantID, "dspm.remediation.failed", map[string]interface{}{
				"remediation_id": rem.ID.String(),
				"failed_step":    stepResult.StepID,
				"error":          stepResult.Error,
			})
		} else if step.FailureHandling == model.FailureHandlingSkip {
			// Skip the failed step and advance.
			rem.CurrentStep++
		}
		// For retry handling, the step remains at the current index for the caller to retry.
	}

	// Transition from open to in_progress on first successful step.
	if rem.Status == model.StatusOpen && rem.CurrentStep > 0 {
		rem.Status = model.StatusInProgress

		_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionStatusChanged, actorID, model.ActorTypeSystem, map[string]interface{}{
			"from_status": string(model.StatusOpen),
			"to_status":   string(model.StatusInProgress),
		})
	}

	// Check if all steps are completed.
	if rem.CurrentStep >= rem.TotalSteps && rem.Status != model.StatusFailed {
		rem.Status = model.StatusCompleted
		completedAt := now
		rem.CompletedAt = &completedAt

		_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionStatusChanged, actorID, model.ActorTypeSystem, map[string]interface{}{
			"from_status": string(model.StatusInProgress),
			"to_status":   string(model.StatusCompleted),
			"total_steps": rem.TotalSteps,
		})

		e.publishRemediationEvent(tenantID, "dspm.remediation.completed", map[string]interface{}{
			"remediation_id": rem.ID.String(),
			"playbook_id":    rem.PlaybookID,
			"total_steps":    rem.TotalSteps,
		})

		e.logger.Info().
			Str("tenant_id", tenantID.String()).
			Str("remediation_id", rem.ID.String()).
			Msg("remediation completed all steps")
	}

	// Persist updates.
	if updateErr := e.remRepo.Update(ctx, rem); updateErr != nil {
		e.logger.Error().Err(updateErr).
			Str("remediation_id", rem.ID.String()).
			Msg("failed to persist remediation updates after step execution")
		return stepResult, fmt.Errorf("execute step: persist updates: %w", updateErr)
	}

	e.publishRemediationEvent(tenantID, "dspm.remediation.step_executed", map[string]interface{}{
		"remediation_id": rem.ID.String(),
		"step_id":        stepResult.StepID,
		"step_status":    string(stepResult.Status),
		"current_step":   rem.CurrentStep,
		"total_steps":    rem.TotalSteps,
	})

	return stepResult, nil
}

// ApproveRemediation transitions a remediation from awaiting_approval to in_progress.
func (e *RemediationEngine) ApproveRemediation(ctx context.Context, tenantID, remediationID, approverID uuid.UUID) error {
	rem, err := e.remRepo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return fmt.Errorf("approve remediation: get: %w", err)
	}

	if rem.Status != model.StatusAwaitingApproval {
		return fmt.Errorf("approve remediation: status is %q; must be awaiting_approval", rem.Status)
	}

	now := time.Now().UTC()
	rem.Status = model.StatusInProgress
	rem.UpdatedAt = now

	if err := e.remRepo.Update(ctx, rem); err != nil {
		return fmt.Errorf("approve remediation: update: %w", err)
	}

	_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionStatusChanged, &approverID, model.ActorTypeUser, map[string]interface{}{
		"from_status": string(model.StatusAwaitingApproval),
		"to_status":   string(model.StatusInProgress),
		"approved_by": approverID.String(),
	})

	e.publishRemediationEvent(tenantID, "dspm.remediation.approved", map[string]interface{}{
		"remediation_id": rem.ID.String(),
		"approved_by":    approverID.String(),
	})

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("remediation_id", rem.ID.String()).
		Str("approver_id", approverID.String()).
		Msg("remediation approved")

	return nil
}

// CancelRemediation sets the remediation status to cancelled and records the reason.
func (e *RemediationEngine) CancelRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, actorID *uuid.UUID, reason string) error {
	rem, err := e.remRepo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return fmt.Errorf("cancel remediation: get: %w", err)
	}

	if rem.Status.IsTerminal() {
		return fmt.Errorf("cancel remediation: cannot cancel a remediation in terminal status %q", rem.Status)
	}

	previousStatus := rem.Status
	now := time.Now().UTC()
	rem.Status = model.StatusCancelled
	rem.UpdatedAt = now

	if err := e.remRepo.Update(ctx, rem); err != nil {
		return fmt.Errorf("cancel remediation: update: %w", err)
	}

	_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionStatusChanged, actorID, model.ActorTypeUser, map[string]interface{}{
		"from_status": string(previousStatus),
		"to_status":   string(model.StatusCancelled),
		"reason":      reason,
	})

	e.publishRemediationEvent(tenantID, "dspm.remediation.cancelled", map[string]interface{}{
		"remediation_id": rem.ID.String(),
		"reason":         reason,
	})

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("remediation_id", rem.ID.String()).
		Str("reason", reason).
		Msg("remediation cancelled")

	return nil
}

// RollbackRemediation validates that rollback is available for the remediation,
// sets the status to rolled_back, and records the rollback in the history chain
// with the pre-action state for auditability.
func (e *RemediationEngine) RollbackRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, actorID *uuid.UUID, reason string) error {
	rem, err := e.remRepo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return fmt.Errorf("rollback remediation: get: %w", err)
	}

	if !rem.RollbackAvailable {
		return fmt.Errorf("rollback remediation: rollback is not available for this remediation")
	}

	previousStatus := rem.Status
	now := time.Now().UTC()
	rem.Status = model.StatusRolledBack
	rem.RolledBack = true
	rem.UpdatedAt = now

	if err := e.remRepo.Update(ctx, rem); err != nil {
		return fmt.Errorf("rollback remediation: update: %w", err)
	}

	// Include pre_action_state in the history for forensic review.
	historyDetails := map[string]interface{}{
		"from_status": string(previousStatus),
		"to_status":   string(model.StatusRolledBack),
		"reason":      reason,
	}
	if len(rem.PreActionState) > 0 {
		historyDetails["pre_action_state"] = json.RawMessage(rem.PreActionState)
	}

	_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionRolledBack, actorID, model.ActorTypeUser, historyDetails)

	e.publishRemediationEvent(tenantID, "dspm.remediation.rolled_back", map[string]interface{}{
		"remediation_id": rem.ID.String(),
		"reason":         reason,
	})

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("remediation_id", rem.ID.String()).
		Str("reason", reason).
		Msg("remediation rolled back")

	return nil
}

// AssignRemediation updates the assignment of a remediation to a user or team.
func (e *RemediationEngine) AssignRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, req *dto.AssignRemediationRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("assign remediation: validation failed: %w", err)
	}

	rem, err := e.remRepo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return fmt.Errorf("assign remediation: get: %w", err)
	}

	now := time.Now().UTC()
	previousAssignedTo := rem.AssignedTo
	previousAssignedTeam := rem.AssignedTeam

	rem.AssignedTo = req.AssignedTo
	rem.AssignedTeam = req.AssignedTeam
	rem.UpdatedAt = now

	if err := e.remRepo.Update(ctx, rem); err != nil {
		return fmt.Errorf("assign remediation: update: %w", err)
	}

	details := map[string]interface{}{
		"assigned_team": req.AssignedTeam,
	}
	if req.AssignedTo != nil {
		details["assigned_to"] = req.AssignedTo.String()
	}
	if previousAssignedTo != nil {
		details["previous_assigned_to"] = previousAssignedTo.String()
	}
	if previousAssignedTeam != "" {
		details["previous_assigned_team"] = previousAssignedTeam
	}

	var actorID *uuid.UUID
	if req.AssignedTo != nil {
		actorID = req.AssignedTo
	}

	_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionAssigned, actorID, model.ActorTypeUser, details)

	e.publishRemediationEvent(tenantID, "dspm.remediation.assigned", map[string]interface{}{
		"remediation_id": rem.ID.String(),
		"assigned_team":  req.AssignedTeam,
	})

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("remediation_id", rem.ID.String()).
		Str("assigned_team", req.AssignedTeam).
		Msg("remediation assigned")

	return nil
}

// GetRemediation retrieves a single remediation by ID with tenant isolation.
func (e *RemediationEngine) GetRemediation(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.Remediation, error) {
	rem, err := e.remRepo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, fmt.Errorf("get remediation: %w", err)
	}
	return rem, nil
}

// ListRemediations returns a paginated, filtered list of remediations for a tenant.
func (e *RemediationEngine) ListRemediations(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) ([]model.Remediation, int, error) {
	params.SetDefaults()
	remediations, total, err := e.remRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list remediations: %w", err)
	}
	return remediations, total, nil
}

// GetHistory returns the paginated audit trail for a remediation.
func (e *RemediationEngine) GetHistory(ctx context.Context, tenantID, remediationID uuid.UUID, page, perPage int) ([]model.RemediationHistory, int, error) {
	entries, total, err := e.histRepo.ListByRemediation(ctx, tenantID, remediationID, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("get history: %w", err)
	}
	return entries, total, nil
}

// GetStats returns aggregated remediation statistics for a tenant.
func (e *RemediationEngine) GetStats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	stats, err := e.remRepo.Stats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return stats, nil
}

// GetDashboard assembles the full remediation dashboard by combining statistics,
// recent remediations, and burndown chart data into a single response.
func (e *RemediationEngine) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*model.RemediationDashboard, error) {
	stats, err := e.remRepo.Stats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: stats: %w", err)
	}

	recentParams := &dto.RemediationListParams{
		Sort:    "created_at",
		Order:   "desc",
		Page:    1,
		PerPage: 10,
	}
	recent, _, err := e.remRepo.List(ctx, tenantID, recentParams)
	if err != nil {
		e.logger.Warn().Err(err).Msg("get dashboard: failed to load recent remediations; using empty list")
		recent = []model.Remediation{}
	}

	burndown, err := e.remRepo.BurndownData(ctx, tenantID, 30)
	if err != nil {
		e.logger.Warn().Err(err).Msg("get dashboard: failed to load burndown data; using empty list")
		burndown = []model.BurndownDataPoint{}
	}

	dashboard := &model.RemediationDashboard{
		Stats:              *stats,
		RecentRemediations: recent,
		BurndownData:       burndown,
	}

	return dashboard, nil
}

// DryRun performs a pre-execution validation of a playbook against a remediation
// target without making any changes. Returns a DryRunResult with validation issues,
// affected scope estimates, and estimated risk reduction.
func (e *RemediationEngine) DryRun(ctx context.Context, tenantID uuid.UUID, playbookID string, assetID *uuid.UUID, identityID string) (*model.DryRunResult, error) {
	result, err := e.playbookValidator.DryRun(ctx, playbookID, assetID, identityID)
	if err != nil {
		return nil, fmt.Errorf("dry run: %w", err)
	}
	return result, nil
}

// CheckSLABreaches scans for remediations that have exceeded their SLA deadline
// but have not yet been marked as breached. For each breach found, it marks the
// remediation, records a history entry, and publishes a breach event. Returns
// the number of newly detected breaches.
func (e *RemediationEngine) CheckSLABreaches(ctx context.Context, tenantID uuid.UUID) (int, error) {
	breached, err := e.remRepo.FindSLABreached(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("check sla breaches: find breached: %w", err)
	}

	if len(breached) == 0 {
		return 0, nil
	}

	count := 0
	for i := range breached {
		rem := &breached[i]

		if markErr := e.remRepo.MarkSLABreached(ctx, tenantID, rem.ID); markErr != nil {
			e.logger.Error().Err(markErr).
				Str("remediation_id", rem.ID.String()).
				Msg("failed to mark SLA breach")
			continue
		}

		_ = e.recordHistory(ctx, tenantID, rem.ID, model.HistoryActionSLABreached, nil, model.ActorTypeScheduler, map[string]interface{}{
			"sla_due_at": rem.SLADueAt,
			"severity":   rem.Severity,
			"status":     string(rem.Status),
		})

		e.publishRemediationEvent(tenantID, "dspm.remediation.sla_breached", map[string]interface{}{
			"remediation_id": rem.ID.String(),
			"severity":       rem.Severity,
			"sla_due_at":     rem.SLADueAt,
		})

		count++

		e.logger.Warn().
			Str("tenant_id", tenantID.String()).
			Str("remediation_id", rem.ID.String()).
			Str("severity", rem.Severity).
			Msg("SLA breach detected")
	}

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("breaches_detected", count).
		Msg("SLA breach check complete")

	return count, nil
}

// EvaluatePolicies retrieves all enabled policies for a tenant, evaluates them
// against the data asset inventory, and for policies with auto_remediate enforcement,
// automatically creates remediation work items. Returns all detected violations.
func (e *RemediationEngine) EvaluatePolicies(ctx context.Context, tenantID uuid.UUID) ([]model.PolicyViolation, error) {
	policies, err := e.policyRepo.ListEnabled(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("evaluate policies: list enabled: %w", err)
	}

	if len(policies) == 0 {
		return []model.PolicyViolation{}, nil
	}

	violations, err := e.policyEngine.EvaluateAll(ctx, tenantID, policies)
	if err != nil {
		return nil, fmt.Errorf("evaluate policies: evaluation: %w", err)
	}

	// Build a map from policy ID to the policy for quick lookup of AutoPlaybookID.
	policyMap := make(map[uuid.UUID]*model.DataPolicy, len(policies))
	for i := range policies {
		policyMap[policies[i].ID] = &policies[i]
	}

	// Update last_evaluated_at for each policy.
	violationsByPolicy := make(map[uuid.UUID]int)
	for _, v := range violations {
		violationsByPolicy[v.PolicyID]++
	}
	for _, pol := range policies {
		count := violationsByPolicy[pol.ID]
		if updateErr := e.policyRepo.UpdateEvaluationResults(ctx, tenantID, pol.ID, count); updateErr != nil {
			e.logger.Error().Err(updateErr).
				Str("policy_id", pol.ID.String()).
				Msg("failed to update policy last_evaluated_at")
		}
	}

	// For auto_remediate violations, create remediations automatically.
	if e.cfg.EnableAutoRemediation {
		for i := range violations {
			v := &violations[i]
			pol, ok := policyMap[v.PolicyID]
			if !ok {
				continue
			}

			action := e.policyEnforcer.DetermineActionWithPlaybook(v, pol.Enforcement, pol.AutoPlaybookID)
			if !action.CreateRemediation || action.PlaybookID == "" {
				continue
			}

			// Create auto-remediation.
			req := &dto.CreateRemediationRequest{
				FindingType:   string(model.FindingPolicyViolation),
				DataAssetID:   &v.AssetID,
				DataAssetName: v.AssetName,
				PlaybookID:    action.PlaybookID,
				Title:         fmt.Sprintf("Auto-remediation: %s - %s", v.PolicyName, v.AssetName),
				Description:   v.Description,
				Severity:      v.Severity,
			}
			if len(v.ComplianceFrameworks) > 0 {
				req.ComplianceTags = v.ComplianceFrameworks
			}

			if _, createErr := e.CreateRemediation(ctx, tenantID, nil, req); createErr != nil {
				e.logger.Error().Err(createErr).
					Str("policy_id", v.PolicyID.String()).
					Str("asset_id", v.AssetID.String()).
					Str("playbook_id", action.PlaybookID).
					Msg("failed to create auto-remediation for policy violation")
			} else {
				e.logger.Info().
					Str("policy_id", v.PolicyID.String()).
					Str("asset_id", v.AssetID.String()).
					Str("playbook_id", action.PlaybookID).
					Msg("auto-remediation created for policy violation")
			}
		}
	}

	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("policies_evaluated", len(policies)).
		Int("violations_found", len(violations)).
		Msg("policy evaluation complete")

	return violations, nil
}

// CheckExceptionExpiry delegates to the ExpiryChecker to find and expire
// risk exceptions that have passed their expiry date. Returns the number
// of exceptions that were expired.
func (e *RemediationEngine) CheckExceptionExpiry(ctx context.Context, tenantID uuid.UUID) (int, error) {
	count, err := e.expiryChecker.Run(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("check exception expiry: %w", err)
	}

	if count > 0 {
		e.publishRemediationEvent(tenantID, "dspm.exception.expired", map[string]interface{}{
			"expired_count": count,
		})
	}

	return count, nil
}

// recordHistory appends a new entry to the tamper-evident audit trail for a
// remediation. It fetches the last entry to compute the hash chain, serializes
// the details, computes the entry hash via model.ComputeEntryHash, and persists
// the entry.
func (e *RemediationEngine) recordHistory(
	ctx context.Context,
	tenantID, remediationID uuid.UUID,
	action model.HistoryAction,
	actorID *uuid.UUID,
	actorType model.ActorType,
	details interface{},
) error {
	// Serialize the details to JSON.
	var detailsJSON json.RawMessage
	if details != nil {
		d, err := json.Marshal(details)
		if err != nil {
			e.logger.Error().Err(err).Msg("record history: failed to marshal details")
			detailsJSON = json.RawMessage("{}")
		} else {
			detailsJSON = d
		}
	} else {
		detailsJSON = json.RawMessage("{}")
	}

	// Retrieve the last history entry to compute the hash chain.
	var prevHash string
	lastEntry, err := e.histRepo.GetLastEntry(ctx, tenantID, remediationID)
	if err != nil {
		e.logger.Error().Err(err).
			Str("remediation_id", remediationID.String()).
			Msg("record history: failed to get last entry for hash chain; using empty prev_hash")
	} else if lastEntry != nil {
		prevHash = lastEntry.EntryHash
	}

	now := time.Now().UTC()
	entryHash := model.ComputeEntryHash(prevHash, action, detailsJSON, now)

	entry := &model.RemediationHistory{
		ID:            uuid.New(),
		TenantID:      tenantID,
		RemediationID: remediationID,
		Action:        action,
		ActorID:       actorID,
		ActorType:     actorType,
		Details:       detailsJSON,
		EntryHash:     entryHash,
		PrevHash:      prevHash,
		CreatedAt:     now,
	}

	if _, insertErr := e.histRepo.Insert(ctx, entry); insertErr != nil {
		e.logger.Error().Err(insertErr).
			Str("remediation_id", remediationID.String()).
			Str("action", string(action)).
			Msg("record history: failed to insert history entry")
		return fmt.Errorf("record history: insert: %w", insertErr)
	}

	return nil
}

// publishRemediationEvent wraps the event producer to publish DSPM remediation
// events. It silently returns if the producer is nil (e.g. in test environments).
func (e *RemediationEngine) publishRemediationEvent(tenantID uuid.UUID, eventType string, data interface{}) {
	if e.producer == nil {
		return
	}

	event, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), data)
	if err != nil {
		e.logger.Error().Err(err).
			Str("event_type", eventType).
			Msg("publish event: failed to create event")
		return
	}

	if pubErr := e.producer.Publish(context.Background(), events.Topics.DSPMEvents, event); pubErr != nil {
		e.logger.Error().Err(pubErr).
			Str("event_type", eventType).
			Str("topic", events.Topics.DSPMEvents).
			Msg("publish event: failed to publish")
	}
}
