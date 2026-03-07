package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
)

type DashboardService struct {
	redis      *redis.Client
	contracts  *repository.ContractRepository
	documents  *repository.DocumentRepository
	alerts     *repository.AlertRepository
	compliance *ComplianceService
	metrics    *metrics.Metrics
	logger     zerolog.Logger
	cacheTTL   time.Duration
}

func NewDashboardService(_ any, redisClient *redis.Client, contracts *repository.ContractRepository, documents *repository.DocumentRepository, alerts *repository.AlertRepository, compliance *ComplianceService, appMetrics *metrics.Metrics, logger zerolog.Logger, cacheTTL time.Duration) *DashboardService {
	if cacheTTL <= 0 {
		cacheTTL = 60 * time.Second
	}
	return &DashboardService{
		redis:      redisClient,
		contracts:  contracts,
		documents:  documents,
		alerts:     alerts,
		compliance: compliance,
		metrics:    appMetrics,
		logger:     logger.With().Str("service", "lex-dashboard").Logger(),
		cacheTTL:   cacheTTL,
	}
}

func (s *DashboardService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*model.LexDashboard, error) {
	cacheKey := fmt.Sprintf("lex:dashboard:%s", tenantID)
	if s.redis != nil {
		if payload, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var cached model.LexDashboard
			if err := json.Unmarshal(payload, &cached); err == nil {
				return &cached, nil
			}
		}
	}

	var (
		stats        *model.ContractStats
		byType       map[string]int
		byStatus     map[string]int
		expiring     []model.ExpiringContractSummary
		highRisk     []model.ContractRiskSummary
		recent       []model.ContractSummary
		alertStatus  map[string]int
		valueBreakdown model.TotalValueBreakdown
		monthly      []model.MonthlyContractActivity
		score        *model.ComplianceScore
		documentTypes map[string]int
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		stats, err = s.contracts.Stats(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		byType, err = s.contracts.CountByType(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		byStatus, err = s.contracts.CountByStatus(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		expiring, err = s.contracts.ListExpiring(gctx, tenantID, 30)
		return err
	})
	g.Go(func() error {
		var err error
		highRisk, err = s.contracts.HighRiskContracts(gctx, tenantID, 10)
		return err
	})
	g.Go(func() error {
		var err error
		recent, err = s.contracts.RecentContracts(gctx, tenantID, 10)
		return err
	})
	g.Go(func() error {
		var err error
		alertStatus, err = s.alerts.CountByStatus(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		valueBreakdown, err = s.contracts.TotalValueBreakdown(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		monthly, err = s.contracts.MonthlyActivity(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		score, err = s.compliance.GetScore(gctx, tenantID)
		return err
	})
	g.Go(func() error {
		var err error
		documentTypes, err = s.documents.CountByType(gctx, tenantID)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, internalError("build dashboard", err)
	}

	totalValue := 0.0
	for _, value := range valueBreakdown.ByType {
		totalValue += value
	}
	kpis := model.LexKPIs{
		ActiveContracts:   byStatus[string(model.ContractStatusActive)],
		ExpiringIn30Days:  stats.Expiring30Days,
		ExpiringIn7Days:   stats.Expiring7Days,
		HighRiskContracts: stats.ByRiskLevel[string(model.RiskLevelCritical)] + stats.ByRiskLevel[string(model.RiskLevelHigh)],
		PendingReview:     byStatus[string(model.ContractStatusInternalReview)] + byStatus[string(model.ContractStatusLegalReview)],
		OpenAlerts:        alertStatus[string(model.ComplianceAlertOpen)] + alertStatus[string(model.ComplianceAlertAcknowledged)] + alertStatus[string(model.ComplianceAlertInvestigating)],
		TotalValue:        totalValue,
		ComplianceScore:   score.Score,
	}
	dashboard := &model.LexDashboard{
		KPIs:                     kpis,
		ContractsByType:          byType,
		ContractsByStatus:        byStatus,
		ExpiringContracts:        expiring,
		HighRiskContracts:        highRisk,
		RecentContracts:          recent,
		ComplianceAlertsByStatus: alertStatus,
		TotalContractValue:       valueBreakdown,
		MonthlyActivity:          monthly,
		CalculatedAt:             time.Now().UTC(),
	}
	if s.metrics != nil {
		for docType, count := range documentTypes {
			s.metrics.DocumentsTotal.WithLabelValues(docType).Set(float64(count))
		}
		for contractType, value := range valueBreakdown.ByType {
			s.metrics.ContractValueTotal.WithLabelValues(contractType, "mixed").Set(value)
		}
	}
	if s.redis != nil {
		if payload, err := json.Marshal(dashboard); err == nil {
			_ = s.redis.Set(ctx, cacheKey, payload, s.cacheTTL).Err()
		}
	}
	return dashboard, nil
}
