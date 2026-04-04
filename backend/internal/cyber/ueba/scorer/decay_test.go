package scorer

import (
	"math"
	"testing"
)

func TestApplyDailyDecay(t *testing.T) {
	if got := ApplyDailyDecay(80, 0.10, 1); math.Abs(got-72) > 0.001 {
		t.Fatalf("ApplyDailyDecay daily = %v, want 72", got)
	}
	if got := ApplyDailyDecay(80, 0.10, 7); math.Abs(got-38.263752) > 0.01 {
		t.Fatalf("ApplyDailyDecay weekly = %v", got)
	}
	if got := ApplyDailyDecay(0.5, 0.10, 7); got < 0 {
		t.Fatalf("ApplyDailyDecay should never go below zero, got %v", got)
	}
}
