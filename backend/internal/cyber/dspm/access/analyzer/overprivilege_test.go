package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

type mockMappingProvider struct {
	mappings []*model.AccessMapping
}

func (m *mockMappingProvider) ListActiveByTenant(_ context.Context, _ uuid.UUID) ([]*model.AccessMapping, error) {
	return m.mappings, nil
}

func newTestAnalyzer(mappings []*model.AccessMapping) *OverprivilegeAnalyzer {
	mock := &mockMappingProvider{mappings: mappings}
	logger := zerolog.Nop()
	return NewOverprivilegeAnalyzer(mock, logger)
}

func TestOverprivilegeAnalyzer_NoMappings(t *testing.T) {
	analyzer := newTestAnalyzer(nil)

	results, err := analyzer.Analyze(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty mappings, got %d", len(results))
	}
}

func TestOverprivilegeAnalyzer_AllUsed(t *testing.T) {
	tenantID := uuid.New()
	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "alice",
			IdentityName:       "Alice",
			DataAssetID:        uuid.New(),
			DataAssetName:      "customers-db",
			DataClassification: "confidential",
			PermissionType:     "read",
			UsageCount90d:      50,
			Status:             "active",
		},
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "bob",
			IdentityName:       "Bob",
			DataAssetID:        uuid.New(),
			DataAssetName:      "logs-bucket",
			DataClassification: "internal",
			PermissionType:     "write",
			UsageCount90d:      10,
			Status:             "active",
		},
	}

	analyzer := newTestAnalyzer(mappings)
	results, err := analyzer.Analyze(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results when all permissions are used, got %d", len(results))
	}
}

func TestOverprivilegeAnalyzer_UnusedPermissions(t *testing.T) {
	tenantID := uuid.New()
	unusedID := uuid.New()
	usedID := uuid.New()

	mappings := []*model.AccessMapping{
		{
			ID:                 usedID,
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "alice",
			IdentityName:       "Alice",
			DataAssetID:        uuid.New(),
			DataAssetName:      "customers-db",
			DataClassification: "confidential",
			PermissionType:     "read",
			UsageCount90d:      50,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-90 * 24 * time.Hour),
		},
		{
			ID:                 unusedID,
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "alice",
			IdentityName:       "Alice",
			DataAssetID:        uuid.New(),
			DataAssetName:      "finance-db",
			DataClassification: "restricted",
			PermissionType:     "write",
			UsageCount90d:      0,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-60 * 24 * time.Hour),
		},
	}

	analyzer := newTestAnalyzer(mappings)
	results, err := analyzer.Analyze(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 overprivileged result, got %d", len(results))
	}

	r := results[0]
	if r.MappingID != unusedID {
		t.Errorf("expected mapping ID %v, got %v", unusedID, r.MappingID)
	}
	if r.IdentityID != "alice" {
		t.Errorf("expected identity ID 'alice', got %q", r.IdentityID)
	}
	if r.PermissionType != "write" {
		t.Errorf("expected permission type 'write', got %q", r.PermissionType)
	}
	// Alice is active (has other used permissions), so confidence should be 0.9.
	if r.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9 (identity is active), got %v", r.Confidence)
	}
	if r.DaysUnused <= 0 {
		t.Errorf("expected positive days unused, got %d", r.DaysUnused)
	}
}

