package service

import "testing"

func TestQuorumPercentage(t *testing.T) {
	got, err := computeQuorumRequired(10, "percentage", 51, nil)
	if err != nil {
		t.Fatalf("computeQuorumRequired returned error: %v", err)
	}
	if got != 6 {
		t.Fatalf("computeQuorumRequired(10, 51%%) = %d, want 6", got)
	}
}

func TestQuorumFixedCount(t *testing.T) {
	fixed := 3
	got, err := computeQuorumRequired(10, "fixed_count", 0, &fixed)
	if err != nil {
		t.Fatalf("computeQuorumRequired returned error: %v", err)
	}
	if got != 3 {
		t.Fatalf("computeQuorumRequired fixed count = %d, want 3", got)
	}
}

func TestQuorumMet(t *testing.T) {
	if !quorumMet(6, 7) {
		t.Fatal("quorumMet(6, 7) = false, want true")
	}
}

func TestQuorumNotMet(t *testing.T) {
	if quorumMet(6, 4) {
		t.Fatal("quorumMet(6, 4) = true, want false")
	}
}

func TestQuorumWithProxy(t *testing.T) {
	presentWithProxy := 5 + 2
	if !quorumMet(6, presentWithProxy) {
		t.Fatalf("quorumMet(6, %d) = false, want true", presentWithProxy)
	}
}
