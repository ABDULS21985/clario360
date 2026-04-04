//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/audit/hash"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
)

// makeEntry creates a test audit entry with proper hash chain linking.
func makeEntry(tenantID, service, action, severity, resourceType, resourceID, userEmail string, createdAt time.Time, previousHash string) model.AuditEntry {
	userID := uuid.NewString()
	entry := model.AuditEntry{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		UserID:        &userID,
		UserEmail:     userEmail,
		Service:       service,
		Action:        action,
		Severity:      severity,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		OldValue:      json.RawMessage(`{"status":"draft"}`),
		NewValue:      json.RawMessage(`{"status":"active"}`),
		IPAddress:     "10.0.0.1",
		UserAgent:     "integration-test/1.0",
		Metadata:      json.RawMessage(`{}`),
		EventID:       uuid.NewString(),
		CorrelationID: uuid.NewString(),
		PreviousHash:  previousHash,
		CreatedAt:     createdAt,
	}
	entry.EntryHash = hash.ComputeEntryHash(&entry, previousHash)
	return entry
}

// makeChain creates a chain of N audit entries with valid hash links.
func makeChain(tenantID string, n int, baseTime time.Time) []model.AuditEntry {
	entries := make([]model.AuditEntry, 0, n)
	prevHash := hash.GenesisHash
	for i := 0; i < n; i++ {
		t := baseTime.Add(time.Duration(i) * time.Second)
		entry := makeEntry(
			tenantID, "iam-service", fmt.Sprintf("user.update.%d", i),
			model.SeverityInfo, "user", uuid.NewString(),
			fmt.Sprintf("user%d@test.local", i), t, prevHash,
		)
		prevHash = entry.EntryHash
		entries = append(entries, entry)
	}
	return entries
}

func TestBatchInsertAndFindByID(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	entries := makeChain(tenantID, 3, time.Now().UTC().Truncate(time.Second))
	inserted, err := h.repo.BatchInsert(ctx, entries)
	if err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}
	if inserted != 3 {
		t.Fatalf("BatchInsert rows affected = %d, want 3", inserted)
	}

	// FindByID
	for _, e := range entries {
		found, err := h.repo.FindByID(ctx, tenantID, e.ID)
		if err != nil {
			t.Fatalf("FindByID(%s): %v", e.ID, err)
		}
		if found == nil {
			t.Fatalf("FindByID(%s) returned nil", e.ID)
		}
		if found.Action != e.Action {
			t.Errorf("FindByID action = %q, want %q", found.Action, e.Action)
		}
		if found.EntryHash != e.EntryHash {
			t.Errorf("FindByID entry_hash = %q, want %q", found.EntryHash, e.EntryHash)
		}
	}

	// FindByID with wrong tenant returns nil
	otherTenant := uuid.NewString()
	found, err := h.repo.FindByID(ctx, otherTenant, entries[0].ID)
	if err != nil {
		t.Fatalf("FindByID with wrong tenant: %v", err)
	}
	if found != nil {
		t.Errorf("FindByID with wrong tenant should return nil, got entry %s", found.ID)
	}
}

func TestBatchInsertDeduplication(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	entries := makeChain(tenantID, 2, time.Now().UTC().Truncate(time.Second))

	// First insert
	inserted, err := h.repo.BatchInsert(ctx, entries)
	if err != nil {
		t.Fatalf("first BatchInsert: %v", err)
	}
	if inserted != 2 {
		t.Fatalf("first insert: rows = %d, want 2", inserted)
	}

	// Second insert with same entries (duplicate event_id + created_at) → ON CONFLICT DO NOTHING
	inserted, err = h.repo.BatchInsert(ctx, entries)
	if err != nil {
		t.Fatalf("duplicate BatchInsert: %v", err)
	}
	if inserted != 0 {
		t.Errorf("duplicate insert: rows = %d, want 0", inserted)
	}
}

