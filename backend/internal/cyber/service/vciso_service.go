package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	cyberrisk "github.com/clario360/platform/internal/cyber/risk"
	"github.com/clario360/platform/internal/cyber/vciso"
	"github.com/clario360/platform/internal/events"
)

type VCISOService struct {
	repo        *repository.VCISORepository
	briefing    *vciso.BriefingGenerator
	recommender *vciso.RecommendationAggregator
	reporter    *vciso.ReportGenerator
	riskScorer  *cyberrisk.RiskScorer
	producer    *events.Producer
	logger      zerolog.Logger
}

func NewVCISOService(
	repo *repository.VCISORepository,
	briefing *vciso.BriefingGenerator,
	recommender *vciso.RecommendationAggregator,
	reporter *vciso.ReportGenerator,
	riskScorer *cyberrisk.RiskScorer,
	producer *events.Producer,
	logger zerolog.Logger,
) *VCISOService {
	return &VCISOService{
		repo:        repo,
		briefing:    briefing,
		recommender: recommender,
		reporter:    reporter,
		riskScorer:  riskScorer,
		producer:    producer,
		logger:      logger.With().Str("service", "vciso").Logger(),
	}
}

func (s *VCISOService) GenerateBriefing(ctx context.Context, tenantID, userID uuid.UUID, periodDays int, actor *Actor) (*model.ExecutiveBriefing, error) {
	briefing, err := s.briefing.GenerateExecutiveBriefing(ctx, tenantID, periodDays)
	if err != nil {
		return nil, err
	}
	riskScore := briefing.RiskPosture.CurrentScore
	if _, err := s.repo.SaveBriefing(ctx, tenantID, userID, "executive", briefing.Period.Start, briefing.Period.End, briefing, &riskScore); err != nil {
		return nil, err
	}
	if err := publishEvent(ctx, s.producer, events.Topics.VCISOEvents, "com.clario360.cyber.vciso.briefing_generated", tenantID, actor, map[string]interface{}{
		"type":      "executive",
		"tenant_id": tenantID.String(),
	}); err != nil {
		s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("publish vciso briefing event")
	}
	return briefing, nil
}

func (s *VCISOService) ListBriefings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListBriefings(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	totalPages := (total + params.PerPage - 1) / params.PerPage
	if totalPages == 0 {
		totalPages = 1
	}
	return &dto.VCISOBriefingHistoryResponse{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (s *VCISOService) Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error) {
	score, err := s.riskScorer.CalculateOrganizationRisk(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.recommender.Generate(ctx, tenantID, score)
}

func (s *VCISOService) GenerateReport(ctx context.Context, tenantID, userID uuid.UUID, req *dto.VCISOReportRequest, actor *Actor) (*dto.VCISOReportResponse, error) {
	report, err := s.reporter.Generate(ctx, tenantID, req.Type, req.PeriodDays)
	if err != nil {
		return nil, err
	}
	riskScore := report.RiskPosture.CurrentScore
	record, err := s.repo.SaveBriefing(ctx, tenantID, userID, req.Type, report.Period.Start, report.Period.End, report, &riskScore)
	if err != nil {
		return nil, err
	}
	if err := publishEvent(ctx, s.producer, events.Topics.VCISOEvents, "com.clario360.cyber.vciso.report_generated", tenantID, actor, map[string]interface{}{
		"report_id": record.ID.String(),
		"type":      req.Type,
		"tenant_id": tenantID.String(),
	}); err != nil {
		s.logger.Error().Err(err).Str("report_id", record.ID.String()).Msg("publish vciso report event")
	}
	return &dto.VCISOReportResponse{JobID: record.ID.String(), Status: "completed"}, nil
}

func (s *VCISOService) PostureSummary(ctx context.Context, tenantID uuid.UUID) (*model.PostureSummary, error) {
	score, err := s.riskScorer.CalculateOrganizationRisk(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	briefing, err := s.briefing.GenerateExecutiveBriefing(ctx, tenantID, 30)
	if err != nil {
		return nil, err
	}

	summary := &model.PostureSummary{
		RiskScore:  score.OverallScore,
		Grade:      score.Grade,
		Trend:      score.Trend,
		TrendDelta: score.TrendDelta,
		DSPMScore:  briefing.ComplianceStatus.DSPMPostureScore,
	}
	for _, issue := range briefing.CriticalIssues {
		summary.TopIssues = append(summary.TopIssues, issue.Title)
		if len(summary.TopIssues) == 3 {
			break
		}
	}
	for severity, count := range briefing.KeyMetrics.AlertsBySeverity {
		if severity == "critical" {
			summary.OpenCriticalAlerts = count
			break
		}
	}
	summary.UnpatchedCriticalVulns = countCriticalVulnIssues(briefing.CriticalIssues)
	summary.ActiveThreats = briefing.ThreatLandscape.NewThreats
	return summary, nil
}

func countCriticalVulnIssues(issues []model.CriticalIssue) int {
	total := 0
	for _, issue := range issues {
		if issue.Type == "vulnerability" {
			total++
		}
	}
	return total
}

func currentWindow(periodDays int) (time.Time, time.Time) {
	end := time.Now().UTC()
	return end.AddDate(0, 0, -periodDays), end
}
