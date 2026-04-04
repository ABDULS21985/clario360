package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/contradiction"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

const contradictionEventsTopic = "data.contradiction.events"

type ContradictionService struct {
	repo     *repository.ContradictionRepository
	detector *contradiction.Detector
	producer *events.Producer
}

func NewContradictionService(repo *repository.ContradictionRepository, detector *contradiction.Detector, producer *events.Producer) *ContradictionService {
	return &ContradictionService{repo: repo, detector: detector, producer: producer}
}

func (s *ContradictionService) Scan(ctx context.Context, tenantID, userID uuid.UUID) (*model.ContradictionScan, error) {
	return s.detector.RunScan(ctx, tenantID, userID)
}

func (s *ContradictionService) ListScans(ctx context.Context, tenantID uuid.UUID, params dto.ListContradictionScansParams) ([]*model.ContradictionScan, int, error) {
	return s.repo.ListScans(ctx, tenantID, params)
}

func (s *ContradictionService) GetScan(ctx context.Context, tenantID, id uuid.UUID) (*model.ContradictionScan, error) {
	return s.repo.GetScan(ctx, tenantID, id)
}

func (s *ContradictionService) List(ctx context.Context, tenantID uuid.UUID, params dto.ListContradictionsParams) ([]*model.Contradiction, int, error) {
	return s.repo.List(ctx, tenantID, params)
}

func (s *ContradictionService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.Contradiction, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func (s *ContradictionService) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.ContradictionStatus) error {
	if !status.IsValid() {
		return fmt.Errorf("%w: invalid contradiction status", ErrValidation)
	}
	if err := s.repo.UpdateStatus(ctx, tenantID, id, status); err != nil {
		return err
	}
	s.publish(ctx, "data.contradiction.status_updated", tenantID, map[string]any{"id": id, "status": status})
	return nil
}

func (s *ContradictionService) Resolve(ctx context.Context, tenantID, id, userID uuid.UUID, action model.ContradictionResolutionAction, notes string) error {
	action = model.ContradictionResolutionAction(strings.TrimSpace(string(action)))
	if !action.IsValid() {
		return fmt.Errorf("%w: invalid resolution action", ErrValidation)
	}
	status := model.ContradictionStatusResolved
	if action == model.ContradictionResolutionFalsePositive {
		status = model.ContradictionStatusFalsePositive
	}
	if err := s.repo.Resolve(ctx, tenantID, id, userID, action, notes, status); err != nil {
		return err
	}
	s.publish(ctx, "data.contradiction.resolved", tenantID, map[string]any{"id": id, "resolution_action": action})
	return nil
}

func (s *ContradictionService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ContradictionStats, error) {
	return s.repo.Stats(ctx, tenantID)
}

func (s *ContradictionService) Dashboard(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	stats, err := s.repo.Stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	recent, _, err := s.repo.List(ctx, tenantID, dto.ListContradictionsParams{Page: 1, PerPage: 10})
	if err != nil {
		return nil, err
	}
	scans, _, err := s.repo.ListScans(ctx, tenantID, dto.ListContradictionScansParams{Page: 1, PerPage: 5})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"stats":        stats,
		"recent_items": recent,
		"recent_scans": scans,
		"generated_at": time.Now().UTC(),
	}, nil
}

func (s *ContradictionService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return
	}
	_ = s.producer.Publish(ctx, contradictionEventsTopic, event)
}
