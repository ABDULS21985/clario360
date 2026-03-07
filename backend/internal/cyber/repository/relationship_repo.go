package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
)

// RelationshipRepository handles asset_relationships table operations.
type RelationshipRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewRelationshipRepository creates a new RelationshipRepository.
func NewRelationshipRepository(db *pgxpool.Pool, logger zerolog.Logger) *RelationshipRepository {
	return &RelationshipRepository{db: db, logger: logger}
}

// Create adds a directed relationship between two assets.
func (r *RelationshipRepository) Create(ctx context.Context, tenantID, sourceAssetID, userID uuid.UUID, req *dto.CreateRelationshipRequest) (*model.AssetRelationship, error) {
	targetID, err := uuid.Parse(req.TargetAssetID)
	if err != nil {
		return nil, fmt.Errorf("invalid target_asset_id: %w", err)
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}

	var id uuid.UUID
	err = r.db.QueryRow(ctx, `
		INSERT INTO asset_relationships (
			id, tenant_id, source_asset_id, target_asset_id,
			relationship_type, metadata, created_by, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, now()
		)
		ON CONFLICT (tenant_id, source_asset_id, target_asset_id, relationship_type)
		DO UPDATE SET metadata = EXCLUDED.metadata
		RETURNING id`,
		tenantID, sourceAssetID, targetID,
		string(req.RelationshipType), metadata, userID,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create relationship: %w", err)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID fetches a relationship by ID.
func (r *RelationshipRepository) GetByID(ctx context.Context, tenantID, relID uuid.UUID) (*model.AssetRelationship, error) {
	row := r.db.QueryRow(ctx, `
		SELECT ar.id, ar.tenant_id, ar.source_asset_id, ar.target_asset_id,
		       ar.relationship_type, ar.metadata, ar.created_by, ar.created_at,
		       sa.name AS source_asset_name, ta.name AS target_asset_name
		FROM asset_relationships ar
		JOIN assets sa ON sa.id = ar.source_asset_id
		JOIN assets ta ON ta.id = ar.target_asset_id
		WHERE ar.tenant_id = $1 AND ar.id = $2`,
		tenantID, relID,
	)
	return scanRelationship(row)
}

// ListForAsset returns all relationships where the asset is either source or target.
func (r *RelationshipRepository) ListForAsset(ctx context.Context, tenantID, assetID uuid.UUID) ([]*model.AssetRelationship, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ar.id, ar.tenant_id, ar.source_asset_id, ar.target_asset_id,
		       ar.relationship_type, ar.metadata, ar.created_by, ar.created_at,
		       sa.name AS source_asset_name, ta.name AS target_asset_name
		FROM asset_relationships ar
		JOIN assets sa ON sa.id = ar.source_asset_id AND sa.deleted_at IS NULL
		JOIN assets ta ON ta.id = ar.target_asset_id AND ta.deleted_at IS NULL
		WHERE ar.tenant_id = $1 AND (ar.source_asset_id = $2 OR ar.target_asset_id = $2)
		ORDER BY ar.created_at DESC`,
		tenantID, assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []*model.AssetRelationship
	for rows.Next() {
		rel, err := scanRelationship(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

// Delete removes a relationship by ID.
func (r *RelationshipRepository) Delete(ctx context.Context, tenantID, assetID, relID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM asset_relationships
		WHERE tenant_id = $1 AND id = $2
		  AND (source_asset_id = $3 OR target_asset_id = $3)`,
		tenantID, relID, assetID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRelationship(row interface{ Scan(dest ...any) error }) (*model.AssetRelationship, error) {
	var rel model.AssetRelationship
	var relType string
	err := row.Scan(
		&rel.ID, &rel.TenantID, &rel.SourceAssetID, &rel.TargetAssetID,
		&relType, &rel.Metadata, &rel.CreatedBy, &rel.CreatedAt,
		&rel.SourceAssetName, &rel.TargetAssetName,
	)
	if err != nil {
		return nil, err
	}
	rel.RelationshipType = model.RelationshipType(relType)
	if rel.Metadata == nil {
		rel.Metadata = json.RawMessage("{}")
	}
	return &rel, nil
}
