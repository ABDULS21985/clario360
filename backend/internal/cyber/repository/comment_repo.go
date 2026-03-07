package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// CommentRepository handles alert comments.
type CommentRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewCommentRepository creates a new CommentRepository.
func NewCommentRepository(db *pgxpool.Pool, logger zerolog.Logger) *CommentRepository {
	return &CommentRepository{db: db, logger: logger}
}

// Create inserts an alert comment.
func (r *CommentRepository) Create(ctx context.Context, comment *model.AlertComment) (*model.AlertComment, error) {
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	if _, err := r.db.Exec(ctx, `
		INSERT INTO alert_comments (
			id, tenant_id, alert_id, user_id, user_name, user_email, content,
			is_system, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, now(), now()
		)`,
		comment.ID, comment.TenantID, comment.AlertID, comment.UserID, comment.UserName,
		comment.UserEmail, comment.Content, comment.IsSystem, ensureRawMessage(comment.Metadata, "{}"),
	); err != nil {
		return nil, fmt.Errorf("insert alert comment: %w", err)
	}
	return r.GetByID(ctx, comment.TenantID, comment.ID)
}

// GetByID fetches a single comment.
func (r *CommentRepository) GetByID(ctx context.Context, tenantID, commentID uuid.UUID) (*model.AlertComment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, alert_id, user_id, user_name, user_email,
			content, is_system, metadata, created_at, updated_at
		FROM alert_comments
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, commentID,
	)
	comment, err := scanAlertComment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get alert comment: %w", err)
	}
	return comment, nil
}

// ListByAlert returns alert comments in chronological order.
func (r *CommentRepository) ListByAlert(ctx context.Context, tenantID, alertID uuid.UUID) ([]*model.AlertComment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, alert_id, user_id, user_name, user_email,
			content, is_system, metadata, created_at, updated_at
		FROM alert_comments
		WHERE tenant_id = $1 AND alert_id = $2
		ORDER BY created_at ASC`,
		tenantID, alertID,
	)
	if err != nil {
		return nil, fmt.Errorf("list alert comments: %w", err)
	}
	defer rows.Close()

	comments := make([]*model.AlertComment, 0)
	for rows.Next() {
		comment, err := scanAlertComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

// ReassignAlert moves comments from one alert to another during merge.
func (r *CommentRepository) ReassignAlert(ctx context.Context, tenantID, fromAlertID, toAlertID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE alert_comments
		SET alert_id = $3, updated_at = now()
		WHERE tenant_id = $1 AND alert_id = $2`,
		tenantID, fromAlertID, toAlertID,
	)
	if err != nil {
		return fmt.Errorf("reassign alert comments: %w", err)
	}
	return nil
}
