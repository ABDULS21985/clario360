package profiler

func EMA(oldValue, newValue, alpha float64) float64 {
	if alpha <= 0 {
		return oldValue
	}
	if alpha >= 1 {
		return newValue
	}
	return alpha*newValue + (1-alpha)*oldValue
}
