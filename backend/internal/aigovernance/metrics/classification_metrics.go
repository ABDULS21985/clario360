package metrics

import aigovmodel "github.com/clario360/platform/internal/aigovernance/model"

func Precision(tp, fp int) float64 {
	return safeDivide(float64(tp), float64(tp+fp))
}

func Recall(tp, fn int) float64 {
	return safeDivide(float64(tp), float64(tp+fn))
}

func F1(precision, recall float64) float64 {
	return safeDivide(2*precision*recall, precision+recall)
}

func FalsePositiveRate(fp, tn int) float64 {
	return safeDivide(float64(fp), float64(fp+tn))
}

func FalseNegativeRate(tp, fn int) float64 {
	return safeDivide(float64(fn), float64(tp+fn))
}

func Accuracy(tp, fp, tn, fn int) float64 {
	return safeDivide(float64(tp+tn), float64(tp+fp+tn+fn))
}

func Summary(matrix ConfusionMatrix, auc float64) aigovmodel.MetricsSummary {
	precision := Precision(matrix.TP, matrix.FP)
	recall := Recall(matrix.TP, matrix.FN)
	return aigovmodel.MetricsSummary{
		DatasetSize:       matrix.TP + matrix.FP + matrix.TN + matrix.FN,
		PositiveCount:     matrix.TP + matrix.FN,
		NegativeCount:     matrix.TN + matrix.FP,
		TruePositives:     matrix.TP,
		FalsePositives:    matrix.FP,
		TrueNegatives:     matrix.TN,
		FalseNegatives:    matrix.FN,
		Precision:         precision,
		Recall:            recall,
		F1Score:           F1(precision, recall),
		FalsePositiveRate: FalsePositiveRate(matrix.FP, matrix.TN),
		FalseNegativeRate: FalseNegativeRate(matrix.TP, matrix.FN),
		Accuracy:          Accuracy(matrix.TP, matrix.FP, matrix.TN, matrix.FN),
		AUC:               auc,
	}
}

func safeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}
