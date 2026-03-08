package security

import (
	"sync"
	"time"
)

// MemStore is an in-memory key-value store with TTL that provides
// rate limiting, session management, and lockout tracking when Redis
// is unavailable. It is NOT a Redis replacement for distributed deployments —
// it works per-process only. In a single-replica dev/staging environment,
// this is sufficient. In production multi-replica deployments, Redis MUST be used.
type MemStore struct {
	mu      sync.RWMutex
	entries map[string]*memEntry
	sets    map[string]map[string]struct{} // for SMEMBERS-like ops
	stopGC  chan struct{}
}

type memEntry struct {
	value     []byte
	expiresAt time.Time
	counter   int64
	sortedSet []sortedSetMember // for sliding window
}

type sortedSetMember struct {
	score  float64
	member string
}

// NewMemStore creates a new in-memory store and starts a background GC goroutine.
func NewMemStore() *MemStore {
	ms := &MemStore{
		entries: make(map[string]*memEntry),
		sets:    make(map[string]map[string]struct{}),
		stopGC:  make(chan struct{}),
	}
	go ms.gc()
	return ms
}

// Close stops the background GC.
func (ms *MemStore) Close() {
	close(ms.stopGC)
}

// gc runs every 30 seconds to evict expired entries.
func (ms *MemStore) gc() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ms.evictExpired()
		case <-ms.stopGC:
			return
		}
	}
}

func (ms *MemStore) evictExpired() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	now := time.Now()
	for key, entry := range ms.entries {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			delete(ms.entries, key)
		}
	}
}

// Set stores a value with TTL.
func (ms *MemStore) Set(key string, value []byte, ttl time.Duration) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.entries[key] = &memEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value. Returns nil if expired or missing.
func (ms *MemStore) Get(key string) ([]byte, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	entry, ok := ms.entries[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

// Del removes a key.
func (ms *MemStore) Del(key string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.entries, key)
	delete(ms.sets, key)
}

// Exists checks if a key exists and is not expired.
func (ms *MemStore) Exists(key string) bool {
	_, ok := ms.Get(key)
	return ok
}

// Incr increments a counter and returns the new value.
func (ms *MemStore) Incr(key string, ttl time.Duration) int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	entry, ok := ms.entries[key]
	if !ok || (!entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)) {
		ms.entries[key] = &memEntry{
			counter:   1,
			expiresAt: time.Now().Add(ttl),
		}
		return 1
	}
	entry.counter++
	return entry.counter
}

// GetCounter returns the current counter value.
func (ms *MemStore) GetCounter(key string) int64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	entry, ok := ms.entries[key]
	if !ok || (!entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)) {
		return 0
	}
	return entry.counter
}

// SlidingWindowAdd adds an entry and returns the count within the window.
func (ms *MemStore) SlidingWindowAdd(key string, window time.Duration) int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	entry, ok := ms.entries[key]
	if !ok {
		entry = &memEntry{
			expiresAt: now.Add(window + time.Minute),
		}
		ms.entries[key] = entry
	}

	// Remove entries outside window
	var kept []sortedSetMember
	for _, m := range entry.sortedSet {
		if time.Unix(0, int64(m.score)) .After(cutoff) {
			kept = append(kept, m)
		}
	}

	// Add current
	kept = append(kept, sortedSetMember{
		score:  float64(now.UnixNano()),
		member: now.String(),
	})

	entry.sortedSet = kept
	entry.expiresAt = now.Add(window + time.Minute)

	return len(kept) - 1 // count before this request
}

// SAdd adds members to a set.
func (ms *MemStore) SAdd(key string, members ...string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.sets[key] == nil {
		ms.sets[key] = make(map[string]struct{})
	}
	for _, m := range members {
		ms.sets[key][m] = struct{}{}
	}
}

// SRem removes a member from a set.
func (ms *MemStore) SRem(key string, member string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.sets[key] != nil {
		delete(ms.sets[key], member)
	}
}

// SMembers returns all members of a set.
func (ms *MemStore) SMembers(key string) []string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	set := ms.sets[key]
	if set == nil {
		return nil
	}
	members := make([]string, 0, len(set))
	for m := range set {
		members = append(members, m)
	}
	return members
}

// SCount returns the size of a set.
func (ms *MemStore) SCount(key string) int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.sets[key])
}
