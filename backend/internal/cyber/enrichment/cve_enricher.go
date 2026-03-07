package enrichment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// CVEEnricherVulnRepo is the minimal repository interface needed by CVEEnricher.
type CVEEnricherVulnRepo interface {
	FindCVEsForAsset(ctx context.Context, os string) ([]*model.CVERecord, error)
	UpsertFromCVE(ctx context.Context, tenantID, assetID uuid.UUID, cve *model.CVERecord) error
}

// CVEEnricher matches CVEs from the local NVD mirror against an asset's OS/software.
type CVEEnricher struct {
	vulnRepo CVEEnricherVulnRepo
	enabled  bool
	logger   zerolog.Logger
}

// NewCVEEnricher creates a CVE enricher.
func NewCVEEnricher(logger zerolog.Logger, vulnRepo CVEEnricherVulnRepo, enabled bool) *CVEEnricher {
	return &CVEEnricher{vulnRepo: vulnRepo, enabled: enabled, logger: logger}
}

// Name implements Enricher.
func (e *CVEEnricher) Name() string { return "cve" }

// Enrich searches the local CVE database for CVEs matching the asset's OS,
// then upserts matching vulnerabilities into the vulnerabilities table.
func (e *CVEEnricher) Enrich(ctx context.Context, asset *model.Asset) (*EnrichmentResult, error) {
	result := &EnrichmentResult{EnricherName: e.Name()}

	if !e.enabled {
		return result, nil
	}
	if asset.OS == nil || *asset.OS == "" {
		return result, nil
	}

	cves, err := e.vulnRepo.FindCVEsForAsset(ctx, *asset.OS)
	if err != nil {
		return result, fmt.Errorf("find CVEs for asset: %w", err)
	}
	if len(cves) == 0 {
		return result, nil
	}

	inserted := 0
	for _, cve := range cves {
		if err := e.vulnRepo.UpsertFromCVE(ctx, asset.TenantID, asset.ID, cve); err != nil {
			e.logger.Warn().
				Err(err).
				Str("asset_id", asset.ID.String()).
				Str("cve_id", cve.CVEID).
				Msg("failed to upsert CVE vulnerability")
			continue
		}
		inserted++
	}

	if inserted > 0 {
		result.FieldsAdded = append(result.FieldsAdded, fmt.Sprintf("vulnerabilities(%d)", inserted))
	}
	return result, nil
}
