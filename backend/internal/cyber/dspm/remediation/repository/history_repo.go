package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// HistoryRepository handles persistence for DSPM remediation audit history entries.
type HistoryRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewHistoryRepository creates a new HistoryRepository.
func NewHistoryRepository(db *pgxpool.Pool, logger zerolog.Logger) *HistoryRepository {
	return &HistoryRepository{db: db, logger: logger}
}

// Insert creates a new history entry and returns it with server-generated fields.
func (r *HistoryRepository) Insert(ctx context.Context, entry *model.RemediationHistory) (*model.RemediationHistory, error) {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if len(entry.Details) == 0 {
		entry.Details = []byte("{}")
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO dspm_remediation_history (
			id, tenant_id, remediation_id, action, actor_id, actor_type,
			details, entry_hash, prev_hash, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, now()
		)
		RETURNING id, tenant_id, remediation_id, action, actor_id, actor_type,
			details, entry_hash, prev_hash, created_at`,
		entry.ID, entry.TenantID, entry.RemediationID, entry.Action,
		entry.ActorID, entry.ActorType,
		entry.Details, entry.EntryHash, entry.PrevHash,
	)

	result, err := scanHistory(row)
	if err != nil {
		return nil, fmt.Errorf("insert remediation history: %w", err)
	}
	return result, nil
}

// ListByRemediation returns paginated history entries for a remediation, ordered chronologically.
func (r *HistoryRepository) ListByRemediation(ctx context.Context, tenantID, remediationID uuid.UUID, page, perPage int) ([]model.RemediationHistory, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 || perPage > 200 {
		perPage = 50
	}

	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM dspm_remediation_history
		WHERE tenant_id = $1 AND remediation_id = $2`,
		tenantID, remediationID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count remediation history: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, remediation_id, action, actor_id, actor_type,
			details, entry_hash, prev_hash, created_at
		FROM dspm_remediation_history
		WHERE tenant_id = $1 AND remediation_id = $2
		ORDER BY created_at ASC
		LIMIT $3 OFFSET $4`,
		tenantID, remediationID, perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list remediation history: %w", err)
	}
	defer rows.Close()

	items := make([]model.RemediationHistory, 0)
	for rows.Next() {
		item, err := scanHistory(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan history row: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate remediation history: %w", err)
	}

	return items, total, nil
}

// GetLastEntry returns the most recent history entry for a remediation, used for hash chain linking.
func (r *HistoryRepository) GetLastEntry(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationHistory, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, remediation_id, action, actor_id, actor_type,
			details, entry_hash, prev_hash, created_at
		FROM dspm_remediation_history
		WHERE tenant_id = $1 AND remediation_id = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, remediationID,
	)

	result, err := scanHistory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("remediation history not found")
		}
		return nil, fmt.Errorf("get last history entry: %w", err)
	}
	return result, nil
}

// scanHistory scans a single history row into a model.RemediationHistory.
func scanHistory(row scanner) (*model.RemediationHistory, error) {
	var item model.RemediationHistory
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.RemediationID,
		&item.Action,
		&item.ActorID,
		&item.ActorType,
		&item.Details,
		&item.EntryHash,
		&item.PrevHash,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	if len(item.Details) == 0 {
		item.Details = []byte("{}")
	}
	return &item, nil
}
