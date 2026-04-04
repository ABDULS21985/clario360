package ai_security

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// ModelDataGovernance provides governance assessments for AI models by
// analyzing all data assets linked to a specific model.
type ModelDataGovernance struct {
	repo   AIUsageRepository
	logger zerolog.Logger
}

// NewModelDataGovernance creates a new model data governance instance.
func NewModelDataGovernance(repo AIUsageRepository, logger zerolog.Logger) *ModelDataGovernance {
	return &ModelDataGovernance{
		repo:   repo,
		logger: logger.With().Str("component", "model_data_governance").Logger(),
	}
}

// AssessModel evaluates all data usages for a specific AI model and produces
// an aggregate governance assessment including PII exposure, consent coverage,
// anonymization coverage, overall risk, and actionable recommendations.
func (g *ModelDataGovernance) AssessModel(ctx context.Context, tenantID uuid.UUID, modelSlug string) (*model.ModelDataAssessment, error) {
	g.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("model_slug", modelSlug).
		Msg("assessing model data governance")

	usages, err := g.repo.ListByModel(ctx, tenantID, modelSlug)
	if err != nil {
		return nil, fmt.Errorf("listing usages for model %s: %w", modelSlug, err)
	}

	if len(usages) == 0 {
		return &model.ModelDataAssessment{
			ModelSlug:       modelSlug,
			Recommendations: []string{"No data usage records found for this model. Verify AI data connections."},
		}, nil
	}

	assessment := &model.ModelDataAssessment{
		ModelSlug:  modelSlug,
		DataUsages: usages,
	}

	// Extract model name from first usage that has one.
	for _, u := range usages {
		if u.ModelName != "" {
			assessment.ModelName = u.ModelName
			break
		}
	}

	var (
		totalTraining    int
		piiTraining      int
		consentCount     int
		anonymizedCount  int
		totalRiskScore   float64
	)

	for _, u := range usages {
		if u.UsageType == model.AIUsageTrainingData ||
			u.UsageType == model.AIUsageEvaluationData ||
			u.UsageType == model.AIUsageFeatureStore {
			totalTraining++
			if u.ContainsPII {
				piiTraining++
			}
		}

		if u.ConsentVerified {
			consentCount++
		}
		if u.AnonymizationLevel != model.AnonymizationNone {
			anonymizedCount++
		}
		totalRiskScore += u.AIRiskScore
	}

	assessment.TrainingDataCount = totalTraining
	assessment.PIITrainingData = piiTraining

	total := len(usages)
	if total > 0 {
		assessment.ConsentCoverage = float64(consentCount) / float64(total) * 100
		assessment.AnonymizationCoverage = float64(anonymizedCount) / float64(total) * 100
		assessment.RiskScore = totalRiskScore / float64(total)
	}

	// Generate recommendations based on assessment.
	assessment.Recommendations = g.generateRecommendations(assessment)

	g.logger.Info().
		Str("model_slug", modelSlug).
		Int("data_usages", total).
		Int("training_data", totalTraining).
		Int("pii_training", piiTraining).
		Float64("risk_score", assessment.RiskScore).
		Msg("model assessment complete")

	return assessment, nil
}

// generateRecommendations produces actionable governance recommendations
// based on the model's data usage assessment.
func (g *ModelDataGovernance) generateRecommendations(assessment *model.ModelDataAssessment) []string {
	var recs []string

	// PII in training data.
	if assessment.PIITrainingData > 0 {
		recs = append(recs,
			fmt.Sprintf("%d training datasets contain PII. Apply anonymization or pseudonymization before model training.", assessment.PIITrainingData))
	}

	// Low consent coverage.
	if assessment.ConsentCoverage < 100 && assessment.ConsentCoverage >= 0 {
		recs = append(recs,
			fmt.Sprintf("Consent verification is at %.0f%%. Ensure data subject consent is obtained for all datasets used in AI/ML.", assessment.ConsentCoverage))
	}

	// Low anonymization coverage.
	if assessment.AnonymizationCoverage < 50 {
		recs = append(recs,
			fmt.Sprintf("Only %.0f%% of datasets have anonymization applied. Implement data minimization and anonymization for PII-containing datasets.", assessment.AnonymizationCoverage))
	}

	// High overall risk.
	if assessment.RiskScore >= 75 {
		recs = append(recs,
			"Model has critical risk score. Conduct an immediate AI risk review and consider suspending data ingestion until risks are mitigated.")
	} else if assessment.RiskScore >= 50 {
		recs = append(recs,
			"Model has high risk score. Schedule an AI governance review within 30 days.")
	}

	// Training data without data minimization.
	dataMinimizationMissing := 0
	for _, u := range assessment.DataUsages {
		if (u.UsageType == model.AIUsageTrainingData || u.UsageType == model.AIUsageEvaluationData) && !u.DataMinimization {
			dataMinimizationMissing++
		}
	}
	if dataMinimizationMissing > 0 {
		recs = append(recs,
			fmt.Sprintf("%d training/evaluation datasets lack data minimization. Apply feature selection to reduce unnecessary data exposure.", dataMinimizationMissing))
	}

	// Retention compliance gaps.
	retentionGaps := 0
	for _, u := range assessment.DataUsages {
		if !u.RetentionCompliant {
			retentionGaps++
		}
	}
	if retentionGaps > 0 {
		recs = append(recs,
			fmt.Sprintf("%d datasets have retention compliance gaps. Review data retention policies for AI/ML workloads.", retentionGaps))
	}

	if len(recs) == 0 {
		recs = append(recs, "Model data governance posture is satisfactory. Continue regular monitoring.")
	}

	return recs
}
