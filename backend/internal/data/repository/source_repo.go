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

type SourceRecord struct {
	Source          *model.DataSource
	EncryptedConfig []byte
}

type SourceRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewSourceRepository(db *pgxpool.Pool, logger zerolog.Logger) *SourceRepository {
	return &SourceRepository{db: db, logger: logger}
}

func (r *SourceRepository) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	query := `SELECT EXISTS (
		SELECT 1 FROM data_sources WHERE tenant_id = $1 AND lower(name) = lower($2) AND deleted_at IS NULL`
	args := []any{tenantID, name}
	if excludeID != nil {
		query += ` AND id <> $3`
		args = append(args, *excludeID)
	}
	query += `)`

	var exists bool
	if err := r.db.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check duplicate data source name: %w", err)
	}
	return exists, nil
}

func (r *SourceRepository) Create(ctx context.Context, record *SourceRecord) error {
	source := record.Source
	query := `
		INSERT INTO data_sources (
			id, tenant_id, name, description, type, connection_config, encryption_key_id,
			status, schema_metadata, schema_discovered_at, last_synced_at, last_sync_status,
			last_sync_error, last_sync_duration_ms, sync_frequency, next_sync_at, table_count,
			total_row_count, total_size_bytes, tags, metadata, created_by, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25
		)`

	schemaPayload := marshalSchema(source.SchemaMetadata)
	_, err := r.db.Exec(ctx, query,
		source.ID, source.TenantID, source.Name, source.Description, source.Type, record.EncryptedConfig, source.EncryptionKeyID,
		source.Status, schemaPayload, source.SchemaDiscoveredAt, source.LastSyncedAt, source.LastSyncStatus,
		source.LastSyncError, source.LastSyncDurationMs, source.SyncFrequency, source.NextSyncAt, source.TableCount,
		source.TotalRowCount, source.TotalSizeBytes, source.Tags, source.Metadata, source.CreatedBy, source.CreatedAt, source.UpdatedAt, source.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert data source: %w", err)
	}
	return nil
}

