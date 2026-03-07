package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type GraphBuilder struct {
	db      *pgxpool.Pool
	repo    *repository.LineageRepository
	logger  zerolog.Logger
}

func NewGraphBuilder(db *pgxpool.Pool, repo *repository.LineageRepository, logger zerolog.Logger) *GraphBuilder {
	return &GraphBuilder{db: db, repo: repo, logger: logger}
}

func (b *GraphBuilder) BuildFullGraph(ctx context.Context, tenantID uuid.UUID) (*model.LineageGraph, error) {
	records, err := b.repo.ListActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return b.buildGraph(ctx, tenantID, records)
}

func (b *GraphBuilder) BuildDirectionalGraph(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int, direction string) (*model.LineageGraph, error) {
	fullGraph, err := b.BuildFullGraph(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if depth <= 0 {
		depth = 3
	}

	state := newGraphState(fullGraph.Nodes, fullGraph.Edges)
	centerKey := nodeKey(entityType, entityID)
	center, ok := state.nodes[centerKey]
	if !ok {
		return &model.LineageGraph{
			Nodes: []model.LineageNode{},
			Edges: []model.LineageEdge{},
			Stats: model.GraphStats{NodesByType: map[string]int{}},
		}, nil
	}

	visited := map[string]int{centerKey: 0}
	queue := []string{centerKey}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		hops := visited[current]
		if hops >= depth {
			continue
		}

		var edges []*model.LineageEdge
		switch direction {
		case "upstream":
			edges = state.incoming[current]
		case "downstream":
			edges = state.outgoing[current]
		default:
			edges = append(edges, state.incoming[current]...)
			edges = append(edges, state.outgoing[current]...)
		}

		for _, edge := range edges {
			next := edge.Target
			if direction == "upstream" {
				next = edge.Source
			}
			if direction == "" {
				if edge.Source == current {
					next = edge.Target
				} else {
					next = edge.Source
				}
			}
			if _, seen := visited[next]; seen {
				continue
			}
			visited[next] = hops + 1
			queue = append(queue, next)
		}
	}

	selectedNodes := make([]model.LineageNode, 0, len(visited))
	for _, id := range sortedNodeIDs(state.nodes) {
		if _, ok := visited[id]; !ok {
			continue
		}
		node := *state.nodes[id]
		if id == centerKey {
			if node.Metadata == nil {
				node.Metadata = map[string]any{}
			}
			node.Metadata["is_center"] = true
		}
		selectedNodes = append(selectedNodes, node)
	}

	selectedEdges := make([]model.LineageEdge, 0)
	for _, edge := range state.edges {
		if _, ok := visited[edge.Source]; !ok {
			continue
		}
		if _, ok := visited[edge.Target]; !ok {
			continue
		}
		selectedEdges = append(selectedEdges, *edge)
	}
	return buildSubgraph(selectedNodes, selectedEdges), nil
}

func (b *GraphBuilder) BuildEntityGraph(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	return b.BuildDirectionalGraph(ctx, tenantID, entityType, entityID, depth, "")
}