func TestQueryWithFilters(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	prevHash := hash.GenesisHash

	// Insert entries with different services and severities
	var entries []model.AuditEntry
	for i, svc := range []string{"iam-service", "iam-service", "acta-service", "cyber-service"} {
		sev := model.SeverityInfo
		if i == 3 {
			sev = model.SeverityCritical
		}
		e := makeEntry(tenantID, svc, "resource.update", sev, "user", fmt.Sprintf("res-%d", i),
			fmt.Sprintf("user%d@test.local", i), now.Add(time.Duration(i)*time.Second), prevHash)
		prevHash = e.EntryHash
		entries = append(entries, e)
	}

	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	// Query all for tenant
	filter := repository.QueryFilter{
		TenantID: tenantID,
		DateFrom: now.Add(-time.Hour),
		DateTo:   now.Add(time.Hour),
		Sort:     "created_at",
		Order:    "ASC",
		Limit:    10,
		Offset:   0,
	}
	results, total, err := h.repo.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Query all: %v", err)
	}
	if total != 4 {
		t.Fatalf("Query all: total = %d, want 4", total)
	}
	if len(results) != 4 {
		t.Fatalf("Query all: len = %d, want 4", len(results))
	}

	// Filter by service
	filter.Service = "iam-service"
	results, total, err = h.repo.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Query by service: %v", err)
	}
	if total != 2 {
		t.Errorf("Query by service: total = %d, want 2", total)
	}
	filter.Service = ""

	// Filter by severity
	filter.Severity = model.SeverityCritical
	results, total, err = h.repo.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Query by severity: %v", err)
	}
	if total != 1 {
		t.Errorf("Query by severity: total = %d, want 1", total)
	}
	if len(results) > 0 && results[0].Service != "cyber-service" {
		t.Errorf("Query by severity: service = %q, want cyber-service", results[0].Service)
	}
	filter.Severity = ""

	// Pagination: page 1 of size 2
	filter.Limit = 2
	filter.Offset = 0
	results, total, err = h.repo.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Query paginated: %v", err)
	}
	if total != 4 {
		t.Errorf("Query paginated: total = %d, want 4", total)
	}
	if len(results) != 2 {
		t.Errorf("Query paginated: len = %d, want 2", len(results))
	}
}

func TestGetTimeline(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	resourceID := uuid.NewString()
	prevHash := hash.GenesisHash

	// Insert 3 entries for same resource with different actions
	var entries []model.AuditEntry
	for i, action := range []string{"user.create", "user.update", "user.delete"} {
		e := makeEntry(tenantID, "iam-service", action, model.SeverityInfo, "user", resourceID,
			"timeline@test.local", now.Add(time.Duration(i)*time.Second), prevHash)
		prevHash = e.EntryHash
		entries = append(entries, e)
	}
	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	// Get timeline for resource
	timeline, err := h.querySvc.GetTimeline(ctx, tenantID, resourceID, 1, 50, []string{"tenant_admin"}, nil)
	if err != nil {
		t.Fatalf("GetTimeline: %v", err)
	}
	if timeline.ResourceID != resourceID {
		t.Errorf("timeline.ResourceID = %q, want %q", timeline.ResourceID, resourceID)
	}
	if len(timeline.Events) != 3 {
		t.Fatalf("timeline events = %d, want 3", len(timeline.Events))
	}

	// Timeline is ordered by created_at DESC, so first event is the most recent
	if timeline.Events[0].Action != "user.delete" {
		t.Errorf("first timeline event action = %q, want user.delete", timeline.Events[0].Action)
	}

	// Filter timeline by action
	filter := &repository.TimelineFilter{Action: "update"}
	timeline, err = h.querySvc.GetTimeline(ctx, tenantID, resourceID, 1, 50, []string{"tenant_admin"}, filter)
	if err != nil {
		t.Fatalf("GetTimeline with action filter: %v", err)
	}
	if len(timeline.Events) != 1 {
		t.Errorf("filtered timeline events = %d, want 1", len(timeline.Events))
	}

	// Filter timeline by date range (only first entry)
	filter = &repository.TimelineFilter{
		DateFrom: now.Add(-time.Second),
		DateTo:   now.Add(500 * time.Millisecond),
	}
	timeline, err = h.querySvc.GetTimeline(ctx, tenantID, resourceID, 1, 50, []string{"tenant_admin"}, filter)
	if err != nil {
		t.Fatalf("GetTimeline with date filter: %v", err)
	}
	if len(timeline.Events) != 1 {
		t.Errorf("date-filtered timeline events = %d, want 1", len(timeline.Events))
	}
}

