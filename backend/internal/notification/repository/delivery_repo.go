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

// DeliveryRepository handles delivery log operations.
type DeliveryRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewDeliveryRepository creates a new DeliveryRepository.
func NewDeliveryRepository(db *pgxpool.Pool, logger zerolog.Logger) *DeliveryRepository {
	return &DeliveryRepository{db: db, logger: logger.With().Str("component", "delivery_repo").Logger()}
}

// Insert creates a delivery log record.
func (r *DeliveryRepository) Insert(ctx context.Context, rec *model.DeliveryRecord) (string, error) {
	metaBytes, err := json.Marshal(rec.Metadata)
	if err != nil {
		metaBytes = []byte("{}")
	}

	query := `
		INSERT INTO notification_delivery_log (notification_id, channel, status, attempt, error_message, metadata, delivered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id string
	err = r.db.QueryRow(ctx, query,
		rec.NotificationID, rec.Channel, rec.Status, rec.Attempt,
		rec.ErrorMessage, metaBytes, rec.DeliveredAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert delivery log: %w", err)
	}
	return id, nil
}

// UpdateStatus updates the status of a delivery record.
func (r *DeliveryRepository) UpdateStatus(ctx context.Context, id, status string, errMsg *string, deliveredAt *time.Time) error {
	query := `
		UPDATE notification_delivery_log
		SET status = $1, error_message = $2, delivered_at = $3
		WHERE id = $4`

	_, err := r.db.Exec(ctx, query, status, errMsg, deliveredAt, id)
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}
	return nil
}

// IncrementAttempt increments the attempt count and updates status/error.
func (r *DeliveryRepository) IncrementAttempt(ctx context.Context, id string, status string, errMsg *string) error {
	query := `
		UPDATE notification_delivery_log
		SET attempt = attempt + 1, status = $1, error_message = $2
		WHERE id = $3`

	_, err := r.db.Exec(ctx, query, status, errMsg, id)
	if err != nil {
		return fmt.Errorf("increment attempt: %w", err)
	}
	return nil
}

// GetFailedRecent returns failed delivery records from the last 24 hours.
func (r *DeliveryRepository) GetFailedRecent(ctx context.Context) ([]model.DeliveryRecord, error) {
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	query := `
		SELECT id, notification_id, channel, status, attempt, error_message, metadata, delivered_at, created_at
		FROM notification_delivery_log
		WHERE status = 'failed' AND created_at >= $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("get failed deliveries: %w", err)
	}
	defer rows.Close()

	var results []model.DeliveryRecord
	for rows.Next() {
		var rec model.DeliveryRecord
		if err := rows.Scan(
			&rec.ID, &rec.NotificationID, &rec.Channel, &rec.Status,
			&rec.Attempt, &rec.ErrorMessage, &rec.Metadata, &rec.DeliveredAt, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		results = append(results, rec)
	}
	return results, rows.Err()
}

