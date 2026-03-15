package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

// TenantListParams holds optional filter/search/sort params for listing tenants.
type TenantListParams struct {
	Search           string
	Status           string
	SubscriptionTier string
	Sort             string
	Order            string
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *model.Tenant) error
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	List(ctx context.Context, page, perPage int, params TenantListParams) ([]model.Tenant, int, error)
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

func (r *tenantRepo) List(ctx context.Context, page, perPage int, params TenantListParams) ([]model.Tenant, int, error) {
	// Build WHERE clause from filters.
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if params.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}
	if params.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, params.Status)
		argIdx++
	}
	if params.SubscriptionTier != "" {
		where += fmt.Sprintf(" AND subscription_tier = $%d", argIdx)
		args = append(args, params.SubscriptionTier)
		argIdx++
	}

	// Count with filters.
	var total int
	countQuery := "SELECT COUNT(*) FROM tenants " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting tenants: %w", err)
	}

	// Determine ORDER BY.
	allowedSorts := map[string]string{
		"name":              "name",
		"slug":              "slug",
		"status":            "status",
		"subscription_tier": "subscription_tier",
		"created_at":        "created_at",
		"updated_at":        "updated_at",
	}
	sortCol := "created_at"
	if col, ok := allowedSorts[params.Sort]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if params.Order == "asc" {
		sortDir = "ASC"
	}

	offset := (page - 1) * perPage
	query := fmt.Sprintf(
		`SELECT id, name, slug, domain, settings, status, subscription_tier, created_at, updated_at
		FROM tenants %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, sortCol, sortDir, argIdx, argIdx+1,
	)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
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
