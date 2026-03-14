package collector

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDeduplicator_UniquePermissions(t *testing.T) {
	dedup := NewDeduplicator()

	assetA := uuid.New()
	assetB := uuid.New()

	perms := []RawPermission{
		{IdentityType: "user", IdentityID: "alice", DataAssetID: assetA, PermissionType: "read"},
		{IdentityType: "user", IdentityID: "bob", DataAssetID: assetA, PermissionType: "write"},
		{IdentityType: "user", IdentityID: "alice", DataAssetID: assetB, PermissionType: "read"},
		{IdentityType: "service_account", IdentityID: "svc-1", DataAssetID: assetA, PermissionType: "read"},
	}

	result := dedup.Deduplicate(perms)
	if len(result) != 4 {
		t.Fatalf("expected 4 unique permissions, got %d", len(result))
	}

	// Verify each original permission is present.
	keys := make(map[string]bool)
	for _, p := range result {
		keys[p.IdentityType+"|"+p.IdentityID+"|"+p.DataAssetID.String()+"|"+p.PermissionType] = true
	}
	for _, p := range perms {
		k := p.IdentityType + "|" + p.IdentityID + "|" + p.DataAssetID.String() + "|" + p.PermissionType
		if !keys[k] {
			t.Errorf("expected permission with key %q to be present", k)
		}
	}
}

func TestDeduplicator_DuplicatesShortestPath(t *testing.T) {
	dedup := NewDeduplicator()

	assetID := uuid.New()
	now := time.Now()

	perms := []RawPermission{
		{
			IdentityType:   "user",
			IdentityID:     "alice",
			DataAssetID:    assetID,
			PermissionType: "read",
			PermissionPath: []string{"group-A", "role-B", "policy-C"},
			GrantedAt:      &now,
		},
		{
			IdentityType:   "user",
			IdentityID:     "alice",
			DataAssetID:    assetID,
			PermissionType: "read",
			PermissionPath: []string{"direct-grant"},
			GrantedAt:      &now,
		},
	}

	result := dedup.Deduplicate(perms)
	if len(result) != 1 {
		t.Fatalf("expected 1 deduplicated result, got %d", len(result))
	}

	// The shorter path (length 1) should win over the longer path (length 3).
	if len(result[0].PermissionPath) != 1 {
		t.Errorf("expected shortest path (len=1), got path of length %d", len(result[0].PermissionPath))
	}
	if result[0].PermissionPath[0] != "direct-grant" {
		t.Errorf("expected path element 'direct-grant', got %q", result[0].PermissionPath[0])
	}
}

func TestDeduplicator_DuplicatesSameLength(t *testing.T) {
	dedup := NewDeduplicator()

	assetID := uuid.New()
	earlier := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	perms := []RawPermission{
		{
			IdentityType:   "user",
			IdentityID:     "bob",
			DataAssetID:    assetID,
			PermissionType: "write",
			PermissionPath: []string{"role-A"},
			GrantedAt:      &earlier,
		},
		{
			IdentityType:   "user",
			IdentityID:     "bob",
			DataAssetID:    assetID,
			PermissionType: "write",
			PermissionPath: []string{"role-B"},
			GrantedAt:      &later,
		},
	}

	result := dedup.Deduplicate(perms)
	if len(result) != 1 {
		t.Fatalf("expected 1 deduplicated result, got %d", len(result))
	}

	// Equal path length: most recently granted should win.
	if result[0].PermissionPath[0] != "role-B" {
		t.Errorf("expected more recently granted permission (role-B), got %q", result[0].PermissionPath[0])
	}
	if !result[0].GrantedAt.Equal(later) {
		t.Errorf("expected granted_at=%v, got %v", later, *result[0].GrantedAt)
	}
}

func TestDeduplicator_EmptyInput(t *testing.T) {
	dedup := NewDeduplicator()

	result := dedup.Deduplicate([]RawPermission{})
	if len(result) != 0 {
		t.Fatalf("expected 0 results for empty input, got %d", len(result))
	}

	result = dedup.Deduplicate(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 results for nil input, got %d", len(result))
	}
}

func TestDeduplicator_SingleItem(t *testing.T) {
	dedup := NewDeduplicator()

	assetID := uuid.New()
	perms := []RawPermission{
		{
			IdentityType:   "user",
			IdentityID:     "alice",
			DataAssetID:    assetID,
			PermissionType: "read",
			PermissionPath: []string{"direct"},
		},
	}

	result := dedup.Deduplicate(perms)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].IdentityID != "alice" {
		t.Errorf("expected IdentityID 'alice', got %q", result[0].IdentityID)
	}
	if result[0].PermissionType != "read" {
		t.Errorf("expected PermissionType 'read', got %q", result[0].PermissionType)
	}
}

func TestDeduplicator_CandidateWithGrantedAtBeatsNilGrantedAt(t *testing.T) {
	dedup := NewDeduplicator()

	assetID := uuid.New()
	now := time.Now()

	perms := []RawPermission{
		{
			IdentityType:   "user",
			IdentityID:     "charlie",
			DataAssetID:    assetID,
			PermissionType: "read",
			PermissionPath: []string{"path-a"},
			GrantedAt:      nil,
		},
		{
			IdentityType:   "user",
			IdentityID:     "charlie",
			DataAssetID:    assetID,
			PermissionType: "read",
			PermissionPath: []string{"path-b"},
			GrantedAt:      &now,
		},
	}

	result := dedup.Deduplicate(perms)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	// Candidate with GrantedAt should replace existing with nil GrantedAt (same path length).
	if result[0].PermissionPath[0] != "path-b" {
		t.Errorf("expected candidate with GrantedAt to win, got path %q", result[0].PermissionPath[0])
	}
}

func TestDeduplicator_ExistingWithGrantedAtBeatsNilCandidate(t *testing.T) {
	dedup := NewDeduplicator()

	assetID := uuid.New()
	now := time.Now()

	perms := []RawPermission{
		{
			IdentityType:   "user",
			IdentityID:     "dave",
			DataAssetID:    assetID,
			PermissionType: "write",
			PermissionPath: []string{"path-a"},
			GrantedAt:      &now,
		},
		{
			IdentityType:   "user",
			IdentityID:     "dave",
			DataAssetID:    assetID,
			PermissionType: "write",
			PermissionPath: []string{"path-b"},
			GrantedAt:      nil,
		},
	}

	result := dedup.Deduplicate(perms)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	// Existing has GrantedAt, candidate does not: existing should remain.
	if result[0].PermissionPath[0] != "path-a" {
		t.Errorf("expected existing with GrantedAt to remain, got path %q", result[0].PermissionPath[0])
	}
}