func (b *GraphBuilder) Search(ctx context.Context, tenantID uuid.UUID, query string, entityType string, limit int) ([]model.LineageNode, error) {
	graph, err := b.BuildFullGraph(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(strings.ToLower(query))
	entityType = strings.TrimSpace(strings.ToLower(entityType))
	if limit <= 0 {
		limit = 25
	}
	results := make([]model.LineageNode, 0, limit)
	for _, node := range graph.Nodes {
		if entityType != "" && strings.ToLower(node.Type) != entityType {
			continue
		}
		if query != "" {
			if !strings.Contains(strings.ToLower(node.Name), query) && !strings.Contains(strings.ToLower(node.ID), query) {
				continue
			}
		}
		results = append(results, node)
		if len(results) == limit {
			break
		}
	}
	return results, nil
}

func (b *GraphBuilder) Stats(ctx context.Context, tenantID uuid.UUID) (*model.LineageStatsSummary, error) {
	graph, err := b.BuildFullGraph(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats := &model.LineageStatsSummary{
		NodeCount:            graph.Stats.NodeCount,
		EdgeCount:            graph.Stats.EdgeCount,
		MaxDepth:             graph.Stats.MaxDepth,
		SourceCount:          graph.Stats.SourceCount,
		ConsumerCount:        graph.Stats.ConsumerCount,
		NodesByType:          graph.Stats.NodesByType,
		LastUpdatedAtUnixSec: time.Now().UTC().Unix(),
	}
	for _, node := range graph.Nodes {
		if node.IsCritical {
			stats.CriticalPathNodes++
		}
	}
	return stats, nil
}

func (b *GraphBuilder) buildGraph(ctx context.Context, tenantID uuid.UUID, records []*model.LineageEdgeRecord) (*model.LineageGraph, error) {
	sanitized, deactivated, err := b.pruneCycles(ctx, tenantID, records)
	if err != nil {
		return nil, err
	}
	if len(deactivated) > 0 {
		b.logger.Warn().Str("tenant_id", tenantID.String()).Int("edge_count", len(deactivated)).Msg("lineage cycle detected; cyclic edges were deactivated")
	}

	nodesByKey, err := b.loadNodes(ctx, tenantID, sanitized)
	if err != nil {
		return nil, err
	}
	edges := make([]model.LineageEdge, 0, len(sanitized))
	for _, edge := range sanitized {
		sourceKey := nodeKey(edge.SourceType, edge.SourceID)
		targetKey := nodeKey(edge.TargetType, edge.TargetID)
		edges = append(edges, model.LineageEdge{
			ID:              edge.ID,
			Source:          sourceKey,
			Target:          targetKey,
			Relationship:    string(edge.Relationship),
			TransformDesc:   stringValue(edge.TransformationDesc),
			ColumnsAffected: edge.ColumnsAffected,
			PipelineID:      edge.PipelineID,
			Active:          edge.Active,
			LastSeenAt:      edge.LastSeenAt,
		})
	}
	return finalizeGraph(nodesByKey, edges), nil
}

func (b *GraphBuilder) pruneCycles(ctx context.Context, tenantID uuid.UUID, edges []*model.LineageEdgeRecord) ([]*model.LineageEdgeRecord, []uuid.UUID, error) {
	remainingNodes := map[string]struct{}{}
	indegree := map[string]int{}
	outgoing := map[string][]*model.LineageEdgeRecord{}
	for _, edge := range edges {
		sourceKey := nodeKey(edge.SourceType, edge.SourceID)
		targetKey := nodeKey(edge.TargetType, edge.TargetID)
		remainingNodes[sourceKey] = struct{}{}
		remainingNodes[targetKey] = struct{}{}
		indegree[targetKey]++
		if _, ok := indegree[sourceKey]; !ok {
			indegree[sourceKey] = indegree[sourceKey]
		}
		outgoing[sourceKey] = append(outgoing[sourceKey], edge)
	}

	queue := make([]string, 0)
	for node := range remainingNodes {
		if indegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	processed := map[string]struct{}{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		processed[current] = struct{}{}
		for _, edge := range outgoing[current] {
			targetKey := nodeKey(edge.TargetType, edge.TargetID)
			indegree[targetKey]--
			if indegree[targetKey] == 0 {
				queue = append(queue, targetKey)
			}
		}
	}
	if len(processed) == len(remainingNodes) {
		return edges, nil, nil
	}

	cyclicNodes := make(map[string]struct{})
	for node := range remainingNodes {
		if _, ok := processed[node]; ok {
			continue
		}
		cyclicNodes[node] = struct{}{}
	}

	activeEdges := make([]*model.LineageEdgeRecord, 0, len(edges))
	deactivated := make([]uuid.UUID, 0)
	for _, edge := range edges {
		sourceKey := nodeKey(edge.SourceType, edge.SourceID)
		targetKey := nodeKey(edge.TargetType, edge.TargetID)
		if _, sourceCyclic := cyclicNodes[sourceKey]; sourceCyclic {
			if _, targetCyclic := cyclicNodes[targetKey]; targetCyclic {
				if err := b.repo.Deactivate(ctx, tenantID, edge.ID); err != nil {
					return nil, nil, err
				}
				deactivated = append(deactivated, edge.ID)
				continue
			}
		}
		activeEdges = append(activeEdges, edge)
	}
	return activeEdges, deactivated, nil
}

func (b *GraphBuilder) loadNodes(ctx context.Context, tenantID uuid.UUID, edges []*model.LineageEdgeRecord) (map[string]model.LineageNode, error) {
	typeSet := map[model.LineageEntityType][]uuid.UUID{}
	nodes := make(map[string]model.LineageNode)
	for _, edge := range edges {
		sourceKey := nodeKey(edge.SourceType, edge.SourceID)
		if _, ok := nodes[sourceKey]; !ok {
			nodes[sourceKey] = model.LineageNode{
				ID:       sourceKey,
				Type:     string(edge.SourceType),
				EntityID: edge.SourceID,
				Name:     edge.SourceName,
			}
		}
		targetKey := nodeKey(edge.TargetType, edge.TargetID)
		if _, ok := nodes[targetKey]; !ok {
			nodes[targetKey] = model.LineageNode{
				ID:       targetKey,
				Type:     string(edge.TargetType),
				EntityID: edge.TargetID,
				Name:     edge.TargetName,
			}
		}
		typeSet[edge.SourceType] = append(typeSet[edge.SourceType], edge.SourceID)
		typeSet[edge.TargetType] = append(typeSet[edge.TargetType], edge.TargetID)
	}

	loaders := map[model.LineageEntityType]func(context.Context, uuid.UUID, []uuid.UUID, map[string]model.LineageNode) error{
		model.LineageEntityDataSource:     b.loadSourceNodes,
		model.LineageEntityDataModel:      b.loadModelNodes,
		model.LineageEntityPipeline:       b.loadPipelineNodes,
		model.LineageEntityQualityRule:    b.loadQualityRuleNodes,
		model.LineageEntityAnalyticsQuery: b.loadAnalyticsQueryNodes,
	}
	for entityType, ids := range typeSet {
		loader, ok := loaders[entityType]
		if !ok {
			continue
		}
		if err := loader(ctx, tenantID, uniqueUUIDs(ids), nodes); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (b *GraphBuilder) loadSourceNodes(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, nodes map[string]model.LineageNode) error {
	rows, err := b.db.Query(ctx, `
		SELECT id, name, type, status, table_count, total_row_count, total_size_bytes,
		       COALESCE(schema_metadata->>'highest_classification', '')
		FROM data_sources
		WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return fmt.Errorf("load lineage source nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name, sourceType, status string
		var tableCount *int
		var totalRows, totalBytes *int64
		var classification string
		if err := rows.Scan(&id, &name, &sourceType, &status, &tableCount, &totalRows, &totalBytes, &classification); err != nil {
			return fmt.Errorf("scan lineage source node: %w", err)
		}
		key := nodeKey(model.LineageEntityDataSource, id)
		node := nodes[key]
		node.Name = name
		node.Status = status
		node.Metadata = map[string]any{
			"source_type":          sourceType,
			"table_count":          intPointerValue(tableCount),
			"total_row_count":      int64PointerValue(totalRows),
			"total_size_bytes":     int64PointerValue(totalBytes),
			"data_classification":  strings.ToLower(strings.TrimSpace(classification)),
		}
		nodes[key] = node
	}
	return rows.Err()
}

func (b *GraphBuilder) loadModelNodes(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, nodes map[string]model.LineageNode) error {
	rows, err := b.db.Query(ctx, `
		SELECT id, display_name, status, data_classification, contains_pii, field_count
		FROM data_models
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, ids,
	)
	if err != nil {
		return fmt.Errorf("load lineage model nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name, status, classification string
		var containsPII bool
		var fieldCount int
		if err := rows.Scan(&id, &name, &status, &classification, &containsPII, &fieldCount); err != nil {
			return fmt.Errorf("scan lineage model node: %w", err)
		}
		key := nodeKey(model.LineageEntityDataModel, id)
		node := nodes[key]
		node.Name = name
		node.Status = status
		node.Metadata = map[string]any{
			"data_classification": strings.ToLower(strings.TrimSpace(classification)),
			"contains_pii":        containsPII,
			"field_count":         fieldCount,
		}
		nodes[key] = node
	}
	return rows.Err()
}

func (b *GraphBuilder) loadPipelineNodes(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, nodes map[string]model.LineageNode) error {
	rows, err := b.db.Query(ctx, `
		SELECT id, name, type, status, schedule
		FROM pipelines
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, ids,
	)
	if err != nil {
		return fmt.Errorf("load lineage pipeline nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name, pipelineType, status string
		var schedule *string
		if err := rows.Scan(&id, &name, &pipelineType, &status, &schedule); err != nil {
			return fmt.Errorf("scan lineage pipeline node: %w", err)
		}
		key := nodeKey(model.LineageEntityPipeline, id)
		node := nodes[key]
		node.Name = name
		node.Status = status
		node.Metadata = map[string]any{
			"pipeline_type": pipelineType,
			"schedule":      stringValue(schedule),
		}
		nodes[key] = node
	}
	return rows.Err()
}

func (b *GraphBuilder) loadQualityRuleNodes(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, nodes map[string]model.LineageNode) error {
	rows, err := b.db.Query(ctx, `
		SELECT id, name, severity, enabled
		FROM quality_rules
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, ids,
	)
	if err != nil {
		return fmt.Errorf("load lineage quality-rule nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name, severity string
		var enabled bool
		if err := rows.Scan(&id, &name, &severity, &enabled); err != nil {
			return fmt.Errorf("scan lineage quality-rule node: %w", err)
		}
		key := nodeKey(model.LineageEntityQualityRule, id)
		node := nodes[key]
		node.Name = name
		if enabled {
			node.Status = "enabled"
		} else {
			node.Status = "disabled"
		}
		node.Metadata = map[string]any{"severity": severity}
		nodes[key] = node
	}
	return rows.Err()
}

func (b *GraphBuilder) loadAnalyticsQueryNodes(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, nodes map[string]model.LineageNode) error {
	rows, err := b.db.Query(ctx, `
		SELECT id, name, visibility, run_count
		FROM saved_queries
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, ids,
	)
	if err != nil {
		return fmt.Errorf("load lineage analytics-query nodes: %w", err)
	}
	defer rows.Close()
	seen := map[uuid.UUID]struct{}{}
	for rows.Next() {
		var id uuid.UUID
		var name, visibility string
		var runCount int
		if err := rows.Scan(&id, &name, &visibility, &runCount); err != nil {
			return fmt.Errorf("scan lineage analytics-query node: %w", err)
		}
		key := nodeKey(model.LineageEntityAnalyticsQuery, id)
		node := nodes[key]
		node.Name = name
		node.Status = visibility
		node.Metadata = map[string]any{"run_count": runCount}
		nodes[key] = node
		seen[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		key := nodeKey(model.LineageEntityAnalyticsQuery, id)
		node := nodes[key]
		if node.Name == "" {
			node.Name = "Ad hoc query"
		}
		nodes[key] = node
	}
	return nil
}

func finalizeGraph(nodesByKey map[string]model.LineageNode, edges []model.LineageEdge) *model.LineageGraph {
	state := newGraphState(valuesFromNodeMap(nodesByKey), edges)
	roots := make([]string, 0)
	indegree := make(map[string]int, len(state.nodes))
	for nodeID := range state.nodes {
		indegree[nodeID] = len(state.incoming[nodeID])
		if indegree[nodeID] == 0 {
			roots = append(roots, nodeID)
		}
	}
	sort.Strings(roots)

	for nodeID, node := range state.nodes {
		node.InDegree = len(state.incoming[nodeID])
		node.OutDegree = len(state.outgoing[nodeID])
		node.Depth = 0
		node.IsCritical = nodeClassification(node) >= classificationRank(string(model.DataClassificationConfidential))
	}

	queue := append([]string(nil), roots...)
	processed := make([]string, 0, len(state.nodes))
	tempIndegree := make(map[string]int, len(indegree))
	for key, value := range indegree {
		tempIndegree[key] = value
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		processed = append(processed, current)
		for _, edge := range state.outgoing[current] {
			sourceNode := state.nodes[edge.Source]
			targetNode := state.nodes[edge.Target]
			if sourceNode.Depth+1 > targetNode.Depth {
				targetNode.Depth = sourceNode.Depth + 1
			}
			if sourceNode.IsCritical {
				targetNode.IsCritical = true
			}
			tempIndegree[edge.Target]--
			if tempIndegree[edge.Target] == 0 {
				queue = append(queue, edge.Target)
			}
		}
	}

	nodeValues := valuesFromState(state.nodes)
	sort.SliceStable(nodeValues, func(i, j int) bool {
		if nodeValues[i].Depth == nodeValues[j].Depth {
			return nodeValues[i].ID < nodeValues[j].ID
		}
		return nodeValues[i].Depth < nodeValues[j].Depth
	})

	edgeValues := make([]model.LineageEdge, 0, len(state.edges))
	for _, edge := range state.edges {
		edgeValues = append(edgeValues, *edge)
	}
	sort.SliceStable(edgeValues, func(i, j int) bool {
		if edgeValues[i].Source == edgeValues[j].Source {
			if edgeValues[i].Target == edgeValues[j].Target {
				return edgeValues[i].Relationship < edgeValues[j].Relationship
			}
			return edgeValues[i].Target < edgeValues[j].Target
		}
		return edgeValues[i].Source < edgeValues[j].Source
	})

	stats := model.GraphStats{
		NodeCount:     len(nodeValues),
		EdgeCount:     len(edgeValues),
		NodesByType:   map[string]int{},
		MaxDepth:      0,
		SourceCount:   0,
		ConsumerCount: 0,
	}
	for _, node := range nodeValues {
		stats.NodesByType[node.Type]++
		if node.Depth > stats.MaxDepth {
			stats.MaxDepth = node.Depth
		}
		if node.InDegree == 0 {
			stats.SourceCount++
		}
		if node.OutDegree == 0 {
			stats.ConsumerCount++
		}
	}

	return &model.LineageGraph{
		Nodes: nodeValues,
		Edges: edgeValues,
		Stats: stats,
	}
}

func buildSubgraph(nodes []model.LineageNode, edges []model.LineageEdge) *model.LineageGraph {
	nodeMap := make(map[string]model.LineageNode, len(nodes))
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}
	return finalizeGraph(nodeMap, edges)
}

func valuesFromNodeMap(nodes map[string]model.LineageNode) []model.LineageNode {
	values := make([]model.LineageNode, 0, len(nodes))
	for _, id := range sortedNodeIDs(func() map[string]*model.LineageNode {
		result := make(map[string]*model.LineageNode, len(nodes))
		for key, node := range nodes {
			value := node
			result[key] = &value
		}
		return result
	}()) {
		values = append(values, nodes[id])
	}
	return values
}

func valuesFromState(nodes map[string]*model.LineageNode) []model.LineageNode {
	values := make([]model.LineageNode, 0, len(nodes))
	for _, id := range sortedNodeIDs(nodes) {
		values = append(values, *nodes[id])
	}
	return values
}

func uniqueUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	unique := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func intPointerValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func int64PointerValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nodeClassification(node *model.LineageNode) int {
	if node.Metadata == nil {
		return 1
	}
	raw, ok := node.Metadata["data_classification"]
	if !ok {
		raw, ok = node.Metadata["classification"]
	}
	if !ok {
		return 1
	}
	switch value := raw.(type) {
	case string:
		return classificationRank(value)
	case json.RawMessage:
		var parsed string
		if err := json.Unmarshal(value, &parsed); err == nil {
			return classificationRank(parsed)
		}
	}
	return 1
}

