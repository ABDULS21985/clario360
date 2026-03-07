package transforms

import "time"

type TransformStats struct {
	InputRows     int
	OutputRows    int
	FilteredRows  int
	DedupedRows   int
	ErrorRows     int
	Duration      time.Duration
}