func (r *SourceRepository) Update(ctx context.Context, record *SourceRecord) error {
	source := record.Source
	query := `
		UPDATE data_sources
		SET name = $3,
		    description = $4,
		    type = $5,
		    connection_config = $6,
		    encryption_key_id = $7,
		    status = $8,
		    sync_frequency = $9,
		    next_sync_at = $10,
		    tags = $11,
		    metadata = $12,
		    updated_at = $13,
		    deleted_at = $14
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`

	tagValues := source.Tags
	if tagValues == nil {
		tagValues = []string{}
	}
	result, err := r.db.Exec(ctx, query,
		source.TenantID, source.ID, source.Name, source.Description, source.Type, record.EncryptedConfig, source.EncryptionKeyID,
		source.Status, source.SyncFrequency, source.NextSyncAt, tagValues, source.Metadata, source.UpdatedAt, source.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("update data source: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SourceRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*SourceRecord, error) {
	query := `
		SELECT id, tenant_id, name, description, type, connection_config, encryption_key_id,
		       status, last_error, schema_metadata, schema_discovered_at, last_synced_at,
		       last_sync_status, last_sync_error, last_sync_duration_ms, sync_frequency,
		       next_sync_at, table_count, total_row_count, total_size_bytes, tags, metadata,
		       created_by, created_at, updated_at, deleted_at
		FROM data_sources
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`

	record, err := scanSourceRecord(r.db.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *SourceRepository) List(ctx context.Context, tenantID uuid.UUID, params dto.ListSourcesParams) ([]*SourceRecord, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.name, a.description, a.type, a.connection_config, a.encryption_key_id,
		       a.status, a.last_error, a.schema_metadata, a.schema_discovered_at, a.last_synced_at,
		       a.last_sync_status, a.last_sync_error, a.last_sync_duration_ms, a.sync_frequency,
		       a.next_sync_at, a.table_count, a.total_row_count, a.total_size_bytes, a.tags, a.metadata,
		       a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM data_sources a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	qb.WhereIf(params.Type != "", "a.type = ?", params.Type)
	qb.WhereIf(params.Status != "", "a.status = ?", params.Status)
	if params.HasSchema != nil {
		if *params.HasSchema {
			qb.Where("a.schema_metadata IS NOT NULL")
		} else {
			qb.Where("a.schema_metadata IS NULL")
		}
	}
	qb.OrderBy(coalesce(params.Sort, "updated_at"), coalesce(params.Order, "desc"), []string{"name", "type", "status", "created_at", "updated_at"})
	qb.Paginate(params.Page, params.PerPage)
	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list data sources: %w", err)
	}
	defer rows.Close()

	items := make([]*SourceRecord, 0)
	for rows.Next() {
		record, err := scanSourceRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate data sources: %w", err)
	}

	countQB := database.NewQueryBuilder(`SELECT COUNT(*) FROM data_sources a`)
	countQB.Where("a.tenant_id = ?", tenantID)
	countQB.Where("a.deleted_at IS NULL")
	countQB.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	countQB.WhereIf(params.Type != "", "a.type = ?", params.Type)
	countQB.WhereIf(params.Status != "", "a.status = ?", params.Status)
	if params.HasSchema != nil {
		if *params.HasSchema {
			countQB.Where("a.schema_metadata IS NOT NULL")
		} else {
			countQB.Where("a.schema_metadata IS NULL")
		}
	}
	countQuery, countArgs := countQB.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count data sources: %w", err)
	}
	return items, total, nil
}

func (r *SourceRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID, deletedAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE data_sources
		SET deleted_at = $3, updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, deletedAt,
	)
	if err != nil {
		return fmt.Errorf("soft delete data source: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SourceRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.DataSourceStatus, lastError *string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE data_sources
		SET status = $3, last_error = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, status, lastError,
	)
	if err != nil {
		return fmt.Errorf("update data source status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SourceRepository) UpdateSchema(ctx context.Context, tenantID, id uuid.UUID, schema *model.DiscoveredSchema, discoveredAt time.Time, estimate *SizeEstimatePatch) error {
	result, err := r.db.Exec(ctx, `
		UPDATE data_sources
		SET schema_metadata = $3,
		    schema_discovered_at = $4,
		    table_count = $5,
		    total_row_count = $6,
		    total_size_bytes = $7,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, marshalSchema(schema), discoveredAt, estimate.TableCount, estimate.TotalRows, estimate.TotalBytes,
	)
	if err != nil {
		return fmt.Errorf("update data source schema: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SourceRepository) UpdateSyncState(ctx context.Context, tenantID, id uuid.UUID, patch SyncStatePatch) error {
	result, err := r.db.Exec(ctx, `
		UPDATE data_sources
		SET status = $3,
		    last_synced_at = $4,
		    last_sync_status = $5,
		    last_sync_error = $6,
		    last_sync_duration_ms = $7,
		    table_count = COALESCE($8, table_count),
		    total_row_count = COALESCE($9, total_row_count),
		    total_size_bytes = COALESCE($10, total_size_bytes),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, patch.Status, patch.LastSyncedAt, patch.LastSyncStatus, patch.LastSyncError,
		patch.LastSyncDurationMs, patch.TableCount, patch.TotalRows, patch.TotalBytes,
	)
	if err != nil {
		return fmt.Errorf("update data source sync state: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SourceRepository) AggregateStats(ctx context.Context, tenantID uuid.UUID) (*dto.AggregateSourceStatsResponse, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*) AS total_sources,
		       COUNT(*) FILTER (WHERE schema_metadata IS NOT NULL) AS sources_with_schema,
		       COALESCE(SUM(total_row_count), 0),
		       COALESCE(SUM(total_size_bytes), 0)
		FROM data_sources
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	)

	stats := &dto.AggregateSourceStatsResponse{
		ByType:   map[string]int{},
		ByStatus: map[string]int{},
	}
	if err := row.Scan(&stats.TotalSources, &stats.SourcesWithSchema, &stats.TotalRows, &stats.TotalSizeBytes); err != nil {
		return nil, fmt.Errorf("aggregate data source stats: %w", err)
	}

	typeRows, err := r.db.Query(ctx, `
		SELECT type, COUNT(*) FROM data_sources
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY type`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("aggregate data sources by type: %w", err)
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var sourceType string
		var count int
		if err := typeRows.Scan(&sourceType, &count); err != nil {
			return nil, fmt.Errorf("scan data source type aggregate: %w", err)
		}
		stats.ByType[sourceType] = count
	}

	statusRows, err := r.db.Query(ctx, `
		SELECT status, COUNT(*) FROM data_sources
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY status`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("aggregate data sources by status: %w", err)
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan data source status aggregate: %w", err)
		}
		stats.ByStatus[status] = count
	}
	return stats, nil
}

func (r *SourceRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*SourceRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, type, connection_config, encryption_key_id,
		       status, last_error, schema_metadata, schema_discovered_at, last_synced_at,
		       last_sync_status, last_sync_error, last_sync_duration_ms, sync_frequency,
		       next_sync_at, table_count, total_row_count, total_size_bytes, tags, metadata,
		       created_by, created_at, updated_at, deleted_at
		FROM data_sources
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND deleted_at IS NULL
		ORDER BY updated_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active data sources: %w", err)
	}
	defer rows.Close()

	items := make([]*SourceRecord, 0)
	for rows.Next() {
		record, scanErr := scanSourceRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active data sources: %w", err)
	}
	return items, nil
}

func (r *SourceRepository) ListActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT tenant_id
		FROM data_sources
		WHERE status = 'active'
		  AND deleted_at IS NULL
		ORDER BY tenant_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active source tenants: %w", err)
	}
	defer rows.Close()

	tenantIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, fmt.Errorf("scan active source tenant: %w", err)
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active source tenants: %w", err)
	}
	return tenantIDs, nil
}

