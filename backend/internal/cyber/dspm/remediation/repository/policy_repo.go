package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// PolicyRepository handles persistence for DSPM data policies.
type PolicyRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPolicyRepository creates a new PolicyRepository.
func NewPolicyRepository(db *pgxpool.Pool, logger zerolog.Logger) *PolicyRepository {
	return &PolicyRepository{db: db, logger: logger}
}

// policyColumns is the shared column list for SELECT queries on dspm_data_policies.
const policyColumns = `id, tenant_id, name, description, category, rule, enforcement,
	auto_playbook_id, severity, scope_classification, scope_asset_types,
	enabled, last_evaluated_at, violation_count, compliance_frameworks,
	created_by, created_at, updated_at`

// Create inserts a new data policy and returns it with server-generated fields.
func (r *PolicyRepository) Create(ctx context.Context, policy *model.DataPolicy) (*model.DataPolicy, error) {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	if len(policy.Rule) == 0 {
		policy.Rule = json.RawMessage("{}")
	}
	if policy.ScopeClassification == nil {
		policy.ScopeClassification = []string{}
	}
	if policy.ScopeAssetTypes == nil {
		policy.ScopeAssetTypes = []string{}
	}
	if policy.ComplianceFrameworks == nil {
		policy.ComplianceFrameworks = []string{}
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO dspm_data_policies (
			id, tenant_id, name, description, category, rule, enforcement,
			auto_playbook_id, severity, scope_classification, scope_asset_types,
			enabled, violation_count, compliance_frameworks,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, now(), now()
		)
		RETURNING `+policyColumns,
		policy.ID, policy.TenantID, policy.Name, policy.Description,
		policy.Category, policy.Rule, policy.Enforcement,
		policy.AutoPlaybookID, policy.Severity, policy.ScopeClassification,
		policy.ScopeAssetTypes,
		policy.Enabled, policy.ViolationCount, policy.ComplianceFrameworks,
		policy.CreatedBy,
	)

	result, err := scanPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("create data policy: %w", err)
	}
	return result, nil
}

// GetByID fetches a single data policy by ID with tenant isolation.
func (r *PolicyRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*model.DataPolicy, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+policyColumns+` FROM dspm_data_policies WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)

	result, err := scanPolicy(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("data policy not found")
		}
		return nil, fmt.Errorf("get data policy: %w", err)
	}
	return result, nil
}

// List returns paginated data policies with filtering.
func (r *PolicyRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.PolicyListParams) ([]model.DataPolicy, int, error) {
	params.SetDefaults()

	var (
		conditions []string
		args       []interface{}
		argIdx     int
	)

	nextArg := func(val interface{}) string {
		argIdx++
		args = append(args, val)
		return fmt.Sprintf("$%d", argIdx)
	}

	conditions = append(conditions, "tenant_id = "+nextArg(tenantID))

	if params.Category != "" {
		conditions = append(conditions, "category = "+nextArg(params.Category))
	}

	if params.Enforcement != "" {
		conditions = append(conditions, "enforcement = "+nextArg(params.Enforcement))
	}

	if params.Enabled != nil {
		conditions = append(conditions, "enabled = "+nextArg(*params.Enabled))
	}

	if strings.TrimSpace(params.Search) != "" {
		search := "%" + strings.TrimSpace(params.Search) + "%"
		conditions = append(conditions, "(name ILIKE "+nextArg(search)+" OR description ILIKE "+nextArg(search)+")")
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query.
	var total int
	countSQL := "SELECT COUNT(*) FROM dspm_data_policies WHERE " + whereClause
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count data policies: %w", err)
	}

	// Data query with pagination.
	offset := (params.Page - 1) * params.PerPage
	dataSQL := `SELECT ` + policyColumns + ` FROM dspm_data_policies WHERE ` + whereClause +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT %s OFFSET %s", nextArg(params.PerPage), nextArg(offset))

	rows, err := r.db.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list data policies: %w", err)
	}
	defer rows.Close()

	items := make([]model.DataPolicy, 0)
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan policy row: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate data policies: %w", err)
	}

	return items, total, nil
}

// Update updates all mutable fields of a data policy.
func (r *PolicyRepository) Update(ctx context.Context, policy *model.DataPolicy) error {
	if len(policy.Rule) == 0 {
		policy.Rule = json.RawMessage("{}")
	}
	if policy.ScopeClassification == nil {
		policy.ScopeClassification = []string{}
	}
	if policy.ScopeAssetTypes == nil {
		policy.ScopeAssetTypes = []string{}
	}
	if policy.ComplianceFrameworks == nil {
		policy.ComplianceFrameworks = []string{}
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_data_policies
		SET
			name = $3,
			description = $4,
			category = $5,
			rule = $6,
			enforcement = $7,
			auto_playbook_id = $8,
			severity = $9,
			scope_classification = $10,
			scope_asset_types = $11,
			enabled = $12,
			compliance_frameworks = $13,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		policy.TenantID, policy.ID,
		policy.Name, policy.Description,
		policy.Category, policy.Rule,
		policy.Enforcement, policy.AutoPlaybookID,
		policy.Severity, policy.ScopeClassification,
		policy.ScopeAssetTypes, policy.Enabled,
		policy.ComplianceFrameworks,
	)
	if err != nil {
		return fmt.Errorf("update data policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("data policy not found")
	}
	return nil
}

// Delete removes a data policy by ID with tenant isolation.
func (r *PolicyRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM dspm_data_policies
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("delete data policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("data policy not found")
	}
	return nil
}

// ListEnabled returns all enabled policies for a tenant (used by the policy engine).
func (r *PolicyRepository) ListEnabled(ctx context.Context, tenantID uuid.UUID) ([]model.DataPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+policyColumns+` FROM dspm_data_policies WHERE tenant_id = $1 AND enabled = true ORDER BY created_at ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled data policies: %w", err)
	}
	defer rows.Close()

	items := make([]model.DataPolicy, 0)
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan enabled policy row: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled data policies: %w", err)
	}
	return items, nil
}

// UpdateEvaluationResults updates the violation count and last evaluated timestamp after a policy evaluation run.
func (r *PolicyRepository) UpdateEvaluationResults(ctx context.Context, tenantID, id uuid.UUID, violationCount int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_data_policies
		SET
			violation_count = $3,
			last_evaluated_at = now(),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, violationCount,
	)
	if err != nil {
		return fmt.Errorf("update policy evaluation results: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("data policy not found")
	}
	return nil
}

// scanPolicy scans a single policy row into a model.DataPolicy.
func scanPolicy(row scanner) (*model.DataPolicy, error) {
	var item model.DataPolicy
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Category,
		&item.Rule,
		&item.Enforcement,
		&item.AutoPlaybookID,
		&item.Severity,
		&item.ScopeClassification,
		&item.ScopeAssetTypes,
		&item.Enabled,
		&item.LastEvaluatedAt,
		&item.ViolationCount,
		&item.ComplianceFrameworks,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(item.Rule) == 0 {
		item.Rule = json.RawMessage("{}")
	}
	if item.ScopeClassification == nil {
		item.ScopeClassification = []string{}
	}
	if item.ScopeAssetTypes == nil {
		item.ScopeAssetTypes = []string{}
	}
	if item.ComplianceFrameworks == nil {
		item.ComplianceFrameworks = []string{}
	}
	return &item, nil
}
