package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

type RoleService struct {
	roleRepo repository.RoleRepository
	userRepo repository.UserRepository
	producer *events.Producer
	logger   zerolog.Logger
}

func NewRoleService(
	roleRepo repository.RoleRepository,
	userRepo repository.UserRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		userRepo: userRepo,
		producer: producer,
		logger:   logger,
	}
}

func (s *RoleService) List(ctx context.Context, tenantID string) ([]dto.RoleResponse, error) {
	roles, err := s.roleRepo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return dto.RolesToResponse(roles), nil
}

func (s *RoleService) GetByID(ctx context.Context, roleID string) (*dto.RoleResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	resp := dto.RoleToResponse(role)
	return &resp, nil
}

func (s *RoleService) Create(ctx context.Context, tenantID string, req *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	// Check if slug already exists
	_, err := s.roleRepo.GetBySlug(ctx, tenantID, req.Slug)
	if err == nil {
		return nil, fmt.Errorf("role slug %s: %w", req.Slug, model.ErrConflict)
	}

	role := &model.Role{
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("creating role: %w", err)
	}

	s.publishEvent(ctx, "role.created", tenantID, "")

	resp := dto.RoleToResponse(role)
	return &resp, nil
}

func (s *RoleService) Update(ctx context.Context, roleID string, req *dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	if role.IsSystemRole {
		return nil, model.ErrSystemRole
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Permissions != nil {
		role.Permissions = req.Permissions
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	updated, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	resp := dto.RoleToResponse(updated)
	return &resp, nil
}

func (s *RoleService) Delete(ctx context.Context, roleID string) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsSystemRole {
		return model.ErrSystemRole
	}
	return s.roleRepo.Delete(ctx, roleID)
}

func (s *RoleService) AssignRole(ctx context.Context, userID string, req *dto.AssignRoleRequest, tenantID, assignedBy string) error {
	// Verify user exists
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return err
	}

	// Verify role exists
	if _, err := s.roleRepo.GetByID(ctx, req.RoleID); err != nil {
		return err
	}

	if err := s.roleRepo.AssignToUser(ctx, userID, req.RoleID, tenantID, assignedBy); err != nil {
		return err
	}

	s.publishEvent(ctx, "role.assigned", tenantID, userID)
	return nil
}

func (s *RoleService) RemoveRole(ctx context.Context, userID, roleID string) error {
	if err := s.roleRepo.RemoveFromUser(ctx, userID, roleID); err != nil {
		return err
	}

	s.publishEvent(ctx, "role.removed", "", userID)
	return nil
}

func (s *RoleService) publishEvent(ctx context.Context, eventType, tenantID, userID string) {
	if s.producer == nil {
		return
	}
	evt, err := events.NewEvent(eventType, "iam-service", tenantID, nil)
	if err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to create event")
		return
	}
	if userID != "" {
		evt.UserID = userID
	}
	if err := s.producer.Publish(ctx, "platform.iam.events", evt); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}
