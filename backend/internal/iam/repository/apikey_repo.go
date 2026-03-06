package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	List(ctx context.Context, tenantID string) ([]model.APIKey, error)
	Revoke(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error
}

type apiKeyRepo struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) APIKeyRepository {
	return &apiKeyRepo{pool: pool}
}

func (r *apiKeyRepo) Create(ctx context.Context, key *model.APIKey) error {
	query := `
		INSERT INTO api_keys (tenant_id, name, key_hash, key_prefix, permissions, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	perms := key.Permissions
	if perms == nil {
		perms = []byte("[]")
	}

	return r.pool.QueryRow(ctx, query,
		key.TenantID, key.Name, key.KeyHash, key.KeyPrefix,
		perms, key.ExpiresAt, key.CreatedBy,
	).Scan(&key.ID, &key.CreatedAt)
}

func (r *apiKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_at, created_by, revoked_at
		FROM api_keys
		WHERE key_hash = $1 AND revoked_at IS NULL`

	k := &model.APIKey{}
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix,
		&k.Permissions, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.CreatedBy, &k.RevokedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("api key: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying api key: %w", err)
	}
	return k, nil
}

func (r *apiKeyRepo) List(ctx context.Context, tenantID string) ([]model.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_at, created_by, revoked_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(
			&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix,
			&k.Permissions, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.CreatedBy, &k.RevokedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *apiKeyRepo) Revoke(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("revoking api key: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("api key %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *apiKeyRepo) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", id)
	return err
}
