package logger

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestNewDebugSampler_ReturnsNonNil(t *testing.T) {
	sampler := NewDebugSampler(50)
	if sampler == nil {
		t.Fatal("NewDebugSampler(50) returned nil")
	}

	ls, ok := sampler.(*zerolog.LevelSampler)
	if !ok {
		t.Fatal("NewDebugSampler did not return a *zerolog.LevelSampler")
	}
	if ls.DebugSampler == nil {
		t.Fatal("DebugSampler field is nil")
	}

	bs, ok := ls.DebugSampler.(*zerolog.BasicSampler)
	if !ok {
		t.Fatal("DebugSampler is not a *zerolog.BasicSampler")
	}
	if bs.N != 50 {
		t.Errorf("BasicSampler.N = %d, want %d", bs.N, 50)
	}
}

func TestNewDebugSampler_DefaultsOnZero(t *testing.T) {
	cases := []struct {
		name  string
		input int
	}{
		{"zero", 0},
		{"negative", -5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sampler := NewDebugSampler(tc.input)
			if sampler == nil {
				t.Fatalf("NewDebugSampler(%d) returned nil", tc.input)
			}

			ls, ok := sampler.(*zerolog.LevelSampler)
			if !ok {
				t.Fatal("NewDebugSampler did not return a *zerolog.LevelSampler")
			}

			bs, ok := ls.DebugSampler.(*zerolog.BasicSampler)
			if !ok {
				t.Fatal("DebugSampler is not a *zerolog.BasicSampler")
			}

			if bs.N != 100 {
				t.Errorf("BasicSampler.N = %d, want default 100", bs.N)
			}
		})
	}
}
