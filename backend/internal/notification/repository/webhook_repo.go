package repository

import (
	"context"
	"encoding/json"
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

const webhookColumns = `id, tenant_id, name, url, secret, event_types, active, headers, retry_policy,
	last_triggered_at, success_count, failure_count, created_by, created_at, updated_at`

func scanWebhook(row pgx.Row) (*model.Webhook, error) {
	var wh model.Webhook
	err := row.Scan(
		&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret,
		&wh.Events, &wh.Active, &wh.HeadersRaw, &wh.RetryPolicyRaw,
		&wh.LastTriggeredAt, &wh.SuccessCount, &wh.FailureCount,
		&wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	wh.ComputeDerived()
	return &wh, nil
}

func scanWebhooks(rows pgx.Rows) ([]model.Webhook, error) {
	var results []model.Webhook
	for rows.Next() {
		var wh model.Webhook
		if err := rows.Scan(
			&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret,
			&wh.Events, &wh.Active, &wh.HeadersRaw, &wh.RetryPolicyRaw,
			&wh.LastTriggeredAt, &wh.SuccessCount, &wh.FailureCount,
			&wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		wh.ComputeDerived()
		results = append(results, wh)
	}
	return results, rows.Err()
}

// Insert creates a new webhook registration.
func (r *WebhookRepository) Insert(ctx context.Context, wh *model.Webhook) (string, error) {
	headersJSON, _ := json.Marshal(wh.Headers)
	if headersJSON == nil {
		headersJSON = []byte("{}")
	}
	rpJSON, _ := json.Marshal(wh.RetryPolicy)
	if rpJSON == nil {
		rpJSON, _ = json.Marshal(model.DefaultRetryPolicy())
	}

	query := `
		INSERT INTO notification_webhooks (tenant_id, name, url, secret, event_types, active, headers, retry_policy, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id string
	err := r.db.QueryRow(ctx, query,
		wh.TenantID, wh.Name, wh.URL, wh.Secret,
		wh.Events, wh.Active, headersJSON, rpJSON, wh.CreatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert webhook: %w", err)
	}
	return id, nil
}

// FindByID returns a webhook by ID, scoped to tenant.
func (r *WebhookRepository) FindByID(ctx context.Context, tenantID, id string) (*model.Webhook, error) {
	query := fmt.Sprintf(`SELECT %s FROM notification_webhooks WHERE id = $1 AND tenant_id = $2`, webhookColumns)

	wh, err := scanWebhook(r.db.QueryRow(ctx, query, id, tenantID))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find webhook: %w", err)
	}
	return wh, nil
}

// ListByTenant returns all webhooks for a tenant.
func (r *WebhookRepository) ListByTenant(ctx context.Context, tenantID string) ([]model.Webhook, error) {
	query := fmt.Sprintf(`SELECT %s FROM notification_webhooks WHERE tenant_id = $1 ORDER BY created_at DESC`, webhookColumns)

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	return scanWebhooks(rows)
}

// ListByTenantPaginated returns webhooks with pagination and total count.
func (r *WebhookRepository) ListByTenantPaginated(ctx context.Context, tenantID string, page, perPage int, search string) ([]model.Webhook, int64, error) {
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM notification_webhooks WHERE tenant_id = $1`
	dataQuery := fmt.Sprintf(`SELECT %s FROM notification_webhooks WHERE tenant_id = $1`, webhookColumns)

	args := []interface{}{tenantID}
	argIdx := 2

	if search != "" {
		filter := fmt.Sprintf(` AND (name ILIKE $%d OR url ILIKE $%d)`, argIdx, argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, "%"+search+"%")
		argIdx++
	}

	dataQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhooks: %w", err)
	}

	dataArgs := append(args, perPage, offset)
	rows, err := r.db.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list webhooks paginated: %w", err)
	}
	defer rows.Close()

	webhooks, err := scanWebhooks(rows)
	if err != nil {
		return nil, 0, err
	}
	return webhooks, total, nil
}

// GetActiveForEvent returns active webhooks for a tenant that subscribe to the given notification type.
func (r *WebhookRepository) GetActiveForEvent(ctx context.Context, tenantID string, notifType string) ([]model.Webhook, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM notification_webhooks
		WHERE tenant_id = $1 AND active = true
		  AND (event_types = '{}' OR $2 = ANY(event_types))`, webhookColumns)

	rows, err := r.db.Query(ctx, query, tenantID, notifType)
	if err != nil {
		return nil, fmt.Errorf("get active webhooks: %w", err)
	}
	defer rows.Close()

	return scanWebhooks(rows)
}

// Update modifies a webhook. Nil fields are not updated.
func (r *WebhookRepository) Update(ctx context.Context, tenantID, id string, name, url, secret *string, events []string, active *bool, headers map[string]string, retryPolicy *model.WebhookRetryPolicy) error {
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
	if events != nil {
		wh.Events = events
	}
	if active != nil {
		wh.Active = *active
	}
	if headers != nil {
		wh.Headers = headers
	}
	if retryPolicy != nil {
		wh.RetryPolicy = *retryPolicy
	}

	headersJSON, _ := json.Marshal(wh.Headers)
	rpJSON, _ := json.Marshal(wh.RetryPolicy)

	query := `
		UPDATE notification_webhooks
		SET name = $1, url = $2, secret = $3, event_types = $4, active = $5,
		    headers = $6, retry_policy = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10`

	_, err = r.db.Exec(ctx, query,
		wh.Name, wh.URL, wh.Secret, wh.Events, wh.Active,
		headersJSON, rpJSON,
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

// IncrementSuccess increments success_count and updates last_triggered_at.
func (r *WebhookRepository) IncrementSuccess(ctx context.Context, tenantID, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notification_webhooks SET success_count = success_count + 1, last_triggered_at = $1, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("increment success: %w", err)
	}
	return nil
}

// IncrementFailure increments failure_count and updates last_triggered_at.
func (r *WebhookRepository) IncrementFailure(ctx context.Context, tenantID, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notification_webhooks SET failure_count = failure_count + 1, last_triggered_at = $1, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("increment failure: %w", err)
	}
	return nil
}

// RotateSecret sets a new secret on a webhook.
func (r *WebhookRepository) RotateSecret(ctx context.Context, tenantID, id, newSecret string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE notification_webhooks SET secret = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
		newSecret, time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("rotate secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}