func TestOverprivilegeAnalyzer_InactiveIdentityConfidence(t *testing.T) {
	tenantID := uuid.New()

	// All permissions unused for this identity.
	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "service_account",
			IdentityID:         "svc-old",
			IdentityName:       "Old Service",
			DataAssetID:        uuid.New(),
			DataAssetName:      "data-lake",
			DataClassification: "internal",
			PermissionType:     "read",
			UsageCount90d:      0,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	analyzer := newTestAnalyzer(mappings)
	results, err := analyzer.Analyze(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Identity is completely inactive, so confidence should be 0.5.
	if results[0].Confidence != 0.5 {
		t.Errorf("expected confidence 0.5 (inactive identity), got %v", results[0].Confidence)
	}
}

func TestOverprivilegeAnalyzer_SeverityClassification(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name           string
		permType       string
		classification string
		wantSeverity   string
	}{
		{
			name:           "write on confidential is high",
			permType:       "write",
			classification: "confidential",
			wantSeverity:   "high",
		},
		{
			name:           "admin on restricted is high",
			permType:       "admin",
			classification: "restricted",
			wantSeverity:   "high",
		},
		{
			name:           "delete on confidential is high",
			permType:       "delete",
			classification: "confidential",
			wantSeverity:   "high",
		},
		{
			name:           "full_control on restricted is high",
			permType:       "full_control",
			classification: "restricted",
			wantSeverity:   "high",
		},
		{
			name:           "alter on restricted is high",
			permType:       "alter",
			classification: "restricted",
			wantSeverity:   "high",
		},
		{
			name:           "read on internal is low",
			permType:       "read",
			classification: "internal",
			wantSeverity:   "low",
		},
		{
			name:           "read on public is medium",
			permType:       "read",
			classification: "public",
			wantSeverity:   "medium",
		},
		{
			name:           "write on internal is medium",
			permType:       "write",
			classification: "internal",
			wantSeverity:   "medium",
		},
		{
			name:           "read on confidential is medium",
			permType:       "read",
			classification: "confidential",
			wantSeverity:   "medium",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mappings := []*model.AccessMapping{
				{
					ID:                 uuid.New(),
					TenantID:           tenantID,
					IdentityType:       "user",
					IdentityID:         "test-user",
					IdentityName:       "Test User",
					DataAssetID:        uuid.New(),
					DataAssetName:      "test-asset",
					DataClassification: tc.classification,
					PermissionType:     tc.permType,
					UsageCount90d:      0,
					Status:             "active",
					DiscoveredAt:       time.Now().Add(-30 * 24 * time.Hour),
				},
			}

			analyzer := newTestAnalyzer(mappings)
			results, err := analyzer.Analyze(context.Background(), tenantID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Severity != tc.wantSeverity {
				t.Errorf("expected severity %q for %s on %s, got %q",
					tc.wantSeverity, tc.permType, tc.classification, results[0].Severity)
			}
		})
	}
}

func TestOverprivilegeAnalyzer_DaysUnusedWithLastUsedAt(t *testing.T) {
	tenantID := uuid.New()
	lastUsed := time.Now().Add(-45 * 24 * time.Hour)

	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "carol",
			IdentityName:       "Carol",
			DataAssetID:        uuid.New(),
			DataAssetName:      "reports",
			DataClassification: "internal",
			PermissionType:     "read",
			UsageCount90d:      0,
			LastUsedAt:         &lastUsed,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-90 * 24 * time.Hour),
		},
	}

	analyzer := newTestAnalyzer(mappings)
	results, err := analyzer.Analyze(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// DaysUnused should be based on LastUsedAt (approximately 45 days).
	if results[0].DaysUnused < 44 || results[0].DaysUnused > 46 {
		t.Errorf("expected ~45 days unused (from LastUsedAt), got %d", results[0].DaysUnused)
	}
}

func TestOverprivilegeAnalyzer_DaysUnusedWithoutLastUsedAt(t *testing.T) {
	tenantID := uuid.New()
	discoveredAt := time.Now().Add(-30 * 24 * time.Hour)

	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "dave",
			IdentityName:       "Dave",
			DataAssetID:        uuid.New(),
			DataAssetName:      "archive",
			DataClassification: "public",
			PermissionType:     "read",
			UsageCount90d:      0,
			LastUsedAt:         nil,
			Status:             "active",
			DiscoveredAt:       discoveredAt,
		},
	}

	analyzer := newTestAnalyzer(mappings)
	results, err := analyzer.Analyze(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// DaysUnused should be based on DiscoveredAt (approximately 30 days).
	if results[0].DaysUnused < 29 || results[0].DaysUnused > 31 {
		t.Errorf("expected ~30 days unused (from DiscoveredAt), got %d", results[0].DaysUnused)
	}
}
