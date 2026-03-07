package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/model"
)

type ComplianceRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewComplianceRepository(db *pgxpool.Pool, logger zerolog.Logger) *ComplianceRepository {
	return &ComplianceRepository{db: db, logger: logger}
}

func (r *ComplianceRepository) CreateRule(ctx context.Context, q Queryer, rule *model.ComplianceRule) error {
	query := `
		INSERT INTO compliance_rules (
			id, tenant_id, name, description, rule_type, severity, config, contract_types, enabled, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING created_at, updated_at`
	return q.QueryRow(ctx, query,
		rule.ID, rule.TenantID, rule.Name, rule.Description, rule.RuleType, rule.Severity, rule.Config, rule.ContractTypes, rule.Enabled, rule.CreatedBy,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
}

func (r *ComplianceRepository) GetRule(ctx context.Context, tenantID, id uuid.UUID) (*model.ComplianceRule, error) {
	query := ruleJSONSelect(`tenant_id = $1 AND id = $2 AND deleted_at IS NULL`)
	return queryRowJSON[model.ComplianceRule](ctx, r.db, query, tenantID, id)
}

func (r *ComplianceRepository) ListRules(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceRule, error) {
	query := ruleJSONSelect(`tenant_id = $1 AND deleted_at IS NULL`) + ` ORDER BY created_at DESC`
	return queryListJSON[model.ComplianceRule](ctx, r.db, query, tenantID)
}

func (r *ComplianceRepository) ListEnabledRules(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceRule, error) {
	query := ruleJSONSelect(`tenant_id = $1 AND enabled = true AND deleted_at IS NULL`) + ` ORDER BY created_at DESC`
	return queryListJSON[model.ComplianceRule](ctx, r.db, query, tenantID)
}

func (r *ComplianceRepository) UpdateRule(ctx context.Context, q Queryer, rule *model.ComplianceRule) error {
	query := `
		UPDATE compliance_rules
		SET name = $3,
		    description = $4,
		    rule_type = $5,
		    severity = $6,
		    config = $7,
		    contract_types = $8,
		    enabled = $9,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		RETURNING updated_at`
	return q.QueryRow(ctx, query,
		rule.TenantID, rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Severity, rule.Config, rule.ContractTypes, rule.Enabled,
	).Scan(&rule.UpdatedAt)
}

func (r *ComplianceRepository) SoftDeleteRule(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `UPDATE compliance_rules SET deleted_at = now(), updated_at = now() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func ruleJSONSelect(where string) string {
	return `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, name, description, rule_type, severity,
			       COALESCE(config, '{}'::jsonb) AS config,
			       COALESCE(contract_types, '{}') AS contract_types,
			       enabled, created_by, created_at, updated_at, deleted_at
			FROM compliance_rules
			WHERE ` + where + `
		) t`
}
