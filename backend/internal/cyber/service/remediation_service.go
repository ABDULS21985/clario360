package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/remediation"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type RemediationService struct {
	repo      *repository.RemediationRepository
	auditRepo *repository.RemediationAuditRepository
	assetRepo *repository.AssetRepository
	executor  *remediation.RemediationExecutor
	audit     *remediation.AuditTrail
	producer  *events.Producer
	logger    zerolog.Logger
}

func NewRemediationService(
	repo *repository.RemediationRepository,
	auditRepo *repository.RemediationAuditRepository,
	assetRepo *repository.AssetRepository,
	executor *remediation.RemediationExecutor,
	audit *remediation.AuditTrail,
	producer *events.Producer,
	logger zerolog.Logger,
) *RemediationService {
	return &RemediationService{
		repo:      repo,
		auditRepo: auditRepo,
		assetRepo: assetRepo,
		executor:  executor,
		audit:     audit,
		producer:  producer,
		logger:    logger.With().Str("service", "remediation").Logger(),
	}
}

func (s *RemediationService) Create(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, req *dto.CreateRemediationRequest) (*model.RemediationAction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.ensureAssetsExist(ctx, tenantID, req.AffectedAssetIDs); err != nil {
		return nil, err
	}
	action, err := s.repo.Create(ctx, tenantID, userID, safeActorLabel(actor), req)
	if err != nil {
		return nil, err
	}
	s.audit.RecordAction(ctx, tenantID, action.ID, "created", actorUUID(actor), safeActorLabel(actor), map[string]interface{}{
		"type":                 action.Type,
		"severity":             action.Severity,
		"affected_asset_count": action.AffectedAssetCount,
	})
	_ = publishEvent(ctx, s.producer, events.Topics.RemediationEvents, "com.clario360.cyber.remediation.created", tenantID, actor, map[string]interface{}{
		"id":                   action.ID.String(),
		"type":                 string(action.Type),
		"title":                action.Title,
		"severity":             action.Severity,
		"affected_asset_count": action.AffectedAssetCount,
	})
	return s.Get(ctx, tenantID, action.ID)
}

func (s *RemediationService) List(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) (*dto.RemediationListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.repo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	return &dto.RemediationListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

func (s *RemediationService) Get(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationAction, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	auditEntries, err := s.auditRepo.ListByRemediation(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	action.AuditTrail = auditEntries
	return action, nil
}

func (s *RemediationService) Update(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.UpdateRemediationRequest) (*model.RemediationAction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	switch action.Status {
	case model.StatusDraft, model.StatusRevisionRequested:
	case model.StatusRejected:
		if action.CreatedBy != actorID {
			return nil, fmt.Errorf("%w: only the original creator can revise a rejected remediation", remediation.ErrInsufficientPermission)
		}
		if err := remediation.ValidateTransition(action, model.StatusDraft, actorRole); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: remediation can only be updated while draft, rejected, or revision_requested", remediation.ErrInvalidTransition)
	}

	if req.AffectedAssetIDs != nil {
		if err := s.ensureAssetsExist(ctx, tenantID, req.AffectedAssetIDs); err != nil {
			return nil, err
		}
	}
	updated, err := s.repo.Update(ctx, tenantID, remediationID, req)
	if err != nil {
		return nil, err
	}
	if action.Status == model.StatusRejected {
		if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusDraft, map[string]interface{}{}); err != nil {
			return nil, err
		}
		s.audit.RecordTransition(ctx, tenantID, remediationID, "revise", &actorID, actorName, model.StatusRejected, model.StatusDraft, nil)
	}
	s.audit.RecordAction(ctx, tenantID, remediationID, "updated", &actorID, actorName, map[string]interface{}{})
	return s.Get(ctx, tenantID, updated.ID)
}

func (s *RemediationService) Delete(ctx context.Context, tenantID, remediationID uuid.UUID, actor *Actor) error {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return err
	}
	if !remediation.IsPreExecutionStatus(action.Status) {
		return fmt.Errorf("%w: only pre-execution remediations can be cancelled", remediation.ErrInvalidTransition)
	}
	if err := s.repo.SoftDelete(ctx, tenantID, remediationID); err != nil {
		return err
	}
	s.audit.RecordAction(ctx, tenantID, remediationID, "deleted", actorUUID(actor), safeActorLabel(actor), nil)
	return nil
}

