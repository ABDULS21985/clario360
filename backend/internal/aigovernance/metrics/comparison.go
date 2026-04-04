package metrics

import aigovmodel "github.com/clario360/platform/internal/aigovernance/model"

func CompareMetrics(candidate, production aigovmodel.MetricsSummary) map[string]float64 {
	return map[string]float64{
		"precision":           candidate.Precision - production.Precision,
		"recall":              candidate.Recall - production.Recall,
		"f1_score":            candidate.F1Score - production.F1Score,
		"false_positive_rate": candidate.FalsePositiveRate - production.FalsePositiveRate,
		"accuracy":            candidate.Accuracy - production.Accuracy,
		"auc":                 candidate.AUC - production.AUC,
	}
}
