package dspm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/model"
)

// DependencyMapper maps cyber asset relationships into DSPM data flow dependencies.
type DependencyMapper struct {
	db *pgxpool.Pool
}

// NewDependencyMapper creates a dependency mapper.
func NewDependencyMapper(db *pgxpool.Pool) *DependencyMapper { return &DependencyMapper{db: db} }

// Counts returns consumer and producer counts for a single data asset.
func (m *DependencyMapper) Counts(ctx context.Context, tenantID, assetID uuid.UUID) (int, int, error) {
	var consumerCount, producerCount int
	if err := m.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE target_asset_id = $2 AND relationship_type IN ('depends_on', 'connects_to'))::int,
			COUNT(*) FILTER (WHERE source_asset_id = $2 AND relationship_type IN ('depends_on', 'connects_to'))::int
		FROM asset_relationships
		WHERE tenant_id = $1`,
		tenantID, assetID,
	).Scan(&consumerCount, &producerCount); err != nil {
		return 0, 0, fmt.Errorf("dependency counts: %w", err)
	}
	return consumerCount, producerCount, nil
}

// MapGraph returns the DSPM dependency graph across all discovered data assets.
func (m *DependencyMapper) MapGraph(ctx context.Context, tenantID uuid.UUID) ([]model.DSPMDependencyNode, error) {
	rows, err := m.db.Query(ctx, `
		SELECT da.asset_id, a.name, a.type, da.data_classification, da.risk_score, da.consumer_count, da.producer_count
		FROM dspm_data_assets da
		JOIN assets a ON a.id = da.asset_id
		WHERE da.tenant_id = $1
		ORDER BY da.risk_score DESC, a.name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list dspm dependency nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]model.DSPMDependencyNode, 0)
	index := make(map[uuid.UUID]int)
	for rows.Next() {
		var node model.DSPMDependencyNode
		if err := rows.Scan(&node.AssetID, &node.AssetName, &node.AssetType, &node.Classification, &node.RiskScore, &node.ConsumerCount, &node.ProducerCount); err != nil {
			return nil, err
		}
		node.Dependencies = make([]model.DSPMDependencyEdge, 0)
		index[node.AssetID] = len(nodes)
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	edgeRows, err := m.db.Query(ctx, `
		SELECT source_asset_id, target_asset_id, relationship_type
		FROM asset_relationships
		WHERE tenant_id = $1
		  AND relationship_type IN ('depends_on', 'connects_to', 'runs_on')`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list dependency edges: %w", err)
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var edge model.DSPMDependencyEdge
		if err := edgeRows.Scan(&edge.FromAssetID, &edge.ToAssetID, &edge.Relationship); err != nil {
			return nil, err
		}
		if idx, ok := index[edge.FromAssetID]; ok {
			nodes[idx].Dependencies = append(nodes[idx].Dependencies, edge)
		}
		if idx, ok := index[edge.ToAssetID]; ok {
			nodes[idx].Dependencies = append(nodes[idx].Dependencies, edge)
		}
	}
	return nodes, edgeRows.Err()
}
