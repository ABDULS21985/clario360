package scorer

import "math"

func ApplyDailyDecay(score, rate, days float64) float64 {
	if score <= 0 || rate <= 0 || days <= 0 {
		return score
	}
	next := score * math.Pow(1-rate, days)
	if next < 0 {
		return 0
	}
	return next
}
