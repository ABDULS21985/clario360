package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeEntryHash(t *testing.T) {
	prevHash := "abc123"
	action := HistoryActionStepCompleted
	details := json.RawMessage(`{"step_id":"step-1","status":"completed"}`)
	timestamp := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	hash1 := ComputeEntryHash(prevHash, action, details, timestamp)
	hash2 := ComputeEntryHash(prevHash, action, details, timestamp)

	assert.NotEmpty(t, hash1, "hash should not be empty")
	assert.Equal(t, hash1, hash2, "same inputs should produce identical hashes (deterministic)")
	assert.Len(t, hash1, 64, "SHA-256 hex digest should be 64 characters")
}

func TestComputeEntryHashDifferentInputs(t *testing.T) {
	timestamp := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	details := json.RawMessage(`{"key":"value"}`)

	t.Run("different_prev_hash", func(t *testing.T) {
		hash1 := ComputeEntryHash("hash-a", HistoryActionStepCompleted, details, timestamp)
		hash2 := ComputeEntryHash("hash-b", HistoryActionStepCompleted, details, timestamp)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("different_action", func(t *testing.T) {
		hash1 := ComputeEntryHash("prev", HistoryActionStepCompleted, details, timestamp)
		hash2 := ComputeEntryHash("prev", HistoryActionStepFailed, details, timestamp)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("different_details", func(t *testing.T) {
		details1 := json.RawMessage(`{"a":1}`)
		details2 := json.RawMessage(`{"b":2}`)
		hash1 := ComputeEntryHash("prev", HistoryActionStepCompleted, details1, timestamp)
		hash2 := ComputeEntryHash("prev", HistoryActionStepCompleted, details2, timestamp)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("different_timestamp", func(t *testing.T) {
		ts1 := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
		ts2 := time.Date(2025, 6, 15, 10, 31, 0, 0, time.UTC)
		hash1 := ComputeEntryHash("prev", HistoryActionStepCompleted, details, ts1)
		hash2 := ComputeEntryHash("prev", HistoryActionStepCompleted, details, ts2)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty_prev_hash", func(t *testing.T) {
		hash1 := ComputeEntryHash("", HistoryActionCreated, details, timestamp)
		hash2 := ComputeEntryHash("something", HistoryActionCreated, details, timestamp)
		assert.NotEqual(t, hash1, hash2)
	})
}

func TestVerifyChainValid(t *testing.T) {
	remediationID := uuid.New()
	tenantID := uuid.New()

	ts1 := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)
	ts3 := time.Date(2025, 6, 15, 10, 10, 0, 0, time.UTC)

	details1 := json.RawMessage(`{"status":"open"}`)
	details2 := json.RawMessage(`{"step_id":"step-1"}`)
	details3 := json.RawMessage(`{"step_id":"step-1","result":"success"}`)

	// Build a valid chain.
	entry1Hash := ComputeEntryHash("", HistoryActionCreated, details1, ts1)
	entry2Hash := ComputeEntryHash(entry1Hash, HistoryActionStepStarted, details2, ts2)
	entry3Hash := ComputeEntryHash(entry2Hash, HistoryActionStepCompleted, details3, ts3)

	entries := []RemediationHistory{
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			RemediationID: remediationID,
			Action:        HistoryActionCreated,
			ActorType:     ActorTypeSystem,
			Details:       details1,
			EntryHash:     entry1Hash,
			PrevHash:      "",
			CreatedAt:     ts1,
		},
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			RemediationID: remediationID,
			Action:        HistoryActionStepStarted,
			ActorType:     ActorTypeSystem,
			Details:       details2,
			EntryHash:     entry2Hash,
			PrevHash:      entry1Hash,
			CreatedAt:     ts2,
		},
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			RemediationID: remediationID,
			Action:        HistoryActionStepCompleted,
			ActorType:     ActorTypeSystem,
			Details:       details3,
			EntryHash:     entry3Hash,
			PrevHash:      entry2Hash,
			CreatedAt:     ts3,
		},
	}

	valid, failedIdx := VerifyChain(entries)
	assert.True(t, valid, "valid chain should verify correctly")
	assert.Equal(t, -1, failedIdx, "failed index should be -1 for valid chain")
}

func TestVerifyChainTampered(t *testing.T) {
	ts1 := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)

	details1 := json.RawMessage(`{"status":"open"}`)
	details2 := json.RawMessage(`{"step_id":"step-1"}`)

	entry1Hash := ComputeEntryHash("", HistoryActionCreated, details1, ts1)
	entry2Hash := ComputeEntryHash(entry1Hash, HistoryActionStepStarted, details2, ts2)

	entries := []RemediationHistory{
		{
			ID:        uuid.New(),
			Action:    HistoryActionCreated,
			ActorType: ActorTypeSystem,
			Details:   details1,
			EntryHash: entry1Hash,
			PrevHash:  "",
			CreatedAt: ts1,
		},
		{
			ID:        uuid.New(),
			Action:    HistoryActionStepStarted,
			ActorType: ActorTypeSystem,
			Details:   details2,
			EntryHash: entry2Hash,
			PrevHash:  entry1Hash,
			CreatedAt: ts2,
		},
	}

	// Tamper with the first entry's details.
	entries[0].Details = json.RawMessage(`{"status":"tampered"}`)

	valid, failedIdx := VerifyChain(entries)
	assert.False(t, valid, "tampered chain should fail verification")
	assert.Equal(t, 0, failedIdx, "tampered entry should be detected at index 0")
}

func TestVerifyChainTamperedMiddleEntry(t *testing.T) {
	ts1 := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)
	ts3 := time.Date(2025, 6, 15, 10, 10, 0, 0, time.UTC)

	details1 := json.RawMessage(`{"status":"open"}`)
	details2 := json.RawMessage(`{"step_id":"step-1"}`)
	details3 := json.RawMessage(`{"step_id":"step-1","result":"done"}`)

	entry1Hash := ComputeEntryHash("", HistoryActionCreated, details1, ts1)
	entry2Hash := ComputeEntryHash(entry1Hash, HistoryActionStepStarted, details2, ts2)
	entry3Hash := ComputeEntryHash(entry2Hash, HistoryActionStepCompleted, details3, ts3)

	entries := []RemediationHistory{
		{
			ID:        uuid.New(),
			Action:    HistoryActionCreated,
			Details:   details1,
			EntryHash: entry1Hash,
			PrevHash:  "",
			CreatedAt: ts1,
		},
		{
			ID:        uuid.New(),
			Action:    HistoryActionStepStarted,
			Details:   details2,
			EntryHash: entry2Hash,
			PrevHash:  entry1Hash,
			CreatedAt: ts2,
		},
		{
			ID:        uuid.New(),
			Action:    HistoryActionStepCompleted,
			Details:   details3,
			EntryHash: entry3Hash,
			PrevHash:  entry2Hash,
			CreatedAt: ts3,
		},
	}

	// Tamper with second entry action.
	entries[1].Action = HistoryActionStepFailed

	valid, failedIdx := VerifyChain(entries)
	assert.False(t, valid, "tampered chain should fail verification")
	assert.Equal(t, 1, failedIdx, "tampered entry should be detected at index 1")
}

