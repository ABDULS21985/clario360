package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id string) (*model.Role, error)
	GetBySlug(ctx context.Context, tenantID, slug string) (*model.Role, error)
	List(ctx context.Context, tenantID string) ([]model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id string) error
	AssignToUser(ctx context.Context, userID, roleID, tenantID, assignedBy string) error
	RemoveFromUser(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]model.Role, error)
	ListUserIDsByRole(ctx context.Context, tenantID, roleSlug string) ([]string, error)
	SeedSystemRoles(ctx context.Context, tenantID string) error
}

type roleRepo struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &roleRepo{pool: pool}
}

func (r *roleRepo) Create(ctx context.Context, role *model.Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshaling permissions: %w", err)
	}

	query := `
		INSERT INTO roles (tenant_id, name, slug, description, is_system_role, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		role.TenantID, role.Name, role.Slug, role.Description,
		role.IsSystemRole, permsJSON,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
}

func (r *roleRepo) GetByID(ctx context.Context, id string) (*model.Role, error) {
	query := `SELECT id, tenant_id, name, slug, description, is_system_role, permissions, created_at, updated_at
		FROM roles WHERE id = $1`

	role := &model.Role{}
	var permsJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
		&role.IsSystemRole, &permsJSON, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role %s: %w", id, model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying role: %w", err)
	}
	if err := json.Unmarshal(permsJSON, &role.Permissions); err != nil {
		return nil, fmt.Errorf("unmarshaling permissions: %w", err)
	}
	return role, nil
}

func (r *roleRepo) GetBySlug(ctx context.Context, tenantID, slug string) (*model.Role, error) {
	query := `SELECT id, tenant_id, name, slug, description, is_system_role, permissions, created_at, updated_at
		FROM roles WHERE tenant_id = $1 AND slug = $2`

	role := &model.Role{}
	var permsJSON []byte
	err := r.pool.QueryRow(ctx, query, tenantID, slug).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
		&role.IsSystemRole, &permsJSON, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role %s: %w", slug, model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying role by slug: %w", err)
	}
	if err := json.Unmarshal(permsJSON, &role.Permissions); err != nil {
		return nil, fmt.Errorf("unmarshaling permissions: %w", err)
	}
	return role, nil
}

func (r *roleRepo) List(ctx context.Context, tenantID string) ([]model.Role, error) {
	query := `SELECT id, tenant_id, name, slug, description, is_system_role, permissions, created_at, updated_at
		FROM roles WHERE tenant_id = $1 ORDER BY is_system_role DESC, name`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		var permsJSON []byte
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
			&role.IsSystemRole, &permsJSON, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning role: %w", err)
		}
		if err := json.Unmarshal(permsJSON, &role.Permissions); err != nil {
			return nil, fmt.Errorf("unmarshaling permissions: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *roleRepo) Update(ctx context.Context, role *model.Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshaling permissions: %w", err)
	}

	query := `UPDATE roles SET name = $2, description = $3, permissions = $4, updated_at = NOW()
		WHERE id = $1 AND is_system_role = false`

	ct, err := r.pool.Exec(ctx, query, role.ID, role.Name, role.Description, permsJSON)
	if err != nil {
		return fmt.Errorf("updating role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("role %s: %w", role.ID, model.ErrNotFound)
	}
	return nil
}

func (r *roleRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM roles WHERE id = $1 AND is_system_role = false`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		// Check if it exists but is a system role
		var isSystem bool
		checkErr := r.pool.QueryRow(ctx, "SELECT is_system_role FROM roles WHERE id = $1", id).Scan(&isSystem)
		if checkErr == pgx.ErrNoRows {
			return fmt.Errorf("role %s: %w", id, model.ErrNotFound)
		}
		if isSystem {
			return model.ErrSystemRole
		}
		return fmt.Errorf("role %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *roleRepo) AssignToUser(ctx context.Context, userID, roleID, tenantID, assignedBy string) error {
	query := `INSERT INTO user_roles (user_id, role_id, tenant_id, assigned_by) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, userID, roleID, tenantID, assignedBy)
	if err != nil {
		return fmt.Errorf("assigning role: %w", err)
	}
	return nil
}

func (r *roleRepo) RemoveFromUser(ctx context.Context, userID, roleID string) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	ct, err := r.pool.Exec(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("removing role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("role assignment: %w", model.ErrNotFound)
	}
	return nil
}

func (r *roleRepo) GetUserRoles(ctx context.Context, userID string) ([]model.Role, error) {
	query := `
		SELECT r.id, r.tenant_id, r.name, r.slug, r.description, r.is_system_role, r.permissions, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying user roles: %w", err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		var permsJSON []byte
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Name, &role.Slug, &role.Description,
			&role.IsSystemRole, &permsJSON, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning role: %w", err)
		}
		if err := json.Unmarshal(permsJSON, &role.Permissions); err != nil {
			return nil, fmt.Errorf("unmarshaling permissions: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *roleRepo) ListUserIDsByRole(ctx context.Context, tenantID, roleSlug string) ([]string, error) {
	query := `
		SELECT u.id
		FROM users u
		INNER JOIN user_roles ur ON ur.user_id = u.id
		INNER JOIN roles ro ON ro.id = ur.role_id
		WHERE ro.tenant_id = $1 AND ro.slug = $2 AND u.deleted_at IS NULL
		ORDER BY u.created_at`

	rows, err := r.pool.Query(ctx, query, tenantID, roleSlug)
	if err != nil {
		return nil, fmt.Errorf("listing users by role: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning user id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *roleRepo) SeedSystemRoles(ctx context.Context, tenantID string) error {
	for _, sr := range model.SystemRoles {
		permsJSON, err := json.Marshal(sr.Permissions)
		if err != nil {
			return fmt.Errorf("marshaling permissions for %s: %w", sr.Slug, err)
		}
		query := `
			INSERT INTO roles (tenant_id, name, slug, description, is_system_role, permissions)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, slug) DO NOTHING`
		if _, err := r.pool.Exec(ctx, query, tenantID, sr.Name, sr.Slug, sr.Description, true, permsJSON); err != nil {
			return fmt.Errorf("seeding role %s: %w", sr.Slug, err)
		}
	}
	return nil
}
