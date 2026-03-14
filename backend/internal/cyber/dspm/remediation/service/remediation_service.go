package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/engine"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// RemediationService is a thin service-layer facade over the RemediationEngine.
// It provides a stable API surface for handlers and other callers while delegating
// all business logic to the engine, which orchestrates playbook execution, policy
// enforcement, exception management, and external integrations.
type RemediationService struct {
	engine *engine.RemediationEngine
}

// New creates a RemediationService wrapping the given engine.
func New(eng *engine.RemediationEngine) *RemediationService {
	return &RemediationService{engine: eng}
}

// ── Remediation CRUD ────────────────────────────────────────────────────────────

// CreateRemediation creates a new remediation work item with playbook steps.
func (s *RemediationService) CreateRemediation(ctx context.Context, tenantID uuid.UUID, createdBy *uuid.UUID, req *dto.CreateRemediationRequest) (*model.Remediation, error) {
	return s.engine.CreateRemediation(ctx, tenantID, createdBy, req)
}

// GetRemediation returns a single remediation by ID.
func (s *RemediationService) GetRemediation(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.Remediation, error) {
	return s.engine.GetRemediation(ctx, tenantID, remediationID)
}

// ListRemediations returns a paginated, filtered list of remediations.
func (s *RemediationService) ListRemediations(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) ([]model.Remediation, int, error) {
	return s.engine.ListRemediations(ctx, tenantID, params)
}

// ── Remediation Lifecycle ───────────────────────────────────────────────────────

// ExecuteStep advances the remediation to the next playbook step.
func (s *RemediationService) ExecuteStep(ctx context.Context, tenantID, remediationID uuid.UUID, actorID *uuid.UUID) (*model.StepResult, error) {
	return s.engine.ExecuteStep(ctx, tenantID, remediationID, actorID)
}

// ApproveRemediation approves a remediation that requires human sign-off.
func (s *RemediationService) ApproveRemediation(ctx context.Context, tenantID, remediationID, approverID uuid.UUID) error {
	return s.engine.ApproveRemediation(ctx, tenantID, remediationID, approverID)
}

// CancelRemediation cancels an in-progress remediation.
func (s *RemediationService) CancelRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, actorID *uuid.UUID, reason string) error {
	return s.engine.CancelRemediation(ctx, tenantID, remediationID, actorID, reason)
}

// RollbackRemediation rolls back a completed remediation to its pre-action state.
func (s *RemediationService) RollbackRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, actorID *uuid.UUID, reason string) error {
	return s.engine.RollbackRemediation(ctx, tenantID, remediationID, actorID, reason)
}

// AssignRemediation assigns or reassigns a remediation to a user/team.
func (s *RemediationService) AssignRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, req *dto.AssignRemediationRequest) error {
	return s.engine.AssignRemediation(ctx, tenantID, remediationID, req)
}

// ── History & Stats ─────────────────────────────────────────────────────────────

// GetHistory returns the tamper-evident audit trail for a remediation.
func (s *RemediationService) GetHistory(ctx context.Context, tenantID, remediationID uuid.UUID, page, perPage int) ([]model.RemediationHistory, int, error) {
	return s.engine.GetHistory(ctx, tenantID, remediationID, page, perPage)
}

// GetStats returns remediation statistics (counts by status, severity, etc.).
func (s *RemediationService) GetStats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	return s.engine.GetStats(ctx, tenantID)
}

// GetDashboard returns the full remediation dashboard payload (KPIs, trends, top items).
func (s *RemediationService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*model.RemediationDashboard, error) {
	return s.engine.GetDashboard(ctx, tenantID)
}

// ── Policy Engine ───────────────────────────────────────────────────────────────

// EvaluatePolicies evaluates all enabled data policies for the tenant and returns violations.
func (s *RemediationService) EvaluatePolicies(ctx context.Context, tenantID uuid.UUID) ([]model.PolicyViolation, error) {
	return s.engine.EvaluatePolicies(ctx, tenantID)
}

// DryRun validates a playbook execution without side effects and returns impact analysis.
func (s *RemediationService) DryRun(ctx context.Context, tenantID uuid.UUID, playbookID string, assetID *uuid.UUID, identityID string) (*model.DryRunResult, error) {
	return s.engine.DryRun(ctx, tenantID, playbookID, assetID, identityID)
}

// ── Scheduled Operations ────────────────────────────────────────────────────────

// CheckSLABreaches detects remediations that have exceeded their SLA deadline.
func (s *RemediationService) CheckSLABreaches(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return s.engine.CheckSLABreaches(ctx, tenantID)
}

// CheckExceptionExpiry finds and expires risk exceptions past their expiry date.
func (s *RemediationService) CheckExceptionExpiry(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return s.engine.CheckExceptionExpiry(ctx, tenantID)
}
