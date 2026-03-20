package ai_security

import (
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AIRiskAssessor calculates AI-specific risk scores for data usage records
// and ranks them by risk severity.
type AIRiskAssessor struct {
	logger zerolog.Logger
}

// NewAIRiskAssessor creates a new AI risk assessor instance.
func NewAIRiskAssessor(logger zerolog.Logger) *AIRiskAssessor {
	return &AIRiskAssessor{
		logger: logger.With().Str("component", "ai_risk_assessor").Logger(),
	}
}

// AssessRisk calculates the AI risk score, risk level, and contributing risk
// factors for a single AI data usage record.
//
// Score calculation:
//   - Classification weight: restricted=40, confidential=25, internal=10, public=0
//   - PII penalty: +30 if contains_pii AND usage is training_data
//   - Consent gap: +20 if consent_verified is false
//   - Anonymization gap: +10 if anonymization is "none" AND contains PII
//   - Score is clamped to [0, 100]
//
// Risk levels: >=75 critical, >=50 high, >=25 medium, else low
func (a *AIRiskAssessor) AssessRisk(usage *model.AIDataUsage) (float64, model.AIRiskLevel, []model.AIRiskFactor) {
	var factors []model.AIRiskFactor
	var score float64

	// Factor 1: Data classification sensitivity.
	classWeight := classificationWeight(usage.DataClassification)
	if classWeight > 0 {
		factors = append(factors, model.AIRiskFactor{
			Factor:      "data_classification",
			Weight:      classWeight,
			Description: formatClassificationDescription(usage.DataClassification, classWeight),
		})
		score += classWeight
	}

	// Factor 2: PII in training data.
	if usage.ContainsPII && usage.UsageType == model.AIUsageTrainingData {
		piiPenalty := 30.0
		factors = append(factors, model.AIRiskFactor{
			Factor:      "pii_in_training",
			Weight:      piiPenalty,
			Description: "PII data used directly in model training increases memorization and leakage risk",
		})
		score += piiPenalty
	}

	// Factor 3: Consent verification gap.
	if !usage.ConsentVerified {
		consentGap := 20.0
		factors = append(factors, model.AIRiskFactor{
			Factor:      "consent_gap",
			Weight:      consentGap,
			Description: "Data subject consent has not been verified for AI/ML usage",
		})
		score += consentGap
	}

	// Factor 4: Anonymization gap.
	if usage.ContainsPII && usage.AnonymizationLevel == model.AnonymizationNone {
		anonGap := 10.0
		factors = append(factors, model.AIRiskFactor{
			Factor:      "anonymization_gap",
			Weight:      anonGap,
			Description: "PII data has no anonymization applied; raw personal data may be exposed to AI systems",
		})
		score += anonGap
	}

	// Clamp score to [0, 100].
	score = clamp(score, 0, 100)

	// Determine risk level from score.
	level := scoreToRiskLevel(score)

	a.logger.Debug().
		Str("asset_id", usage.DataAssetID.String()).
		Float64("score", score).
		Str("level", string(level)).
		Int("factors", len(factors)).
		Msg("AI risk assessment completed")

	return score, level, factors
}

// RankByRisk sorts AI data usages by risk score in descending order (highest risk first).
func (a *AIRiskAssessor) RankByRisk(usages []model.AIDataUsage) []model.AIDataUsage {
	ranked := make([]model.AIDataUsage, len(usages))
	copy(ranked, usages)

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].AIRiskScore > ranked[j].AIRiskScore
	})

	return ranked
}

// scoreToRiskLevel converts a numeric score to a risk level category.
func scoreToRiskLevel(score float64) model.AIRiskLevel {
	switch {
	case score >= 75:
		return model.AIRiskCritical
	case score >= 50:
		return model.AIRiskHigh
	case score >= 25:
		return model.AIRiskMedium
	default:
		return model.AIRiskLow
	}
}

// formatClassificationDescription returns a human-readable description for the
// classification risk factor.
func formatClassificationDescription(classification string, weight float64) string {
	cl := strings.ToLower(classification)
	switch cl {
	case "restricted":
		return "Restricted data classification carries maximum sensitivity weight (40)"
	case "confidential":
		return "Confidential data classification carries high sensitivity weight (25)"
	case "internal":
		return "Internal data classification carries moderate sensitivity weight (10)"
	default:
		return "Data classification contributes to risk score"
	}
}