func TestVerifyChainEmpty(t *testing.T) {
	valid, failedIdx := VerifyChain([]RemediationHistory{})
	assert.True(t, valid, "empty chain should verify correctly")
	assert.Equal(t, -1, failedIdx)
}

func TestVerifyChainSingleEntry(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	details := json.RawMessage(`{"status":"open"}`)
	hash := ComputeEntryHash("", HistoryActionCreated, details, ts)

	entries := []RemediationHistory{
		{
			ID:        uuid.New(),
			Action:    HistoryActionCreated,
			Details:   details,
			EntryHash: hash,
			PrevHash:  "",
			CreatedAt: ts,
		},
	}

	valid, failedIdx := VerifyChain(entries)
	assert.True(t, valid, "single valid entry should verify correctly")
	assert.Equal(t, -1, failedIdx)
}

func TestVerifyChainBrokenLink(t *testing.T) {
	ts1 := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)

	details1 := json.RawMessage(`{"status":"open"}`)
	details2 := json.RawMessage(`{"step_id":"step-1"}`)

	entry1Hash := ComputeEntryHash("", HistoryActionCreated, details1, ts1)

	// Entry 2 uses a WRONG prev_hash (not entry1Hash).
	wrongPrevHash := "wrong_prev_hash_value"
	entry2Hash := ComputeEntryHash(wrongPrevHash, HistoryActionStepStarted, details2, ts2)

	entries := []RemediationHistory{
		{
			ID:        uuid.New(),
			Action:    HistoryActionCreated,
			Details:   details1,
			EntryHash: entry1Hash,
			PrevHash:  "",
			CreatedAt: ts1,
		},
		{
			ID:        uuid.New(),
			Action:    HistoryActionStepStarted,
			Details:   details2,
			EntryHash: entry2Hash,
			PrevHash:  wrongPrevHash, // does not match entry1Hash
			CreatedAt: ts2,
		},
	}

	valid, failedIdx := VerifyChain(entries)
	assert.False(t, valid, "broken prev_hash link should fail verification")
	assert.Equal(t, 1, failedIdx, "broken link should be detected at index 1")
}

