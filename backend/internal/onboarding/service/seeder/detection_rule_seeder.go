package seeder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	cyberservice "github.com/clario360/platform/internal/cyber/service"
)

type DetectionRuleSeeder struct {
	pool *pgxpool.Pool
}

func NewDetectionRuleSeeder(pool *pgxpool.Pool) *DetectionRuleSeeder {
	return &DetectionRuleSeeder{pool: pool}
}

func (s *DetectionRuleSeeder) Seed(ctx context.Context, tenantID, adminUserID uuid.UUID) error {
	for _, rule := range cyberservice.BuiltinTenantRuleSeeds() {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO detection_rules (
				id, tenant_id, name, description, rule_type, severity, enabled, rule_content,
				mitre_tactic_ids, mitre_technique_ids, base_confidence, false_positive_count,
				true_positive_count, trigger_count, tags, is_template, template_id, created_by,
				created_at, updated_at
			)
			SELECT
				$1, $2, $3, $4, $5, $6, true, $7::jsonb,
				$8, $9, $10, 0, 0, 0, $11, false, NULL, $12, now(), now()
			WHERE NOT EXISTS (
				SELECT 1
				FROM detection_rules
				WHERE tenant_id = $2
				  AND name = $3
				  AND deleted_at IS NULL
			)`,
			uuid.New(),
			tenantID,
			rule.Name,
			rule.Description,
			string(rule.RuleType),
			string(rule.Severity),
			rule.RuleContent,
			stringArray(rule.MITRETacticIDs),
			stringArray(rule.MITRETechniqueIDs),
			rule.BaseConfidence,
			stringArray(rule.Tags),
			adminUserID,
		); err != nil {
			return fmt.Errorf("seed detection rule %s: %w", rule.Name, err)
		}
	}
	return nil
}

func stringArray(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
