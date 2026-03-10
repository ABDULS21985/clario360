package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// DetectionResult contains all detected shadow copies for a tenant.
type DetectionResult struct {
	TenantID     uuid.UUID     `json:"tenant_id"`
	Matches      []ShadowMatch `json:"matches"`
	SourcesCount int           `json:"sources_count"`
	TablesCount  int           `json:"tables_count"`
	Duration     time.Duration `json:"duration"`
	Summary      string        `json:"summary"`
}

// ShadowMatch is a confirmed or suspected shadow copy.
type ShadowMatch struct {
	SourceAssetID   uuid.UUID `json:"source_asset_id"`
	SourceAssetName string    `json:"source_asset_name"`
	SourceTable     string    `json:"source_table"`
	TargetAssetID   uuid.UUID `json:"target_asset_id"`
	TargetAssetName string    `json:"target_asset_name"`
	TargetTable     string    `json:"target_table"`
	Fingerprint     string    `json:"fingerprint"`
	MatchType       string    `json:"match_type"`
	Similarity      float64   `json:"similarity"`
	HasLineage      bool      `json:"has_lineage"`
}

// Detector detects unauthorized shadow copies across data sources.
type Detector struct {
	cyberDB *pgxpool.Pool
	dataDB  *pgxpool.Pool
	logger  zerolog.Logger
}

// NewDetector creates a shadow copy detector.
func NewDetector(cyberDB, dataDB *pgxpool.Pool, logger zerolog.Logger) *Detector {
	return &Detector{
		cyberDB: cyberDB,
		dataDB:  dataDB,
		logger:  logger.With().Str("component", "shadow-detector").Logger(),
	}
}

// Detect scans all data sources in a tenant for unauthorized duplicates.
func (d *Detector) Detect(ctx context.Context, tenantID uuid.UUID) (*DetectionResult, error) {
	start := time.Now()

	// Step 1: Load all assets with schema info
	fingerprints, err := d.loadAssetFingerprints(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(fingerprints) == 0 {
		return &DetectionResult{
			TenantID: tenantID,
			Matches:  []ShadowMatch{},
			Duration: time.Since(start),
		}, nil
	}

	// Step 2: Compare fingerprints across all sources
	var allFingerprints []TableFingerprint
	for _, fps := range fingerprints {
		allFingerprints = append(allFingerprints, fps...)
	}

	matches := CompareUniqueFingerprints(allFingerprints, 0.8)

	// Step 3: Check lineage for each match
	var shadowMatches []ShadowMatch
	tablesCount := len(allFingerprints)
	seenMatches := make(map[string]struct{}, len(matches))
	lineageCache := make(map[string]bool)

	for _, match := range matches {
		// Skip self-matches (same source)
		if match.SourceFingerprint.SourceID == match.TargetFingerprint.SourceID {
			continue
		}

		matchKey := unorderedMatchKey(match)
		if _, seen := seenMatches[matchKey]; seen {
			continue
		}
		seenMatches[matchKey] = struct{}{}

		lineageKey := unorderedAssetPairKey(match.SourceFingerprint.SourceID, match.TargetFingerprint.SourceID)
		hasLineage, ok := lineageCache[lineageKey]
		if !ok {
			hasLineage = d.checkLineageExists(ctx, tenantID, match.SourceFingerprint.SourceID, match.TargetFingerprint.SourceID)
			lineageCache[lineageKey] = hasLineage
		}

		shadowMatches = append(shadowMatches, ShadowMatch{
			SourceAssetID:   uuidFromString(match.SourceFingerprint.SourceID),
			SourceAssetName: match.SourceFingerprint.SourceName,
			SourceTable:     match.SourceFingerprint.TableName,
			TargetAssetID:   uuidFromString(match.TargetFingerprint.SourceID),
			TargetAssetName: match.TargetFingerprint.SourceName,
			TargetTable:     match.TargetFingerprint.TableName,
			Fingerprint:     match.SourceFingerprint.Hash,
			MatchType:       match.MatchType,
			Similarity:      match.Similarity,
			HasLineage:      hasLineage,
		})
	}

	return &DetectionResult{
		TenantID:     tenantID,
		Matches:      shadowMatches,
		SourcesCount: len(fingerprints),
		TablesCount:  tablesCount,
		Duration:     time.Since(start),
		Summary:      fmt.Sprintf("%d potential shadow copies across %d sources and %d tables", len(shadowMatches), len(fingerprints), tablesCount),
	}, nil
}

// loadAssetFingerprints loads schema info from DSPM data assets and computes fingerprints.
func (d *Detector) loadAssetFingerprints(ctx context.Context, tenantID uuid.UUID) (map[string][]TableFingerprint, error) {
	rows, err := d.cyberDB.Query(ctx, `
		SELECT COALESCE(asset_id, id) AS source_id, name, schema_info
		FROM dspm_data_assets
		WHERE tenant_id = $1 AND schema_info IS NOT NULL AND schema_info != '{}'::jsonb
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]TableFingerprint)

	for rows.Next() {
		var (
			assetID   uuid.UUID
			assetName string
			schemaRaw json.RawMessage
		)
		if err := rows.Scan(&assetID, &assetName, &schemaRaw); err != nil {
			d.logger.Warn().Err(err).Msg("skip asset with bad schema")
			continue
		}

		fps := ExtractColumnsFromJSON(schemaRaw)
		for i := range fps {
			fps[i].SourceID = assetID.String()
			fps[i].SourceName = assetName
		}

		if len(fps) > 0 {
			result[assetID.String()] = fps
		}
	}

	return result, rows.Err()
}

// checkLineageExists queries data lineage to see if a pipeline connects source→target.
func (d *Detector) checkLineageExists(ctx context.Context, tenantID uuid.UUID, sourceID, targetID string) bool {
	if d.dataDB == nil {
		return false
	}

	var count int
	err := d.dataDB.QueryRow(ctx, `
		SELECT COUNT(*) FROM data_lineage_edges
		WHERE tenant_id = $1
		  AND active = true
		  AND (
		    (source_id = $2 AND target_id = $3)
		    OR (source_id = $3 AND target_id = $2)
		  )
	`, tenantID, sourceID, targetID).Scan(&count)
	if err != nil {
		d.logger.Debug().Err(err).Msg("lineage check failed, assuming no lineage")
		return false
	}
	return count > 0
}

func uuidFromString(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func unorderedMatchKey(match MatchResult) string {
	left := match.SourceFingerprint.SourceID + "|" + match.SourceFingerprint.TableName
	right := match.TargetFingerprint.SourceID + "|" + match.TargetFingerprint.TableName
	if left > right {
		left, right = right, left
	}
	return left + "->" + right + "|" + match.MatchType
}

func unorderedAssetPairKey(sourceID, targetID string) string {
	if sourceID > targetID {
		sourceID, targetID = targetID, sourceID
	}
	return sourceID + "->" + targetID
}
