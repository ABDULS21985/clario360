package explainer

import (
	"math"
	"sort"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type SHAPExplainer struct{}

func NewSHAPExplainer() *SHAPExplainer {
	return &SHAPExplainer{}
}

func (e *SHAPExplainer) FromWeights(
	values map[string]float64,
	baseline map[string]float64,
	weights map[string]float64,
	raw map[string]any,
) []predictmodel.FeatureContribution {
	out := make([]predictmodel.FeatureContribution, 0, len(weights))
	for feature, weight := range weights {
		delta := values[feature] - baseline[feature]
		contribution := delta * weight
		direction := "stable"
		switch {
		case contribution > 0:
			direction = "increase"
		case contribution < 0:
			direction = "decrease"
		}
		out = append(out, predictmodel.FeatureContribution{
			Feature:   feature,
			SHAPValue: round4(contribution),
			Direction: direction,
			Value:     raw[feature],
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai := math.Abs(out[i].SHAPValue)
		aj := math.Abs(out[j].SHAPValue)
		if ai == aj {
			return out[i].Feature < out[j].Feature
		}
		return ai > aj
	})
	return out
}

func (e *SHAPExplainer) TopN(items []predictmodel.FeatureContribution, limit int) []predictmodel.FeatureContribution {
	if limit <= 0 || len(items) <= limit {
		return append([]predictmodel.FeatureContribution(nil), items...)
	}
	return append([]predictmodel.FeatureContribution(nil), items[:limit]...)
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
