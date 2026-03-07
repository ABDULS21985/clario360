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

type DarkDataRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewDarkDataRepository(db *pgxpool.Pool, logger zerolog.Logger) *DarkDataRepository {
	return &DarkDataRepository{db: db, logger: logger}
}

func (r *DarkDataRepository) CreateScan(ctx context.Context, scan *model.DarkDataScan) error {
	if scan.ID == uuid.Nil {
		scan.ID = uuid.New()
	}
	if scan.CreatedAt.IsZero() {
		scan.CreatedAt = time.Now().UTC()
	}
	if scan.StartedAt.IsZero() {
		scan.StartedAt = scan.CreatedAt
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO dark_data_scans (
			id, tenant_id, status, sources_scanned, storage_scanned, assets_discovered, by_reason, by_type,
			pii_assets_found, high_risk_found, total_size_bytes, started_at, completed_at, duration_ms, triggered_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16
		)`,
		scan.ID, scan.TenantID, scan.Status, scan.SourcesScanned, scan.StorageScanned, scan.AssetsDiscovered, scan.ByReason, scan.ByType,
		scan.PIIAssetsFound, scan.HighRiskFound, scan.TotalSizeBytes, scan.StartedAt, scan.CompletedAt, scan.DurationMs, scan.TriggeredBy, scan.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert dark data scan: %w", err)
	}
	return nil
}

func (r *DarkDataRepository) UpdateScan(ctx context.Context, scan *model.DarkDataScan) error {
	result, err := r.db.Exec(ctx, `
		UPDATE dark_data_scans
		SET status = $3,
		    sources_scanned = $4,
		    storage_scanned = $5,
		    assets_discovered = $6,
		    by_reason = $7,
		    by_type = $8,
		    pii_assets_found = $9,
		    high_risk_found = $10,
		    total_size_bytes = $11,
		    completed_at = $12,
		    duration_ms = $13
		WHERE tenant_id = $1 AND id = $2`,
		scan.TenantID, scan.ID, scan.Status, scan.SourcesScanned, scan.StorageScanned, scan.AssetsDiscovered, scan.ByReason, scan.ByType,
		scan.PIIAssetsFound, scan.HighRiskFound, scan.TotalSizeBytes, scan.CompletedAt, scan.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("update dark data scan: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *DarkDataRepository) GetScan(ctx context.Context, tenantID, id uuid.UUID) (*model.DarkDataScan, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, status, sources_scanned, storage_scanned, assets_discovered, by_reason, by_type,
		       pii_assets_found, high_risk_found, total_size_bytes, started_at, completed_at, duration_ms, triggered_by, created_at
		FROM dark_data_scans
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanDarkDataScan(row)
}

func (r *DarkDataRepository) ListScans(ctx context.Context, tenantID uuid.UUID, params dto.ListDarkDataScansParams) ([]*model.DarkDataScan, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.status, a.sources_scanned, a.storage_scanned, a.assets_discovered, a.by_reason, a.by_type,
		       a.pii_assets_found, a.high_risk_found, a.total_size_bytes, a.started_at, a.completed_at, a.duration_ms, a.triggered_by, a.created_at
		FROM dark_data_scans a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(params.Status != "", "a.status = ?", params.Status)
	qb.OrderBy("created_at", "desc", []string{"created_at", "started_at"})
	qb.Paginate(params.Page, params.PerPage)
	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list dark data scans: %w", err)
	}
	defer rows.Close()

	items := make([]*model.DarkDataScan, 0)
	for rows.Next() {
		item, err := scanDarkDataScan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate dark data scans: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count dark data scans: %w", err)
	}
	return items, total, nil
}

func (r *DarkDataRepository) UpsertAsset(ctx context.Context, asset *model.DarkDataAsset) error {
	if asset.ID == uuid.Nil {
		existingID, err := r.findAssetID(ctx, asset)
		if err != nil && !errorsIsNoRows(err) {
			return err
		}
		if existingID != nil {
			asset.ID = *existingID
		} else {
			asset.ID = uuid.New()
		}
	}
	now := time.Now().UTC()
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = now
	}
	asset.UpdatedAt = now
	if asset.DiscoveredAt.IsZero() {
		asset.DiscoveredAt = now
	}
	if asset.Metadata == nil {
		asset.Metadata = json.RawMessage(`{}`)
	}
	if len(asset.RiskFactors) == 0 {
		asset.RiskFactors = []model.RiskFactor{}
	}

	existingID, err := r.findAssetID(ctx, asset)
	if err != nil && !errorsIsNoRows(err) {
		return err
	}
	if existingID == nil {
		_, err = r.db.Exec(ctx, `
			INSERT INTO dark_data_assets (
				id, tenant_id, scan_id, name, asset_type, source_id, source_name, schema_name, table_name, file_path, reason,
				estimated_row_count, estimated_size_bytes, column_count, contains_pii, pii_types, inferred_classification,
				last_accessed_at, last_modified_at, days_since_access, risk_score, risk_factors, governance_status, governance_notes,
				reviewed_by, reviewed_at, linked_model_id, metadata, discovered_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16, $17,
				$18, $19, $20, $21, $22, $23, $24,
				$25, $26, $27, $28, $29, $30, $31
			)`,
			asset.ID, asset.TenantID, asset.ScanID, asset.Name, asset.AssetType, asset.SourceID, asset.SourceName, asset.SchemaName, asset.TableName, asset.FilePath, asset.Reason,
			asset.EstimatedRowCount, asset.EstimatedSizeBytes, asset.ColumnCount, asset.ContainsPII, ensureStringSlice(asset.PIITypes), asset.InferredClassification,
			asset.LastAccessedAt, asset.LastModifiedAt, asset.DaysSinceAccess, asset.RiskScore, marshalJSONValue(asset.RiskFactors), asset.GovernanceStatus, asset.GovernanceNotes,
			asset.ReviewedBy, asset.ReviewedAt, asset.LinkedModelID, asset.Metadata, asset.DiscoveredAt, asset.CreatedAt, asset.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert dark data asset: %w", err)
		}
		return nil
	}

	asset.ID = *existingID
	result, err := r.db.Exec(ctx, `
		UPDATE dark_data_assets
		SET scan_id = $3,
		    name = $4,
		    source_name = $5,
		    estimated_row_count = $6,
		    estimated_size_bytes = $7,
		    column_count = $8,
		    contains_pii = $9,
		    pii_types = $10,
		    inferred_classification = $11,
		    last_accessed_at = $12,
		    last_modified_at = $13,
		    days_since_access = $14,
		    risk_score = $15,
		    risk_factors = $16,
		    metadata = $17,
		    discovered_at = $18,
		    updated_at = $19
		WHERE tenant_id = $1 AND id = $2`,
		asset.TenantID, asset.ID, asset.ScanID, asset.Name, asset.SourceName, asset.EstimatedRowCount, asset.EstimatedSizeBytes, asset.ColumnCount,
		asset.ContainsPII, ensureStringSlice(asset.PIITypes), asset.InferredClassification, asset.LastAccessedAt, asset.LastModifiedAt, asset.DaysSinceAccess,
		asset.RiskScore, marshalJSONValue(asset.RiskFactors), asset.Metadata, asset.DiscoveredAt, asset.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update dark data asset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *DarkDataRepository) GetAsset(ctx context.Context, tenantID, id uuid.UUID) (*model.DarkDataAsset, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, scan_id, name, asset_type, source_id, source_name, schema_name, table_name, file_path, reason,
		       estimated_row_count, estimated_size_bytes, column_count, contains_pii, pii_types, inferred_classification,
		       last_accessed_at, last_modified_at, days_since_access, risk_score, risk_factors, governance_status, governance_notes,
		       reviewed_by, reviewed_at, linked_model_id, metadata, discovered_at, created_at, updated_at
		FROM dark_data_assets
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanDarkDataAsset(row)
}

func (r *DarkDataRepository) ListAssets(ctx context.Context, tenantID uuid.UUID, params dto.ListDarkDataParams) ([]*model.DarkDataAsset, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.scan_id, a.name, a.asset_type, a.source_id, a.source_name, a.schema_name, a.table_name, a.file_path, a.reason,
		       a.estimated_row_count, a.estimated_size_bytes, a.column_count, a.contains_pii, a.pii_types, a.inferred_classification,
		       a.last_accessed_at, a.last_modified_at, a.days_since_access, a.risk_score, a.risk_factors, a.governance_status, a.governance_notes,
		       a.reviewed_by, a.reviewed_at, a.linked_model_id, a.metadata, a.discovered_at, a.created_at, a.updated_at
		FROM dark_data_assets a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "(a.name ILIKE ? OR COALESCE(a.table_name, '') ILIKE ? OR COALESCE(a.file_path, '') ILIKE ?)", "%"+strings.TrimSpace(params.Search)+"%", "%"+strings.TrimSpace(params.Search)+"%", "%"+strings.TrimSpace(params.Search)+"%")
	qb.WhereIf(params.Reason != "", "a.reason = ?", params.Reason)
	qb.WhereIf(params.AssetType != "", "a.asset_type = ?", params.AssetType)
	qb.WhereIf(params.GovernanceStatus != "", "a.governance_status = ?", params.GovernanceStatus)
	if params.ContainsPII != nil {
		qb.Where("a.contains_pii = ?", *params.ContainsPII)
	}
	if params.MinRiskScore != nil {
		qb.Where("a.risk_score >= ?", *params.MinRiskScore)
	}
	qb.OrderBy(coalesce(params.Sort, "risk_score"), coalesce(params.Order, "desc"), []string{"name", "asset_type", "reason", "risk_score", "created_at", "updated_at", "discovered_at"})
	qb.Paginate(params.Page, params.PerPage)
	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list dark data assets: %w", err)
	}
	defer rows.Close()

	items := make([]*model.DarkDataAsset, 0)
	for rows.Next() {
		item, err := scanDarkDataAsset(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate dark data assets: %w", err)
	}
	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count dark data assets: %w", err)
	}
	return items, total, nil
}

func (r *DarkDataRepository) UpdateGovernance(ctx context.Context, tenantID, id uuid.UUID, status model.DarkDataGovernanceStatus, notes *string, reviewedBy *uuid.UUID, linkedModelID *uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE dark_data_assets
		SET governance_status = $3,
		    governance_notes = $4,
		    reviewed_by = $5::uuid,
		    reviewed_at = CASE WHEN $5::uuid IS NOT NULL THEN now() ELSE reviewed_at END,
		    linked_model_id = COALESCE($6::uuid, linked_model_id),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status, notes, reviewedBy, linkedModelID,
	)
	if err != nil {
		return fmt.Errorf("update dark data governance: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *DarkDataRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.DarkDataStatsSummary, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE contains_pii = true),
		       COUNT(*) FILTER (WHERE risk_score >= 70),
		       COALESCE(SUM(estimated_size_bytes), 0),
		       COALESCE(AVG(risk_score), 0),
		       COUNT(*) FILTER (WHERE governance_status = 'governed'),
		       COUNT(*) FILTER (WHERE governance_status = 'scheduled_deletion')
		FROM dark_data_assets
		WHERE tenant_id = $1`,
		tenantID,
	)
	stats := &model.DarkDataStatsSummary{
		ByReason:           map[string]int{},
		ByType:             map[string]int{},
		ByGovernanceStatus: map[string]int{},
	}
	if err := row.Scan(
		&stats.TotalAssets,
		&stats.PIIAssets,
		&stats.HighRiskAssets,
		&stats.TotalSizeBytes,
		&stats.AverageRiskScore,
		&stats.GovernedAssets,
		&stats.ScheduledDeletionCount,
	); err != nil {
		return nil, fmt.Errorf("query dark data stats: %w", err)
	}
	for _, field := range []struct {
		query string
		dest  map[string]int
	}{
		{`SELECT reason, COUNT(*) FROM dark_data_assets WHERE tenant_id = $1 GROUP BY reason`, stats.ByReason},
		{`SELECT asset_type, COUNT(*) FROM dark_data_assets WHERE tenant_id = $1 GROUP BY asset_type`, stats.ByType},
		{`SELECT governance_status, COUNT(*) FROM dark_data_assets WHERE tenant_id = $1 GROUP BY governance_status`, stats.ByGovernanceStatus},
	} {
		rows, err := r.db.Query(ctx, field.query, tenantID)
		if err != nil {
			return nil, fmt.Errorf("query dark data stats breakdown: %w", err)
		}
		for rows.Next() {
			var key string
			var count int
			if err := rows.Scan(&key, &count); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan dark data stats breakdown: %w", err)
			}
			field.dest[key] = count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate dark data stats breakdown: %w", err)
		}
		rows.Close()
	}
	return stats, nil
}

