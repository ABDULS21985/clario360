package scorer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type profileRepository interface {
	GetByEntity(ctx context.Context, tenantID uuid.UUID, entityID string) (*model.UEBAProfile, error)
	UpdateRisk(ctx context.Context, profile *model.UEBAProfile) error
	DecayRiskScores(ctx context.Context, tenantID uuid.UUID, rate float64, now time.Time) (int64, error)
}

type alertRepository interface {
	ListByEntitySince(ctx context.Context, tenantID uuid.UUID, entityID string, since time.Time) ([]*model.UEBAAlert, error)
	UpdateRiskImpact(ctx context.Context, tenantID uuid.UUID, alertID uuid.UUID, before, after float64) error
}

type EntityRiskScorer struct {
	profileRepo profileRepository
	alertRepo   alertRepository
	decayRate   float64
	logger      zerolog.Logger
}

func New(profileRepo profileRepository, alertRepo alertRepository, decayRate float64, logger zerolog.Logger) *EntityRiskScorer {
	return &EntityRiskScorer{
		profileRepo: profileRepo,
		alertRepo:   alertRepo,
		decayRate:   decayRate,
		logger:      logger.With().Str("component", "ueba-risk-scorer").Logger(),
	}
}

func (s *EntityRiskScorer) UpdateRiskScore(ctx context.Context, tenantID uuid.UUID, entityID string, newAlerts []model.UEBAAlert) error {
	profile, err := s.profileRepo.GetByEntity(ctx, tenantID, entityID)
	if err != nil {
		return fmt.Errorf("load ueba profile: %w", err)
	}
	now := time.Now().UTC()
	alerts, err := s.alertRepo.ListByEntitySince(ctx, tenantID, entityID, now.Add(-30*24*time.Hour))
	if err != nil {
		return fmt.Errorf("load ueba alerts for scoring: %w", err)
	}

	totalImpact := 0.0
	factors := make([]model.RiskFactor, 0, len(alerts))
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		impact := alertSeverityImpact(alert.Severity) * recencyWeight(alert.CreatedAt, now) * alert.Confidence
		totalImpact += impact
		factors = append(factors, model.RiskFactor{
			AlertID:     alert.ID,
			AlertType:   string(alert.AlertType),
			Severity:    alert.Severity,
			Confidence:  alert.Confidence,
			Impact:      impact,
			Description: alert.Description,
			CreatedAt:   alert.CreatedAt,
			SignalTypes: extractSignalTypes(alert.TriggeringSignals),
			EventCount:  len(alert.TriggeringEventIDs),
		})
	}
	if totalImpact > 100 {
		totalImpact = 100
	}

	before := profile.RiskScore
	profile.RiskScore = totalImpact
	profile.RiskLevel = riskLevelForScore(totalImpact)
	profile.RiskFactors = factors
	profile.RiskLastUpdated = &now
	profile.AlertCount7D = countAlertsSince(alerts, now.Add(-7*24*time.Hour))
	profile.AlertCount30D = len(alerts)
	if len(newAlerts) > 0 {
		profile.LastAlertAt = &now
	}

	if err := s.profileRepo.UpdateRisk(ctx, profile); err != nil {
		return fmt.Errorf("persist ueba profile risk: %w", err)
	}
	for _, alert := range newAlerts {
		if alert.ID == uuid.Nil {
			continue
		}
		if err := s.alertRepo.UpdateRiskImpact(ctx, tenantID, alert.ID, before, totalImpact); err != nil {
			s.logger.Warn().Err(err).Str("alert_id", alert.ID.String()).Msg("update ueba alert risk impact")
		}
	}
	return nil
}

func (s *EntityRiskScorer) RunDailyDecay(ctx context.Context, tenantID uuid.UUID) error {
	_, err := s.profileRepo.DecayRiskScores(ctx, tenantID, s.decayRate, time.Now().UTC())
	return err
}

func extractSignalTypes(signals []model.AnomalySignal) []string {
	out := make([]string, 0, len(signals))
	seen := make(map[string]struct{}, len(signals))
	for _, signal := range signals {
		key := string(signal.SignalType)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func countAlertsSince(alerts []*model.UEBAAlert, since time.Time) int {
	count := 0
	for _, alert := range alerts {
		if alert != nil && !alert.CreatedAt.Before(since) {
			count++
		}
	}
	return count
}
