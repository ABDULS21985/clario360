package lineage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AssetLister retrieves active data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// LineageRepository persists and retrieves lineage edges.
type LineageRepository interface {
	Upsert(ctx context.Context, edge *model.LineageEdge) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.LineageGraphParams) ([]model.LineageEdge, error)
	GetUpstream(ctx context.Context, tenantID, assetID uuid.UUID, depth int) ([]model.LineageEdge, error)
	GetDownstream(ctx context.Context, tenantID, assetID uuid.UUID, depth int) ([]model.LineageEdge, error)
}

// LineageEngine orchestrates lineage discovery by combining SQL parsing,
// pipeline tracking, and schema-similarity inference.
type LineageEngine struct {
	assets   AssetLister
	lineage  LineageRepository
	sqlParser *SQLParser
	inferrer  *InferredLineageDetector
	logger    zerolog.Logger
}

// NewLineageEngine creates a LineageEngine with the given dependencies.
func NewLineageEngine(assets AssetLister, lineage LineageRepository, logger zerolog.Logger) *LineageEngine {
	return &LineageEngine{
		assets:    assets,
		lineage:   lineage,
		sqlParser: NewSQLParser(),
		inferrer:  NewInferredLineageDetector(logger),
		logger:    logger.With().Str("component", "lineage_engine").Logger(),
	}
}

// BuildLineage discovers lineage relationships for all active assets in a
// tenant by running schema-similarity inference. Inferred edges are persisted
// via the LineageRepository.
func (e *LineageEngine) BuildLineage(ctx context.Context, tenantID uuid.UUID) error {
	e.logger.Info().
		Str("tenant_id", tenantID.String()).
		Msg("starting lineage build")

	assets, err := e.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("listing active assets: %w", err)
	}

	if len(assets) < 2 {
		e.logger.Info().
			Int("assets", len(assets)).
			Msg("not enough assets for lineage inference")
		return nil
	}

	// Detect inferred lineage from schema similarity.
	inferredEdges := e.inferrer.DetectSimilarSchemas(assets)

	e.logger.Info().
		Int("inferred_edges", len(inferredEdges)).
		Msg("schema similarity analysis complete")

	// Populate tenant ID and persist each inferred edge.
	persisted := 0
	for i := range inferredEdges {
		inferredEdges[i].TenantID = tenantID

		// Look up asset names for the edge.
		inferredEdges[i].SourceAssetName = findAssetName(assets, inferredEdges[i].SourceAssetID)
		inferredEdges[i].TargetAssetName = findAssetName(assets, inferredEdges[i].TargetAssetID)

		// Look up classifications.
		srcAsset := findAsset(assets, inferredEdges[i].SourceAssetID)
		tgtAsset := findAsset(assets, inferredEdges[i].TargetAssetID)
		if srcAsset != nil {
			inferredEdges[i].SourceClassification = srcAsset.DataClassification
		}
		if tgtAsset != nil {
			inferredEdges[i].TargetClassification = tgtAsset.DataClassification
		}
		inferredEdges[i].ClassificationChanged =
			inferredEdges[i].SourceClassification != inferredEdges[i].TargetClassification

		// Merge PII types from both assets.
		piiSet := make(map[string]bool)
		if srcAsset != nil {
			for _, p := range srcAsset.PIITypes {
				piiSet[p] = true
			}
		}
		if tgtAsset != nil {
			for _, p := range tgtAsset.PIITypes {
				piiSet[p] = true
			}
		}
		var piiTypes []string
		for p := range piiSet {
			piiTypes = append(piiTypes, p)
		}
		inferredEdges[i].PIITypesTransferred = piiTypes

		if err := e.lineage.Upsert(ctx, &inferredEdges[i]); err != nil {
			e.logger.Error().
				Err(err).
				Str("source", inferredEdges[i].SourceAssetID.String()).
				Str("target", inferredEdges[i].TargetAssetID.String()).
				Msg("failed to persist inferred lineage edge")
			continue
		}
		persisted++
	}

	e.logger.Info().
		Int("persisted", persisted).
		Int("total_inferred", len(inferredEdges)).
		Msg("lineage build complete")

	return nil
}

// GetGraph retrieves the full lineage graph for a tenant, optionally filtered
// by the given parameters.
func (e *LineageEngine) GetGraph(ctx context.Context, tenantID uuid.UUID, params *dto.LineageGraphParams) (*model.LineageGraph, error) {
	edges, err := e.lineage.ListByTenant(ctx, tenantID, params)
	if err != nil {
		return nil, fmt.Errorf("listing lineage edges: %w", err)
	}

	assets, err := e.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for graph: %w", err)
	}

	graphOps := NewGraphOperations(e.logger)
	graph := graphOps.BuildGraph(edges, assets)

	return graph, nil
}

// findAssetName finds an asset name by ID in the given slice.
func findAssetName(assets []*cybermodel.DSPMDataAsset, id uuid.UUID) string {
	for _, a := range assets {
		if a.ID == id || a.AssetID == id {
			return a.AssetName
		}
	}
	return ""
}

// findAsset finds an asset by ID in the given slice.
func findAsset(assets []*cybermodel.DSPMDataAsset, id uuid.UUID) *cybermodel.DSPMDataAsset {
	for _, a := range assets {
		if a.ID == id || a.AssetID == id {
			return a
		}
	}
	return nil
}
