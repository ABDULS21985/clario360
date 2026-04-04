package lineage

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// GraphOperations provides graph traversal and analysis algorithms over
// lineage edges and asset nodes.
type GraphOperations struct {
	logger zerolog.Logger
}

// NewGraphOperations creates a GraphOperations instance.
func NewGraphOperations(logger zerolog.Logger) *GraphOperations {
	return &GraphOperations{
		logger: logger.With().Str("component", "lineage_graph").Logger(),
	}
}

// BuildGraph constructs a LineageGraph from edges and assets, computing
// aggregate statistics (total nodes, edges, PII flow count, inferred count).
func (g *GraphOperations) BuildGraph(edges []model.LineageEdge, assets []*cybermodel.DSPMDataAsset) *model.LineageGraph {
	graph := &model.LineageGraph{
		Nodes:      []model.LineageNode{},
		Edges:      edges,
		TotalEdges: len(edges),
	}

	if len(edges) == 0 {
		graph.Edges = []model.LineageEdge{}
		return graph
	}

	// Collect all asset IDs referenced in edges.
	referencedIDs := make(map[uuid.UUID]bool)
	piiFlowCount := 0
	inferredCount := 0

	for _, e := range edges {
		referencedIDs[e.SourceAssetID] = true
		referencedIDs[e.TargetAssetID] = true

		if len(e.PIITypesTransferred) > 0 {
			piiFlowCount++
		}
		if e.EdgeType == model.EdgeTypeInferred {
			inferredCount++
		}
	}

	// Build asset lookup map.
	assetMap := make(map[uuid.UUID]*cybermodel.DSPMDataAsset, len(assets))
	for _, a := range assets {
		assetMap[a.ID] = a
		assetMap[a.AssetID] = a
	}

	// Create nodes for all referenced assets.
	seenNodes := make(map[uuid.UUID]bool)
	for id := range referencedIDs {
		if seenNodes[id] {
			continue
		}
		seenNodes[id] = true

		node := model.LineageNode{
			AssetID: id,
		}

		if asset, ok := assetMap[id]; ok {
			node.AssetName = asset.AssetName
			node.AssetType = asset.AssetType
			node.Classification = asset.DataClassification
			node.ContainsPII = asset.ContainsPII
			node.PIITypes = asset.PIITypes
			node.RiskScore = asset.RiskScore
			node.PostureScore = asset.PostureScore
		}

		graph.Nodes = append(graph.Nodes, node)
	}

	graph.TotalNodes = len(graph.Nodes)
	graph.PIIFlowCount = piiFlowCount
	graph.InferredCount = inferredCount

	return graph
}

// Upstream performs a BFS traversal from the given asset ID following edges
// in reverse (target -> source) up to the specified depth. Returns the
// discovered nodes and traversed edges.
func (g *GraphOperations) Upstream(edges []model.LineageEdge, assetID uuid.UUID, depth int) ([]model.LineageNode, []model.LineageEdge) {
	if depth <= 0 {
		depth = 3
	}

	// Build reverse adjacency: target -> list of edges.
	reverseAdj := make(map[uuid.UUID][]model.LineageEdge)
	for _, e := range edges {
		reverseAdj[e.TargetAssetID] = append(reverseAdj[e.TargetAssetID], e)
	}

	visited := make(map[uuid.UUID]bool)
	visited[assetID] = true

	type queueItem struct {
		id    uuid.UUID
		depth int
	}

	queue := []queueItem{{id: assetID, depth: 0}}
	var resultNodes []model.LineageNode
	var resultEdges []model.LineageEdge

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= depth {
			continue
		}

		for _, edge := range reverseAdj[current.id] {
			resultEdges = append(resultEdges, edge)

			if !visited[edge.SourceAssetID] {
				visited[edge.SourceAssetID] = true
				node := model.LineageNode{
					AssetID:        edge.SourceAssetID,
					AssetName:      edge.SourceAssetName,
					Classification: edge.SourceClassification,
					Depth:          current.depth + 1,
				}
				resultNodes = append(resultNodes, node)
				queue = append(queue, queueItem{id: edge.SourceAssetID, depth: current.depth + 1})
			}
		}
	}

	return resultNodes, resultEdges
}

