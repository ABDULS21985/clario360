package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	intdto "github.com/clario360/platform/internal/integration/dto"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

type TicketLinkRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewTicketLinkRepository(db *pgxpool.Pool, logger zerolog.Logger) *TicketLinkRepository {
	return &TicketLinkRepository{db: db, logger: logger.With().Str("component", "ticket_link_repo").Logger()}
}

func (r *TicketLinkRepository) Create(ctx context.Context, link *intmodel.ExternalTicketLink) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO external_ticket_links (
			tenant_id, integration_id, entity_type, entity_id, external_system, external_id,
			external_key, external_url, external_status, external_priority, sync_direction,
			last_synced_at, last_sync_direction, sync_error
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`,
		link.TenantID, link.IntegrationID, link.EntityType, link.EntityID, link.ExternalSystem,
		link.ExternalID, link.ExternalKey, link.ExternalURL, link.ExternalStatus, link.ExternalPriority,
		string(link.SyncDirection), link.LastSyncedAt, link.LastSyncDirection, link.SyncError,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create ticket link: %w", err)
	}
	return id, nil
}

func (r *TicketLinkRepository) List(ctx context.Context, tenantID string, query *intdto.TicketLinkQuery) ([]intmodel.ExternalTicketLink, error) {
	where := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argIdx := 2
	if query != nil {
		if query.EntityType != "" {
			where = append(where, fmt.Sprintf("entity_type = $%d", argIdx))
			args = append(args, query.EntityType)
			argIdx++
		}
		if query.EntityID != "" {
			where = append(where, fmt.Sprintf("entity_id::text = $%d", argIdx))
			args = append(args, query.EntityID)
			argIdx++
		}
		if query.ExternalSystem != "" {
			where = append(where, fmt.Sprintf("external_system = $%d", argIdx))
			args = append(args, query.ExternalSystem)
			argIdx++
		}
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, integration_id, entity_type, entity_id::text, external_system, external_id,
		       external_key, external_url, external_status, external_priority, sync_direction,
		       last_synced_at, last_sync_direction, sync_error, created_at, updated_at
		FROM external_ticket_links
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY updated_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list ticket links: %w", err)
	}
	defer rows.Close()

	var items []intmodel.ExternalTicketLink
	for rows.Next() {
		item, err := scanTicketLink(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *TicketLinkRepository) GetByID(ctx context.Context, tenantID, id string) (*intmodel.ExternalTicketLink, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, integration_id, entity_type, entity_id::text, external_system, external_id,
		       external_key, external_url, external_status, external_priority, sync_direction,
		       last_synced_at, last_sync_direction, sync_error, created_at, updated_at
		FROM external_ticket_links
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return scanTicketLink(row)
}

func (r *TicketLinkRepository) GetByExternal(ctx context.Context, externalSystem, externalID string) (*intmodel.ExternalTicketLink, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, integration_id, entity_type, entity_id::text, external_system, external_id,
		       external_key, external_url, external_status, external_priority, sync_direction,
		       last_synced_at, last_sync_direction, sync_error, created_at, updated_at
		FROM external_ticket_links
		WHERE external_system = $1 AND external_id = $2`, externalSystem, externalID)
	return scanTicketLink(row)
}

func (r *TicketLinkRepository) UpdateSync(ctx context.Context, link *intmodel.ExternalTicketLink) error {
	_, err := r.db.Exec(ctx, `
		UPDATE external_ticket_links
		SET external_status = $2,
		    external_priority = $3,
		    last_synced_at = $4,
		    last_sync_direction = $5,
		    sync_error = $6,
		    updated_at = now()
		WHERE id = $1`,
		link.ID, link.ExternalStatus, link.ExternalPriority, link.LastSyncedAt, link.LastSyncDirection, link.SyncError,
	)
	if err != nil {
		return fmt.Errorf("update ticket link sync: %w", err)
	}
	return nil
}

func scanTicketLink(row interface {
	Scan(dest ...any) error
}) (*intmodel.ExternalTicketLink, error) {
	var item intmodel.ExternalTicketLink
	var direction string
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.IntegrationID,
		&item.EntityType,
		&item.EntityID,
		&item.ExternalSystem,
		&item.ExternalID,
		&item.ExternalKey,
		&item.ExternalURL,
		&item.ExternalStatus,
		&item.ExternalPriority,
		&direction,
		&item.LastSyncedAt,
		&item.LastSyncDirection,
		&item.SyncError,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("scan ticket link: %w", err)
	}
	item.SyncDirection = intmodel.SyncDirection(direction)
	return &item, nil
}