type SizeEstimatePatch struct {
	TableCount int
	TotalRows  int64
	TotalBytes int64
}

type SyncStatePatch struct {
	Status             model.DataSourceStatus
	LastSyncedAt       *time.Time
	LastSyncStatus     *string
	LastSyncError      *string
	LastSyncDurationMs *int64
	TableCount         *int
	TotalRows          *int64
	TotalBytes         *int64
}

type sourceScanner interface {
	Scan(dest ...any) error
}

func scanSourceRecord(scanner sourceScanner) (*SourceRecord, error) {
	source := &model.DataSource{}
	var schemaBytes []byte
	var metadata []byte
	var tags []string
	var lastError *string
	var lastSyncStatus *string
	var lastSyncError *string

	record := &SourceRecord{Source: source}
	if err := scanner.Scan(
		&source.ID, &source.TenantID, &source.Name, &source.Description, &source.Type, &record.EncryptedConfig, &source.EncryptionKeyID,
		&source.Status, &lastError, &schemaBytes, &source.SchemaDiscoveredAt, &source.LastSyncedAt,
		&lastSyncStatus, &lastSyncError, &source.LastSyncDurationMs, &source.SyncFrequency,
		&source.NextSyncAt, &source.TableCount, &source.TotalRowCount, &source.TotalSizeBytes, &tags, &metadata,
		&source.CreatedBy, &source.CreatedAt, &source.UpdatedAt, &source.DeletedAt,
	); err != nil {
		return nil, err
	}
	source.LastError = lastError
	source.LastSyncStatus = lastSyncStatus
	source.LastSyncError = lastSyncError
	source.Tags = tags
	source.Metadata = metadata
	if len(schemaBytes) > 0 && string(schemaBytes) != "null" {
		var schema model.DiscoveredSchema
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			return nil, fmt.Errorf("decode source schema metadata: %w", err)
		}
		source.SchemaMetadata = &schema
	}
	return record, nil
}

func marshalSchema(schema *model.DiscoveredSchema) []byte {
	if schema == nil {
		return nil
	}
	payload, _ := json.Marshal(schema)
	return payload
}

func coalesce(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
