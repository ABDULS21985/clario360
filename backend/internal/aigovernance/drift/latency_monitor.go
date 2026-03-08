package drift

func LatencyChangePct(reference, current *float64) *float64 {
	if reference == nil || current == nil || *reference == 0 {
		return nil
	}
	value := ((*current - *reference) / *reference) * 100
	return &value
}
