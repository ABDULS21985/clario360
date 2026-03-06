package logger

import "github.com/rs/zerolog"

// NewDebugSampler creates a zerolog.LevelSampler that samples only debug-level logs.
//
// In production, debug logs are voluminous. This sampler emits every Nth debug entry.
// Info, warn, error, and fatal levels are never sampled — they are always emitted.
//
// Usage:
//
//	logger = logger.Sample(NewDebugSampler(100))
func NewDebugSampler(n int) zerolog.Sampler {
	if n <= 0 {
		n = 100
	}
	return &zerolog.LevelSampler{
		DebugSampler: &zerolog.BasicSampler{N: uint32(n)},
	}
}
