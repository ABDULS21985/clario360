package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
)

// WebhookRepository handles webhook registration CRUD.
type WebhookRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewWebhookRepository creates a new WebhookRepository.
func NewWebhookRepository(db *pgxpool.Pool, logger zerolog.Logger) *WebhookRepository {
	return &WebhookRepository{db: db, logger: logger.With().Str("component", "webhook_repo").Logger()}
}

// Insert creates a new webhook registration.
func (r *WebhookRepository) Insert(ctx context.Context, wh *model.Webhook) (string, error) {
	query := `
		INSERT INTO notification_webhooks (tenant_id, name, url, secret, event_types, active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id string
	err := r.db.QueryRow(ctx, query,
		wh.TenantID, wh.Name, wh.URL, wh.Secret,
		wh.EventTypes, wh.Active, wh.CreatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert webhook: %w", err)
	}
	return id, nil
}

// FindByID returns a webhook by ID, scoped to tenant.
func (r *WebhookRepository) FindByID(ctx context.Context, tenantID, id string) (*model.Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, event_types, active, created_by, created_at, updated_at
		FROM notification_webhooks
		WHERE id = $1 AND tenant_id = $2`

	var wh model.Webhook
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret,
		&wh.EventTypes, &wh.Active, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find webhook: %w", err)
	}
	return &wh, nil
}

// ListByTenant returns all webhooks for a tenant.
func (r *WebhookRepository) ListByTenant(ctx context.Context, tenantID string) ([]model.Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, event_types, active, created_by, created_at, updated_at
		FROM notification_webhooks
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var results []model.Webhook
	for rows.Next() {
		var wh model.Webhook
		if err := rows.Scan(
			&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret,
			&wh.EventTypes, &wh.Active, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		results = append(results, wh)
	}
	return results, rows.Err()
}

// GetActiveForEvent returns active webhooks for a tenant that subscribe to the given notification type.
func (r *WebhookRepository) GetActiveForEvent(ctx context.Context, tenantID string, notifType string) ([]model.Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, event_types, active, created_by, created_at, updated_at
		FROM notification_webhooks
		WHERE tenant_id = $1 AND active = true
		  AND (event_types = '{}' OR $2 = ANY(event_types))`

	rows, err := r.db.Query(ctx, query, tenantID, notifType)
	if err != nil {
		return nil, fmt.Errorf("get active webhooks: %w", err)
	}
	defer rows.Close()

	var results []model.Webhook
	for rows.Next() {
		var wh model.Webhook
		if err := rows.Scan(
			&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret,
			&wh.EventTypes, &wh.Active, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		results = append(results, wh)
	}
	return results, rows.Err()
}

// Update modifies a webhook. Nil fields are not updated.
func (r *WebhookRepository) Update(ctx context.Context, tenantID, id string, name, url, secret *string, eventTypes []string, active *bool) error {
	wh, err := r.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if wh == nil {
		return fmt.Errorf("webhook not found")
	}

	if name != nil {
		wh.Name = *name
	}
	if url != nil {
		wh.URL = *url
	}
	if secret != nil {
		wh.Secret = secret
	}
	if eventTypes != nil {
		wh.EventTypes = eventTypes
	}
	if active != nil {
		wh.Active = *active
	}

	query := `
		UPDATE notification_webhooks
		SET name = $1, url = $2, secret = $3, event_types = $4, active = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8`

	_, err = r.db.Exec(ctx, query,
		wh.Name, wh.URL, wh.Secret, wh.EventTypes, wh.Active,
		time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update webhook: %w", err)
	}
	return nil
}

// Deactivate sets active=false on a webhook (soft delete).
func (r *WebhookRepository) Deactivate(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx,
		"UPDATE notification_webhooks SET active = false, updated_at = $1 WHERE id = $2 AND tenant_id = $3",
		time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("deactivate webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}
