package analyzer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// CrossAssetAnalyzer identifies identities with access spanning multiple sensitive
// data domains. An identity accessing 3+ distinct restricted/confidential assets
// is flagged as having "broad sensitive access".
type CrossAssetAnalyzer struct {
	repo   MappingProvider
	logger zerolog.Logger
}

// NewCrossAssetAnalyzer creates a new cross-asset access correlator.
func NewCrossAssetAnalyzer(repo MappingProvider, logger zerolog.Logger) *CrossAssetAnalyzer {
	return &CrossAssetAnalyzer{
		repo:   repo,
		logger: logger.With().Str("analyzer", "cross_asset").Logger(),
	}
}

// Analyze finds identities with access spanning multiple sensitive data domains.
// Flags any identity accessing 3+ distinct restricted/confidential assets.
func (a *CrossAssetAnalyzer) Analyze(ctx context.Context, tenantID uuid.UUID) ([]model.CrossAssetResult, error) {
	mappings, err := a.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Group by identity.
	type identityAccess struct {
		name            string
		identityType    string
		classifications map[string]bool
		assetTypes      map[string]bool
		sensitiveCount  int
	}

	byIdentity := make(map[string]*identityAccess)
	for _, m := range mappings {
		key := m.IdentityType + "|" + m.IdentityID
		ia, ok := byIdentity[key]
		if !ok {
			ia = &identityAccess{
				name:            m.IdentityName,
				identityType:    m.IdentityType,
				classifications: make(map[string]bool),
				assetTypes:      make(map[string]bool),
			}
			byIdentity[key] = ia
		}

		ia.classifications[m.DataClassification] = true
		// Use data_asset_name suffix as a proxy for asset type when explicit type is unavailable.
		ia.assetTypes[m.DataClassification+"_"+m.DataAssetID.String()] = true

		if m.DataClassification == "restricted" || m.DataClassification == "confidential" {
			ia.sensitiveCount++
		}
	}

	var results []model.CrossAssetResult
	for key, ia := range byIdentity {
		if ia.sensitiveCount < 3 {
			continue
		}

		identityID := key[len(ia.identityType)+1:]
		distinctClassifications := len(ia.classifications)
		distinctAssetTypes := len(ia.assetTypes)
		breadthScore := distinctClassifications * distinctAssetTypes

		recommendation := fmt.Sprintf(
			"Review separation of duties — identity accesses %d sensitive assets across %d classification levels",
			ia.sensitiveCount, distinctClassifications,
		)

		results = append(results, model.CrossAssetResult{
			IdentityType:            ia.identityType,
			IdentityID:              identityID,
			IdentityName:            ia.name,
			DistinctClassifications: distinctClassifications,
			DistinctAssetTypes:      distinctAssetTypes,
			BreadthScore:            breadthScore,
			SensitiveAssetCount:     ia.sensitiveCount,
			Recommendation:          recommendation,
		})
	}

	return results, nil
}
