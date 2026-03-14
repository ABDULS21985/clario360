package lineage

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// InferredLineageDetector discovers lineage relationships by comparing schema
// information across data assets and detecting similar column structures.
type InferredLineageDetector struct {
	similarityThreshold float64
	logger              zerolog.Logger
}

// NewInferredLineageDetector creates an InferredLineageDetector with a default
// similarity threshold of 0.7.
func NewInferredLineageDetector(logger zerolog.Logger) *InferredLineageDetector {
	return &InferredLineageDetector{
		similarityThreshold: 0.7,
		logger:              logger.With().Str("component", "inferred_lineage").Logger(),
	}
}

// DetectSimilarSchemas compares schema_info between all pairs of assets and
// creates inferred lineage edges for pairs whose column overlap ratio meets
// or exceeds the similarity threshold. The confidence of each edge equals
// the overlap ratio.
func (d *InferredLineageDetector) DetectSimilarSchemas(assets []*cybermodel.DSPMDataAsset) []model.LineageEdge {
	if len(assets) < 2 {
		return nil
	}

	// Pre-extract columns for each asset.
	type assetColumns struct {
		asset   *cybermodel.DSPMDataAsset
		columns map[string]bool
	}

	var assetCols []assetColumns
	for _, a := range assets {
		cols := extractSchemaColumns(a)
		if len(cols) == 0 {
			continue
		}
		colSet := make(map[string]bool, len(cols))
		for _, c := range cols {
			colSet[strings.ToLower(c)] = true
		}
		assetCols = append(assetCols, assetColumns{asset: a, columns: colSet})
	}

	var edges []model.LineageEdge
	now := time.Now().UTC()

	// Compare all pairs.
	for i := 0; i < len(assetCols); i++ {
		for j := i + 1; j < len(assetCols); j++ {
			a := assetCols[i]
			b := assetCols[j]

			overlap, overlapCols := columnOverlap(a.columns, b.columns)
			if overlap < d.similarityThreshold {
				continue
			}

			// Determine directionality: larger asset is likely the source.
			sourceAsset := a.asset
			targetAsset := b.asset
			if len(b.columns) > len(a.columns) {
				sourceAsset = b.asset
				targetAsset = a.asset
			}

			edge := model.LineageEdge{
				ID:                    uuid.New(),
				TenantID:              sourceAsset.TenantID,
				SourceAssetID:         sourceAsset.ID,
				SourceAssetName:       sourceAsset.AssetName,
				TargetAssetID:         targetAsset.ID,
				TargetAssetName:       targetAsset.AssetName,
				EdgeType:              model.EdgeTypeInferred,
				SourceClassification:  sourceAsset.DataClassification,
				TargetClassification:  targetAsset.DataClassification,
				ClassificationChanged: sourceAsset.DataClassification != targetAsset.DataClassification,
				Confidence:            overlap,
				Status:                model.EdgeStatusActive,
				Evidence: map[string]interface{}{
					"source":            "schema_similarity",
					"overlap_ratio":     overlap,
					"column_overlap":    overlapCols,
					"source_col_count":  len(a.columns),
					"target_col_count":  len(b.columns),
					"overlap_col_count": len(overlapCols),
				},
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Merge PII types transferred.
			piiSet := make(map[string]bool)
			for _, p := range sourceAsset.PIITypes {
				piiSet[p] = true
			}
			for _, p := range targetAsset.PIITypes {
				piiSet[p] = true
			}
			for p := range piiSet {
				edge.PIITypesTransferred = append(edge.PIITypesTransferred, p)
			}
			sort.Strings(edge.PIITypesTransferred)

			edges = append(edges, edge)

			d.logger.Debug().
				Str("source", sourceAsset.AssetName).
				Str("target", targetAsset.AssetName).
				Float64("overlap", overlap).
				Int("shared_cols", len(overlapCols)).
				Msg("inferred lineage edge detected")
		}
	}

	d.logger.Info().
		Int("assets_analyzed", len(assetCols)).
		Int("edges_detected", len(edges)).
		Float64("threshold", d.similarityThreshold).
		Msg("schema similarity analysis complete")

	return edges
}

// extractSchemaColumns extracts column names from an asset's SchemaInfo map.
func extractSchemaColumns(asset *cybermodel.DSPMDataAsset) []string {
	if asset.SchemaInfo == nil {
		return nil
	}

	var columns []string

	// Try tables[].columns pattern.
	if tables, ok := asset.SchemaInfo["tables"].([]interface{}); ok {
		for _, t := range tables {
			tbl, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			if cols, ok := tbl["columns"].([]interface{}); ok {
				for _, c := range cols {
					switch v := c.(type) {
					case string:
						columns = append(columns, v)
					case map[string]interface{}:
						if name, ok := v["name"].(string); ok {
							columns = append(columns, name)
						}
					}
				}
			}
		}
	}

	// Try direct columns array.
	if cols, ok := asset.SchemaInfo["columns"].([]interface{}); ok {
		for _, c := range cols {
			switch v := c.(type) {
			case string:
				columns = append(columns, v)
			case map[string]interface{}:
				if name, ok := v["name"].(string); ok {
					columns = append(columns, name)
				}
			}
		}
	}

	return columns
}

// columnOverlap computes the Jaccard-like overlap ratio between two sets of
// column names and returns the ratio along with the list of overlapping columns.
func columnOverlap(a, b map[string]bool) (float64, []string) {
	if len(a) == 0 || len(b) == 0 {
		return 0, nil
	}

	var overlapCols []string
	for col := range a {
		if b[col] {
			overlapCols = append(overlapCols, col)
		}
	}

	if len(overlapCols) == 0 {
		return 0, nil
	}

	sort.Strings(overlapCols)

	// Use the smaller set as the denominator to get the containment ratio,
	// which better captures derived datasets that are subsets.
	smaller := len(a)
	if len(b) < smaller {
		smaller = len(b)
	}

	ratio := float64(len(overlapCols)) / float64(smaller)
	return ratio, overlapCols
}
