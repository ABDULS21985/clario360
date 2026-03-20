package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
)

type DashboardKPI struct {
	TotalModels    int   `json:"total_models"`
	InProduction   int   `json:"in_production"`
	ShadowTesting  int   `json:"shadow_testing"`
	Predictions24h int64 `json:"predictions_24h"`
	DriftAlerts    int   `json:"drift_alerts"`
}

type DashboardModelRow struct {
	ID                uuid.UUID                `json:"id"`
	Name              string                   `json:"name"`
	Slug              string                   `json:"slug"`
	Suite             aigovmodel.ModelSuite    `json:"suite"`
	Type              aigovmodel.ModelType     `json:"type"`
	RiskTier          aigovmodel.RiskTier      `json:"risk_tier"`
	Status            aigovmodel.ModelStatus   `json:"status"`
	ProductionVersion *aigovmodel.ModelVersion `json:"production_version,omitempty"`
	ShadowVersion     *aigovmodel.ModelVersion `json:"shadow_version,omitempty"`
	Predictions24h    int64                    `json:"predictions_24h"`
	AvgConfidence     *float64                 `json:"avg_confidence,omitempty"`
	DriftStatus       aigovmodel.DriftLevel    `json:"drift_status"`
}

type DashboardData struct {
	KPIs   DashboardKPI       `json:"kpis"`
	Models []DashboardModelRow `json:"models"`
}

type DashboardService struct {
	registryRepo   *repository.ModelRegistryRepository
	predictionRepo *repository.PredictionLogRepository
	driftRepo      *repository.DriftReportRepository
	logger         zerolog.Logger
}

func NewDashboardService(registryRepo *repository.ModelRegistryRepository, predictionRepo *repository.PredictionLogRepository, driftRepo *repository.DriftReportRepository, logger zerolog.Logger) *DashboardService {
	return &DashboardService{
		registryRepo:   registryRepo,
		predictionRepo: predictionRepo,
		driftRepo:      driftRepo,
		logger:         logger.With().Str("component", "ai_dashboard_service").Logger(),
	}
}

func (s *DashboardService) Get(ctx context.Context, tenantID uuid.UUID) (*DashboardData, error) {
	models, _, err := s.registryRepo.ListModels(ctx, tenantID, repository.ListModelsParams{Page: 1, PerPage: 250})
	if err != nil {
		return nil, err
	}
	recentStats, err := s.predictionRepo.RecentModelStats(ctx, tenantID, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	out := &DashboardData{
		Models: make([]DashboardModelRow, 0, len(models)),
	}
	for _, item := range models {
		row := DashboardModelRow{
			ID:       item.ID,
			Name:     item.Name,
			Slug:     item.Slug,
			Suite:    item.Suite,
			Type:     item.ModelType,
			RiskTier: item.RiskTier,
			Status:   item.Status,
		}
		if production, err := s.registryRepo.GetCurrentProductionVersion(ctx, tenantID, item.ID); err == nil {
			row.ProductionVersion = production
			out.KPIs.InProduction++
		}
		if shadowVersion, err := s.registryRepo.GetCurrentShadowVersion(ctx, tenantID, item.ID); err == nil {
			row.ShadowVersion = shadowVersion
			out.KPIs.ShadowTesting++
		}
		if stat, ok := recentStats[item.ID]; ok {
			row.Predictions24h = stat.Predictions24h
			row.AvgConfidence = stat.AvgConfidence
			out.KPIs.Predictions24h += stat.Predictions24h
		}
		if drift, err := s.driftRepo.LatestByModel(ctx, tenantID, item.ID); err == nil {
			row.DriftStatus = drift.OutputDriftLevel
			out.KPIs.DriftAlerts += drift.AlertCount
		}
		out.Models = append(out.Models, row)
	}
	out.KPIs.TotalModels = len(models)
	return out, nil
}