func TestComputeEntryHashWithNilDetails(t *testing.T) {
	prevHash := ""
	action := HistoryActionCreated
	timestamp := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	hash := ComputeEntryHash(prevHash, action, nil, timestamp)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestVerifyChainLongerSequence(t *testing.T) {
	baseTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	actions := []HistoryAction{
		HistoryActionCreated,
		HistoryActionAssigned,
		HistoryActionStepStarted,
		HistoryActionStepCompleted,
		HistoryActionStepStarted,
		HistoryActionStepCompleted,
		HistoryActionStatusChanged,
	}

	var entries []RemediationHistory
	prevHash := ""

	for i, action := range actions {
		ts := baseTime.Add(time.Duration(i) * 5 * time.Minute)
		details := json.RawMessage(`{"index":` + string(rune('0'+i)) + `}`)
		// Use a safe JSON encoding for the index.
		detailsJSON, _ := json.Marshal(map[string]int{"index": i})
		details = json.RawMessage(detailsJSON)

		hash := ComputeEntryHash(prevHash, action, details, ts)
		entries = append(entries, RemediationHistory{
			ID:        uuid.New(),
			Action:    action,
			Details:   details,
			EntryHash: hash,
			PrevHash:  prevHash,
			CreatedAt: ts,
		})
		prevHash = hash
	}

	require.Len(t, entries, 7)

	valid, failedIdx := VerifyChain(entries)
	assert.True(t, valid, "7-entry chain should verify correctly")
	assert.Equal(t, -1, failedIdx)

	// Tamper with entry at index 4.
	entries[4].Details = json.RawMessage(`{"tampered":true}`)

	valid, failedIdx = VerifyChain(entries)
	assert.False(t, valid, "tampered chain should fail")
	assert.Equal(t, 4, failedIdx)
}

func TestHistoryActionConstants(t *testing.T) {
	// Verify all action constants have non-empty values.
	actions := []HistoryAction{
		HistoryActionStepStarted,
		HistoryActionStepCompleted,
		HistoryActionStepFailed,
		HistoryActionAssigned,
		HistoryActionStatusChanged,
		HistoryActionRolledBack,
		HistoryActionSLABreached,
		HistoryActionExceptionGranted,
		HistoryActionNoteAdded,
		HistoryActionCreated,
	}

	for _, action := range actions {
		assert.NotEmpty(t, string(action), "action constant should not be empty")
	}
}

func TestActorTypeConstants(t *testing.T) {
	actors := []ActorType{
		ActorTypeUser,
		ActorTypeSystem,
		ActorTypePolicyEngine,
		ActorTypeScheduler,
	}

	for _, actor := range actors {
		assert.NotEmpty(t, string(actor), "actor type constant should not be empty")
	}
}
