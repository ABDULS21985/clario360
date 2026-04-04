package proliferation

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// unauthorizedEdgeTypes defines edge types that indicate unauthorized data copies.
var unauthorizedEdgeTypes = map[model.LineageEdgeType]bool{
	model.EdgeTypeManualCopy: true,
	model.EdgeTypeInferred:   true,
}

// LineageRepository retrieves lineage edges for proliferation tracking.
type LineageRepository interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.LineageGraphParams) ([]model.LineageEdge, error)
}

// ProliferationTracker uses data lineage information to track how sensitive
// data proliferates across systems, identifying unauthorized copies and
// data sprawl.
type ProliferationTracker struct {
	lineageRepo LineageRepository
	logger      zerolog.Logger
}

// NewProliferationTracker creates a new proliferation tracker instance.
func NewProliferationTracker(lineageRepo LineageRepository, logger zerolog.Logger) *ProliferationTracker {
	return &ProliferationTracker{
		lineageRepo: lineageRepo,
		logger:      logger.With().Str("component", "proliferation_tracker").Logger(),
	}
}

// Track analyzes data proliferation across all sensitive assets for a tenant.
// It uses lineage edges to discover copies of sensitive data and categorize
// them as authorized or unauthorized.
func (t *ProliferationTracker) Track(ctx context.Context, tenantID uuid.UUID, assets []*cybermodel.DSPMDataAsset) (*model.ProliferationOverview, error) {
	t.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("assets", len(assets)).
		Msg("starting proliferation tracking")

	// Retrieve all lineage edges for the tenant.
	edges, err := t.lineageRepo.ListByTenant(ctx, tenantID, &dto.LineageGraphParams{})
	if err != nil {
		return nil, err
	}

	// Build an index of edges by source asset ID.
	edgesBySource := make(map[uuid.UUID][]model.LineageEdge)
	for _, edge := range edges {
		edgesBySource[edge.SourceAssetID] = append(edgesBySource[edge.SourceAssetID], edge)
	}

	// Build a set of sensitive asset IDs for quick lookup.
	sensitiveAssets := make(map[uuid.UUID]*cybermodel.DSPMDataAsset)
	for _, asset := range assets {
		if asset.ContainsPII || asset.DataClassification == "restricted" || asset.DataClassification == "confidential" {
			sensitiveAssets[asset.AssetID] = asset
		}
	}

	overview := &model.ProliferationOverview{
		TotalSensitiveAssets: len(sensitiveAssets),
	}

	var proliferations []model.DataProliferation
	for assetID, asset := range sensitiveAssets {
		assetEdges := edgesBySource[assetID]
		if len(assetEdges) == 0 {
			continue
		}

		dp := t.TrackAsset(ctx, tenantID, assetID, assetEdges)
		if dp == nil || dp.TotalCopies == 0 {
			continue
		}

		dp.AssetName = asset.AssetName
		dp.Classification = asset.DataClassification
		dp.PIITypes = asset.PIITypes

		proliferations = append(proliferations, *dp)
		overview.AssetsWithCopies++
		overview.TotalUnauthorizedCopies += dp.UnauthorizedCopies
	}

	// Sort by unauthorized copies descending and take top proliferators.
	sort.Slice(proliferations, func(i, j int) bool {
		return proliferations[i].UnauthorizedCopies > proliferations[j].UnauthorizedCopies
	})

	// Top 10 proliferators for the overview.
	topCount := len(proliferations)
	if topCount > 10 {
		topCount = 10
	}
	overview.TopProliferators = proliferations[:topCount]

	// Build spread trend from all events.
	var allEvents []model.SpreadEvent
	for _, dp := range proliferations {
		allEvents = append(allEvents, dp.SpreadEvents...)
	}
	visualizer := NewSpreadVisualizer(t.logger)
	overview.SpreadTrend = visualizer.BuildSpreadTrend(allEvents, 30)

	t.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("sensitive_assets", overview.TotalSensitiveAssets).
		Int("assets_with_copies", overview.AssetsWithCopies).
		Int("unauthorized_copies", overview.TotalUnauthorizedCopies).
		Msg("proliferation tracking complete")

	return overview, nil
}

// TrackAsset analyzes proliferation for a single asset using its lineage edges.
// It categorizes each downstream copy as authorized or unauthorized based on
// the edge type.
func (t *ProliferationTracker) TrackAsset(ctx context.Context, tenantID uuid.UUID, assetID uuid.UUID, edges []model.LineageEdge) *model.DataProliferation {
	if len(edges) == 0 {
		return nil
	}

	dp := &model.DataProliferation{
		AssetID: assetID,
	}

	var (
		firstDetected time.Time
		lastDetected  time.Time
	)

	for _, edge := range edges {
		// Only count active edges as proliferation.
		if edge.Status != model.EdgeStatusActive {
			continue
		}

		isUnauthorized := unauthorizedEdgeTypes[edge.EdgeType]

		status := model.ProliferationAuthorized
		if isUnauthorized {
			status = model.ProliferationUnauthorized
			dp.UnauthorizedCopies++
		} else {
			dp.AuthorizedCopies++
		}
		dp.TotalCopies++

		event := model.SpreadEvent{
			ID:              edge.ID,
			SourceAssetID:   edge.SourceAssetID,
			SourceAssetName: edge.SourceAssetName,
			TargetAssetID:   edge.TargetAssetID,
			TargetAssetName: edge.TargetAssetName,
			EdgeType:        string(edge.EdgeType),
			Status:          status,
			DetectedAt:      edge.CreatedAt,
			Similarity:      edge.Confidence,
		}
		dp.SpreadEvents = append(dp.SpreadEvents, event)

		// Track detection timestamps.
		if firstDetected.IsZero() || edge.CreatedAt.Before(firstDetected) {
			firstDetected = edge.CreatedAt
		}
		if edge.CreatedAt.After(lastDetected) {
			lastDetected = edge.CreatedAt
		}
	}

	dp.FirstDetectedAt = firstDetected
	dp.LastDetectedAt = lastDetected

	return dp
}