// GetFailedRecentFiltered returns failed deliveries with optional channel filter.
func (r *DeliveryRepository) GetFailedRecentFiltered(ctx context.Context, channel string, since time.Time) ([]model.DeliveryRecord, error) {
	query := `
		SELECT id, notification_id, channel, status, attempt, error_message, metadata, delivered_at, created_at
		FROM notification_delivery_log
		WHERE status = 'failed' AND created_at >= $1`
	args := []interface{}{since}

	if channel != "" {
		query += ` AND channel = $2`
		args = append(args, channel)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get failed deliveries filtered: %w", err)
	}
	defer rows.Close()

	var results []model.DeliveryRecord
	for rows.Next() {
		var rec model.DeliveryRecord
		if err := rows.Scan(
			&rec.ID, &rec.NotificationID, &rec.Channel, &rec.Status,
			&rec.Attempt, &rec.ErrorMessage, &rec.Metadata, &rec.DeliveredAt, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		results = append(results, rec)
	}
	return results, rows.Err()
}

// GetDeliveryStats returns aggregated delivery statistics.
func (r *DeliveryRepository) GetDeliveryStats(ctx context.Context, tenantID string, since time.Time) ([]model.DeliveryStats, error) {
	query := `
		SELECT dl.channel, dl.status, COUNT(*) as count
		FROM notification_delivery_log dl
		JOIN notifications n ON n.id = dl.notification_id
		WHERE n.tenant_id = $1 AND dl.created_at >= $2
		GROUP BY dl.channel, dl.status
		ORDER BY dl.channel, dl.status`

	rows, err := r.db.Query(ctx, query, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("get delivery stats: %w", err)
	}
	defer rows.Close()

	var stats []model.DeliveryStats
	for rows.Next() {
		var s model.DeliveryStats
		if err := rows.Scan(&s.Channel, &s.Status, &s.Count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetRichDeliveryStats computes the full frontend-compatible stats response.
func (r *DeliveryRepository) GetRichDeliveryStats(ctx context.Context, tenantID string, since time.Time, period string, channel string) (*model.RichDeliveryStats, error) {
	result := &model.RichDeliveryStats{
		Period:    period,
		ByChannel: make(map[string]model.ChannelStats),
		ByType:    make(map[string]int64),
		ByDay:     []model.DayStats{},
	}

	// 1) Per-channel aggregates
	channelQuery := `
		SELECT dl.channel, dl.status, COUNT(*) as count
		FROM notification_delivery_log dl
		JOIN notifications n ON n.id = dl.notification_id
		WHERE n.tenant_id = $1 AND dl.created_at >= $2`
	channelArgs := []interface{}{tenantID, since}
	argIdx := 3
	if channel != "" {
		channelQuery += fmt.Sprintf(` AND dl.channel = $%d`, argIdx)
		channelArgs = append(channelArgs, channel)
	}
	channelQuery += ` GROUP BY dl.channel, dl.status ORDER BY dl.channel, dl.status`

	rows, err := r.db.Query(ctx, channelQuery, channelArgs...)
	if err != nil {
		return nil, fmt.Errorf("get channel stats: %w", err)
	}
	for rows.Next() {
		var ch, status string
		var count int64
		if err := rows.Scan(&ch, &status, &count); err != nil {
			rows.Close()
			return nil, err
		}
		cs := result.ByChannel[ch]
		cs.Sent += count
		if status == model.DeliveryDelivered {
			cs.Delivered += count
			result.Delivered += count
		} else if status == model.DeliveryFailed {
			cs.Failed += count
			result.Failed += count
		}
		result.TotalSent += count
		result.ByChannel[ch] = cs
	}
	rows.Close()

	if result.TotalSent > 0 {
		result.DeliveryRate = float64(result.Delivered) / float64(result.TotalSent)
	}

	// 2) By notification type
	typeQuery := `
		SELECT n.type, COUNT(*) as count
		FROM notification_delivery_log dl
		JOIN notifications n ON n.id = dl.notification_id
		WHERE n.tenant_id = $1 AND dl.created_at >= $2`
	typeArgs := []interface{}{tenantID, since}
	if channel != "" {
		typeQuery += ` AND dl.channel = $3`
		typeArgs = append(typeArgs, channel)
	}
	typeQuery += ` GROUP BY n.type ORDER BY count DESC`

	rows, err = r.db.Query(ctx, typeQuery, typeArgs...)
	if err != nil {
		return nil, fmt.Errorf("get type stats: %w", err)
	}
	for rows.Next() {
		var ntype string
		var count int64
		if err := rows.Scan(&ntype, &count); err != nil {
			rows.Close()
			return nil, err
		}
		result.ByType[ntype] = count
	}
	rows.Close()

	// 3) By day
	dayQuery := `
		SELECT dl.created_at::date as day,
		       COUNT(*) as sent,
		       COUNT(*) FILTER (WHERE dl.status = 'delivered') as delivered,
		       COUNT(*) FILTER (WHERE dl.status = 'failed') as failed
		FROM notification_delivery_log dl
		JOIN notifications n ON n.id = dl.notification_id
		WHERE n.tenant_id = $1 AND dl.created_at >= $2`
	dayArgs := []interface{}{tenantID, since}
	if channel != "" {
		dayQuery += ` AND dl.channel = $3`
		dayArgs = append(dayArgs, channel)
	}
	dayQuery += ` GROUP BY day ORDER BY day ASC`

	rows, err = r.db.Query(ctx, dayQuery, dayArgs...)
	if err != nil {
		return nil, fmt.Errorf("get day stats: %w", err)
	}
	for rows.Next() {
		var ds model.DayStats
		var day time.Time
		if err := rows.Scan(&day, &ds.Sent, &ds.Delivered, &ds.Failed); err != nil {
			rows.Close()
			return nil, err
		}
		ds.Date = day.Format("2006-01-02")
		result.ByDay = append(result.ByDay, ds)
	}
	rows.Close()

	// 4) Average delivery time (from delivery records that have duration_ms)
	avgQuery := `
		SELECT COALESCE(AVG(dl.duration_ms), 0)::bigint
		FROM notification_delivery_log dl
		JOIN notifications n ON n.id = dl.notification_id
		WHERE n.tenant_id = $1 AND dl.created_at >= $2 AND dl.duration_ms IS NOT NULL`
	avgArgs := []interface{}{tenantID, since}
	if channel != "" {
		avgQuery += ` AND dl.channel = $3`
		avgArgs = append(avgArgs, channel)
	}
	_ = r.db.QueryRow(ctx, avgQuery, avgArgs...).Scan(&result.AvgDeliveryTimeMS)

	return result, nil
}

// GetWebhookDeliveries returns paginated delivery records for a specific webhook.
func (r *DeliveryRepository) GetWebhookDeliveries(ctx context.Context, webhookID string, page, perPage int, status string) ([]model.WebhookDelivery, int64, error) {
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM notification_delivery_log WHERE webhook_id = $1`
	dataQuery := `SELECT id, webhook_id, COALESCE(event_type,'') as event_type, status,
		COALESCE(request_url,'') as request_url, COALESCE(request_body,'{}')::jsonb as request_body,
		response_status, response_body, duration_ms, attempt, next_retry_at, created_at
		FROM notification_delivery_log WHERE webhook_id = $1`

	args := []interface{}{webhookID}
	argIdx := 2

	if status != "" {
		filter := fmt.Sprintf(` AND status = $%d`, argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, status)
		argIdx++
	}

	dataQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}

	dataArgs := append(args, perPage, offset)
	rows, err := r.db.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()

	var results []model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := rows.Scan(
			&d.ID, &d.WebhookID, &d.EventType, &d.Status,
			&d.RequestURL, &d.RequestBody, &d.ResponseStatus, &d.ResponseBody,
			&d.DurationMS, &d.AttemptCount, &d.NextRetryAt, &d.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan webhook delivery: %w", err)
		}
		results = append(results, d)
	}
	return results, total, rows.Err()
}

// GetByID retrieves a delivery record by its ID.
func (r *DeliveryRepository) GetByID(ctx context.Context, id string) (*model.DeliveryRecord, error) {
	query := `
		SELECT id, notification_id, channel, status, attempt, error_message, metadata, delivered_at, created_at
		FROM notification_delivery_log
		WHERE id = $1`

	var rec model.DeliveryRecord
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.NotificationID, &rec.Channel, &rec.Status,
		&rec.Attempt, &rec.ErrorMessage, &rec.Metadata, &rec.DeliveredAt, &rec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get delivery by id: %w", err)
	}
	return &rec, nil
}

// GetNotificationByID retrieves a notification by ID (no user scope, for internal use).
func (r *DeliveryRepository) GetNotificationByID(ctx context.Context, notifID string) (*model.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id, read_at, created_at
		FROM notifications WHERE id = $1`

	var n model.Notification
	err := r.db.QueryRow(ctx, query, notifID).Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Category, &n.Priority,
		&n.Title, &n.Body, &n.Data, &n.ActionURL, &n.SourceEventID, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	n.ComputeRead()
	return &n, nil
}

// RetryDelivery re-queues a specific delivery record.
func (r *DeliveryRepository) RetryDelivery(ctx context.Context, deliveryID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notification_delivery_log SET status = 'pending', error_message = NULL WHERE id = $1 AND status = 'failed'`,
		deliveryID,
	)
	if err != nil {
		return fmt.Errorf("retry delivery: %w", err)
	}
	return nil
}

// FindDeliveryByID returns a delivery record for retry purposes, including the webhook_id.
func (r *DeliveryRepository) FindDeliveryByID(ctx context.Context, id string) (*model.DeliveryRecord, string, error) {
	query := `
		SELECT id, notification_id, channel, status, attempt, error_message, metadata, delivered_at, created_at, COALESCE(webhook_id::text, '')
		FROM notification_delivery_log
		WHERE id = $1`

	var rec model.DeliveryRecord
	var webhookID string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.NotificationID, &rec.Channel, &rec.Status,
		&rec.Attempt, &rec.ErrorMessage, &rec.Metadata, &rec.DeliveredAt, &rec.CreatedAt,
		&webhookID,
	)
	if err == pgx.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("find delivery by id: %w", err)
	}
	return &rec, webhookID, nil
}
