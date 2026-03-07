package quality

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type Scorer struct {
	ruleRepo   *repository.QualityRuleRepository
	resultRepo *repository.QualityResultRepository
	modelRepo  *repository.ModelRepository
}

func NewScorer(ruleRepo *repository.QualityRuleRepository, resultRepo *repository.QualityResultRepository, modelRepo *repository.ModelRepository) *Scorer {
	return &Scorer{ruleRepo: ruleRepo, resultRepo: resultRepo, modelRepo: modelRepo}
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
	sort.Slice(score.TopFailures, func(i, j int) bool {
		return score.TopFailures[i].RecordsFailed > score.TopFailures[j].RecordsFailed
	})
	if len(score.TopFailures) > 5 {
		score.TopFailures = score.TopFailures[:5]
	}
	return score, nil
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
