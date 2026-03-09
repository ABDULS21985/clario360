package rca

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ImpactAssessor evaluates the blast radius of an incident.
type ImpactAssessor struct {
	cyberDB *pgxpool.Pool
	logger  zerolog.Logger
}

// NewImpactAssessor creates an impact assessor.
func NewImpactAssessor(cyberDB *pgxpool.Pool, logger zerolog.Logger) *ImpactAssessor {
	return &ImpactAssessor{
		cyberDB: cyberDB,
		logger:  logger.With().Str("component", "rca-impact").Logger(),
	}
}

// AssessForAlert evaluates impact for a security alert.
func (ia *ImpactAssessor) AssessForAlert(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) (*ImpactAssessment, error) {
	if len(assetIDs) == 0 {
		return &ImpactAssessment{
			DirectAssets:    []AffectedAsset{},
			TransitiveAssets: []AffectedAsset{},
			DataAtRisk:      []DataRisk{},
			Summary:         "No assets directly affected.",
		}, nil
	}

	// Get direct assets
	directAssets, err := ia.loadAssets(ctx, tenantID, assetIDs)
	if err != nil {
		return nil, err
	}

	// Get transitive assets (connected via relationships)
	transitiveAssets, err := ia.loadTransitiveAssets(ctx, tenantID, assetIDs)
	if err != nil {
		ia.logger.Warn().Err(err).Msg("load transitive assets")
		transitiveAssets = []AffectedAsset{}
	}

	// Get data at risk from DSPM
	dataAtRisk, err := ia.loadDataRisk(ctx, tenantID, assetIDs)
	if err != nil {
		ia.logger.Warn().Err(err).Msg("load data risk")
		dataAtRisk = []DataRisk{}
	}

	// Count users at risk
	usersAtRisk := ia.countUsersAtRisk(ctx, tenantID, assetIDs)

	total := len(directAssets) + len(transitiveAssets)
	businessImpact := assessBusinessImpact(directAssets, transitiveAssets, dataAtRisk)

	return &ImpactAssessment{
		DirectAssets:    directAssets,
		TransitiveAssets: transitiveAssets,
		TotalAffected:   total,
		DataAtRisk:      dataAtRisk,
		UsersAtRisk:     usersAtRisk,
		BusinessImpact:  businessImpact,
		Summary:         buildImpactSummary(total, usersAtRisk, len(dataAtRisk), businessImpact),
	}, nil
}

// AssessForPipeline evaluates impact for a pipeline failure.
func (ia *ImpactAssessor) AssessForPipeline(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) (*ImpactAssessment, error) {
	return ia.AssessForAlert(ctx, tenantID, assetIDs)
}

func (ia *ImpactAssessor) loadAssets(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) ([]AffectedAsset, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	rows, err := ia.cyberDB.Query(ctx, `
		SELECT id, name, type::text, criticality::text
		FROM assets
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL
	`, tenantID, assetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []AffectedAsset
	for rows.Next() {
		var a AffectedAsset
		if err := rows.Scan(&a.AssetID, &a.AssetName, &a.AssetType, &a.Criticality); err != nil {
			continue
		}
		a.ImpactType = "direct"
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (ia *ImpactAssessor) loadTransitiveAssets(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) ([]AffectedAsset, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	rows, err := ia.cyberDB.Query(ctx, `
		SELECT DISTINCT a.id, a.name, a.type::text, a.criticality::text
		FROM asset_relationships r
		JOIN assets a ON (a.id = r.target_asset_id OR a.id = r.source_asset_id) AND a.tenant_id = $1
		WHERE r.tenant_id = $1
		  AND (r.source_asset_id = ANY($2) OR r.target_asset_id = ANY($2))
		  AND a.id != ALL($2)
		  AND a.deleted_at IS NULL
	`, tenantID, assetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []AffectedAsset
	for rows.Next() {
		var a AffectedAsset
		if err := rows.Scan(&a.AssetID, &a.AssetName, &a.AssetType, &a.Criticality); err != nil {
			continue
		}
		a.ImpactType = "transitive"
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (ia *ImpactAssessor) loadDataRisk(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) ([]DataRisk, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	rows, err := ia.cyberDB.Query(ctx, `
		SELECT asset_id, asset_name, data_classification, contains_pii, pii_types
		FROM dspm_data_assets
		WHERE tenant_id = $1 AND asset_id = ANY($2)
	`, tenantID, assetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var risks []DataRisk
	for rows.Next() {
		var r DataRisk
		if err := rows.Scan(&r.AssetID, &r.AssetName, &r.Classification, &r.ContainsPII, &r.PIITypes); err != nil {
			continue
		}
		risks = append(risks, r)
	}
	return risks, rows.Err()
}

func (ia *ImpactAssessor) countUsersAtRisk(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) int {
	if len(assetIDs) == 0 {
		return 0
	}

	var count int
	err := ia.cyberDB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT entity_id)
		FROM ueba_profiles
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func assessBusinessImpact(direct, transitive []AffectedAsset, dataRisk []DataRisk) string {
	hasCriticalAsset := false
	for _, a := range append(direct, transitive...) {
		if a.Criticality == "critical" {
			hasCriticalAsset = true
			break
		}
	}

	hasRestrictedData := false
	for _, d := range dataRisk {
		if d.Classification == "restricted" {
			hasRestrictedData = true
			break
		}
	}

	if hasCriticalAsset && hasRestrictedData {
		return "critical"
	}
	if hasCriticalAsset || hasRestrictedData {
		return "high"
	}
	if len(direct) > 3 || len(dataRisk) > 0 {
		return "medium"
	}
	return "low"
}

func buildImpactSummary(totalAffected, usersAtRisk, dataRiskCount int, businessImpact string) string {
	return "Impact assessment: " +
		itoa(totalAffected) + " assets affected, " +
		itoa(usersAtRisk) + " users potentially at risk, " +
		itoa(dataRiskCount) + " data classification records at risk. " +
		"Business impact: " + businessImpact + "."
}

