package profiler

import (
	"testing"
	"time"
)

func TestLRU_Capacity(t *testing.T) {
	lru := NewLRUFrequencySet(20)
	alpha := 0.05
	base := time.Now().UTC()

	// Insert 25 items into a set with capacity 20.
	for i := 0; i < 25; i++ {
		lru.Access("item-"+string(rune('A'+i)), base.Add(time.Duration(i)*time.Second), alpha)
	}

	values := lru.Values()
	if len(values) != 20 {
		t.Fatalf("len = %d, want 20 (capacity)", len(values))
	}

	// The 5 lowest-frequency items should have been evicted. Since all items were
	// accessed exactly once with the same initial frequency (alpha), eviction is
	// by oldest LastSeen — items A-E should be gone.
	for _, item := range values {
		if item.Key >= "item-A" && item.Key <= "item-E" {
			t.Fatalf("item %s should have been evicted", item.Key)
		}
	}
}

func TestLRU_FrequencyEviction(t *testing.T) {
	lru := NewLRUFrequencySet(3)
	alpha := 0.05
	base := time.Now().UTC()

	// Insert 3 items.
	lru.Access("rare", base, alpha)
	lru.Access("medium", base.Add(time.Second), alpha)
	lru.Access("frequent", base.Add(2*time.Second), alpha)

	// Boost "frequent" and "medium" frequencies via repeated access so they survive eviction.
	for i := 0; i < 20; i++ {
		ts := base.Add(time.Duration(3+i) * time.Second)
		lru.Access("frequent", ts, alpha)
		if i%3 == 0 {
			lru.Access("medium", ts, alpha)
		}
	}

	// Add a new item — "rare" should be evicted (lowest frequency).
	lru.Access("newcomer", base.Add(30*time.Second), alpha)
	values := lru.Values()
	if len(values) != 3 {
		t.Fatalf("len = %d, want 3", len(values))
	}

	keys := make(map[string]bool, len(values))
	for _, item := range values {
		keys[item.Key] = true
	}
	if keys["rare"] {
		t.Fatal("rare should have been evicted (lowest frequency)")
	}
	if !keys["frequent"] || !keys["medium"] || !keys["newcomer"] {
		t.Fatalf("expected frequent, medium, newcomer; got %v", keys)
	}
}

func TestLRU_AccessMovesFront(t *testing.T) {
	lru := NewLRUFrequencySet(5)
	alpha := 0.05
	base := time.Now().UTC()

	lru.Access("first", base, alpha)
	lru.Access("second", base.Add(time.Second), alpha)
	lru.Access("third", base.Add(2*time.Second), alpha)

	// "first" is now at the back. Access it again to move to front.
	lru.Access("first", base.Add(3*time.Second), alpha)

	values := lru.Values()
	if values[0].Key != "first" {
		t.Fatalf("front item = %s, want first (after re-access)", values[0].Key)
	}

	// Verify frequency was updated via EMA: initial=alpha=0.05, second access EMA(0.05, 1, 0.05) ≈ 0.0975.
	if values[0].Frequency <= alpha {
		t.Fatalf("frequency = %v, want > %v after re-access", values[0].Frequency, alpha)
	}
}
