package discovery

import "github.com/clario360/platform/internal/data/model"

func MaxClassification(values ...model.DataClassification) model.DataClassification {
	max := model.DataClassificationPublic
	for _, value := range values {
		if classificationRank(value) > classificationRank(max) {
			max = value
		}
	}
	return max
}

func classificationRank(value model.DataClassification) int {
	switch value {
	case model.DataClassificationRestricted:
		return 4
	case model.DataClassificationConfidential:
		return 3
	case model.DataClassificationInternal:
		return 2
	default:
		return 1
	}
}
