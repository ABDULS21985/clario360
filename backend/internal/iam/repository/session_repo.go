package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error)
	GetByUserID(ctx context.Context, userID string) ([]model.Session, error)
	UpdateLastActive(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type sessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) SessionRepository {
	return &sessionRepo{pool: pool}
}

func (r *sessionRepo) Create(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (user_id, tenant_id, refresh_token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4::inet, $5, $6)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		session.UserID, session.TenantID, session.RefreshTokenHash,
		session.IPAddress, session.UserAgent, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
}

func (r *sessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	query := `
		SELECT id, user_id, tenant_id, refresh_token_hash, host(ip_address), user_agent, expires_at, created_at, last_active_at
		FROM sessions
		WHERE refresh_token_hash = $1 AND expires_at > NOW()`

	s := &model.Session{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID, &s.UserID, &s.TenantID, &s.RefreshTokenHash,
		&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt, &s.LastActiveAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying session: %w", err)
	}
	return s, nil
}

func (r *sessionRepo) GetByUserID(ctx context.Context, userID string) ([]model.Session, error) {
	query := `
		SELECT id, user_id, tenant_id, refresh_token_hash, host(ip_address), user_agent, expires_at, created_at, last_active_at
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY last_active_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var s model.Session
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.TenantID, &s.RefreshTokenHash,
			&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt, &s.LastActiveAt,
		); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *sessionRepo) UpdateLastActive(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE sessions SET last_active_at = NOW() WHERE id = $1", id)
	return err
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

func (r *sessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}

func (r *sessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	ct, err := r.pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at <= NOW()")
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
