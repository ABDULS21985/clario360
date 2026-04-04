package governance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

type mockRecommendationMappingProvider struct {
	mappings []*model.AccessMapping
}

func (m *mockRecommendationMappingProvider) ListActiveByTenant(_ context.Context, _ uuid.UUID) ([]*model.AccessMapping, error) {
	return m.mappings, nil
}

func newTestRecommendationEngine(mappings []*model.AccessMapping) *RecommendationEngine {
	mock := &mockRecommendationMappingProvider{mappings: mappings}
	logger := zerolog.Nop()
	return NewRecommendationEngine(mock, logger)
}

func TestRecommendationEngine_RevokeUnused(t *testing.T) {
	tenantID := uuid.New()
	identityID := "alice"
	mappingID := uuid.New()

	mappings := []*model.AccessMapping{
		{
			ID:                 mappingID,
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         identityID,
			IdentityName:       "Alice",
			DataAssetID:        uuid.New(),
			DataAssetName:      "hr-database",
			DataClassification: "restricted",
			PermissionType:     "write",
			UsageCount90d:      0,
			SensitivityWeight:  10.0,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-60 * 24 * time.Hour),
		},
	}

	engine := newTestRecommendationEngine(mappings)
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, identityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce a "revoke" recommendation for unused permission on restricted data.
	found := false
	for _, r := range recs {
		if r.Type == "revoke" {
			found = true
			if r.MappingID != mappingID {
				t.Errorf("expected mapping ID %v, got %v", mappingID, r.MappingID)
			}
			if r.PermissionType != "write" {
				t.Errorf("expected permission type 'write', got %q", r.PermissionType)
			}
			// risk_reduction = sensitivity_weight * permission_breadth = 10 * 2 = 20
			if r.RiskReduction != 20.0 {
				t.Errorf("expected risk reduction 20.0, got %v", r.RiskReduction)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected a 'revoke' recommendation, got types: %v", recTypes(recs))
	}
}

func TestRecommendationEngine_DowngradeAdmin(t *testing.T) {
	tenantID := uuid.New()
	identityID := "bob"
	mappingID := uuid.New()

	mappings := []*model.AccessMapping{
		{
			ID:                 mappingID,
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         identityID,
			IdentityName:       "Bob",
			DataAssetID:        uuid.New(),
			DataAssetName:      "analytics-db",
			DataClassification: "internal",
			PermissionType:     "admin",
			UsageCount90d:      15,
			SensitivityWeight:  2.0,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-90 * 24 * time.Hour),
		},
	}

	engine := newTestRecommendationEngine(mappings)
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, identityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range recs {
		if r.Type == "downgrade" {
			found = true
			if r.MappingID != mappingID {
				t.Errorf("expected mapping ID %v, got %v", mappingID, r.MappingID)
			}
			// risk_reduction = sensitivity_weight * (admin_breadth - read_breadth) = 2 * (4 - 1) = 6
			if r.RiskReduction != 6.0 {
				t.Errorf("expected risk reduction 6.0, got %v", r.RiskReduction)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected a 'downgrade' recommendation, got types: %v", recTypes(recs))
	}
}

func TestRecommendationEngine_TimeBound(t *testing.T) {
	tenantID := uuid.New()
	identityID := "carol"
	mappingID := uuid.New()

	mappings := []*model.AccessMapping{
		{
			ID:                 mappingID,
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         identityID,
			IdentityName:       "Carol",
			DataAssetID:        uuid.New(),
			DataAssetName:      "pii-store",
			DataClassification: "restricted",
			PermissionType:     "read",
			UsageCount90d:      30,
			SensitivityWeight:  10.0,
			ExpiresAt:          nil, // Indefinite access.
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-90 * 24 * time.Hour),
		},
	}

	engine := newTestRecommendationEngine(mappings)
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, identityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range recs {
		if r.Type == "time_bound" {
			found = true
			if r.MappingID != mappingID {
				t.Errorf("expected mapping ID %v, got %v", mappingID, r.MappingID)
			}
			// risk_reduction = sensitivity_weight * 0.3 = 10 * 0.3 = 3.0
			if r.RiskReduction != 3.0 {
				t.Errorf("expected risk reduction 3.0, got %v", r.RiskReduction)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected a 'time_bound' recommendation, got types: %v", recTypes(recs))
	}
}

func TestRecommendationEngine_ReviewLowUsage(t *testing.T) {
	tenantID := uuid.New()
	identityID := "dave"
	mappingID := uuid.New()
	expires := time.Now().Add(30 * 24 * time.Hour)

	mappings := []*model.AccessMapping{
		{
			ID:                 mappingID,
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         identityID,
			IdentityName:       "Dave",
			DataAssetID:        uuid.New(),
			DataAssetName:      "finance-reports",
			DataClassification: "confidential",
			PermissionType:     "read",
			UsageCount90d:      2,
			SensitivityWeight:  5.0,
			ExpiresAt:          &expires, // Has expiry, so no time_bound recommendation.
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-90 * 24 * time.Hour),
		},
	}

	engine := newTestRecommendationEngine(mappings)
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, identityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range recs {
		if r.Type == "review" {
			found = true
			if r.MappingID != mappingID {
				t.Errorf("expected mapping ID %v, got %v", mappingID, r.MappingID)
			}
			// risk_reduction = sensitivity_weight * 0.1 = 5 * 0.1 = 0.5
			if r.RiskReduction != 0.5 {
				t.Errorf("expected risk reduction 0.5, got %v", r.RiskReduction)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected a 'review' recommendation, got types: %v", recTypes(recs))
	}
}

func TestRecommendationEngine_NoRecommendations(t *testing.T) {
	tenantID := uuid.New()
	identityID := "eve"
	expires := time.Now().Add(30 * 24 * time.Hour)

	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         identityID,
			IdentityName:       "Eve",
			DataAssetID:        uuid.New(),
			DataAssetName:      "public-docs",
			DataClassification: "public",
			PermissionType:     "read",
			UsageCount90d:      100,
			SensitivityWeight:  1.0,
			ExpiresAt:          &expires,
			Status:             "active",
			DiscoveredAt:       time.Now().Add(-90 * 24 * time.Hour),
		},
	}

	engine := newTestRecommendationEngine(mappings)
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, identityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(recs) != 0 {
		t.Errorf("expected no recommendations for well-used read on public data, got %d: %v", len(recs), recTypes(recs))
	}
}

func TestRecommendationEngine_FiltersToTargetIdentity(t *testing.T) {
	tenantID := uuid.New()

	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "alice",
			IdentityName:       "Alice",
			DataAssetID:        uuid.New(),
			DataAssetName:      "secret-db",
			DataClassification: "restricted",
			PermissionType:     "write",
			UsageCount90d:      0,
			SensitivityWeight:  10.0,
			Status:             "active",
		},
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         "bob",
			IdentityName:       "Bob",
			DataAssetID:        uuid.New(),
			DataAssetName:      "another-db",
			DataClassification: "restricted",
			PermissionType:     "admin",
			UsageCount90d:      0,
			SensitivityWeight:  10.0,
			Status:             "active",
		},
	}

	engine := newTestRecommendationEngine(mappings)

	// Only request recommendations for "alice".
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return recommendations for alice, not bob.
	for _, r := range recs {
		if r.IdentityID != "alice" {
			t.Errorf("expected recommendation for alice only, got identity %q", r.IdentityID)
		}
	}
}

