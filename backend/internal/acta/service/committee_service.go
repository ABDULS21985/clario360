package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/dto"
	"github.com/clario360/platform/internal/acta/metrics"
	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
)

type CommitteeService struct {
	store     *repository.Store
	publisher Publisher
	metrics   *metrics.Metrics
	logger    zerolog.Logger
}

func NewCommitteeService(store *repository.Store, publisher Publisher, metrics *metrics.Metrics, logger zerolog.Logger) *CommitteeService {
	return &CommitteeService{
		store:     store,
		publisher: publisherOrNoop(publisher),
		metrics:   metrics,
		logger:    logger.With().Str("component", "acta_committee_service").Logger(),
	}
}

func (s *CommitteeService) CreateCommittee(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateCommitteeRequest) (*model.Committee, error) {
	req.Normalize()
	if req.Name == "" {
		return nil, validationError("name is required", map[string]string{"name": "required"})
	}
	if req.ChairUserID == uuid.Nil {
		return nil, validationError("chair_user_id is required", map[string]string{"chair_user_id": "required"})
	}
	if req.ChairName == "" || req.ChairEmail == "" {
		return nil, validationError("chair name and email are required", map[string]string{"chair_name": "required", "chair_email": "required"})
	}
	if existing, err := s.store.GetCommitteeByName(ctx, tenantID, req.Name); err == nil && existing != nil {
		return nil, conflictError("committee name already exists for tenant")
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, internalError("failed to validate committee uniqueness", err)
	}

	if _, err := computeQuorumRequired(1, req.QuorumType, req.QuorumPercentage, req.QuorumFixedCount); err != nil {
		return nil, validationError(err.Error(), nil)
	}

	now := time.Now().UTC()
	committee := &model.Committee{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             req.Name,
		Type:             model.CommitteeType(req.Type),
		Description:      req.Description,
		ChairUserID:      req.ChairUserID,
		ViceChairUserID:  req.ViceChairUserID,
		SecretaryUserID:  req.SecretaryUserID,
		MeetingFrequency: model.MeetingFrequency(req.MeetingFrequency),
		QuorumPercentage: req.QuorumPercentage,
		QuorumType:       model.QuorumType(req.QuorumType),
		QuorumFixedCount: req.QuorumFixedCount,
		Charter:          req.Charter,
		EstablishedDate:  req.EstablishedDate,
		DissolutionDate:  req.DissolutionDate,
		Status:           model.CommitteeStatusActive,
		Tags:             req.Tags,
		Metadata:         req.Metadata,
		CreatedBy:        userID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if err := s.store.CreateCommittee(ctx, tx, committee); err != nil {
			return err
		}
		members := []model.CommitteeMember{
			{
				ID:          uuid.New(),
				TenantID:    tenantID,
				CommitteeID: committee.ID,
				UserID:      req.ChairUserID,
				UserName:    req.ChairName,
				UserEmail:   req.ChairEmail,
				Role:        model.CommitteeMemberRoleChair,
				JoinedAt:    now,
				Active:      true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}
		if req.ViceChairUserID != nil && req.ViceChairName != nil && req.ViceChairEmail != nil {
			members = append(members, model.CommitteeMember{
				ID:          uuid.New(),
				TenantID:    tenantID,
				CommitteeID: committee.ID,
				UserID:      *req.ViceChairUserID,
				UserName:    *req.ViceChairName,
				UserEmail:   *req.ViceChairEmail,
				Role:        model.CommitteeMemberRoleViceChair,
				JoinedAt:    now,
				Active:      true,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
		if req.SecretaryUserID != nil && req.SecretaryName != nil && req.SecretaryEmail != nil {
			members = append(members, model.CommitteeMember{
				ID:          uuid.New(),
				TenantID:    tenantID,
				CommitteeID: committee.ID,
				UserID:      *req.SecretaryUserID,
				UserName:    *req.SecretaryName,
				UserEmail:   *req.SecretaryEmail,
				Role:        model.CommitteeMemberRoleSecretary,
				JoinedAt:    now,
				Active:      true,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
		for _, member := range members {
			member := member
			if err := s.store.UpsertCommitteeMember(ctx, tx, &member); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, internalError("failed to create committee", err)
	}
	if s.metrics != nil {
		s.metrics.CommitteesTotal.WithLabelValues(tenantID.String(), string(committee.Type), string(committee.Status)).Inc()
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.committee.created", tenantID, &userID, map[string]any{
		"id":   committee.ID,
		"name": committee.Name,
		"type": committee.Type,
	}, s.logger)
	return s.GetCommittee(ctx, tenantID, committee.ID)
}

func (s *CommitteeService) ListCommittees(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) ([]model.Committee, int, error) {
	return s.store.ListCommittees(ctx, tenantID, search, page, perPage)
}

func (s *CommitteeService) GetCommittee(ctx context.Context, tenantID, committeeID uuid.UUID) (*model.Committee, error) {
	committee, err := s.store.GetCommittee(ctx, tenantID, committeeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	members, err := s.store.ListCommitteeMembers(ctx, tenantID, committeeID, true)
	if err != nil {
		return nil, internalError("failed to load committee members", err)
	}
	stats, err := s.store.GetCommitteeStats(ctx, tenantID, committeeID)
	if err != nil {
		return nil, internalError("failed to load committee stats", err)
	}
	committee.Members = members
	committee.Stats = stats
	return committee, nil
}

func (s *CommitteeService) UpdateCommittee(ctx context.Context, tenantID, userID, committeeID uuid.UUID, req dto.UpdateCommitteeRequest) (*model.Committee, error) {
	req.Normalize()
	committee, err := s.store.GetCommittee(ctx, tenantID, committeeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	if req.ChairUserID != uuid.Nil {
		isMember, err := s.store.UserIsCommitteeMember(ctx, tenantID, committeeID, req.ChairUserID)
		if err != nil {
			return nil, internalError("failed to validate committee chair", err)
		}
		if !isMember {
			return nil, validationError("chair_user_id must be an active committee member", nil)
		}
		committee.ChairUserID = req.ChairUserID
	}
	committee.Name = req.Name
	committee.Type = model.CommitteeType(req.Type)
	committee.Description = req.Description
	committee.ViceChairUserID = req.ViceChairUserID
	committee.SecretaryUserID = req.SecretaryUserID
	committee.MeetingFrequency = model.MeetingFrequency(req.MeetingFrequency)
	committee.QuorumPercentage = req.QuorumPercentage
	committee.QuorumType = model.QuorumType(req.QuorumType)
	committee.QuorumFixedCount = req.QuorumFixedCount
	committee.Charter = req.Charter
	committee.EstablishedDate = req.EstablishedDate
	committee.DissolutionDate = req.DissolutionDate
	committee.Status = model.CommitteeStatus(req.Status)
	committee.Tags = req.Tags
	committee.Metadata = req.Metadata
	committee.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateCommittee(ctx, s.store.DB(), committee); err != nil {
		return nil, internalError("failed to update committee", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.committee.updated", tenantID, &userID, map[string]any{
		"id":   committee.ID,
		"name": committee.Name,
	}, s.logger)
	return s.GetCommittee(ctx, tenantID, committeeID)
}

func (s *CommitteeService) DeleteCommittee(ctx context.Context, tenantID, committeeID uuid.UUID) error {
	pending, err := s.store.CommitteeHasPendingMeetings(ctx, tenantID, committeeID)
	if err != nil {
		return internalError("failed to validate pending meetings", err)
	}
	if pending {
		return validationError("committee cannot be deleted while pending meetings exist", nil)
	}
	if err := s.store.SoftDeleteCommittee(ctx, tenantID, committeeID, time.Now().UTC()); err != nil {
		return internalError("failed to delete committee", err)
	}
	return nil
}

func (s *CommitteeService) AddOrUpdateMember(ctx context.Context, tenantID, userID, committeeID uuid.UUID, req dto.UpsertCommitteeMemberRequest) (*model.Committee, error) {
	req.Normalize()
	if req.UserID == uuid.Nil {
		return nil, validationError("user_id is required", map[string]string{"user_id": "required"})
	}
	committee, err := s.store.GetCommittee(ctx, tenantID, committeeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	existingMember, err := s.store.GetCommitteeMember(ctx, tenantID, committeeID, req.UserID)
	switch {
	case err == nil:
		if req.UserName == "" {
			req.UserName = existingMember.UserName
		}
		if req.UserEmail == "" {
			req.UserEmail = existingMember.UserEmail
		}
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return nil, internalError("failed to inspect committee member", err)
	}
	if req.UserName == "" || req.UserEmail == "" {
		return nil, validationError("user_name and user_email are required", nil)
	}
	role := model.CommitteeMemberRole(req.Role)
	now := time.Now().UTC()
	member := &model.CommitteeMember{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CommitteeID: committeeID,
		UserID:      req.UserID,
		UserName:    req.UserName,
		UserEmail:   req.UserEmail,
		Role:        role,
		JoinedAt:    now,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if role == model.CommitteeMemberRoleChair || role == model.CommitteeMemberRoleViceChair || role == model.CommitteeMemberRoleSecretary {
			members, err := s.store.ListCommitteeMembers(ctx, tenantID, committeeID, true)
			if err != nil {
				return err
			}
			for _, existing := range members {
				if existing.Role == role && existing.UserID != req.UserID {
					existing.Role = model.CommitteeMemberRoleMember
					existing.UpdatedAt = now
					if err := s.store.UpsertCommitteeMember(ctx, tx, &existing); err != nil {
						return err
					}
				}
			}
		}
		if err := s.store.UpsertCommitteeMember(ctx, tx, member); err != nil {
			return err
		}
		switch role {
		case model.CommitteeMemberRoleChair:
			committee.ChairUserID = req.UserID
		case model.CommitteeMemberRoleViceChair:
			committee.ViceChairUserID = &req.UserID
		case model.CommitteeMemberRoleSecretary:
			committee.SecretaryUserID = &req.UserID
		}
		committee.UpdatedAt = now
		return s.store.UpdateCommittee(ctx, tx, committee)
	}); err != nil {
		return nil, internalError("failed to add committee member", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.committee.member_added", tenantID, &userID, map[string]any{
		"committee_id": committeeID,
		"user_id":      req.UserID,
		"role":         req.Role,
	}, s.logger)
	return s.GetCommittee(ctx, tenantID, committeeID)
}

func (s *CommitteeService) RemoveMember(ctx context.Context, tenantID, userID, committeeID, memberUserID uuid.UUID) (*model.Committee, error) {
	committee, err := s.store.GetCommittee(ctx, tenantID, committeeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	if committee.ChairUserID == memberUserID {
		return nil, forbiddenError("committee chair must be reassigned before removal")
	}
	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if err := s.store.DeactivateCommitteeMember(ctx, tx, tenantID, committeeID, memberUserID, time.Now().UTC()); err != nil {
			return err
		}
		if committee.ViceChairUserID != nil && *committee.ViceChairUserID == memberUserID {
			committee.ViceChairUserID = nil
		}
		if committee.SecretaryUserID != nil && *committee.SecretaryUserID == memberUserID {
			committee.SecretaryUserID = nil
		}
		committee.UpdatedAt = time.Now().UTC()
		return s.store.UpdateCommittee(ctx, tx, committee)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundError("committee member not found")
		}
		return nil, internalError("failed to remove committee member", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.committee.member_removed", tenantID, &userID, map[string]any{
		"committee_id": committeeID,
		"user_id":      memberUserID,
	}, s.logger)
	return s.GetCommittee(ctx, tenantID, committeeID)
}
