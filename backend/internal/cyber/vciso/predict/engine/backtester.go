package engine

import (
	"fmt"
	"math"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type Backtester struct{}

func NewBacktester() *Backtester {
	return &Backtester{}
}

func (b *Backtester) Classification(predicted, actual []float64, threshold float64) (predictmodel.BacktestMetrics, error) {
	if len(predicted) == 0 || len(predicted) != len(actual) {
		return predictmodel.BacktestMetrics{}, fmt.Errorf("classification backtest requires aligned predicted and actual slices")
	}
	if threshold <= 0 {
		threshold = 0.5
	}
	tp := 0.0
	fp := 0.0
	tn := 0.0
	fn := 0.0
	for idx := range predicted {
		actualPositive := actual[idx] >= threshold
		predPositive := predicted[idx] >= threshold
		switch {
		case predPositive && actualPositive:
			tp++
		case predPositive && !actualPositive:
			fp++
		case !predPositive && !actualPositive:
			tn++
		default:
			fn++
		}
	}
	precision := safeDivide(tp, tp+fp)
	recall := safeDivide(tp, tp+fn)
	accuracy := safeDivide(tp+tn, tp+tn+fp+fn)
	f1 := safeDivide(2*precision*recall, precision+recall)
	return predictmodel.BacktestMetrics{
		Accuracy:  accuracy,
		Precision: precision,
		Recall:    recall,
		F1:        f1,
		Count:     len(predicted),
	}, nil
}

func (b *Backtester) Regression(predicted, actual []float64) (predictmodel.BacktestMetrics, error) {
	if len(predicted) == 0 || len(predicted) != len(actual) {
		return predictmodel.BacktestMetrics{}, fmt.Errorf("regression backtest requires aligned predicted and actual slices")
	}
	totalMAPE := 0.0
	totalError := 0.0
	for idx := range predicted {
		diff := math.Abs(actual[idx] - predicted[idx])
		totalError += diff
		denominator := math.Abs(actual[idx])
		if denominator < 1 {
			denominator = 1
		}
		totalMAPE += diff / denominator
	}
	mae := totalError / float64(len(predicted))
	mape := totalMAPE / float64(len(predicted))
	accuracy := clamp(1-mape, 0, 1)
	return predictmodel.BacktestMetrics{
		Accuracy: accuracy,
		MAPE:     mape,
		Count:    len(predicted),
		Recall:   mae,
	}, nil
}

func (b *Backtester) ClusterQuality(cohesion []float64) (predictmodel.BacktestMetrics, error) {
	if len(cohesion) == 0 {
		return predictmodel.BacktestMetrics{}, fmt.Errorf("cluster quality requires at least one cohesion score")
	}
	value := mean(cohesion)
	return predictmodel.BacktestMetrics{
		Accuracy: clamp(value, 0, 1),
		Count:    len(cohesion),
	}, nil
}

func safeDivide(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}
