package risk

import (
	"testing"
	"time"
)

func TestNextTwoAMUTC(t *testing.T) {
	t.Parallel()

	sameDay := nextTwoAMUTC(time.Date(2026, 3, 7, 1, 30, 0, 0, time.UTC))
	if !sameDay.Equal(time.Date(2026, 3, 7, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected same-day 02:00 UTC, got %v", sameDay)
	}

	nextDay := nextTwoAMUTC(time.Date(2026, 3, 7, 3, 0, 0, 0, time.UTC))
	if !nextDay.Equal(time.Date(2026, 3, 8, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected next-day 02:00 UTC, got %v", nextDay)
	}
}
