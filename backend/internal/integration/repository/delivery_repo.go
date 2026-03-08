package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	intdto "github.com/clario360/platform/internal/integration/dto"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

type DeliveryRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewDeliveryRepository(db *pgxpool.Pool, logger zerolog.Logger) *DeliveryRepository {
	return &DeliveryRepository{db: db, logger: logger.With().Str("component", "integration_delivery_repo").Logger()}
}

func (r *DeliveryRepository) Create(ctx context.Context, record *intmodel.DeliveryRecord) (string, error) {
	query := `
		INSERT INTO integration_deliveries (
			tenant_id, integration_id, event_type, event_id, event_data, status, attempts, max_attempts,
			response_code, response_body, last_error, error_category, next_retry_at, latency_ms,
			delivered_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, COALESCE($16, now())
		)
		RETURNING id`

	var id string
	err := r.db.QueryRow(
		ctx, query,
		record.TenantID, record.IntegrationID, record.EventType, record.EventID, record.EventData,
		string(record.Status), record.Attempts, record.MaxAttempts, record.ResponseCode, record.ResponseBody,
		record.LastError, record.ErrorCategory, record.NextRetryAt, record.LatencyMS, record.DeliveredAt, record.CreatedAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create delivery record: %w", err)
	}
	return id, nil
}

func (r *DeliveryRepository) ListByIntegration(ctx context.Context, tenantID, integrationID string, query *intdto.DeliveryQuery) ([]intmodel.DeliveryRecord, int, error) {
	where := []string{"tenant_id = $1", "integration_id = $2"}
	args := []any{tenantID, integrationID}
	argIdx := 3

	if query.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, query.Status)
		argIdx++
	}
	if query.EventType != "" {
		where = append(where, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, query.EventType)
		argIdx++
	}
	if query.DateFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *query.DateFrom)
		argIdx++
	}
	if query.DateTo != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *query.DateTo)
		argIdx++
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM integration_deliveries"+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deliveries: %w", err)
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, integration_id, event_type, event_id, event_data, status, attempts, max_attempts,
		       response_code, response_body, last_error, error_category, next_retry_at, latency_ms,
		       delivered_at, created_at
		FROM integration_deliveries%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereSQL, argIdx, argIdx+1), append(args, query.PerPage, query.Offset())...)
	if err != nil {
		return nil, 0, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()

	items := make([]intmodel.DeliveryRecord, 0, query.PerPage)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *DeliveryRepository) ListDue(ctx context.Context, limit int) ([]intmodel.DeliveryRecord, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, integration_id, event_type, event_id, event_data, status, attempts, max_attempts,
		       response_code, response_body, last_error, error_category, next_retry_at, latency_ms,
		       delivered_at, created_at
		FROM integration_deliveries
		WHERE status IN ('pending', 'retrying')
		  AND (next_retry_at IS NULL OR next_retry_at <= now())
		ORDER BY COALESCE(next_retry_at, created_at) ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list due deliveries: %w", err)
	}
	defer rows.Close()

	items := make([]intmodel.DeliveryRecord, 0, limit)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *DeliveryRepository) MarkDelivered(ctx context.Context, id string, responseCode int, responseBody string, latencyMS int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integration_deliveries
		SET status = 'delivered',
		    attempts = attempts + 1,
		    response_code = $2,
		    response_body = $3,
		    latency_ms = $4,
		    delivered_at = now(),
		    next_retry_at = NULL,
		    last_error = NULL,
		    error_category = NULL
		WHERE id = $1`, id, responseCode, truncate(responseBody, 1000), latencyMS)
	if err != nil {
		return fmt.Errorf("mark delivery delivered: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) ScheduleRetry(ctx context.Context, id string, nextRetryAt time.Time, lastError, category string, responseCode *int, responseBody *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integration_deliveries
		SET status = 'retrying',
		    attempts = attempts + 1,
		    last_error = $2,
		    error_category = $3,
		    next_retry_at = $4,
		    response_code = $5,
		    response_body = $6
		WHERE id = $1`,
		id, truncate(lastError, 1000), category, nextRetryAt, responseCode, truncatePtr(responseBody, 1000),
	)
	if err != nil {
		return fmt.Errorf("schedule delivery retry: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) MarkFailed(ctx context.Context, id string, lastError, category string, responseCode *int, responseBody *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integration_deliveries
		SET status = 'failed',
		    attempts = attempts + 1,
		    last_error = $2,
		    error_category = $3,
		    next_retry_at = NULL,
		    response_code = $4,
		    response_body = $5
		WHERE id = $1`,
		id, truncate(lastError, 1000), category, responseCode, truncatePtr(responseBody, 1000),
	)
	if err != nil {
		return fmt.Errorf("mark delivery failed: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) RetryFailedByIntegration(ctx context.Context, tenantID, integrationID string) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE integration_deliveries
		SET status = 'retrying',
		    next_retry_at = now(),
		    last_error = NULL
		WHERE tenant_id = $1 AND integration_id = $2 AND status = 'failed'`,
		tenantID, integrationID,
	)
	if err != nil {
		return 0, fmt.Errorf("retry failed deliveries: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *DeliveryRepository) CancelPendingByIntegration(ctx context.Context, tenantID, integrationID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integration_deliveries
		SET status = 'failed',
		    last_error = 'integration disabled'
		WHERE tenant_id = $1 AND integration_id = $2 AND status IN ('pending', 'retrying')`,
		tenantID, integrationID,
	)
	if err != nil {
		return fmt.Errorf("cancel pending deliveries: %w", err)
	}
	return nil
}

func (r *DeliveryRepository) GetByID(ctx context.Context, id string) (*intmodel.DeliveryRecord, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, integration_id, event_type, event_id, event_data, status, attempts, max_attempts,
		       response_code, response_body, last_error, error_category, next_retry_at, latency_ms,
		       delivered_at, created_at
		FROM integration_deliveries
		WHERE id = $1`, id)
	item, err := scanDelivery(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("get delivery by id: %w", err)
	}
	return item, nil
}

func scanDelivery(row interface {
	Scan(dest ...any) error
}) (*intmodel.DeliveryRecord, error) {
	var item intmodel.DeliveryRecord
	var status string
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.IntegrationID,
		&item.EventType,
		&item.EventID,
		&item.EventData,
		&status,
		&item.Attempts,
		&item.MaxAttempts,
		&item.ResponseCode,
		&item.ResponseBody,
		&item.LastError,
		&item.ErrorCategory,
		&item.NextRetryAt,
		&item.LatencyMS,
		&item.DeliveredAt,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	item.Status = intmodel.DeliveryStatus(status)
	return &item, nil
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func truncatePtr(value *string, max int) *string {
	if value == nil {
		return nil
	}
	truncated := truncate(*value, max)
	return &truncated
}
