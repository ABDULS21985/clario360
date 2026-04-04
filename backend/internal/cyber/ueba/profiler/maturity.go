package profiler

import "github.com/clario360/platform/internal/cyber/ueba/model"

func ClassifyMaturity(observationCount int64, daysActive int) model.ProfileMaturity {
	switch {
	case observationCount < 100 || daysActive < 30:
		return model.ProfileMaturityLearning
	case observationCount < 1000 || daysActive < 90:
		return model.ProfileMaturityBaseline
	default:
		return model.ProfileMaturityMature
	}
}
