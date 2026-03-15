package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
)

// ScanRepository handles scan_history table operations.
type ScanRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewScanRepository creates a new ScanRepository.
func NewScanRepository(db *pgxpool.Pool, logger zerolog.Logger) *ScanRepository {
	return &ScanRepository{db: db, logger: logger}
}

// Create inserts a new scan_history row and returns it.
func (r *ScanRepository) Create(ctx context.Context, tenantID, userID uuid.UUID, scanType model.ScanType, cfg *model.ScanConfig) (*model.ScanHistory, error) {
	id := uuid.New()
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO scan_history (
			id, tenant_id, scan_type, config, status,
			assets_discovered, assets_new, assets_updated, error_count,
			started_at, created_by, created_at
		) VALUES ($1,$2,$3,$4,'running',0,0,0,0,now(),$5,now())`,
		id, tenantID, string(scanType), cfgJSON, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create scan: %w", err)
	}
	return r.GetByID(ctx, tenantID, id)
}

// GetByID fetches a scan history record by ID.
func (r *ScanRepository) GetByID(ctx context.Context, tenantID, scanID uuid.UUID) (*model.ScanHistory, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, scan_type, config, status,
		       assets_discovered, assets_new, assets_updated, error_count,
		       errors, started_at, completed_at, duration_ms, created_by, created_at
		FROM scan_history
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, scanID,
	)
	scan, err := scanScanHistory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return scan, err
}

// List returns paginated scan history for a tenant.
func (r *ScanRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.ScanListParams) ([]*model.ScanHistory, int, error) {
	args := []any{tenantID}
	conditions := []string{"tenant_id = $1"}
	idx := 2

	if params.ScanType != nil {
		conditions = append(conditions, fmt.Sprintf("scan_type = $%d", idx))
		args = append(args, *params.ScanType)
		idx++
	}
	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, *params.Status)
		idx++
	}

	whereClause := buildWhere(conditions)

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scan_history "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := params.PerPage
	offset := (params.Page - 1) * params.PerPage
	args = append(args, limit, offset)
	listSQL := fmt.Sprintf(`
		SELECT id, tenant_id, scan_type, config, status,
		       assets_discovered, assets_new, assets_updated, error_count,
		       errors, started_at, completed_at, duration_ms, created_by, created_at
		FROM scan_history %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, idx, idx+1)

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var scans []*model.ScanHistory
	for rows.Next() {
		s, err := scanScanHistory(rows)
		if err != nil {
			return nil, 0, err
		}
		scans = append(scans, s)
	}
	return scans, total, rows.Err()
}

// Complete marks a scan as completed and records final counts/duration.
func (r *ScanRepository) Complete(ctx context.Context, tenantID, scanID uuid.UUID, result *model.ScanResult) error {
	errorsJSON, _ := json.Marshal(result.Errors)
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE scan_history
		SET status = $3, assets_discovered = $4, assets_new = $5, assets_updated = $6,
		    error_count = $7, errors = $8, completed_at = $9, duration_ms = $10
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, scanID,
		string(result.Status),
		result.AssetsDiscovered, result.AssetsNew, result.AssetsUpdated,
		len(result.Errors), errorsJSON, now, result.DurationMs,
	)
	return err
}

// Cancel marks a scan as cancelled.
func (r *ScanRepository) Cancel(ctx context.Context, tenantID, scanID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE scan_history
		SET status = 'cancelled', completed_at = now()
		WHERE tenant_id = $1 AND id = $2 AND status = 'running'`,
		tenantID, scanID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LastScanAt returns the timestamp of the most recent completed scan for a tenant.
func (r *ScanRepository) LastScanAt(ctx context.Context, tenantID uuid.UUID) (*time.Time, error) {
	var t *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT MAX(completed_at) FROM scan_history WHERE tenant_id = $1 AND status = 'completed'`,
		tenantID,
	).Scan(&t)
	return t, err
}

func scanScanHistory(row interface{ Scan(dest ...any) error }) (*model.ScanHistory, error) {
	var s model.ScanHistory
	var scanType, status string
	err := row.Scan(
		&s.ID, &s.TenantID, &scanType, &s.Config, &status,
		&s.AssetsDiscovered, &s.AssetsNew, &s.AssetsUpdated, &s.ErrorCount,
		&s.Errors, &s.StartedAt, &s.CompletedAt, &s.DurationMs,
		&s.CreatedBy, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.ScanType = model.ScanType(scanType)
	s.Status = model.ScanStatus(status)
	if s.Config == nil {
		s.Config = json.RawMessage("{}")
	}
	s.ComputeDerived()
	return &s, nil
}

func buildWhere(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	result := " WHERE "
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}