func TestRecommendationEngine_RevokeSkipsContinue(t *testing.T) {
	// When a revoke recommendation is generated, the loop continues and skips
	// other recommendation types for that mapping.
	tenantID := uuid.New()
	identityID := "frank"

	mappings := []*model.AccessMapping{
		{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IdentityType:       "user",
			IdentityID:         identityID,
			IdentityName:       "Frank",
			DataAssetID:        uuid.New(),
			DataAssetName:      "secret-data",
			DataClassification: "restricted",
			PermissionType:     "admin",
			UsageCount90d:      0,
			SensitivityWeight:  10.0,
			ExpiresAt:          nil, // Would trigger time_bound too, but revoke continues.
			Status:             "active",
		},
	}

	engine := newTestRecommendationEngine(mappings)
	recs, err := engine.GenerateForIdentity(context.Background(), tenantID, identityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only produce a revoke (not time_bound or downgrade) due to continue.
	types := recTypes(recs)
	if len(recs) != 1 {
		t.Fatalf("expected exactly 1 recommendation, got %d: %v", len(recs), types)
	}
	if recs[0].Type != "revoke" {
		t.Errorf("expected 'revoke' type, got %q", recs[0].Type)
	}
}

// recTypes is a helper that extracts recommendation types for error messages.
func recTypes(recs []model.Recommendation) []string {
	types := make([]string, len(recs))
	for i, r := range recs {
		types[i] = r.Type
	}
	return types
}
