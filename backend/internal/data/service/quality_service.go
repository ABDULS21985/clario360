package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/quality"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/sqlutil"
	"github.com/clario360/platform/internal/events"
)

const qualityEventsTopic = "data.quality.events"

type QualityService struct {
	ruleRepo   *repository.QualityRuleRepository
	resultRepo *repository.QualityResultRepository
	modelRepo  *repository.ModelRepository
	executor   *quality.QualityExecutor
	scorer     *quality.Scorer
	producer   *events.Producer
}

func NewQualityService(
	ruleRepo *repository.QualityRuleRepository,
	resultRepo *repository.QualityResultRepository,
	modelRepo *repository.ModelRepository,
	executor *quality.QualityExecutor,
	scorer *quality.Scorer,
	producer *events.Producer,
) *QualityService {
	return &QualityService{
		ruleRepo:   ruleRepo,
		resultRepo: resultRepo,
		modelRepo:  modelRepo,
		executor:   executor,
		scorer:     scorer,
		producer:   producer,
	}
}

func (s *QualityService) CreateRule(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateQualityRuleRequest) (*model.QualityRule, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if _, err := s.modelRepo.Get(ctx, tenantID, req.ModelID); err != nil {
		return nil, err
	}
	ruleType := model.QualityRuleType(strings.TrimSpace(req.RuleType))
	if !ruleType.IsValid() {
		return nil, fmt.Errorf("%w: invalid rule_type", ErrValidation)
	}
	severity := model.QualitySeverity(strings.TrimSpace(req.Severity))
	if !severity.IsValid() {
		return nil, fmt.Errorf("%w: invalid severity", ErrValidation)
	}
	if ruleType == model.QualityRuleTypeCustomSQL {
		if err := validateCustomSQLConfig(req.Config); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC()
	item := &model.QualityRule{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ModelID:     req.ModelID,
		Name:        req.Name,
		Description: req.Description,
		RuleType:    ruleType,
		Severity:    severity,
		ColumnName:  req.ColumnName,
		Config:      req.Config,
		Schedule:    req.Schedule,
		Enabled:     enabled,
		Tags:        req.Tags,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.ruleRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.publish(ctx, "data.quality.rule.created", tenantID, map[string]any{"id": item.ID, "model_id": item.ModelID})
	return item, nil
}

func (s *QualityService) ListRules(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityRulesParams) ([]*model.QualityRule, int, error) {
	return s.ruleRepo.List(ctx, tenantID, params)
}

func (s *QualityService) GetRule(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityRule, error) {
	return s.ruleRepo.Get(ctx, tenantID, id)
}

func (s *QualityService) UpdateRule(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateQualityRuleRequest) (*model.QualityRule, error) {
	item, err := s.ruleRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, fmt.Errorf("%w: name is required", ErrValidation)
		}
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Severity != nil {
		severity := model.QualitySeverity(strings.TrimSpace(*req.Severity))
		if !severity.IsValid() {
			return nil, fmt.Errorf("%w: invalid severity", ErrValidation)
		}
		item.Severity = severity
	}
	if req.ColumnName != nil {
		item.ColumnName = req.ColumnName
	}
	if len(req.Config) > 0 {
		if item.RuleType == model.QualityRuleTypeCustomSQL {
			if err := validateCustomSQLConfig(req.Config); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrValidation, err)
			}
		}
		item.Config = req.Config
	}
	if req.Schedule != nil {
		item.Schedule = req.Schedule
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if req.Tags != nil {
		item.Tags = req.Tags
	}
	item.UpdatedAt = time.Now().UTC()
	if err := s.ruleRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	s.publish(ctx, "data.quality.rule.updated", tenantID, map[string]any{"id": item.ID})
	return item, nil
}

func (s *QualityService) DeleteRule(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.ruleRepo.SoftDelete(ctx, tenantID, id, time.Now().UTC()); err != nil {
		return err
	}
	s.publish(ctx, "data.quality.rule.deleted", tenantID, map[string]any{"id": id})
	return nil
}

func (s *QualityService) RunRule(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error) {
	before, _ := s.scorer.CalculateScore(ctx, tenantID)
	result, err := s.executor.RunCheck(ctx, tenantID, id, nil)
	if err != nil {
		return nil, err
	}
	after, _ := s.scorer.CalculateScore(ctx, tenantID)
	if before != nil && after != nil && s.producer != nil && before.OverallScore != after.OverallScore {
		s.publish(ctx, "data.quality.score_changed", tenantID, map[string]any{
			"tenant_id": tenantID,
			"old_score": before.OverallScore,
			"new_score": after.OverallScore,
		})
	}
	return result, nil
}

func (s *QualityService) ListResults(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityResultsParams) ([]*model.QualityResult, int, error) {
	return s.resultRepo.List(ctx, tenantID, params)
}

func (s *QualityService) GetResult(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error) {
	return s.resultRepo.Get(ctx, tenantID, id)
}

func (s *QualityService) Score(ctx context.Context, tenantID uuid.UUID) (*model.QualityScore, error) {
	return s.scorer.CalculateScore(ctx, tenantID)
}

func (s *QualityService) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.QualityTrendPoint, error) {
	return s.resultRepo.Trend(ctx, tenantID, days)
}

func (s *QualityService) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.QualityDashboard, error) {
	score, err := s.scorer.CalculateScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	trend, err := s.resultRepo.Trend(ctx, tenantID, 30)
	if err != nil {
		return nil, err
	}
	recentRules, _, err := s.ruleRepo.List(ctx, tenantID, dto.ListQualityRulesParams{Page: 1, PerPage: 10})
	if err != nil {
		return nil, err
	}
	values := make([]model.QualityRule, 0, len(recentRules))
	for _, item := range recentRules {
		values = append(values, *item)
	}
	return &model.QualityDashboard{
		Score:       score,
		RecentRules: values,
		TopFailures: score.TopFailures,
		Trend:       trend,
	}, nil
}

func (s *QualityService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return
	}
	_ = s.producer.Publish(ctx, qualityEventsTopic, event)
}

func validateCustomSQLConfig(raw []byte) error {
	var payload struct {
		SQL string `json:"sql"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return sqlutil.ValidateReadOnlySQL(payload.SQL)
}
