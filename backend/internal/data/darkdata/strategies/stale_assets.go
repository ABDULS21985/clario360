package strategies

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/data/darkdata"
	"github.com/clario360/platform/internal/data/model"
)

type StaleAssetsStrategy struct {
	db *pgxpool.Pool
}

func NewStaleAssetsStrategy(db *pgxpool.Pool) *StaleAssetsStrategy {
	return &StaleAssetsStrategy{db: db}
}

func (s *StaleAssetsStrategy) Name() string {
	return "stale_assets"
}

func (s *StaleAssetsStrategy) Scan(ctx context.Context, tenantID uuid.UUID) ([]darkdata.RawDarkDataAsset, error) {
	cutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)
	results := make([]darkdata.RawDarkDataAsset, 0)

	sourceRows, err := s.db.Query(ctx, `
		SELECT id, name, type, last_synced_at
		FROM data_sources
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND (last_synced_at IS NULL OR last_synced_at < $2)`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query stale data sources: %w", err)
	}
	for sourceRows.Next() {
		var id uuid.UUID
		var name string
		var sourceType model.DataSourceType
		var lastSyncedAt *time.Time
		if err := sourceRows.Scan(&id, &name, &sourceType, &lastSyncedAt); err != nil {
			sourceRows.Close()
			return nil, fmt.Errorf("scan stale data source: %w", err)
		}
		results = append(results, darkdata.RawDarkDataAsset{
			Name:           name,
			AssetType:      sourceTypeToDarkAssetType(sourceType),
			SourceID:       &id,
			Reason:         model.DarkDataReasonStale,
			LastAccessedAt: lastSyncedAt,
			Metadata: map[string]any{
				"entity_kind": "data_source",
				"source_type": sourceType,
			},
		})
	}
	if err := sourceRows.Err(); err != nil {
		sourceRows.Close()
		return nil, fmt.Errorf("iterate stale data sources: %w", err)
	}
	sourceRows.Close()

	modelRows, err := s.db.Query(ctx, `
		SELECT m.id, m.display_name, m.source_id, m.source_table, m.updated_at
		FROM data_models m
		WHERE m.tenant_id = $1
		  AND m.deleted_at IS NULL
		  AND m.updated_at < $2
		  AND NOT EXISTS (
		      SELECT 1 FROM analytics_audit_log a
		      WHERE a.tenant_id = m.tenant_id
		        AND a.model_id = m.id
		        AND a.executed_at >= $2
		  )`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query stale data models: %w", err)
	}
	for modelRows.Next() {
		var id uuid.UUID
		var name string
		var sourceID *uuid.UUID
		var sourceTable *string
		var updatedAt time.Time
		if err := modelRows.Scan(&id, &name, &sourceID, &sourceTable, &updatedAt); err != nil {
			modelRows.Close()
			return nil, fmt.Errorf("scan stale data model: %w", err)
		}
		results = append(results, darkdata.RawDarkDataAsset{
			Name:           name,
			AssetType:      model.DarkDataAssetDatabaseTable,
			SourceID:       sourceID,
			TableName:      sourceTable,
			Reason:         model.DarkDataReasonStale,
			LastAccessedAt: &updatedAt,
			LinkedModelID:  &id,
			Metadata: map[string]any{
				"entity_kind": "data_model",
			},
		})
	}
	if err := modelRows.Err(); err != nil {
		modelRows.Close()
		return nil, fmt.Errorf("iterate stale data models: %w", err)
	}
	modelRows.Close()

	return results, nil
}

func sourceTypeToDarkAssetType(sourceType model.DataSourceType) model.DarkDataAssetType {
	switch sourceType {
	case model.DataSourceTypeCSV, model.DataSourceTypeS3:
		return model.DarkDataAssetFile
	case model.DataSourceTypeAPI:
		return model.DarkDataAssetAPIEndpoint
	case model.DataSourceTypeStream:
		return model.DarkDataAssetStreamTopic
	default:
		return model.DarkDataAssetDatabaseTable
	}
}