func (s *RemediationService) Submit(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if action.Status == model.StatusRevisionRequested && action.CreatedBy != actorID {
		return nil, fmt.Errorf("%w: only the original creator can resubmit a revision-requested remediation", remediation.ErrInsufficientPermission)
	}
	if err := remediation.ValidateTransition(action, model.StatusPendingApproval, actorRole); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusPendingApproval, map[string]interface{}{
		"submitted_by": actorID,
		"submitted_at": now,
	}); err != nil {
		return nil, err
	}
	eventType := "com.clario360.cyber.remediation.submitted"
	actionName := "submit"
	if action.Status == model.StatusRevisionRequested {
		actionName = "resubmit"
	}
	s.audit.RecordTransition(ctx, tenantID, remediationID, actionName, &actorID, actorName, action.Status, model.StatusPendingApproval, nil)
	_ = publishEvent(ctx, s.producer, events.Topics.RemediationEvents, eventType, tenantID, &Actor{UserID: actorID, UserName: actorName}, map[string]interface{}{
		"id":           remediationID.String(),
		"submitted_by": actorID.String(),
	})
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) Approve(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ApproveRemediationRequest) (*model.RemediationAction, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if err := remediation.ValidateTransition(action, model.StatusApproved, actorRole); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusApproved, map[string]interface{}{
		"approved_by":    actorID,
		"approved_at":    now,
		"approval_notes": nullableString(req.Notes),
	}); err != nil {
		return nil, err
	}
	s.audit.RecordTransition(ctx, tenantID, remediationID, "approve", &actorID, actorName, action.Status, model.StatusApproved, map[string]interface{}{"notes": req.Notes})
	_ = publishEvent(ctx, s.producer, events.Topics.RemediationEvents, "com.clario360.cyber.remediation.approved", tenantID, &Actor{UserID: actorID, UserName: actorName}, map[string]interface{}{
		"id":          remediationID.String(),
		"approved_by": actorID.String(),
	})
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) Reject(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RejectRemediationRequest) (*model.RemediationAction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if err := remediation.ValidateTransition(action, model.StatusRejected, actorRole); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusRejected, map[string]interface{}{
		"rejected_by":      actorID,
		"rejected_at":      now,
		"rejection_reason": req.Reason,
	}); err != nil {
		return nil, err
	}
	s.audit.RecordTransition(ctx, tenantID, remediationID, "reject", &actorID, actorName, action.Status, model.StatusRejected, map[string]interface{}{"reason": req.Reason})
	_ = publishEvent(ctx, s.producer, events.Topics.RemediationEvents, "com.clario360.cyber.remediation.rejected", tenantID, &Actor{UserID: actorID, UserName: actorName}, map[string]interface{}{
		"id":          remediationID.String(),
		"rejected_by": actorID.String(),
		"reason":      req.Reason,
	})
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) RequestRevision(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RequestRevisionRequest) (*model.RemediationAction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if err := remediation.ValidateTransition(action, model.StatusRevisionRequested, actorRole); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusRevisionRequested, map[string]interface{}{
		"approval_notes": req.Notes,
	}); err != nil {
		return nil, err
	}
	s.audit.RecordTransition(ctx, tenantID, remediationID, "request_revision", &actorID, actorName, action.Status, model.StatusRevisionRequested, map[string]interface{}{"notes": req.Notes})
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) DryRun(ctx context.Context, tenantID, remediationID uuid.UUID, actorID uuid.UUID, actorName, actorRole string) (*model.DryRunResult, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	target := model.StatusDryRunRunning
	if err := remediation.ValidateTransition(action, target, actorRole); err != nil {
		return nil, err
	}
	return s.executor.DryRun(ctx, action, &actorID, actorName)
}

func (s *RemediationService) GetDryRun(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.DryRunResult, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if action.DryRunResult == nil {
		return &model.DryRunResult{
			Success:          false,
			SimulatedChanges: []model.SimulatedChange{},
			Warnings:         []string{},
			Blockers:         []string{},
			AffectedServices: []string{},
		}, nil
	}
	return action.DryRunResult, nil
}

