package mapper

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// EffectiveAccessResolver computes the union of all permissions for an identity
// across all permission paths (direct grants + role inheritance + group membership
// + wildcard expansion).
type EffectiveAccessResolver struct {
	repo   MappingLister
	logger zerolog.Logger
}

// NewEffectiveAccessResolver creates a new resolver.
func NewEffectiveAccessResolver(repo MappingLister, logger zerolog.Logger) *EffectiveAccessResolver {
	return &EffectiveAccessResolver{
		repo:   repo,
		logger: logger.With().Str("component", "effective_access").Logger(),
	}
}

// Resolve computes effective access for a single identity. For each data asset
// the identity can access, it returns the highest permission level across all paths.
func (e *EffectiveAccessResolver) Resolve(ctx context.Context, tenantID uuid.UUID, identityID string) (*model.EffectiveAccess, error) {
	allMappings, err := e.repo.ListActiveByTenant(ctx, tenantID)
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

	if len(identityMappings) == 0 {
		return &model.EffectiveAccess{
			IdentityID: identityID,
		}, nil
	}

	first := identityMappings[0]
	result := &model.EffectiveAccess{
		IdentityType: first.IdentityType,
		IdentityID:   identityID,
		IdentityName: first.IdentityName,
	}

	// Group by data asset, keeping the highest permission level.
	type assetInfo struct {
		maxLevel int
		mapping  *model.AccessMapping
		count    int
		isStale  bool
	}
	assetMap := make(map[uuid.UUID]*assetInfo)

	for _, m := range identityMappings {
		level := model.PermissionLevel(m.PermissionType)
		if existing, ok := assetMap[m.DataAssetID]; ok {
			existing.count++
			if level > existing.maxLevel {
				existing.maxLevel = level
				existing.mapping = m
			}
			// Asset is stale only if ALL permissions to it are stale.
			if !m.IsStale {
				existing.isStale = false
			}
		} else {
			assetMap[m.DataAssetID] = &assetInfo{
				maxLevel: level,
				mapping:  m,
				count:    1,
				isStale:  m.IsStale,
			}
		}
	}

	maxOverallLevel := 0
	for _, info := range assetMap {
		m := info.mapping
		access := model.AssetAccess{
			DataAssetID:        m.DataAssetID,
			DataAssetName:      m.DataAssetName,
			DataClassification: m.DataClassification,
			MaxPermissionLevel: m.PermissionType,
			PermissionCount:    info.count,
			SensitivityWeight:  m.SensitivityWeight,
			IsStale:            info.isStale,
		}
		result.Assets = append(result.Assets, access)
		if info.maxLevel > maxOverallLevel {
			maxOverallLevel = info.maxLevel
			result.MaxLevel = m.PermissionType
		}
	}

	result.TotalAssets = len(assetMap)
	return result, nil
}