// Downstream performs a BFS traversal from the given asset ID following edges
// forward (source -> target) up to the specified depth. Returns the
// discovered nodes and traversed edges.
func (g *GraphOperations) Downstream(edges []model.LineageEdge, assetID uuid.UUID, depth int) ([]model.LineageNode, []model.LineageEdge) {
	if depth <= 0 {
		depth = 3
	}

	// Build forward adjacency: source -> list of edges.
	forwardAdj := make(map[uuid.UUID][]model.LineageEdge)
	for _, e := range edges {
		forwardAdj[e.SourceAssetID] = append(forwardAdj[e.SourceAssetID], e)
	}

	visited := make(map[uuid.UUID]bool)
	visited[assetID] = true

	type queueItem struct {
		id    uuid.UUID
		depth int
	}

	queue := []queueItem{{id: assetID, depth: 0}}
	var resultNodes []model.LineageNode
	var resultEdges []model.LineageEdge

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= depth {
			continue
		}

		for _, edge := range forwardAdj[current.id] {
			resultEdges = append(resultEdges, edge)

			if !visited[edge.TargetAssetID] {
				visited[edge.TargetAssetID] = true
				node := model.LineageNode{
					AssetID:        edge.TargetAssetID,
					AssetName:      edge.TargetAssetName,
					Classification: edge.TargetClassification,
					Depth:          current.depth + 1,
				}
				resultNodes = append(resultNodes, node)
				queue = append(queue, queueItem{id: edge.TargetAssetID, depth: current.depth + 1})
			}
		}
	}

	return resultNodes, resultEdges
}

// ImpactAnalysis computes the downstream impact of a change to the given asset.
// It counts downstream assets, PII exposure, and calculates risk amplification
// as the sum of risk scores across all downstream assets.
func (g *GraphOperations) ImpactAnalysis(edges []model.LineageEdge, assets []*cybermodel.DSPMDataAsset, assetID uuid.UUID) *model.ImpactResult {
	// Build asset lookup.
	assetMap := make(map[uuid.UUID]*cybermodel.DSPMDataAsset, len(assets))
	for _, a := range assets {
		assetMap[a.ID] = a
		assetMap[a.AssetID] = a
	}

	// Find the source asset.
	sourceAsset := assetMap[assetID]
	result := &model.ImpactResult{
		AssetID:       assetID,
		AffectedNodes: []model.LineageNode{},
		AffectedEdges: []model.LineageEdge{},
	}
	if sourceAsset != nil {
		result.AssetName = sourceAsset.AssetName
	}

	// BFS downstream with unlimited depth to find all affected assets.
	downstreamNodes, downstreamEdges := g.Downstream(edges, assetID, 100)

	result.AffectedNodes = downstreamNodes
	result.AffectedEdges = downstreamEdges
	result.DownstreamAssets = len(downstreamNodes)

	// Calculate max depth.
	maxDepth := 0
	for _, n := range downstreamNodes {
		if n.Depth > maxDepth {
			maxDepth = n.Depth
		}
	}
	result.MaxDepth = maxDepth

	// Count PII assets and compute risk amplification.
	var riskSum float64
	piiCount := 0

	for _, node := range downstreamNodes {
		downstream := assetMap[node.AssetID]
		if downstream != nil {
			riskSum += downstream.RiskScore
			if downstream.ContainsPII {
				piiCount++
			}
			// Enrich node data.
			node.ContainsPII = downstream.ContainsPII
			node.PIITypes = downstream.PIITypes
			node.RiskScore = downstream.RiskScore
			node.PostureScore = downstream.PostureScore
			node.AssetType = downstream.AssetType
		}
	}

	result.DownstreamPIIAssets = piiCount
	result.RiskAmplification = riskSum

	g.logger.Info().
		Str("asset_id", assetID.String()).
		Int("downstream_assets", result.DownstreamAssets).
		Int("downstream_pii", result.DownstreamPIIAssets).
		Int("max_depth", result.MaxDepth).
		Float64("risk_amplification", result.RiskAmplification).
		Msg("impact analysis complete")

	return result
}
