package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// AccessPolicyRepository handles dspm_access_policies table operations.
type AccessPolicyRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAccessPolicyRepository creates a new access policy repository.
func NewAccessPolicyRepository(db *pgxpool.Pool, logger zerolog.Logger) *AccessPolicyRepository {
	return &AccessPolicyRepository{db: db, logger: logger}
}

// Create inserts a new access policy.
func (r *AccessPolicyRepository) Create(ctx context.Context, policy *model.AccessPolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	now := time.Now().UTC()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO dspm_access_policies (
			id, tenant_id, name, description, policy_type, rule_config,
			enforcement, severity, enabled, created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, policy.ID, policy.TenantID, policy.Name, policy.Description, policy.PolicyType,
		policy.RuleConfig, policy.Enforcement, policy.Severity, policy.Enabled,
		policy.CreatedBy, policy.CreatedAt, policy.UpdatedAt,
	)
	return err
}

// Update modifies an existing access policy.
func (r *AccessPolicyRepository) Update(ctx context.Context, policy *model.AccessPolicy) error {
	now := time.Now().UTC()
	policy.UpdatedAt = now

	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_access_policies SET
			name = $1, description = $2, rule_config = $3,
			enforcement = $4, severity = $5, enabled = $6,
			updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`, policy.Name, policy.Description, policy.RuleConfig,
		policy.Enforcement, policy.Severity, policy.Enabled,
		policy.UpdatedAt, policy.ID, policy.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an access policy.
func (r *AccessPolicyRepository) Delete(ctx context.Context, tenantID, policyID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM dspm_access_policies WHERE id = $1 AND tenant_id = $2
	`, policyID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByID returns a single access policy.
func (r *AccessPolicyRepository) GetByID(ctx context.Context, tenantID, policyID uuid.UUID) (*model.AccessPolicy, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, policy_type, rule_config,
			enforcement, severity, enabled, created_by, created_at, updated_at
		FROM dspm_access_policies
		WHERE id = $1 AND tenant_id = $2
	`, policyID, tenantID)

	p, err := scanPolicy(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

// ListAll returns all access policies for a tenant.
func (r *AccessPolicyRepository) ListAll(ctx context.Context, tenantID uuid.UUID) ([]model.AccessPolicy, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, policy_type, rule_config,
			enforcement, severity, enabled, created_by, created_at, updated_at
		FROM dspm_access_policies
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.AccessPolicy
	for rows.Next() {
		p, err := scanPolicyRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *p)
	}
	return results, rows.Err()
}

// ListEnabled returns only enabled policies for a tenant.
func (r *AccessPolicyRepository) ListEnabled(ctx context.Context, tenantID uuid.UUID) ([]model.AccessPolicy, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, policy_type, rule_config,
			enforcement, severity, enabled, created_by, created_at, updated_at
		FROM dspm_access_policies
		WHERE tenant_id = $1 AND enabled = true
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.AccessPolicy
	for rows.Next() {
		p, err := scanPolicyRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *p)
	}
	return results, rows.Err()
}

// CountViolations counts current policy violations (approximate from stale + overprivileged + overdue review).
func (r *AccessPolicyRepository) CountViolations(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM dspm_access_mappings WHERE tenant_id = $1 AND status = 'active' AND is_stale = true) +
			(SELECT COUNT(*) FROM dspm_identity_profiles WHERE tenant_id = $1 AND status = 'active' AND next_review_due IS NOT NULL AND next_review_due < now())
	`, tenantID).Scan(&count)
	return count, err
}

func scanPolicy(row pgx.Row) (*model.AccessPolicy, error) {
	return scanPolicyRow(row)
}

func scanPolicyRow(row pgx.Row) (*model.AccessPolicy, error) {
	p := &model.AccessPolicy{}
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description, &p.PolicyType, &p.RuleConfig,
		&p.Enforcement, &p.Severity, &p.Enabled, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}
