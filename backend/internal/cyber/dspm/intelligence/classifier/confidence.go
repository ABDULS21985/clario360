package classifier

// ConfidenceCalculator computes a weighted confidence score from three
// classification layers: pattern matching, content inspection, and
// statistical analysis.
type ConfidenceCalculator struct {
	PatternWeight     float64
	ContentWeight     float64
	StatisticalWeight float64
}

// NewConfidenceCalculator creates a ConfidenceCalculator with default weights.
func NewConfidenceCalculator() *ConfidenceCalculator {
	return &ConfidenceCalculator{
		PatternWeight:     0.3,
		ContentWeight:     0.5,
		StatisticalWeight: 0.2,
	}
}

// Calculate returns the weighted average confidence across three layers.
// Each input should be in [0.0, 1.0]. If total weight is zero the result is 0.
func (cc *ConfidenceCalculator) Calculate(patternConf, contentConf, statisticalConf float64) float64 {
	totalWeight := cc.PatternWeight + cc.ContentWeight + cc.StatisticalWeight
	if totalWeight == 0 {
		return 0
	}

	weighted := (patternConf * cc.PatternWeight) +
		(contentConf * cc.ContentWeight) +
		(statisticalConf * cc.StatisticalWeight)

	result := weighted / totalWeight

	// Clamp to [0, 1].
	if result > 1.0 {
		return 1.0
	}
	if result < 0.0 {
		return 0.0
	}
	return result
}

// NeedsHumanReview returns true when the confidence score is below the
// threshold where automated classification is considered unreliable.
func (cc *ConfidenceCalculator) NeedsHumanReview(confidence float64) bool {
	return confidence < 0.5
}