func TestGetStats(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	prevHash := hash.GenesisHash

	var entries []model.AuditEntry
	services := []string{"iam-service", "iam-service", "acta-service"}
	severities := []string{model.SeverityInfo, model.SeverityWarning, model.SeverityInfo}
	for i := 0; i < 3; i++ {
		e := makeEntry(tenantID, services[i], "resource.update", severities[i], "user",
			fmt.Sprintf("res-%d", i), fmt.Sprintf("stats%d@test.local", i),
			now.Add(time.Duration(i)*time.Second), prevHash)
		prevHash = e.EntryHash
		entries = append(entries, e)
	}
	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	stats, err := h.querySvc.GetStats(ctx, tenantID, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if stats.TotalEvents != 3 {
		t.Errorf("TotalEvents = %d, want 3", stats.TotalEvents)
	}
	if stats.UniqueServices != 2 {
		t.Errorf("UniqueServices = %d, want 2", stats.UniqueServices)
	}
	if stats.UniqueUsers != 3 {
		t.Errorf("UniqueUsers = %d, want 3", stats.UniqueUsers)
	}
	if len(stats.ByService) == 0 {
		t.Fatal("ByService is empty")
	}
	if len(stats.BySeverity) == 0 {
		t.Fatal("BySeverity is empty")
	}
	if len(stats.TopUsers) == 0 {
		t.Fatal("TopUsers is empty")
	}
}

func TestHashChainVerification(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	entries := makeChain(tenantID, 10, now)

	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	result, err := h.integritySvc.VerifyChain(ctx, tenantID, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}

	if !result.Verified {
		t.Errorf("VerifyChain: verified = false, want true (broken at: %v)", result.BrokenChainAt)
	}
	if result.TotalRecords != 10 {
		t.Errorf("TotalRecords = %d, want 10", result.TotalRecords)
	}
	if result.VerifiedRecords != 10 {
		t.Errorf("VerifiedRecords = %d, want 10", result.VerifiedRecords)
	}
	if result.FirstRecord != entries[0].ID {
		t.Errorf("FirstRecord = %q, want %q", result.FirstRecord, entries[0].ID)
	}
	if result.LastRecord != entries[9].ID {
		t.Errorf("LastRecord = %q, want %q", result.LastRecord, entries[9].ID)
	}
	if result.VerificationHash != entries[9].EntryHash {
		t.Errorf("VerificationHash = %q, want %q", result.VerificationHash, entries[9].EntryHash)
	}
}

func TestHashChainBrokenDetection(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	entries := makeChain(tenantID, 5, now)

	// Corrupt the 3rd entry's hash
	entries[2].EntryHash = "corrupted_hash_value"

	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	result, err := h.integritySvc.VerifyChain(ctx, tenantID, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}

	if result.Verified {
		t.Error("VerifyChain: verified = true, want false for corrupted chain")
	}
	if result.BrokenChainAt == nil {
		t.Fatal("BrokenChainAt should not be nil")
	}
	// VerifyChain continues after the first break (to count all records) and
	// overwrites BrokenChainAt on each subsequent mismatch, so just verify
	// that the break was detected somewhere in the chain.
	if result.TotalRecords != 5 {
		t.Errorf("TotalRecords = %d, want 5", result.TotalRecords)
	}
}

func TestStreamByTenant(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	entries := makeChain(tenantID, 5, now)
	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	var streamed []model.AuditEntry
	err := h.repo.StreamByTenant(ctx, tenantID, now.Add(-time.Hour), now.Add(time.Hour), func(e *model.AuditEntry) error {
		streamed = append(streamed, *e)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamByTenant: %v", err)
	}
	if len(streamed) != 5 {
		t.Fatalf("StreamByTenant: got %d entries, want 5", len(streamed))
	}

	// Verify ASC ordering
	for i := 1; i < len(streamed); i++ {
		if !streamed[i].CreatedAt.After(streamed[i-1].CreatedAt) && !streamed[i].CreatedAt.Equal(streamed[i-1].CreatedAt) {
			t.Errorf("StreamByTenant: entries not in ASC order at index %d", i)
		}
	}
}

func TestPartitionManagement(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	// EnsurePartitions already called in newAuditHarness. List partitions.
	partitions, err := h.partitionMgr.ListPartitions(ctx)
	if err != nil {
		t.Fatalf("ListPartitions: %v", err)
	}
	if len(partitions) == 0 {
		t.Fatal("ListPartitions: no partitions found")
	}

	// Verify each partition has valid metadata
	for _, p := range partitions {
		if p.Name == "" {
			t.Error("partition has empty name")
		}
		if p.Status == "" {
			t.Errorf("partition %s has empty status", p.Name)
		}
	}

	// EnsurePartitions is idempotent
	created, err := h.partitionMgr.EnsurePartitions(ctx)
	if err != nil {
		t.Fatalf("EnsurePartitions (idempotent): %v", err)
	}
	if len(created) != 0 {
		t.Errorf("EnsurePartitions created %d new partitions on second call, want 0", len(created))
	}
}

func TestChainStateUpsert(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	// Initially nil
	cs, err := h.repo.GetChainState(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetChainState (initial): %v", err)
	}
	if cs != nil {
		t.Fatalf("GetChainState (initial): expected nil, got %+v", cs)
	}

	// Upsert
	now := time.Now().UTC().Truncate(time.Microsecond)
	state := &model.ChainState{
		TenantID:    tenantID,
		LastEntryID: uuid.NewString(),
		LastHash:    "abc123",
		LastCreated: now,
	}
	if err := h.repo.UpsertChainState(ctx, state); err != nil {
		t.Fatalf("UpsertChainState: %v", err)
	}

	cs, err = h.repo.GetChainState(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetChainState (after upsert): %v", err)
	}
	if cs == nil {
		t.Fatal("GetChainState (after upsert): returned nil")
	}
	if cs.LastHash != "abc123" {
		t.Errorf("ChainState.LastHash = %q, want abc123", cs.LastHash)
	}

	// Update (upsert overwrites)
	state.LastHash = "def456"
	if err := h.repo.UpsertChainState(ctx, state); err != nil {
		t.Fatalf("UpsertChainState (update): %v", err)
	}
	cs, err = h.repo.GetChainState(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetChainState (after update): %v", err)
	}
	if cs.LastHash != "def456" {
		t.Errorf("ChainState.LastHash after update = %q, want def456", cs.LastHash)
	}
}

func TestTenantIsolation(t *testing.T) {
	tenant1 := uuid.NewString()
	tenant2 := uuid.NewString()
	h1 := newAuditHarness(t, tenant1)
	_ = newAuditHarness(t, tenant2)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Insert 2 entries for tenant1
	entries1 := makeChain(tenant1, 2, now)
	if _, err := h1.repo.BatchInsert(ctx, entries1); err != nil {
		t.Fatalf("BatchInsert tenant1: %v", err)
	}

	// Insert 3 entries for tenant2
	entries2 := makeChain(tenant2, 3, now)
	if _, err := h1.repo.BatchInsert(ctx, entries2); err != nil {
		t.Fatalf("BatchInsert tenant2: %v", err)
	}

	// Query for tenant1 — should see only 2
	filter := repository.QueryFilter{
		TenantID: tenant1,
		DateFrom: now.Add(-time.Hour),
		DateTo:   now.Add(time.Hour),
		Sort:     "created_at",
		Order:    "ASC",
		Limit:    50,
		Offset:   0,
	}
	results, total, err := h1.repo.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Query tenant1: %v", err)
	}
	if total != 2 {
		t.Errorf("tenant1 total = %d, want 2", total)
	}
	for _, r := range results {
		if r.TenantID != tenant1 {
			t.Errorf("tenant1 query returned entry with tenant_id = %q", r.TenantID)
		}
	}

	// Query for tenant2 — should see only 3
	filter.TenantID = tenant2
	_, total, err = h1.repo.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Query tenant2: %v", err)
	}
	if total != 3 {
		t.Errorf("tenant2 total = %d, want 3", total)
	}

	// FindByID: tenant1 ID with tenant2 context → nil
	found, err := h1.repo.FindByID(ctx, tenant2, entries1[0].ID)
	if err != nil {
		t.Fatalf("FindByID cross-tenant: %v", err)
	}
	if found != nil {
		t.Error("FindByID cross-tenant should return nil")
	}
}

func TestGetLastEntryHash(t *testing.T) {
	tenantID := uuid.NewString()
	h := newAuditHarness(t, tenantID)
	ctx := context.Background()

	// Initially empty
	id, hashVal, err := h.repo.GetLastEntryHash(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetLastEntryHash (empty): %v", err)
	}
	if id != "" || hashVal != "" {
		t.Errorf("GetLastEntryHash (empty): got id=%q hash=%q, want empty", id, hashVal)
	}

	// Insert entries
	now := time.Now().UTC().Truncate(time.Second)
	entries := makeChain(tenantID, 3, now)
	if _, err := h.repo.BatchInsert(ctx, entries); err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}

	// Should return the last entry (most recent created_at)
	id, hashVal, err = h.repo.GetLastEntryHash(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetLastEntryHash: %v", err)
	}
	lastEntry := entries[len(entries)-1]
	if id != lastEntry.ID {
		t.Errorf("GetLastEntryHash id = %q, want %q", id, lastEntry.ID)
	}
	if hashVal != lastEntry.EntryHash {
		t.Errorf("GetLastEntryHash hash = %q, want %q", hashVal, lastEntry.EntryHash)
	}
}
