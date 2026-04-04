package seeder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ComplianceRuleSeeder struct {
	pool *pgxpool.Pool
}

func NewComplianceRuleSeeder(pool *pgxpool.Pool) *ComplianceRuleSeeder {
	return &ComplianceRuleSeeder{pool: pool}
}

func (s *ComplianceRuleSeeder) Seed(ctx context.Context, tenantID, adminUserID uuid.UUID) error {
	rules := []struct {
		Name        string
		Description string
		RuleType    string
		Severity    string
		Config      any
	}{
		{
			Name:        "Data protection clause required for service agreements",
			Description: "Service agreements must contain a data protection clause.",
			RuleType:    "data_protection_required",
			Severity:    "high",
			Config:      map[string]any{"required_clause": "data protection", "contract_types": []string{"service_agreement"}},
		},
		{
			Name:        "Limitation of liability required for vendor contracts",
			Description: "Vendor contracts must contain a limitation of liability clause.",
			RuleType:    "missing_clause",
			Severity:    "medium",
			Config:      map[string]any{"required_clause": "limitation of liability", "contract_types": []string{"vendor_contract"}},
		},
		{
			Name:        "90-day expiry notification threshold",
			Description: "Flag contracts expiring within 90 days.",
			RuleType:    "expiry_warning",
			Severity:    "medium",
			Config:      map[string]any{"threshold_days": 90},
		},
		{
			Name:        "Contracts > 1M SAR require extra approval",
			Description: "Large-value contracts must receive extra legal approval.",
			RuleType:    "value_threshold",
			Severity:    "high",
			Config:      map[string]any{"amount": 1000000, "currency": "SAR"},
		},
		{
			Name:        "Auto-renewal contracts must be reviewed 60 days before renewal",
			Description: "Auto-renewal contracts need pre-renewal review.",
			RuleType:    "review_overdue",
			Severity:    "medium",
			Config:      map[string]any{"review_window_days": 60, "requires_auto_renewal": true},
		},
	}
	for _, rule := range rules {
		var exists bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM compliance_rules
				WHERE tenant_id = $1
				  AND name = $2
				  AND deleted_at IS NULL
			)`,
			tenantID,
			rule.Name,
		).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO compliance_rules (
				id, tenant_id, name, description, rule_type, severity, config, contract_types, enabled, created_by
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7::jsonb, $8, true, $9
			)`,
			uuid.New(),
			tenantID,
			rule.Name,
			rule.Description,
			rule.RuleType,
			rule.Severity,
			marshalJSON(rule.Config),
			[]string{},
			adminUserID,
		); err != nil {
			return fmt.Errorf("seed compliance rule %s: %w", rule.Name, err)
		}
	}
	return nil
}
