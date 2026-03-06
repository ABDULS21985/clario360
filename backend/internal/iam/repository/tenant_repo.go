package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *model.Tenant) error
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	List(ctx context.Context, page, perPage int) ([]model.Tenant, int, error)
	Update(ctx context.Context, tenant *model.Tenant) error
}

type tenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) TenantRepository {
	return &tenantRepo{pool: pool}
}

func (r *tenantRepo) Create(ctx context.Context, tenant *model.Tenant) error {
	query := `
		INSERT INTO tenants (name, slug, domain, settings, status, subscription_tier)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	settings := tenant.Settings
	if settings == nil {
		settings = []byte("{}")
	}

	return r.pool.QueryRow(ctx, query,
		tenant.Name, tenant.Slug, tenant.Domain, settings,
		tenant.Status, tenant.SubscriptionTier,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)
}

func (r *tenantRepo) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	query := `SELECT id, name, slug, domain, settings, status, subscription_tier, created_at, updated_at
		FROM tenants WHERE id = $1`

	t := &model.Tenant{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings,
		&t.Status, &t.SubscriptionTier, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant %s: %w", id, model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying tenant: %w", err)
	}
	return t, nil
}

func (r *tenantRepo) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	query := `SELECT id, name, slug, domain, settings, status, subscription_tier, created_at, updated_at
		FROM tenants WHERE slug = $1`

	t := &model.Tenant{}
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings,
		&t.Status, &t.SubscriptionTier, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant %s: %w", slug, model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying tenant by slug: %w", err)
	}
	return t, nil
}

func (r *tenantRepo) List(ctx context.Context, page, perPage int) ([]model.Tenant, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting tenants: %w", err)
	}

	offset := (page - 1) * perPage
	query := `SELECT id, name, slug, domain, settings, status, subscription_tier, created_at, updated_at
		FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tenants: %w", err)
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.Domain, &t.Settings,
			&t.Status, &t.SubscriptionTier, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

func (r *tenantRepo) Update(ctx context.Context, tenant *model.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, domain = $3, settings = $4, status = $5, subscription_tier = $6, updated_at = NOW()
		WHERE id = $1`

	settings := tenant.Settings
	if settings == nil {
		settings = []byte("{}")
	}

	ct, err := r.pool.Exec(ctx, query,
		tenant.ID, tenant.Name, tenant.Domain, settings,
		tenant.Status, tenant.SubscriptionTier,
	)
	if err != nil {
		return fmt.Errorf("updating tenant: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("tenant %s: %w", tenant.ID, model.ErrNotFound)
	}
	return nil
}
