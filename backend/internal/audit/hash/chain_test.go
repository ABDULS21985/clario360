package hash

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/clario360/platform/internal/audit/model"
)

func makeEntry(id, tenantID, service, action string) *model.AuditEntry {
	userID := "user-123"
	return &model.AuditEntry{
		ID:           id,
		TenantID:     tenantID,
		UserID:       &userID,
		UserEmail:    "test@example.com",
		Service:      service,
		Action:       action,
		Severity:     "info",
		ResourceType: "user",
		ResourceID:   "res-456",
		OldValue:     json.RawMessage(`{"name":"old"}`),
		NewValue:     json.RawMessage(`{"name":"new"}`),
		EventID:      "evt-789",
		CreatedAt:    time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC),
	}
}

func TestComputeEntryHash_Deterministic(t *testing.T) {
	entry := makeEntry("id-1", "tenant-1", "iam-service", "user.created")

	hash1 := ComputeEntryHash(entry, GenesisHash)
	hash2 := ComputeEntryHash(entry, GenesisHash)

	if hash1 != hash2 {
		t.Errorf("expected identical hashes, got %s and %s", hash1, hash2)
	}
	if len(hash1) != 64 {
		t.Errorf("expected 64-char hex SHA-256, got length %d", len(hash1))
	}
}

func TestComputeEntryHash_DifferentInputs(t *testing.T) {
	entry1 := makeEntry("id-1", "tenant-1", "iam-service", "user.created")
	entry2 := makeEntry("id-2", "tenant-1", "iam-service", "user.created")

	hash1 := ComputeEntryHash(entry1, GenesisHash)
	hash2 := ComputeEntryHash(entry2, GenesisHash)

	if hash1 == hash2 {
		t.Error("expected different hashes for different IDs")
	}

	// Different action
	entry3 := makeEntry("id-1", "tenant-1", "iam-service", "user.deleted")
	hash3 := ComputeEntryHash(entry3, GenesisHash)
	if hash1 == hash3 {
		t.Error("expected different hashes for different actions")
	}

	// Different tenant
	entry4 := makeEntry("id-1", "tenant-2", "iam-service", "user.created")
	hash4 := ComputeEntryHash(entry4, GenesisHash)
	if hash1 == hash4 {
		t.Error("expected different hashes for different tenants")
	}

	// Different previous hash
	hash5 := ComputeEntryHash(entry1, "some-other-previous-hash")
	if hash1 == hash5 {
		t.Error("expected different hashes for different previous hashes")
	}
}

func TestComputeEntryHash_NilFields(t *testing.T) {
	entry := &model.AuditEntry{
		ID:           "id-1",
		TenantID:     "tenant-1",
		UserID:       nil, // nil user_id
		Service:      "iam-service",
		Action:       "user.created",
		ResourceType: "user",
		OldValue:     nil, // nil old_value
		NewValue:     nil, // nil new_value
		CreatedAt:    time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC),
	}

	hash := ComputeEntryHash(entry, GenesisHash)
	if hash == "" {
		t.Error("expected non-empty hash for entry with nil fields")
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex SHA-256, got length %d", len(hash))
	}

	// Verify deterministic with nil fields
	hash2 := ComputeEntryHash(entry, GenesisHash)
	if hash != hash2 {
		t.Error("expected deterministic hash with nil fields")
	}
}

func TestComputeEntryHash_Genesis(t *testing.T) {
	entry := makeEntry("id-1", "tenant-1", "iam-service", "user.created")

	hashGenesis := ComputeEntryHash(entry, GenesisHash)
	hashEmpty := ComputeEntryHash(entry, "")

	if hashGenesis == hashEmpty {
		t.Error("GENESIS and empty string should produce different hashes")
	}
}

func TestComputeEntryHash_JSONCompaction(t *testing.T) {
	entry1 := makeEntry("id-1", "tenant-1", "iam-service", "user.created")
	entry1.OldValue = json.RawMessage(`{"name": "old",  "age":  30}`)

	entry2 := makeEntry("id-1", "tenant-1", "iam-service", "user.created")
	entry2.OldValue = json.RawMessage(`{"name":"old","age":30}`)

	hash1 := ComputeEntryHash(entry1, GenesisHash)
	hash2 := ComputeEntryHash(entry2, GenesisHash)

	if hash1 != hash2 {
		t.Error("whitespace in JSON should not affect hash after compaction")
	}
}

func TestComputeEntryHash_ChainIntegrity(t *testing.T) {
	// Simulate a chain of 3 entries
	entry1 := makeEntry("id-1", "tenant-1", "iam-service", "user.created")
	entry1.CreatedAt = time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	hash1 := ComputeEntryHash(entry1, GenesisHash)

	entry2 := makeEntry("id-2", "tenant-1", "iam-service", "user.updated")
	entry2.CreatedAt = time.Date(2026, 3, 6, 12, 1, 0, 0, time.UTC)
	hash2 := ComputeEntryHash(entry2, hash1)

	entry3 := makeEntry("id-3", "tenant-1", "iam-service", "user.deleted")
	entry3.CreatedAt = time.Date(2026, 3, 6, 12, 2, 0, 0, time.UTC)
	hash3 := ComputeEntryHash(entry3, hash2)

	// All hashes should be unique
	if hash1 == hash2 || hash2 == hash3 || hash1 == hash3 {
		t.Error("all entries in chain should have unique hashes")
	}

	// Verify chain is reproducible
	h1 := ComputeEntryHash(entry1, GenesisHash)
	h2 := ComputeEntryHash(entry2, h1)
	h3 := ComputeEntryHash(entry3, h2)

	if h1 != hash1 || h2 != hash2 || h3 != hash3 {
		t.Error("hash chain should be reproducible")
	}
}
