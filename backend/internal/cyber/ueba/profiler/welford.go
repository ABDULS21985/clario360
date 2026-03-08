package profiler

import "math"

func WelfordUpdate(count int64, mean, m2, newValue float64) (newMean, newM2, newStddev float64) {
	nextCount := count + 1
	delta := newValue - mean
	newMean = mean + delta/float64(nextCount)
	delta2 := newValue - newMean
	newM2 = m2 + delta*delta2
	if nextCount > 1 {
		newStddev = math.Sqrt(newM2 / float64(nextCount-1))
	}
	return newMean, newM2, newStddev
}