func (r *DarkDataRepository) findAssetID(ctx context.Context, asset *model.DarkDataAsset) (*uuid.UUID, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id
		FROM dark_data_assets
		WHERE tenant_id = $1
		  AND asset_type = $2
		  AND COALESCE(source_id, '00000000-0000-0000-0000-000000000000'::uuid) = COALESCE($3, '00000000-0000-0000-0000-000000000000'::uuid)
		  AND COALESCE(schema_name, '') = COALESCE($4, '')
		  AND COALESCE(table_name, '') = COALESCE($5, '')
		  AND COALESCE(file_path, '') = COALESCE($6, '')
		  AND reason = $7`,
		asset.TenantID, asset.AssetType, asset.SourceID, asset.SchemaName, asset.TableName, asset.FilePath, asset.Reason,
	)
	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	return &id, nil
}

type darkDataAssetScanner interface {
	Scan(dest ...any) error
}

func scanDarkDataAsset(scanner darkDataAssetScanner) (*model.DarkDataAsset, error) {
	item := &model.DarkDataAsset{}
	var piiTypes []string
	var riskJSON []byte
	var metadata []byte
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.ScanID, &item.Name, &item.AssetType, &item.SourceID, &item.SourceName, &item.SchemaName, &item.TableName, &item.FilePath, &item.Reason,
		&item.EstimatedRowCount, &item.EstimatedSizeBytes, &item.ColumnCount, &item.ContainsPII, &piiTypes, &item.InferredClassification,
		&item.LastAccessedAt, &item.LastModifiedAt, &item.DaysSinceAccess, &item.RiskScore, &riskJSON, &item.GovernanceStatus, &item.GovernanceNotes,
		&item.ReviewedBy, &item.ReviewedAt, &item.LinkedModelID, &metadata, &item.DiscoveredAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.PIITypes = piiTypes
	item.Metadata = metadata
	if len(riskJSON) > 0 {
		if err := json.Unmarshal(riskJSON, &item.RiskFactors); err != nil {
			return nil, fmt.Errorf("decode dark data risk factors: %w", err)
		}
	}
	return item, nil
}

type darkDataScanScanner interface {
	Scan(dest ...any) error
}

func scanDarkDataScan(scanner darkDataScanScanner) (*model.DarkDataScan, error) {
	item := &model.DarkDataScan{}
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.Status, &item.SourcesScanned, &item.StorageScanned, &item.AssetsDiscovered, &item.ByReason, &item.ByType,
		&item.PIIAssetsFound, &item.HighRiskFound, &item.TotalSizeBytes, &item.StartedAt, &item.CompletedAt, &item.DurationMs, &item.TriggeredBy, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}

func errorsIsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
