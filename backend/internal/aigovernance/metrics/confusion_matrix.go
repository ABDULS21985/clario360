package metrics

type BinarySample struct {
	PredictedPositive bool
	ActualPositive    bool
}

type ConfusionMatrix struct {
	TP int
	FP int
	TN int
	FN int
}

func CalculateConfusionMatrix(samples []BinarySample) ConfusionMatrix {
	var matrix ConfusionMatrix
	for _, sample := range samples {
		switch {
		case sample.PredictedPositive && sample.ActualPositive:
			matrix.TP++
		case sample.PredictedPositive && !sample.ActualPositive:
			matrix.FP++
		case !sample.PredictedPositive && sample.ActualPositive:
			matrix.FN++
		default:
			matrix.TN++
		}
	}
	return matrix
}
