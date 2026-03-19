package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/dto"
	"github.com/clario360/platform/internal/notification/model"
)

// NotificationRepository handles notification CRUD operations.
type NotificationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(db *pgxpool.Pool, logger zerolog.Logger) *NotificationRepository {
	return &NotificationRepository{db: db, logger: logger.With().Str("component", "notification_repo").Logger()}
}

// InsertWithDedup inserts a notification and returns its ID. If a duplicate exists (same tenant+user+source_event_id),
// returns empty string and no error (dedup).
func (r *NotificationRepository) InsertWithDedup(ctx context.Context, n *model.Notification) (string, error) {
	dataBytes, err := json.Marshal(n.Data)
	if err != nil {
		dataBytes = []byte("{}")
	}

	query := `
		INSERT INTO notifications (tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT ON CONSTRAINT idx_notif_dedup DO NOTHING
		RETURNING id`

	var id string
	err = r.db.QueryRow(ctx, query,
		n.TenantID, n.UserID, string(n.Type), n.Category, n.Priority,
		n.Title, n.Body, dataBytes, n.ActionURL, n.SourceEventID,
	).Scan(&id)

	if err == pgx.ErrNoRows {
		// Duplicate — dedup triggered
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("insert notification: %w", err)
	}
	return id, nil
}

// Insert inserts a notification without dedup (no source_event_id).
func (r *NotificationRepository) Insert(ctx context.Context, n *model.Notification) (string, error) {
	dataBytes, err := json.Marshal(n.Data)
	if err != nil {
		dataBytes = []byte("{}")
	}

	query := `
		INSERT INTO notifications (tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id string
	err = r.db.QueryRow(ctx, query,
		n.TenantID, n.UserID, string(n.Type), n.Category, n.Priority,
		n.Title, n.Body, dataBytes, n.ActionURL, n.SourceEventID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert notification: %w", err)
	}
	return id, nil
}

// FindByID returns a single notification by ID, scoped to tenant and user.
func (r *NotificationRepository) FindByID(ctx context.Context, tenantID, userID, id string) (*model.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id, read_at, created_at
		FROM notifications
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3`

	return r.scanNotification(r.db.QueryRow(ctx, query, id, tenantID, userID))
}

// Query lists notifications with filtering and pagination.
func (r *NotificationRepository) Query(ctx context.Context, params *dto.QueryParams) ([]model.Notification, int, error) {
	where, args := r.buildWhere(params)

	countQuery := "SELECT COUNT(*) FROM notifications" + where
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	orderCol := "created_at"
	if params.Sort == "priority" {
		orderCol = "CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END"
	}
	orderDir := "DESC"
	if params.Order == "asc" {
		orderDir = "ASC"
	}

	argIdx := len(args) + 1
	dataQuery := fmt.Sprintf(
		"SELECT id, tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id, read_at, created_at FROM notifications%s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		where, orderCol, orderDir, argIdx, argIdx+1,
	)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	results := make([]model.Notification, 0)
	for rows.Next() {
		n, err := r.scanNotificationFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, *n)
	}
	return results, total, rows.Err()
}

// UnreadCount returns the number of unread notifications for a user in a tenant.
func (r *NotificationRepository) UnreadCount(ctx context.Context, tenantID, userID string) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications WHERE tenant_id = $1 AND user_id = $2 AND read_at IS NULL",
		tenantID, userID,
	).Scan(&count)
	return count, err
}

// MarkRead marks a single notification as read.
func (r *NotificationRepository) MarkRead(ctx context.Context, tenantID, userID, id string) error {
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx,
		"UPDATE notifications SET read_at = $1 WHERE id = $2 AND tenant_id = $3 AND user_id = $4 AND read_at IS NULL",
		now, id, tenantID, userID,
	)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification not found or already read")
	}
	return nil
}

// MarkAllRead marks all unread notifications as read for a user in a tenant.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, tenantID, userID string) (int64, error) {
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx,
		"UPDATE notifications SET read_at = $1 WHERE tenant_id = $2 AND user_id = $3 AND read_at IS NULL",
		now, tenantID, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("mark all read: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Delete hard-deletes a notification.
func (r *NotificationRepository) Delete(ctx context.Context, tenantID, userID, id string) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM notifications WHERE id = $1 AND tenant_id = $2 AND user_id = $3",
		id, tenantID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

// BulkDelete deletes multiple notifications by ID, scoped to tenant and user.
func (r *NotificationRepository) BulkDelete(ctx context.Context, tenantID, userID string, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	// Build parameterized query with ANY($3)
	tag, err := r.db.Exec(ctx,
		"DELETE FROM notifications WHERE tenant_id = $1 AND user_id = $2 AND id = ANY($3)",
		tenantID, userID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("bulk delete notifications: %w", err)
	}
	return tag.RowsAffected(), nil
}

// GetUnreadForDigest returns unread notifications created since a cutoff for a list of users.
func (r *NotificationRepository) GetUnreadForDigest(ctx context.Context, tenantID string, since time.Time) ([]model.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id, read_at, created_at
		FROM notifications
		WHERE tenant_id = $1 AND read_at IS NULL AND created_at >= $2
		ORDER BY user_id, created_at DESC`

	rows, err := r.db.Query(ctx, query, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("get unread for digest: %w", err)
	}
	defer rows.Close()

	results := make([]model.Notification, 0)
	for rows.Next() {
		n, err := r.scanNotificationFromRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *n)
	}
	return results, rows.Err()
}

func (r *NotificationRepository) buildWhere(params *dto.QueryParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	idx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", idx))
	args = append(args, params.TenantID)
	idx++

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", idx))
	args = append(args, params.UserID)
	idx++

	if params.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", idx))
		args = append(args, params.Category)
		idx++
	}

	if params.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", idx))
		args = append(args, params.Type)
		idx++
	}

	if params.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", idx))
		args = append(args, params.Priority)
		idx++
	}

	if params.Read != nil {
		if *params.Read {
			conditions = append(conditions, "read_at IS NOT NULL")
		} else {
			conditions = append(conditions, "read_at IS NULL")
		}
	}

	if params.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *params.DateFrom)
		idx++
	}

	if params.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *params.DateTo)
		idx++
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *NotificationRepository) scanNotification(row pgx.Row) (*model.Notification, error) {
	var n model.Notification
	err := row.Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Category, &n.Priority,
		&n.Title, &n.Body, &n.Data, &n.ActionURL, &n.SourceEventID, &n.ReadAt, &n.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan notification: %w", err)
	}
	n.ComputeRead()
	return &n, nil
}

func (r *NotificationRepository) scanNotificationFromRow(rows pgx.Rows) (*model.Notification, error) {
	var n model.Notification
	err := rows.Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Category, &n.Priority,
		&n.Title, &n.Body, &n.Data, &n.ActionURL, &n.SourceEventID, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan notification row: %w", err)
	}
	n.ComputeRead()
	return &n, nil
}
