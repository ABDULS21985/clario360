package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type FeatureRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewFeatureRepository(db *pgxpool.Pool, logger zerolog.Logger) *FeatureRepository {
	return &FeatureRepository{
		db:     db,
		logger: logger.With().Str("component", "vciso_feature_repo").Logger(),
	}
}

func (r *FeatureRepository) SaveSnapshot(ctx context.Context, tenantID uuid.UUID, featureSet string, entityType string, entityID *string, vector any) error {
	payload, err := json.Marshal(vector)
	if err != nil {
		return fmt.Errorf("marshal feature vector: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO vciso_feature_snapshots (
			id, tenant_id, feature_set, entity_type, entity_id, vector_json, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New(), tenantID, featureSet, entityType, entityID, payload, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("save feature snapshot: %w", err)
	}
	return nil
}

func (r *FeatureRepository) ListSnapshots(ctx context.Context, tenantID uuid.UUID, featureSet string, limit int) ([]predictmodel.FeatureSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, feature_set, entity_type, entity_id, vector_json, created_at
		FROM vciso_feature_snapshots
		WHERE tenant_id = $1 AND feature_set = $2
		ORDER BY created_at DESC
		LIMIT $3`,
		tenantID, featureSet, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list feature snapshots: %w", err)
	}
	defer rows.Close()
	items := make([]predictmodel.FeatureSnapshot, 0, limit)
	for rows.Next() {
		var item predictmodel.FeatureSnapshot
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.FeatureSet,
			&item.EntityType,
			&item.EntityID,
			&item.VectorJSON,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
