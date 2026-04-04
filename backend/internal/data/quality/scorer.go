package quality

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type Scorer struct {
	ruleRepo         *repository.QualityRuleRepository
	resultRepo       *repository.QualityResultRepository
	modelRepo        *repository.ModelRepository
	predictionLogger *aigovmiddleware.PredictionLogger
}

func NewScorer(ruleRepo *repository.QualityRuleRepository, resultRepo *repository.QualityResultRepository, modelRepo *repository.ModelRepository) *Scorer {
	return &Scorer{ruleRepo: ruleRepo, resultRepo: resultRepo, modelRepo: modelRepo}
}

func (s *Scorer) SetPredictionLogger(predictionLogger *aigovmiddleware.PredictionLogger) {
	s.predictionLogger = predictionLogger
}

func (s *Scorer) CalculateScore(ctx context.Context, tenantID uuid.UUID) (*model.QualityScore, error) {
	rules, _, err := s.ruleRepo.List(ctx, tenantID, repositoryRuleParams())
	if err != nil {
		return nil, err
	}
	modelScores := make(map[uuid.UUID]*model.ModelQualityScore)
	score := &model.QualityScore{
		ModelScores:  make([]model.ModelQualityScore, 0),
		TopFailures:  make([]model.TopFailure, 0),
		CalculatedAt: time.Now().UTC(),
	}

	totalWeighted := 0.0
	totalModelWeights := 0.0
	for _, rule := range rules {
		result, err := s.resultRepo.LatestByRule(ctx, tenantID, rule.ID)
		if err != nil {
			continue
		}
		modelItem, err := s.modelRepo.Get(ctx, tenantID, rule.ModelID)
		if err != nil {
			continue
		}
		modelScore := modelScores[rule.ModelID]
		if modelScore == nil {
			modelScore = &model.ModelQualityScore{
				ModelID:              rule.ModelID,
				ModelName:            modelItem.DisplayName,
				Classification:       string(modelItem.DataClassification),
				ClassificationWeight: classificationWeight(modelItem.DataClassification),
			}
			modelScores[rule.ModelID] = modelScore
		}
		weight := severityWeight(rule.Severity)
		score.TotalRules++
		modelScore.TotalRules++
		switch result.Status {
		case model.QualityResultPassed:
			score.PassedRules++
			modelScore.PassedRules++
			modelScore.Score += float64(weight)
		case model.QualityResultWarning:
			score.WarningRules++
			modelScore.WarningRules++
			modelScore.Score += float64(weight)
		default:
			score.FailedRules++
			modelScore.FailedRules++
			score.TopFailures = append(score.TopFailures, model.TopFailure{
				RuleID:        rule.ID,
				RuleName:      rule.Name,
				ModelID:       rule.ModelID,
				ModelName:     modelItem.DisplayName,
				Severity:      string(rule.Severity),
				Status:        string(result.Status),
				RecordsFailed: result.RecordsFailed,
			})
		}
		totalWeighted += float64(weight)
	}

	for modelID, modelScore := range modelScores {
		modelRules, err := s.ruleRepo.ListEnabledByModel(ctx, tenantID, modelID)
		if err != nil || len(modelRules) == 0 {
			continue
		}
		possible := 0.0
		for _, rule := range modelRules {
			possible += float64(severityWeight(rule.Severity))
		}
		if possible > 0 {
			modelScore.Score = (modelScore.Score / possible) * 100
			totalModelWeights += modelScore.ClassificationWeight
			score.OverallScore += modelScore.Score * modelScore.ClassificationWeight
		}
		score.ModelScores = append(score.ModelScores, *modelScore)
	}
	if totalModelWeights > 0 {
		score.OverallScore = score.OverallScore / totalModelWeights
	}
	if score.TotalRules > 0 {
		score.PassRate = float64(score.PassedRules+score.WarningRules) / float64(score.TotalRules) * 100
	}
	score.Grade = grade(score.OverallScore)
	score.Trend = "stable"
	if trendPts, err := s.resultRepo.Trend(ctx, tenantID, 12); err == nil && len(trendPts) > 0 {
		hist := make([]float64, len(trendPts))
		for i, pt := range trendPts {
			hist[i] = pt.Score
		}
		score.History = hist
	}
	sort.Slice(score.TopFailures, func(i, j int) bool {
		return score.TopFailures[i].RecordsFailed > score.TopFailures[j].RecordsFailed
	})
	if len(score.TopFailures) > 5 {
		score.TopFailures = score.TopFailures[:5]
	}
	s.recordPrediction(ctx, tenantID, score)
	return score, nil
}

func (s *Scorer) recordPrediction(ctx context.Context, tenantID uuid.UUID, score *model.QualityScore) {
	if s.predictionLogger == nil || score == nil {
		return
	}

	componentScores := make(map[string]any, len(score.ModelScores))
	componentWeights := make(map[string]any, len(score.ModelScores))
	for _, modelScore := range score.ModelScores {
		key := modelScore.ModelName
		if key == "" {
			key = modelScore.ModelID.String()
		}
		componentScores[key] = modelScore.Score
		componentWeights[key] = modelScore.ClassificationWeight
	}

	input := map[string]any{
		"total_rules":   score.TotalRules,
		"passed_rules":  score.PassedRules,
		"warning_rules": score.WarningRules,
		"failed_rules":  score.FailedRules,
	}
	confidence := clampQualityConfidence(score.PassRate / 100)
	tenantEntityID := tenantID
	_, _ = s.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "data-quality-scorer",
		UseCase:      "quality_scoring",
		EntityType:   "tenant",
		EntityID:     &tenantEntityID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output:     score,
				Confidence: confidence,
				Metadata: map[string]any{
					"overall_score":     score.OverallScore,
					"grade":             score.Grade,
					"component_scores":  componentScores,
					"component_weights": componentWeights,
					"total_rules":       score.TotalRules,
					"failed_rules":      score.FailedRules,
					"top_failures":      formatTopFailures(score.TopFailures),
				},
			}, nil
		},
	})
}

func clampQualityConfidence(value float64) float64 {
	switch {
	case value < 0.55:
		return 0.55
	case value > 0.98:
		return 0.98
	default:
		return value
	}
}

func formatTopFailures(items []model.TopFailure) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s:%s:%d", item.ModelName, item.RuleName, item.RecordsFailed))
	}
	return out
}

func repositoryRuleParams() dto.ListQualityRulesParams {
	return dto.ListQualityRulesParams{Page: 1, PerPage: 1000}
}

func severityWeight(value model.QualitySeverity) int {
	switch value {
	case model.QualitySeverityCritical:
		return 4
	case model.QualitySeverityHigh:
		return 3
	case model.QualitySeverityMedium:
		return 2
	default:
		return 1
	}
}

func classificationWeight(value model.DataClassification) float64 {
	switch value {
	case model.DataClassificationRestricted:
		return 3
	case model.DataClassificationConfidential:
		return 2
	case model.DataClassificationInternal:
		return 1
	default:
		return 0.5
	}
}

func grade(score float64) string {
	switch {
	case score >= 85:
		return "A"
	case score >= 70:
		return "B"
	case score >= 55:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}
