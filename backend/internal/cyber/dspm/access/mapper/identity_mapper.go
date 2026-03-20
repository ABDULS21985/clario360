package mapper

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/collector"
	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// MappingRepository is the persistence interface the mapper needs.
type MappingRepository interface {
	UpsertMapping(ctx context.Context, mapping *model.AccessMapping) error
	MarkUnseen(ctx context.Context, tenantID uuid.UUID, verifiedBefore time.Time) (int, error)
}

// IdentityMapper converts raw permissions into persisted access mappings and
// calculates per-mapping risk scores. It is the core identity-to-data-asset
// bridge, and marks mappings absent from the current collection cycle as revoked.
type IdentityMapper struct {
	repo   MappingRepository
	logger zerolog.Logger
}

// NewIdentityMapper creates a new identity mapper.
func NewIdentityMapper(repo MappingRepository, logger zerolog.Logger) *IdentityMapper {
	return &IdentityMapper{
		repo:   repo,
		logger: logger.With().Str("component", "identity_mapper").Logger(),
	}
}

// BuildMappings persists access mappings from raw permissions and calculates risk
// scores. It upserts each permission as an AccessMapping and marks any mappings
// not seen in this cycle as status='revoked'.
func (m *IdentityMapper) BuildMappings(ctx context.Context, tenantID uuid.UUID, permissions []collector.RawPermission) error {
	cycleStart := time.Now().UTC()

	for _, perm := range permissions {
		weight := model.SensitivityWeight(perm.DataClassification)
		breadth := model.PermissionBreadth(perm.PermissionType)
		staleness := 0.5 // Default for new mappings: recently active.
		riskScore := weight * breadth * staleness

		mapping := &model.AccessMapping{
			TenantID:           tenantID,
			IdentityType:       perm.IdentityType,
			IdentityID:         perm.IdentityID,
			IdentityName:       perm.IdentityName,
			IdentitySource:     perm.IdentitySource,
			DataAssetID:        perm.DataAssetID,
			DataAssetName:      perm.DataAssetName,
			DataClassification: perm.DataClassification,
			PermissionType:     perm.PermissionType,
			PermissionSource:   perm.PermissionSource,
			PermissionPath:     perm.PermissionPath,
			IsWildcard:         perm.IsWildcard,
			SensitivityWeight:  weight,
			AccessRiskScore:    math.Round(riskScore*100) / 100,
			Status:             "active",
			ExpiresAt:          perm.ExpiresAt,
			LastVerifiedAt:     cycleStart,
		}

		if err := m.repo.UpsertMapping(ctx, mapping); err != nil {
			m.logger.Warn().Err(err).
				Str("identity", perm.IdentityID).
				Str("asset", perm.DataAssetName).
				Msg("failed to upsert mapping")
			continue
		}
	}

	// Mark mappings not seen in this collection cycle as revoked.
	revoked, err := m.repo.MarkUnseen(ctx, tenantID, cycleStart)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to mark unseen mappings as revoked")
		return err
	}
	if revoked > 0 {
		m.logger.Info().Int("revoked", revoked).Msg("revoked stale mappings")
	}

	m.logger.Info().Int("count", len(permissions)).Msg("built access mappings")
	return nil
}

// UpdateUsageCounts recalculates usage_count_30d, usage_count_90d, is_stale,
// and access_risk_score for all active mappings in a tenant based on audit data.
func (m *IdentityMapper) UpdateUsageCounts(ctx context.Context, tenantID uuid.UUID, mappings []*model.AccessMapping, auditCounts map[string]usageInfo) error {
	for _, mapping := range mappings {
		key := mappingUsageKey(mapping)
		info := auditCounts[key]

		mapping.UsageCount30d = info.Count30d
		mapping.UsageCount90d = info.Count90d
		if info.LastUsed != nil {
			mapping.LastUsedAt = info.LastUsed
		}

		// Update staleness.
		mapping.IsStale = mapping.UsageCount90d == 0

		// Recalculate access_risk_score with staleness factor.
		staleness := stalenessFactor(mapping.LastUsedAt)
		mapping.AccessRiskScore = math.Round(
			mapping.SensitivityWeight*model.PermissionBreadth(mapping.PermissionType)*staleness*100,
		) / 100

		if err := m.repo.UpsertMapping(ctx, mapping); err != nil {
			m.logger.Warn().Err(err).Str("mapping_id", mapping.ID.String()).Msg("failed to update usage counts")
		}
	}
	return nil
}

type usageInfo struct {
	Count30d int
	Count90d int
	LastUsed *time.Time
}

func mappingUsageKey(m *model.AccessMapping) string {
	return m.IdentityType + "|" + m.IdentityID + "|" + m.DataAssetID.String() + "|" + m.PermissionType
}

// stalenessFactor returns a risk multiplier based on when the permission was last used.
// Higher staleness = higher risk (unused permissions are unnecessary attack surface).
func stalenessFactor(lastUsed *time.Time) float64 {
	if lastUsed == nil {
		return 1.0 // Never used: maximum staleness risk.
	}
	days := time.Since(*lastUsed).Hours() / 24
	switch {
	case days <= 7:
		return 0.5
	case days <= 30:
		return 0.7
	case days <= 90:
		return 0.9
	default:
		return 1.0
	}
}
