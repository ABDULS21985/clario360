package strategies

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/data/darkdata"
	"github.com/clario360/platform/internal/data/model"
)

type UngovernedDataStrategy struct {
	db *pgxpool.Pool
}

func NewUngovernedDataStrategy(db *pgxpool.Pool) *UngovernedDataStrategy {
	return &UngovernedDataStrategy{db: db}
}

func (s *UngovernedDataStrategy) Name() string {
	return "ungoverned_data"
}

func (s *UngovernedDataStrategy) Scan(ctx context.Context, tenantID uuid.UUID) ([]darkdata.RawDarkDataAsset, error) {
	rows, err := s.db.Query(ctx, `
		SELECT m.id, m.display_name, m.source_id, m.source_table, m.contains_pii, m.data_classification,
		       COALESCE(jsonb_array_length(m.quality_rules), 0) AS embedded_rule_count,
		       EXISTS (
		           SELECT 1
		           FROM quality_rules q
		           WHERE q.tenant_id = m.tenant_id
		             AND q.model_id = m.id
		             AND q.deleted_at IS NULL
		       ) AS has_quality_rules
		FROM data_models m
		WHERE m.tenant_id = $1
		  AND m.deleted_at IS NULL`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("query ungoverned data models: %w", err)
	}
	defer rows.Close()

	results := make([]darkdata.RawDarkDataAsset, 0)
	for rows.Next() {
		var (
			modelID           uuid.UUID
			name              string
			sourceID          *uuid.UUID
			sourceTable       *string
			containsPII       bool
			classification    model.DataClassification
			embeddedRuleCount int
			hasQualityRules   bool
		)
		if err := rows.Scan(&modelID, &name, &sourceID, &sourceTable, &containsPII, &classification, &embeddedRuleCount, &hasQualityRules); err != nil {
			return nil, fmt.Errorf("scan ungoverned data model: %w", err)
		}
		if !hasQualityRules && embeddedRuleCount == 0 {
			results = append(results, darkdata.RawDarkDataAsset{
				Name:          name,
				AssetType:     model.DarkDataAssetDatabaseTable,
				SourceID:      sourceID,
				TableName:     sourceTable,
				Reason:        model.DarkDataReasonUngoverned,
				LinkedModelID: &modelID,
				Metadata: map[string]any{
					"contains_pii": containsPII,
				},
			})
		}
		if classification == model.DataClassificationInternal {
			results = append(results, darkdata.RawDarkDataAsset{
				Name:          name,
				AssetType:     model.DarkDataAssetDatabaseTable,
				SourceID:      sourceID,
				TableName:     sourceTable,
				Reason:        model.DarkDataReasonUnclassified,
				LinkedModelID: &modelID,
				Metadata: map[string]any{
					"contains_pii": containsPII,
				},
			})
		}
	}
	return results, rows.Err()
}
