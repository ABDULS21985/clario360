package drift

func VolumeChangePct(reference, current int64) *float64 {
	if reference == 0 {
		if current == 0 {
			value := 0.0
			return &value
		}
		value := 100.0
		return &value
	}
	value := ((float64(current) - float64(reference)) / float64(reference)) * 100
	return &value
}
