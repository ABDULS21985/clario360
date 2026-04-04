package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	GetByID(ctx context.Context, id, tenantID string) (*model.APIKey, error)
	List(ctx context.Context, tenantID string) ([]model.APIKey, error)
	ListPaginated(ctx context.Context, tenantID string, page, perPage int, search, status, sort, order string) ([]model.APIKey, int, error)
	Revoke(ctx context.Context, id, tenantID string) error
	RotateKey(ctx context.Context, id, tenantID, newKeyHash, newKeyPrefix string) error
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

func (r *apiKeyRepo) GetByID(ctx context.Context, id, tenantID string) (*model.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_at, created_by, revoked_at
		FROM api_keys
		WHERE id = $1 AND tenant_id = $2`

	k := &model.APIKey{}
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
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

// validAPIKeySortColumns is an allowlist of columns that can be used for sorting API keys.
var validAPIKeySortColumns = map[string]string{
	"name":         "name",
	"created_at":   "created_at",
	"last_used_at": "last_used_at",
	"expires_at":   "expires_at",
}

func (r *apiKeyRepo) ListPaginated(ctx context.Context, tenantID string, page, perPage int, search, status, sort, order string) ([]model.APIKey, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	where := "tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR key_prefix ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Support comma-separated multi-status: "active,revoked"
	if status != "" {
		statusValues := strings.Split(status, ",")
		var statusConds []string
		for _, sv := range statusValues {
			switch strings.TrimSpace(sv) {
			case "active":
				statusConds = append(statusConds, "(revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW()))")
			case "revoked":
				statusConds = append(statusConds, "revoked_at IS NOT NULL")
			case "expired":
				statusConds = append(statusConds, "(revoked_at IS NULL AND expires_at IS NOT NULL AND expires_at <= NOW())")
			}
		}
		if len(statusConds) > 0 {
			where += " AND (" + strings.Join(statusConds, " OR ") + ")"
		}
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM api_keys WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting api keys: %w", err)
	}
	if total == 0 {
		return []model.APIKey{}, 0, nil
	}

	// Determine sort column and direction from allowlist.
	orderCol := "created_at"
	if col, ok := validAPIKeySortColumns[sort]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.EqualFold(order, "asc") {
		orderDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_at, created_by, revoked_at
		FROM api_keys
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, where, orderCol, orderDir, argIdx, argIdx+1)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing api keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(
			&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix,
			&k.Permissions, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.CreatedBy, &k.RevokedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, total, nil
}

func (r *apiKeyRepo) Revoke(ctx context.Context, id, tenantID string) error {
	query := `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("revoking api key: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("api key %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *apiKeyRepo) RotateKey(ctx context.Context, id, tenantID, newKeyHash, newKeyPrefix string) error {
	query := `UPDATE api_keys SET key_hash = $3, key_prefix = $4 WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, tenantID, newKeyHash, newKeyPrefix)
	if err != nil {
		return fmt.Errorf("rotating api key: %w", err)
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
