package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/iam/model"
)

type UserFilter struct {
	Status  *string
	Search  *string
	Page    int
	PerPage int
}

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*model.User, error)
	List(ctx context.Context, tenantID string, filter UserFilter) ([]model.User, int, error)
	Update(ctx context.Context, user *model.User) error
	SoftDelete(ctx context.Context, id, deletedBy string) error
	UpdateStatus(ctx context.Context, id string, status model.UserStatus, updatedBy string) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	UpdateMFA(ctx context.Context, id string, enabled bool, secret *string) error
	UpdateLastLogin(ctx context.Context, id string) error
	CountByTenant(ctx context.Context, tenantID string) (int, error)
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (tenant_id, email, password_hash, first_name, last_name, avatar_url, status, mfa_enabled, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		user.TenantID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.AvatarURL,
		user.Status, user.MFAEnabled, user.CreatedBy,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT u.id, u.tenant_id, u.email, u.password_hash, u.first_name, u.last_name,
		       u.avatar_url, u.status, u.mfa_enabled, u.mfa_secret, u.last_login_at,
		       u.created_at, u.updated_at, u.created_by, u.updated_by, u.deleted_at
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.AvatarURL, &user.Status,
		&user.MFAEnabled, &user.MFASecret, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user %s: %w", id, model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying user: %w", err)
	}

	roles, err := r.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, tenantID, email string) (*model.User, error) {
	query := `
		SELECT u.id, u.tenant_id, u.email, u.password_hash, u.first_name, u.last_name,
		       u.avatar_url, u.status, u.mfa_enabled, u.mfa_secret, u.last_login_at,
		       u.created_at, u.updated_at, u.created_by, u.updated_by, u.deleted_at
		FROM users u
		WHERE u.tenant_id = $1 AND u.email = $2 AND u.deleted_at IS NULL`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, tenantID, email).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.AvatarURL, &user.Status,
		&user.MFAEnabled, &user.MFASecret, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user with email %s: %w", email, model.ErrNotFound)
		}
		return nil, fmt.Errorf("querying user by email: %w", err)
	}

	roles, err := r.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *userRepo) List(ctx context.Context, tenantID string, filter UserFilter) ([]model.User, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("u.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "u.deleted_at IS NULL")

	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("u.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(u.email ILIKE $%d OR u.first_name ILIKE $%d OR u.last_name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, search)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users u WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	// Fetch page
	offset := (filter.Page - 1) * filter.PerPage
	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.tenant_id, u.email, u.password_hash, u.first_name, u.last_name,
		       u.avatar_url, u.status, u.mfa_enabled, u.mfa_secret, u.last_login_at,
		       u.created_at, u.updated_at, u.created_by, u.updated_by, u.deleted_at
		FROM users u
		WHERE %s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Email, &u.PasswordHash,
			&u.FirstName, &u.LastName, &u.AvatarURL, &u.Status,
			&u.MFAEnabled, &u.MFASecret, &u.LastLoginAt,
			&u.CreatedAt, &u.UpdatedAt, &u.CreatedBy, &u.UpdatedBy, &u.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (r *userRepo) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET first_name = $2, last_name = $3, avatar_url = $4, updated_by = $5, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	ct, err := r.pool.Exec(ctx, query, user.ID, user.FirstName, user.LastName, user.AvatarURL, user.UpdatedBy)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", user.ID, model.ErrNotFound)
	}
	return nil
}

func (r *userRepo) SoftDelete(ctx context.Context, id, deletedBy string) error {
	query := `UPDATE users SET deleted_at = NOW(), updated_by = $2 WHERE id = $1 AND deleted_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, deletedBy)
	if err != nil {
		return fmt.Errorf("soft-deleting user: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *userRepo) UpdateStatus(ctx context.Context, id string, status model.UserStatus, updatedBy string) error {
	query := `UPDATE users SET status = $2, updated_by = $3, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, status, updatedBy)
	if err != nil {
		return fmt.Errorf("updating user status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *userRepo) UpdateMFA(ctx context.Context, id string, enabled bool, secret *string) error {
	query := `UPDATE users SET mfa_enabled = $2, mfa_secret = $3, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, enabled, secret)
	if err != nil {
		return fmt.Errorf("updating mfa: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", id, model.ErrNotFound)
	}
	return nil
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id string) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *userRepo) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND deleted_at IS NULL", tenantID).Scan(&count)
	return count, err
}

func (r *userRepo) getUserRoles(ctx context.Context, userID string) ([]model.Role, error) {
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
			return nil, fmt.Errorf("scanning role row: %w", err)
		}
		if err := json.Unmarshal(permsJSON, &role.Permissions); err != nil {
			return nil, fmt.Errorf("unmarshaling permissions: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}
