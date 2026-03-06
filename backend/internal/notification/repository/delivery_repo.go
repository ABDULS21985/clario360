package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	return &n, nil
}
