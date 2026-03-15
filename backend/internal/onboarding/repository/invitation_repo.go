package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

type InvitationRepository struct {
	pool *pgxpool.Pool
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

func (r *InvitationRepository) CountPending(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM invitations
		WHERE tenant_id = $1 AND status = 'pending'`,
		tenantID,
	).Scan(&count)
	return count, err
}

func (r *InvitationRepository) Create(ctx context.Context, invitation *onboardingmodel.Invitation) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO invitations (
			tenant_id, email, role_slug, token_hash, token_prefix, status,
			invited_by, invited_by_name, expires_at, message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`,
		invitation.TenantID,
		invitation.Email,
		invitation.RoleSlug,
		invitation.TokenHash,
		invitation.TokenPrefix,
		invitation.Status,
		invitation.InvitedBy,
		invitation.InvitedByName,
		invitation.ExpiresAt,
		invitation.Message,
	).Scan(&invitation.ID, &invitation.CreatedAt, &invitation.UpdatedAt)
	if isUniqueViolation(err) {
		return fmt.Errorf("pending invitation already exists")
	}
	return err
}

func (r *InvitationRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.Invitation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, email, role_slug, token_hash, token_prefix, status,
		       invited_by, invited_by_name, accepted_at, accepted_by, expires_at,
		       message, created_at, updated_at
		FROM invitations
		WHERE tenant_id = $1
		ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]onboardingmodel.Invitation, 0)
	for rows.Next() {
		item, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *InvitationRepository) GetByID(ctx context.Context, tenantID, invitationID uuid.UUID) (*onboardingmodel.Invitation, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, role_slug, token_hash, token_prefix, status,
		       invited_by, invited_by_name, accepted_at, accepted_by, expires_at,
		       message, created_at, updated_at
		FROM invitations
		WHERE id = $1 AND tenant_id = $2`,
		invitationID,
		tenantID,
	)
	return scanInvitation(row)
}

func (r *InvitationRepository) ListByPrefix(ctx context.Context, tokenPrefix string) ([]onboardingmodel.Invitation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, email, role_slug, token_hash, token_prefix, status,
		       invited_by, invited_by_name, accepted_at, accepted_by, expires_at,
		       message, created_at, updated_at
		FROM invitations
		WHERE token_prefix = $1`,
		tokenPrefix,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]onboardingmodel.Invitation, 0)
	for rows.Next() {
		item, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *InvitationRepository) MarkAccepted(ctx context.Context, invitationID, acceptedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE invitations
		SET status = 'accepted',
		    accepted_at = now(),
		    accepted_by = $2,
		    updated_at = now()
		WHERE id = $1`,
		invitationID,
		acceptedBy,
	)
	return err
}

func (r *InvitationRepository) UpdateStatus(ctx context.Context, tenantID, invitationID uuid.UUID, status onboardingmodel.InvitationStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE invitations
		SET status = $3, updated_at = now()
		WHERE id = $1 AND tenant_id = $2`,
		invitationID,
		tenantID,
		status,
	)
	return err
}

func (r *InvitationRepository) Refresh(ctx context.Context, tenantID, invitationID uuid.UUID, tokenHash, tokenPrefix string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE invitations
		SET token_hash = $3,
		    token_prefix = $4,
		    expires_at = $5,
		    status = 'pending',
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2`,
		invitationID,
		tenantID,
		tokenHash,
		tokenPrefix,
		expiresAt,
	)
	return err
}

func (r *InvitationRepository) ExpirePastDue(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE invitations
		SET status = 'expired', updated_at = now()
		WHERE status = 'pending' AND expires_at < now()`)
	return err
}

func (r *InvitationRepository) HasPendingForEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM invitations
			WHERE tenant_id = $1
			  AND lower(email) = lower($2)
			  AND status = 'pending'
		)`,
		tenantID,
		email,
	).Scan(&exists)
	return exists, err
}

// allowedSortColumns is a whitelist of column names that can be used for ORDER BY
// to prevent SQL injection.
var allowedSortColumns = map[string]string{
	"email":      "email",
	"role_slug":  "role_slug",
	"status":     "status",
	"expires_at": "expires_at",
	"created_at": "created_at",
}

func (r *InvitationRepository) ListByTenantPaginated(ctx context.Context, tenantID uuid.UUID, page, perPage int, sort, order, search, status string) ([]onboardingmodel.Invitation, int, error) {
	// Validate and sanitize pagination
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 100 {
		perPage = 100
	}

	// Validate sort column against whitelist
	sortCol, ok := allowedSortColumns[sort]
	if !ok {
		sortCol = "created_at"
	}

	// Validate order direction
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	// Build WHERE clause
	where := "WHERE tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if search != "" {
		where += fmt.Sprintf(" AND email ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// Count total matching rows
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM invitations %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch the page
	offset := (page - 1) * perPage
	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, email, role_slug, token_hash, token_prefix, status,
		       invited_by, invited_by_name, accepted_at, accepted_by, expires_at,
		       message, created_at, updated_at
		FROM invitations
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, sortCol, order, argIdx, argIdx+1,
	)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]onboardingmodel.Invitation, 0)
	for rows.Next() {
		item, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *InvitationRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[onboardingmodel.InvitationStatus]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM invitations
		WHERE tenant_id = $1
		GROUP BY status`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[onboardingmodel.InvitationStatus]int)
	for rows.Next() {
		var status onboardingmodel.InvitationStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func (r *InvitationRepository) GetByStatusAndEmail(ctx context.Context, tenantID uuid.UUID, email string, status onboardingmodel.InvitationStatus) (*onboardingmodel.Invitation, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, role_slug, token_hash, token_prefix, status,
		       invited_by, invited_by_name, accepted_at, accepted_by, expires_at,
		       message, created_at, updated_at
		FROM invitations
		WHERE tenant_id = $1 AND lower(email) = lower($2) AND status = $3
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID,
		email,
		status,
	)
	return scanInvitation(row)
}
