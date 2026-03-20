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

type ContradictionRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewContradictionRepository(db *pgxpool.Pool, logger zerolog.Logger) *ContradictionRepository {
	return &ContradictionRepository{db: db, logger: logger}
}

func (r *ContradictionRepository) Create(ctx context.Context, item *model.Contradiction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO contradictions (
			id, tenant_id, scan_id, type, severity, confidence_score, title, description, source_a, source_b,
			entity_key_column, entity_key_value, affected_records, sample_records, resolution_guidance,
			authoritative_source, status, resolved_by, resolved_at, resolution_notes, resolution_action,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21,
			$22, $23, $24
		)`,
		item.ID, item.TenantID, item.ScanID, item.Type, item.Severity, item.ConfidenceScore, item.Title, item.Description,
		marshalJSONValue(item.SourceA), marshalJSONValue(item.SourceB), item.EntityKeyColumn, item.EntityKeyValue, item.AffectedRecords,
		item.SampleRecords, item.ResolutionGuidance, item.AuthoritativeSource, item.Status, item.ResolvedBy, item.ResolvedAt,
		item.ResolutionNotes, item.ResolutionAction, item.Metadata, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert contradiction: %w", err)
	}
	return nil
}

func (r *ContradictionRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.Contradiction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, scan_id, type, severity, confidence_score, title, description, source_a, source_b,
		       entity_key_column, entity_key_value, affected_records, sample_records, resolution_guidance,
		       authoritative_source, status, resolved_by, resolved_at, resolution_notes, resolution_action,
		       metadata, created_at, updated_at
		FROM contradictions
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanContradiction(row)
}

func (r *ContradictionRepository) List(ctx context.Context, tenantID uuid.UUID, params dto.ListContradictionsParams) ([]*model.Contradiction, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.scan_id, a.type, a.severity, a.confidence_score, a.title, a.description, a.source_a, a.source_b,
		       a.entity_key_column, a.entity_key_value, a.affected_records, a.sample_records, a.resolution_guidance,
		       a.authoritative_source, a.status, a.resolved_by, a.resolved_at, a.resolution_notes, a.resolution_action,
		       a.metadata, a.created_at, a.updated_at
		FROM contradictions a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(params.Type != "", "a.type = ?", params.Type)
	qb.WhereIf(params.Severity != "", "a.severity = ?", params.Severity)
	qb.WhereIf(params.Status != "", "a.status = ?", params.Status)
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "(a.title ILIKE ? OR a.description ILIKE ?)", "%"+strings.TrimSpace(params.Search)+"%", "%"+strings.TrimSpace(params.Search)+"%")
	qb.OrderBy(coalesce(params.Sort, "created_at"), coalesce(params.Order, "desc"), []string{"created_at", "updated_at", "severity", "type"})
	qb.Paginate(params.Page, params.PerPage)

	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list contradictions: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Contradiction, 0)
	for rows.Next() {
		item, err := scanContradiction(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate contradictions: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contradictions: %w", err)
	}
	return items, total, nil
}

func (r *ContradictionRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.ContradictionStatus) error {
	result, err := r.db.Exec(ctx, `
		UPDATE contradictions SET status = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status,
	)
	if err != nil {
		return fmt.Errorf("update contradiction status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContradictionRepository) Resolve(ctx context.Context, tenantID, id, userID uuid.UUID, action model.ContradictionResolutionAction, notes string, status model.ContradictionStatus) error {
	now := time.Now().UTC()
	result, err := r.db.Exec(ctx, `
		UPDATE contradictions
		SET status = $3,
		    resolved_by = $4,
		    resolved_at = $5,
		    resolution_notes = $6,
		    resolution_action = $7,
		    updated_at = $5
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status, userID, now, notes, action,
	)
	if err != nil {
		return fmt.Errorf("resolve contradiction: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContradictionRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ContradictionStats, error) {
	stats := &model.ContradictionStats{
		ByStatus:  map[string]int{},
		ByType:    map[string]int{},
		BySeverity: map[string]int{},
		UpdatedAt: time.Now().UTC(),
	}
	rows, err := r.db.Query(ctx, `
		SELECT status, type, severity, COUNT(*)
		FROM contradictions
		WHERE tenant_id = $1
		GROUP BY status, type, severity`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("query contradiction stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status, typ, severity string
		var count int
		if err := rows.Scan(&status, &typ, &severity, &count); err != nil {
			return nil, fmt.Errorf("scan contradiction stats: %w", err)
		}
		stats.Total += count
		stats.ByStatus[status] += count
		stats.ByType[typ] += count
		stats.BySeverity[severity] += count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contradiction stats: %w", err)
	}
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(confidence_score), 0)
		FROM contradictions
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&stats.AverageConfidence); err != nil {
		return nil, fmt.Errorf("query contradiction average confidence: %w", err)
	}
	stats.OpenContradictions = stats.ByStatus[string(model.ContradictionStatusDetected)] + stats.ByStatus[string(model.ContradictionStatusInvestigating)]
	return stats, nil
}

func (r *ContradictionRepository) CreateScan(ctx context.Context, item *model.ContradictionScan) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO contradiction_scans (
			id, tenant_id, status, models_scanned, model_pairs_compared, contradictions_found,
			by_type, by_severity, started_at, completed_at, duration_ms, triggered_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13
		)`,
		item.ID, item.TenantID, item.Status, item.ModelsScanned, item.ModelPairsCompared, item.ContradictionsFound,
		item.ByType, item.BySeverity, item.StartedAt, item.CompletedAt, item.DurationMs, item.TriggeredBy, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert contradiction scan: %w", err)
	}
	return nil
}

func (r *ContradictionRepository) UpdateScan(ctx context.Context, item *model.ContradictionScan) error {
	result, err := r.db.Exec(ctx, `
		UPDATE contradiction_scans
		SET status = $3,
		    models_scanned = $4,
		    model_pairs_compared = $5,
		    contradictions_found = $6,
		    by_type = $7,
		    by_severity = $8,
		    started_at = $9,
		    completed_at = $10,
		    duration_ms = $11
		WHERE tenant_id = $1 AND id = $2`,
		item.TenantID, item.ID, item.Status, item.ModelsScanned, item.ModelPairsCompared, item.ContradictionsFound,
		item.ByType, item.BySeverity, item.StartedAt, item.CompletedAt, item.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("update contradiction scan: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContradictionRepository) GetScan(ctx context.Context, tenantID, id uuid.UUID) (*model.ContradictionScan, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, status, models_scanned, model_pairs_compared, contradictions_found,
		       by_type, by_severity, started_at, completed_at, duration_ms, triggered_by, created_at
		FROM contradiction_scans
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanContradictionScan(row)
}

func (r *ContradictionRepository) ListScans(ctx context.Context, tenantID uuid.UUID, params dto.ListContradictionScansParams) ([]*model.ContradictionScan, int, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 20
	}
	query := `
		SELECT id, tenant_id, status, models_scanned, model_pairs_compared, contradictions_found,
		       by_type, by_severity, started_at, completed_at, duration_ms, triggered_by, created_at
		FROM contradiction_scans
		WHERE tenant_id = $1
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(ctx, query, tenantID, params.Status, params.PerPage, (params.Page-1)*params.PerPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list contradiction scans: %w", err)
	}
	defer rows.Close()

	items := make([]*model.ContradictionScan, 0)
	for rows.Next() {
		item, err := scanContradictionScan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate contradiction scans: %w", err)
	}

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contradiction_scans
		WHERE tenant_id = $1
		  AND ($2 = '' OR status = $2)`,
		tenantID, params.Status,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contradiction scans: %w", err)
	}
	return items, total, nil
}

func scanContradiction(scanner interface{ Scan(dest ...any) error }) (*model.Contradiction, error) {
	item := &model.Contradiction{}
	var sourceAJSON []byte
	var sourceBJSON []byte
	var action *string
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.ScanID, &item.Type, &item.Severity, &item.ConfidenceScore, &item.Title, &item.Description, &sourceAJSON, &sourceBJSON,
		&item.EntityKeyColumn, &item.EntityKeyValue, &item.AffectedRecords, &item.SampleRecords, &item.ResolutionGuidance,
		&item.AuthoritativeSource, &item.Status, &item.ResolvedBy, &item.ResolvedAt, &item.ResolutionNotes, &action,
		&item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(sourceAJSON) > 0 && string(sourceAJSON) != "null" {
		if err := json.Unmarshal(sourceAJSON, &item.SourceA); err != nil {
			return nil, fmt.Errorf("decode contradiction source_a: %w", err)
		}
	}
	if len(sourceBJSON) > 0 && string(sourceBJSON) != "null" {
		if err := json.Unmarshal(sourceBJSON, &item.SourceB); err != nil {
			return nil, fmt.Errorf("decode contradiction source_b: %w", err)
		}
	}
	if action != nil {
		value := model.ContradictionResolutionAction(*action)
		item.ResolutionAction = &value
	}
	return item, nil
}

func scanContradictionScan(scanner interface{ Scan(dest ...any) error }) (*model.ContradictionScan, error) {
	item := &model.ContradictionScan{}
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.Status, &item.ModelsScanned, &item.ModelPairsCompared, &item.ContradictionsFound,
		&item.ByType, &item.BySeverity, &item.StartedAt, &item.CompletedAt, &item.DurationMs, &item.TriggeredBy, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}

