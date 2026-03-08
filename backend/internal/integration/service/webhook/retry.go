package webhook

import "time"

func BackoffForAttempt(attempt int) time.Duration {
	switch attempt {
	case 1:
		return time.Second
	case 2:
		return 5 * time.Second
	default:
		return 25 * time.Second
	}
}
