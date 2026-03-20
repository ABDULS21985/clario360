package collector

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

// PermissionSource collects raw permissions from a single source (database, cloud, application).
type PermissionSource interface {
	Name() string
	CollectPermissions(ctx context.Context, tenantID uuid.UUID, assets []*cybermodel.DSPMDataAsset) ([]RawPermission, error)
}

// RawPermission is an unprocessed permission record collected from a source.
type RawPermission struct {
	IdentityType     string
	IdentityID       string
	IdentityName     string
	IdentitySource   string
	DataAssetID      uuid.UUID
	DataAssetName    string
	DataClassification string
	PermissionType   string
	PermissionSource string
	PermissionPath   []string
	IsWildcard       bool
	GrantedAt        *time.Time
	ExpiresAt        *time.Time
}

// AssetLister loads active DSPM data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// MappingUpserter persists access mappings.
type MappingUpserter interface {
	UpsertMapping(ctx context.Context, mapping *model.AccessMapping) error
	MarkUnseen(ctx context.Context, tenantID uuid.UUID, verifiedBefore time.Time) (int, error)
}

// PermissionCollector orchestrates collection from all registered sources.
type PermissionCollector struct {
	sources      []PermissionSource
	assetLister  AssetLister
	deduplicator *Deduplicator
	logger       zerolog.Logger
}

// NewPermissionCollector creates a new collector with the given sources.
func NewPermissionCollector(
	assetLister AssetLister,
	logger zerolog.Logger,
	sources ...PermissionSource,
) *PermissionCollector {
	return &PermissionCollector{
		sources:      sources,
		assetLister:  assetLister,
		deduplicator: NewDeduplicator(),
		logger:       logger.With().Str("component", "permission_collector").Logger(),
	}
}

// CollectAll loads all data assets for the tenant and collects permissions from every
// registered source. Each source is given a 30-second timeout. Duplicate permissions
// (same identity + asset + permission type) are deduplicated with shortest-path wins.
func (c *PermissionCollector) CollectAll(ctx context.Context, tenantID uuid.UUID) ([]RawPermission, error) {
	assets, err := c.assetLister.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		return nil, nil
	}

	var all []RawPermission
	for _, src := range c.sources {
		srcCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		perms, srcErr := src.CollectPermissions(srcCtx, tenantID, assets)
		cancel()
		if srcErr != nil {
			c.logger.Warn().Err(srcErr).Str("source", src.Name()).Msg("permission collection failed for source")
			continue
		}
		c.logger.Info().Str("source", src.Name()).Int("count", len(perms)).Msg("collected permissions")
		all = append(all, perms...)
	}

	deduped := c.deduplicator.Deduplicate(all)
	c.logger.Info().
		Int("raw_count", len(all)).
		Int("deduped_count", len(deduped)).
		Msg("permission collection complete")
	return deduped, nil
}
