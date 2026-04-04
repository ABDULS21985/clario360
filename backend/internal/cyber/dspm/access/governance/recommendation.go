package governance

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// RecommendationEngine generates actionable least-privilege recommendations for an identity.
type RecommendationEngine struct {
	repo   MappingProvider
	logger zerolog.Logger
}

// NewRecommendationEngine creates a recommendation generator.
func NewRecommendationEngine(repo MappingProvider, logger zerolog.Logger) *RecommendationEngine {
	return &RecommendationEngine{
		repo:   repo,
		logger: logger.With().Str("component", "recommendation_engine").Logger(),
	}
}

// GenerateForIdentity analyzes an identity's permission profile and generates recommendations.
// Types: "revoke" (unused), "downgrade" (excessive level), "time_bound" (indefinite on sensitive), "review" (ambiguous).
func (r *RecommendationEngine) GenerateForIdentity(ctx context.Context, tenantID uuid.UUID, identityID string) ([]model.Recommendation, error) {
	allMappings, err := r.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Filter to target identity.
	var identityMappings []*model.AccessMapping
	for _, m := range allMappings {
		if m.IdentityID == identityID {
			identityMappings = append(identityMappings, m)
		}
	}

	var recommendations []model.Recommendation

	for _, m := range identityMappings {
		// 1. Revoke: unused permissions on sensitive data.
		if m.UsageCount90d == 0 && (m.DataClassification == "confidential" || m.DataClassification == "restricted") {
			riskReduction := m.SensitivityWeight * model.PermissionBreadth(m.PermissionType)
			recommendations = append(recommendations, model.Recommendation{
				Type:           "revoke",
				MappingID:      m.ID,
				IdentityID:     m.IdentityID,
				IdentityName:   m.IdentityName,
				DataAssetName:  m.DataAssetName,
				PermissionType: m.PermissionType,
				Reason:         fmt.Sprintf("Permission %s on %s (%s) has never been used in 90 days", m.PermissionType, m.DataAssetName, m.DataClassification),
				Impact:         fmt.Sprintf("Removes %s access to %s", m.PermissionType, m.DataAssetName),
				RiskReduction:  math.Round(riskReduction*100) / 100,
			})
			continue
		}

		// 2. Downgrade: admin/write used only for reads.
		if isHighPermission(m.PermissionType) && m.UsageCount90d > 0 {
			// If the identity has used this permission but only for read operations,
			// the permission level is excessive. Since we can't inspect individual
			// query types from the mapping alone, we flag admin/full_control for
			// downgrade consideration when usage count is low relative to read count.
			if m.PermissionType == "admin" || m.PermissionType == "full_control" {
				riskReduction := m.SensitivityWeight * (model.PermissionBreadth(m.PermissionType) - model.PermissionBreadth("read"))
				recommendations = append(recommendations, model.Recommendation{
					Type:           "downgrade",
					MappingID:      m.ID,
					IdentityID:     m.IdentityID,
					IdentityName:   m.IdentityName,
					DataAssetName:  m.DataAssetName,
					PermissionType: m.PermissionType,
					Reason:         fmt.Sprintf("Current %s permission may be excessive — consider downgrading to read", m.PermissionType),
					Impact:         fmt.Sprintf("Reduces access level from %s to read on %s", m.PermissionType, m.DataAssetName),
					RiskReduction:  math.Round(riskReduction*100) / 100,
				})
			}
		}

		// 3. Time-bound: indefinite grants on restricted data.
		if m.ExpiresAt == nil && m.DataClassification == "restricted" {
			recommendations = append(recommendations, model.Recommendation{
				Type:           "time_bound",
				MappingID:      m.ID,
				IdentityID:     m.IdentityID,
				IdentityName:   m.IdentityName,
				DataAssetName:  m.DataAssetName,
				PermissionType: m.PermissionType,
				Reason:         fmt.Sprintf("Indefinite %s access to restricted asset %s — should have an expiry", m.PermissionType, m.DataAssetName),
				Impact:         "Adds 30-day expiry to reduce standing access risk",
				RiskReduction:  m.SensitivityWeight * 0.3,
			})
		}

		// 4. Review: ambiguous usage patterns (low usage on sensitive data).
		if m.UsageCount90d > 0 && m.UsageCount90d <= 3 && (m.DataClassification == "confidential" || m.DataClassification == "restricted") {
			recommendations = append(recommendations, model.Recommendation{
				Type:           "review",
				MappingID:      m.ID,
				IdentityID:     m.IdentityID,
				IdentityName:   m.IdentityName,
				DataAssetName:  m.DataAssetName,
				PermissionType: m.PermissionType,
				Reason:         fmt.Sprintf("Low usage (%d accesses in 90 days) on %s %s — verify if access is still needed", m.UsageCount90d, m.DataClassification, m.DataAssetName),
				Impact:         "Human review to determine if access is appropriate",
				RiskReduction:  m.SensitivityWeight * 0.1,
			})
		}
	}

	return recommendations, nil
}

func isHighPermission(permType string) bool {
	return permType == "admin" || permType == "full_control" || permType == "write" || permType == "delete" || permType == "alter"
}
