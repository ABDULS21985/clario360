package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *InvitationRepository) ListPendingByPrefix(ctx context.Context, tokenPrefix string) ([]onboardingmodel.Invitation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, email, role_slug, token_hash, token_prefix, status,
		       invited_by, invited_by_name, accepted_at, accepted_by, expires_at,
		       message, created_at, updated_at
		FROM invitations
		WHERE token_prefix = $1 AND status = 'pending'`,
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
