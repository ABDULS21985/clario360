package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// LineageRepository handles persistence for data lineage edges.
type LineageRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewLineageRepository creates a new LineageRepository.
func NewLineageRepository(db *pgxpool.Pool, logger zerolog.Logger) *LineageRepository {
	return &LineageRepository{db: db, logger: logger}
}

// Upsert inserts or updates a lineage edge.
func (r *LineageRepository) Upsert(ctx context.Context, edge *model.LineageEdge) error {
	if edge.ID == uuid.Nil {
		edge.ID = uuid.New()
	}
	now := time.Now().UTC()
	edge.UpdatedAt = now

	evidenceJSON, err := json.Marshal(edge.Evidence)
	if err != nil {
		return fmt.Errorf("marshal evidence: %w", err)
	}
	if edge.PIITypesTransferred == nil {
		edge.PIITypesTransferred = []string{}
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO dspm_data_lineage (
			id, tenant_id, source_asset_id, source_asset_name, source_table,
			target_asset_id, target_asset_name, target_table,
			edge_type, transformation, pipeline_id, pipeline_name,
			source_classification, target_classification, classification_changed,
			pii_types_transferred, confidence, evidence, status,
			last_transfer_at, transfer_count_30d,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$22
		)
		ON CONFLICT (tenant_id, source_asset_id, target_asset_id, COALESCE(source_table,''), COALESCE(target_table,''), edge_type)
		DO UPDATE SET
			source_asset_name = EXCLUDED.source_asset_name,
			target_asset_name = EXCLUDED.target_asset_name,
			transformation = EXCLUDED.transformation,
			pipeline_id = EXCLUDED.pipeline_id,
			pipeline_name = EXCLUDED.pipeline_name,
			source_classification = EXCLUDED.source_classification,
			target_classification = EXCLUDED.target_classification,
			classification_changed = EXCLUDED.classification_changed,
			pii_types_transferred = EXCLUDED.pii_types_transferred,
			confidence = EXCLUDED.confidence,
			evidence = EXCLUDED.evidence,
			status = EXCLUDED.status,
			last_transfer_at = EXCLUDED.last_transfer_at,
			transfer_count_30d = EXCLUDED.transfer_count_30d,
			updated_at = EXCLUDED.updated_at`,
		edge.ID, edge.TenantID, edge.SourceAssetID, edge.SourceAssetName, edge.SourceTable,
		edge.TargetAssetID, edge.TargetAssetName, edge.TargetTable,
		edge.EdgeType, edge.Transformation, edge.PipelineID, edge.PipelineName,
		edge.SourceClassification, edge.TargetClassification, edge.ClassificationChanged,
		edge.PIITypesTransferred, edge.Confidence, evidenceJSON, edge.Status,
		edge.LastTransferAt, edge.TransferCount30d,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert lineage edge: %w", err)
	}
	return nil
}

// ListByTenant returns all lineage edges for a tenant, optionally filtered.
func (r *LineageRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.LineageGraphParams) ([]model.LineageEdge, error) {
	conds := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	i := 2

	if params != nil {
		if params.Classification != nil && *params.Classification != "" {
			conds = append(conds, fmt.Sprintf("(source_classification = $%d OR target_classification = $%d)", i, i))
			args = append(args, *params.Classification)
			i++
		}
		if params.EdgeType != nil && *params.EdgeType != "" {
			conds = append(conds, fmt.Sprintf("edge_type = $%d", i))
			args = append(args, *params.EdgeType)
			i++
		}
		if params.ShowInferred != nil && !*params.ShowInferred {
			conds = append(conds, "edge_type != 'inferred'")
		}
		if params.PIIOnly != nil && *params.PIIOnly {
			conds = append(conds, "array_length(pii_types_transferred, 1) > 0")
		}
	}

	where := "WHERE " + strings.Join(conds, " AND ")
	query := fmt.Sprintf(`
		SELECT id, tenant_id, source_asset_id, source_asset_name, source_table,
		       target_asset_id, target_asset_name, target_table,
		       edge_type, transformation, pipeline_id, pipeline_name,
		       source_classification, target_classification, classification_changed,
		       pii_types_transferred, confidence, evidence, status,
		       last_transfer_at, transfer_count_30d,
		       created_at, updated_at
		FROM dspm_data_lineage
		%s
		ORDER BY updated_at DESC
		LIMIT 1000`, where)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lineage edges: %w", err)
	}
	defer rows.Close()

	edges := make([]model.LineageEdge, 0)
	for rows.Next() {
		edge, err := scanLineageEdge(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, *edge)
	}
	return edges, rows.Err()
}

// GetUpstream returns edges where the target matches the given asset, recursively up to depth levels.
func (r *LineageRepository) GetUpstream(ctx context.Context, tenantID, assetID uuid.UUID, depth int) ([]model.LineageEdge, error) {
	if depth < 1 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	query := `
		WITH RECURSIVE upstream AS (
			SELECT id, tenant_id, source_asset_id, source_asset_name, source_table,
			       target_asset_id, target_asset_name, target_table,
			       edge_type, transformation, pipeline_id, pipeline_name,
			       source_classification, target_classification, classification_changed,
			       pii_types_transferred, confidence, evidence, status,
			       last_transfer_at, transfer_count_30d,
			       created_at, updated_at,
			       1 AS depth
			FROM dspm_data_lineage
			WHERE tenant_id = $1 AND target_asset_id = $2 AND status = 'active'

			UNION ALL

			SELECT l.id, l.tenant_id, l.source_asset_id, l.source_asset_name, l.source_table,
			       l.target_asset_id, l.target_asset_name, l.target_table,
			       l.edge_type, l.transformation, l.pipeline_id, l.pipeline_name,
			       l.source_classification, l.target_classification, l.classification_changed,
			       l.pii_types_transferred, l.confidence, l.evidence, l.status,
			       l.last_transfer_at, l.transfer_count_30d,
			       l.created_at, l.updated_at,
			       u.depth + 1
			FROM dspm_data_lineage l
			JOIN upstream u ON l.target_asset_id = u.source_asset_id
			WHERE l.tenant_id = $1 AND l.status = 'active' AND u.depth < $3
		)
		SELECT id, tenant_id, source_asset_id, source_asset_name, source_table,
		       target_asset_id, target_asset_name, target_table,
		       edge_type, transformation, pipeline_id, pipeline_name,
		       source_classification, target_classification, classification_changed,
		       pii_types_transferred, confidence, evidence, status,
		       last_transfer_at, transfer_count_30d,
		       created_at, updated_at
		FROM upstream
		LIMIT 500`

	rows, err := r.db.Query(ctx, query, tenantID, assetID, depth)
	if err != nil {
		return nil, fmt.Errorf("get upstream lineage: %w", err)
	}
	defer rows.Close()

	edges := make([]model.LineageEdge, 0)
	for rows.Next() {
		edge, err := scanLineageEdge(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, *edge)
	}
	return edges, rows.Err()
}

// GetDownstream returns edges where the source matches the given asset, recursively up to depth levels.
func (r *LineageRepository) GetDownstream(ctx context.Context, tenantID, assetID uuid.UUID, depth int) ([]model.LineageEdge, error) {
	if depth < 1 {
		depth = 3
	}
	if depth > 10 {
		depth = 10
	}

	query := `
		WITH RECURSIVE downstream AS (
			SELECT id, tenant_id, source_asset_id, source_asset_name, source_table,
			       target_asset_id, target_asset_name, target_table,
			       edge_type, transformation, pipeline_id, pipeline_name,
			       source_classification, target_classification, classification_changed,
			       pii_types_transferred, confidence, evidence, status,
			       last_transfer_at, transfer_count_30d,
			       created_at, updated_at,
			       1 AS depth
			FROM dspm_data_lineage
			WHERE tenant_id = $1 AND source_asset_id = $2 AND status = 'active'

			UNION ALL

			SELECT l.id, l.tenant_id, l.source_asset_id, l.source_asset_name, l.source_table,
			       l.target_asset_id, l.target_asset_name, l.target_table,
			       l.edge_type, l.transformation, l.pipeline_id, l.pipeline_name,
			       l.source_classification, l.target_classification, l.classification_changed,
			       l.pii_types_transferred, l.confidence, l.evidence, l.status,
			       l.last_transfer_at, l.transfer_count_30d,
			       l.created_at, l.updated_at,
			       d.depth + 1
			FROM dspm_data_lineage l
			JOIN downstream d ON l.source_asset_id = d.target_asset_id
			WHERE l.tenant_id = $1 AND l.status = 'active' AND d.depth < $3
		)
		SELECT id, tenant_id, source_asset_id, source_asset_name, source_table,
		       target_asset_id, target_asset_name, target_table,
		       edge_type, transformation, pipeline_id, pipeline_name,
		       source_classification, target_classification, classification_changed,
		       pii_types_transferred, confidence, evidence, status,
		       last_transfer_at, transfer_count_30d,
		       created_at, updated_at
		FROM downstream
		LIMIT 500`

	rows, err := r.db.Query(ctx, query, tenantID, assetID, depth)
	if err != nil {
		return nil, fmt.Errorf("get downstream lineage: %w", err)
	}
	defer rows.Close()

	edges := make([]model.LineageEdge, 0)
	for rows.Next() {
		edge, err := scanLineageEdge(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, *edge)
	}
	return edges, rows.Err()
}

// scanLineageEdge scans a single lineage edge row.
func scanLineageEdge(row interface{ Scan(...interface{}) error }) (*model.LineageEdge, error) {
	var edge model.LineageEdge
	var evidenceJSON []byte
	var piiTypes []string

	err := row.Scan(
		&edge.ID, &edge.TenantID, &edge.SourceAssetID, &edge.SourceAssetName, &edge.SourceTable,
		&edge.TargetAssetID, &edge.TargetAssetName, &edge.TargetTable,
		&edge.EdgeType, &edge.Transformation, &edge.PipelineID, &edge.PipelineName,
		&edge.SourceClassification, &edge.TargetClassification, &edge.ClassificationChanged,
		&piiTypes, &edge.Confidence, &evidenceJSON, &edge.Status,
		&edge.LastTransferAt, &edge.TransferCount30d,
		&edge.CreatedAt, &edge.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("lineage edge not found")
		}
		return nil, fmt.Errorf("scan lineage edge: %w", err)
	}

	edge.PIITypesTransferred = piiTypes
	if edge.PIITypesTransferred == nil {
		edge.PIITypesTransferred = []string{}
	}

	if len(evidenceJSON) > 0 {
		_ = json.Unmarshal(evidenceJSON, &edge.Evidence)
	}
	if edge.Evidence == nil {
		edge.Evidence = make(map[string]interface{})
	}

	return &edge, nil
}
