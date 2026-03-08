package detection

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/aigovernance"
	"github.com/clario360/platform/internal/cyber/model"
)

func (e *DetectionEngine) recordGovernedRuleEvaluation(ctx context.Context, tenantID uuid.UUID, loadedRule *LoadedRule, eventsBatch []model.SecurityEvent, matches []model.RuleMatch) {
	if e.predictionLogger == nil || loadedRule == nil || loadedRule.Rule == nil {
		return
	}

	switch loadedRule.Rule.RuleType {
	case model.RuleTypeSigma:
		e.recordSigmaEvaluation(ctx, tenantID, loadedRule.Rule, eventsBatch, matches)
	case model.RuleTypeAnomaly:
		e.recordAnomalyEvaluation(ctx, tenantID, loadedRule.Rule, loadedRule.Compiled, eventsBatch, matches)
	}
}

func (e *DetectionEngine) recordSigmaEvaluation(ctx context.Context, tenantID uuid.UUID, rule *model.DetectionRule, eventsBatch []model.SecurityEvent, matches []model.RuleMatch) {
	matched := len(matches) > 0
	matchedConditions := aggregateMatchConditionKeys(matches)
	matchedRules := []string{}
	ruleWeights := map[string]any{}
	confidence := 0.35
	if matched {
		matchedRules = []string{rule.Name}
		confidence = maxFloat(0.8, rule.BaseConfidence)
		ruleWeights[rule.Name] = confidence
	}
	input := map[string]any{
		"rule_id":     rule.ID.String(),
		"rule_name":   rule.Name,
		"event_count": len(eventsBatch),
		"match_count": len(matches),
	}
	_, _ = e.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "cyber-sigma-evaluator",
		UseCase:      "threat_detection",
		EntityType:   "detection_rule",
		EntityID:     &rule.ID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"rule_id":     rule.ID,
					"rule_name":   rule.Name,
					"matched":     matched,
					"match_count": len(matches),
					"event_count": len(eventsBatch),
				},
				Confidence: confidence,
				Metadata: map[string]any{
					"matched":            matched,
					"rule_name":          rule.Name,
					"matched_rules":      matchedRules,
					"matched_conditions": matchedConditions,
					"rule_weights":       ruleWeights,
					"event_count":        len(eventsBatch),
					"match_count":        len(matches),
				},
			}, nil
		},
	})
}

func (e *DetectionEngine) recordAnomalyEvaluation(ctx context.Context, tenantID uuid.UUID, rule *model.DetectionRule, compiled any, eventsBatch []model.SecurityEvent, matches []model.RuleMatch) {
	matched := len(matches) > 0
	currentValue := 0.0
	baselineMean := 0.0
	stdDev := 0.0
	zScore := 0.0
	metricName := ""
	if matched {
		match := matches[0]
		currentValue = numericMatchDetail(match.MatchDetails["current_value"])
		baselineMean = numericMatchDetail(match.MatchDetails["mean"])
		stdDev = numericMatchDetail(match.MatchDetails["std_dev"])
		zScore = numericMatchDetail(match.MatchDetails["z_score"])
		metricName = fmt.Sprint(match.MatchDetails["metric"])
	}
	threshold := 0.0
	if cfg, ok := compiled.(*compiledAnomalyRule); ok && cfg != nil {
		threshold = cfg.ZScoreThreshold
		if metricName == "" {
			metricName = cfg.Metric
		}
	}
	if threshold == 0 {
		threshold = math.Abs(zScore)
	}
	input := map[string]any{
		"rule_id":     rule.ID.String(),
		"rule_name":   rule.Name,
		"event_count": len(eventsBatch),
		"match_count": len(matches),
		"metric":      metricName,
	}
	_, _ = e.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "cyber-anomaly-detector",
		UseCase:      "anomaly_detection",
		EntityType:   "detection_rule",
		EntityID:     &rule.ID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"rule_id":          rule.ID,
					"rule_name":        rule.Name,
					"anomaly_detected": matched,
					"match_count":      len(matches),
					"event_count":      len(eventsBatch),
					"metric":           metricName,
				},
				Confidence: anomalyConfidence(matched, rule.BaseConfidence),
				Metadata: map[string]any{
					"anomaly_detected": matched,
					"current_value":    currentValue,
					"baseline_mean":    baselineMean,
					"baseline_stddev":  stdDev,
					"z_score":          zScore,
					"threshold":        threshold,
					"match_count":      len(matches),
					"metric":           metricName,
				},
			}, nil
		},
	})
}

func aggregateMatchConditionKeys(matches []model.RuleMatch) []string {
	set := make(map[string]struct{})
	for _, match := range matches {
		for key := range match.MatchDetails {
			set[key] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func numericMatchDetail(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func anomalyConfidence(matched bool, base float64) float64 {
	if matched {
		return maxFloat(0.82, base)
	}
	if base > 0 {
		return minFloat(base, 0.6)
	}
	return 0.55
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