func (s *RemediationService) Execute(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ExecuteRemediationRequest) (*model.RemediationAction, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	switch action.Status {
	case model.StatusDraft, model.StatusPendingApproval, model.StatusRejected, model.StatusRevisionRequested:
		return nil, fmt.Errorf("%w: approval is required before execution", remediation.ErrInsufficientPermission)
	}
	if action.ApprovedBy == nil || action.ApprovedAt == nil {
		return nil, fmt.Errorf("%w: approval is required before execution", remediation.ErrInsufficientPermission)
	}
	if action.DryRunResult == nil || action.DryRunAt == nil {
		return nil, fmt.Errorf("%w: dry-run must be completed before execution", remediation.ErrPreConditionFailed)
	}
	if !action.DryRunResult.Success {
		return nil, fmt.Errorf("%w: cannot execute: dry-run reported failures. Fix issues and re-run dry-run.", remediation.ErrPreConditionFailed)
	}
	if action.Type == model.RemediationTypeCustom {
		if req == nil || req.ManualConfirmation == nil || strings.TrimSpace(*req.ManualConfirmation) != "I have manually performed the remediation steps." {
			return nil, fmt.Errorf("%w: manual confirmation is required for custom remediations", remediation.ErrPreConditionFailed)
		}
	}
	if action.Status == model.StatusDryRunCompleted {
		if err := remediation.ValidateTransition(action, model.StatusExecutionPending, actorRole); err != nil {
			return nil, err
		}
		if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusExecutionPending, map[string]interface{}{}); err != nil {
			return nil, err
		}
		s.audit.RecordTransition(ctx, tenantID, remediationID, "queue_execution", &actorID, actorName, action.Status, model.StatusExecutionPending, nil)
		action.Status = model.StatusExecutionPending
	}
	if _, err := s.executor.Execute(ctx, action, actorID, actorName); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) Verify(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.VerifyRemediationRequest) (*model.RemediationAction, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if action.Type == model.RemediationTypeCustom {
		if req == nil || req.ManualConfirmation == nil || strings.TrimSpace(*req.ManualConfirmation) == "" {
			return nil, fmt.Errorf("%w: manual verification confirmation is required for custom remediations", remediation.ErrPreConditionFailed)
		}
	}
	if action.Status == model.StatusExecuted {
		if err := remediation.ValidateTransition(action, model.StatusVerificationPending, actorRole); err != nil {
			return nil, err
		}
	}
	if _, err := s.executor.Verify(ctx, action, &actorID, actorName); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) Rollback(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RollbackRequest) (*model.RemediationAction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if action.Status != model.StatusRollbackPending {
		if err := remediation.ValidateTransition(action, model.StatusRollbackPending, actorRole); err != nil {
			return nil, err
		}
		if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusRollbackPending, map[string]interface{}{
			"rollback_reason": req.Reason,
		}); err != nil {
			return nil, err
		}
		s.audit.RecordTransition(ctx, tenantID, remediationID, "request_rollback", &actorID, actorName, action.Status, model.StatusRollbackPending, map[string]interface{}{"reason": req.Reason})
		_ = publishEvent(ctx, s.producer, events.Topics.RemediationEvents, "com.clario360.cyber.remediation.rollback_requested", tenantID, &Actor{UserID: actorID, UserName: actorName}, map[string]interface{}{
			"id":           remediationID.String(),
			"reason":       req.Reason,
			"requested_by": actorID.String(),
		})
		action.Status = model.StatusRollbackPending
	}

	if !roleAtLeast(actorRole, "security_manager") {
		return s.Get(ctx, tenantID, remediationID)
	}
	refreshed, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if err := remediation.ValidateTransition(refreshed, model.StatusRollingBack, actorRole); err != nil {
		return nil, err
	}
	if err := s.executor.Rollback(ctx, refreshed, req.Reason, actorID, actorName); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) Close(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error) {
	action, err := s.repo.GetByID(ctx, tenantID, remediationID)
	if err != nil {
		return nil, err
	}
	if err := remediation.ValidateTransition(action, model.StatusClosed, actorRole); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, tenantID, remediationID, model.StatusClosed, map[string]interface{}{}); err != nil {
		return nil, err
	}
	s.audit.RecordTransition(ctx, tenantID, remediationID, "close", &actorID, actorName, action.Status, model.StatusClosed, nil)
	_ = publishEvent(ctx, s.producer, events.Topics.RemediationEvents, "com.clario360.cyber.remediation.closed", tenantID, &Actor{UserID: actorID, UserName: actorName}, map[string]interface{}{
		"id": remediationID.String(),
	})
	return s.Get(ctx, tenantID, remediationID)
}

func (s *RemediationService) AuditTrail(ctx context.Context, tenantID, remediationID uuid.UUID) ([]model.RemediationAuditEntry, error) {
	return s.auditRepo.ListByRemediation(ctx, tenantID, remediationID)
}

func (s *RemediationService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	return s.repo.Stats(ctx, tenantID)
}

func (s *RemediationService) ensureAssetsExist(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) error {
	for _, assetID := range assetIDs {
		if _, err := s.assetRepo.GetByID(ctx, tenantID, assetID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return fmt.Errorf("%w: asset %s not found in tenant scope", repository.ErrNotFound, assetID.String())
			}
			return err
		}
	}
	return nil
}

func roleAtLeast(role, required string) bool {
	levels := map[string]int{
		"viewer":           0,
		"security_analyst": 1,
		"analyst":          1,
		"security_manager": 2,
		"tenant_admin":     3,
		"ciso":             4,
		"admin":            5,
	}
	return levels[role] >= levels[required]
}

func safeActorLabel(actor *Actor) string {
	if actor == nil {
		return "system"
	}
	if actor.UserName != "" {
		return actor.UserName
	}
	if actor.UserEmail != "" {
		return actor.UserEmail
	}
	return actor.UserID.String()
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
