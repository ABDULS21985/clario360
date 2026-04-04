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

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/database"
)

type ModelRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewModelRepository(db *pgxpool.Pool, logger zerolog.Logger) *ModelRepository {
	return &ModelRepository{db: db, logger: logger}
}

func (r *ModelRepository) Create(ctx context.Context, item *model.DataModel) error {
	query := `
		INSERT INTO data_models (
			id, tenant_id, name, display_name, description, status, schema_definition,
			source_id, source_table, quality_rules, data_classification, contains_pii,
			pii_columns, field_count, version, previous_version_id, tags, metadata,
			created_by, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22
		)`
	_, err := r.db.Exec(ctx, query,
		item.ID, item.TenantID, item.Name, item.DisplayName, item.Description, item.Status, marshalJSON(item.SchemaDefinition),
		item.SourceID, item.SourceTable, marshalJSON(item.QualityRules), item.DataClassification, item.ContainsPII,
		item.PIIColumns, item.FieldCount, item.Version, item.PreviousVersionID, item.Tags, item.Metadata,
		item.CreatedBy, item.CreatedAt, item.UpdatedAt, item.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert data model: %w", err)
	}
	return nil
}

func (r *ModelRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DataModel, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, display_name, description, status, schema_definition,
		       source_id, source_table, quality_rules, data_classification, contains_pii,
		       pii_columns, field_count, version, previous_version_id, tags, metadata,
		       created_by, created_at, updated_at, deleted_at
		FROM data_models
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	)
	return scanDataModel(row)
}

func (r *ModelRepository) List(ctx context.Context, tenantID uuid.UUID, params dto.ListModelsParams) ([]*model.DataModel, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.name, a.display_name, a.description, a.status, a.schema_definition,
		       a.source_id, a.source_table, a.quality_rules, a.data_classification, a.contains_pii,
		       a.pii_columns, a.field_count, a.version, a.previous_version_id, a.tags, a.metadata,
		       a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM data_models a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	qb.WhereIn("a.status", params.Statuses)
	qb.WhereIf(params.SourceID != "", "a.source_id = ?", params.SourceID)
	qb.WhereIn("a.data_classification", params.DataClassifications)
	if params.ContainsPII != nil {
		qb.Where("a.contains_pii = ?", *params.ContainsPII)
	}
	qb.OrderBy(coalesce(params.Sort, "updated_at"), coalesce(params.Order, "desc"), []string{"name", "status", "version", "created_at", "updated_at"})
	qb.Paginate(params.Page, params.PerPage)

	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list data models: %w", err)
	}
	defer rows.Close()

	items := make([]*model.DataModel, 0)
	for rows.Next() {
		item, err := scanDataModel(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate data models: %w", err)
	}

	countQB := database.NewQueryBuilder(`SELECT COUNT(*) FROM data_models a`)
	countQB.Where("a.tenant_id = ?", tenantID)
	countQB.Where("a.deleted_at IS NULL")
	countQB.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	countQB.WhereIn("a.status", params.Statuses)
	countQB.WhereIf(params.SourceID != "", "a.source_id = ?", params.SourceID)
	countQB.WhereIn("a.data_classification", params.DataClassifications)
	if params.ContainsPII != nil {
		countQB.Where("a.contains_pii = ?", *params.ContainsPII)
	}
	countQuery, countArgs := countQB.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count data models: %w", err)
	}
	return items, total, nil
}

func (r *ModelRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID, deletedAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE data_models SET deleted_at = $3, updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, deletedAt,
	)
	if err != nil {
		return fmt.Errorf("soft delete data model: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ModelRepository) NextVersion(ctx context.Context, tenantID uuid.UUID, name string) (int, *uuid.UUID, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0), (
			SELECT id FROM data_models
			WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL
			ORDER BY version DESC
			LIMIT 1
		)
		FROM data_models
		WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL`,
		tenantID, name,
	)

	var maxVersion int
	var previousVersionID *uuid.UUID
	if err := row.Scan(&maxVersion, &previousVersionID); err != nil {
		return 0, nil, fmt.Errorf("get data model next version: %w", err)
	}
	return maxVersion + 1, previousVersionID, nil
}

func (r *ModelRepository) ListVersions(ctx context.Context, tenantID, id uuid.UUID) ([]*model.DataModel, error) {
	current, err := r.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, display_name, description, status, schema_definition,
		       source_id, source_table, quality_rules, data_classification, contains_pii,
		       pii_columns, field_count, version, previous_version_id, tags, metadata,
		       created_by, created_at, updated_at, deleted_at
		FROM data_models
		WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL
		ORDER BY version DESC`,
		tenantID, current.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("list data model versions: %w", err)
	}
	defer rows.Close()

	values := make([]*model.DataModel, 0)
	for rows.Next() {
		item, err := scanDataModel(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, item)
	}
	return values, rows.Err()
}

type modelScanner interface {
	Scan(dest ...any) error
}

func scanDataModel(scanner modelScanner) (*model.DataModel, error) {
	item := &model.DataModel{}
	var schemaJSON []byte
	var rulesJSON []byte
	var metadata []byte
	var piiColumns []string
	var tags []string
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.DisplayName, &item.Description, &item.Status, &schemaJSON,
		&item.SourceID, &item.SourceTable, &rulesJSON, &item.DataClassification, &item.ContainsPII,
		&piiColumns, &item.FieldCount, &item.Version, &item.PreviousVersionID, &tags, &metadata,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.PIIColumns = piiColumns
	item.Tags = tags
	item.Metadata = metadata
	if len(schemaJSON) > 0 {
		if err := json.Unmarshal(schemaJSON, &item.SchemaDefinition); err != nil {
			// Handle {"fields": [...]} wrapper format from seeded data.
			var wrapper struct {
				Fields []model.ModelField `json:"fields"`
			}
			if err2 := json.Unmarshal(schemaJSON, &wrapper); err2 != nil {
				return nil, fmt.Errorf("decode model schema_definition: %w", err)
			}
			item.SchemaDefinition = wrapper.Fields
		}
	}
	if len(rulesJSON) > 0 {
		if err := json.Unmarshal(rulesJSON, &item.QualityRules); err != nil {
			return nil, fmt.Errorf("decode model quality_rules: %w", err)
		}
	}
	return item, nil
}

func marshalJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}
