package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ctem"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

type ExposureService struct {
	scoring        *ctem.ScoringEngine
	snapshotRepo   *repository.CTEMSnapshotRepository
	assessmentRepo *repository.CTEMAssessmentRepository
}

func NewExposureService(
	scoring *ctem.ScoringEngine,
	snapshotRepo *repository.CTEMSnapshotRepository,
	assessmentRepo *repository.CTEMAssessmentRepository,
) *ExposureService {
	return &ExposureService{
		scoring:        scoring,
		snapshotRepo:   snapshotRepo,
		assessmentRepo: assessmentRepo,
	}
}

func (s *ExposureService) Calculate(ctx context.Context, tenantID uuid.UUID) (*model.ExposureScore, error) {
	return s.scoring.CalculateExposureScore(ctx, tenantID)
}

func (s *ExposureService) CalculateAndSnapshot(ctx context.Context, tenantID uuid.UUID, assessmentID *uuid.UUID, snapshotType string, assetCount, vulnCount, findingCount int) (*model.ExposureScore, error) {
	score, err := s.scoring.CalculateExposureScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.snapshotRepo.Create(ctx, tenantID, assessmentID, snapshotType, score, assetCount, vulnCount, findingCount); err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	return score, nil
}

func (s *ExposureService) History(ctx context.Context, tenantID uuid.UUID, sinceDays int) ([]model.TimeSeriesPoint, error) {
	params := struct{ Days int }{Days: sinceDays}
	if params.Days == 0 {
		params.Days = 90
	}
	return s.snapshotRepo.History(ctx, tenantID, time.Now().UTC().AddDate(0, 0, -params.Days))
}
