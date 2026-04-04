package seeder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	iammodel "github.com/clario360/platform/internal/iam/model"
)

type RoleSeeder struct {
	pool *pgxpool.Pool
}

func NewRoleSeeder(pool *pgxpool.Pool) *RoleSeeder {
	return &RoleSeeder{pool: pool}
}

func (s *RoleSeeder) Seed(ctx context.Context, tenantID uuid.UUID) error {
	roles := []iammodel.Role{
		{Name: "Tenant Admin", Slug: "tenant-admin", Description: "Full tenant administration access.", Permissions: []string{"tenant:*", "users:*", "roles:*", "apikeys:*", "cyber:*", "data:*", "acta:*", "lex:*", "visus:*", "reports:*", "files:*", "workflows:*"}},
		{Name: "Security Manager", Slug: "security-manager", Description: "Security operations management.", Permissions: []string{"cyber:*", "alerts:*", "remediation:*", "visus:read"}},
		{Name: "Security Analyst", Slug: "security-analyst", Description: "Security monitoring and triage.", Permissions: []string{"cyber:read", "cyber:write", "alerts:*", "remediation:read"}},
		{Name: "Data Steward", Slug: "data-steward", Description: "Data governance and quality management.", Permissions: []string{"data:read", "data:write", "quality:*", "lineage:*"}},
		{Name: "Data Analyst", Slug: "data-analyst", Description: "Data analysis and reporting.", Permissions: []string{"data:read", "quality:read", "analytics:*", "reports:read"}},
		{Name: "Legal Manager", Slug: "legal-manager", Description: "Legal operations management.", Permissions: []string{"lex:*", "reports:read"}},
		{Name: "Legal Analyst", Slug: "legal-analyst", Description: "Legal analysis and contract operations.", Permissions: []string{"lex:read", "lex:write"}},
		{Name: "Board Secretary", Slug: "board-secretary", Description: "Board governance and meeting administration.", Permissions: []string{"acta:*", "files:*"}},
		{Name: "Executive", Slug: "executive", Description: "Executive cross-suite visibility.", Permissions: []string{"visus:*", "reports:read", "acta:read", "lex:read", "cyber:read", "data:read"}},
		{Name: "Auditor", Slug: "auditor", Description: "Read-only audit and oversight access.", Permissions: []string{"*:read"}},
		{Name: "Viewer", Slug: "viewer", Description: "Read-only tenant access.", Permissions: []string{"*:read"}},
	}
	for _, role := range roles {
		permsJSON, err := json.Marshal(role.Permissions)
		if err != nil {
			return fmt.Errorf("marshal permissions for %s: %w", role.Slug, err)
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO roles (tenant_id, name, slug, description, is_system_role, permissions)
			VALUES ($1, $2, $3, $4, true, $5::jsonb)
			ON CONFLICT (tenant_id, slug)
			DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				is_system_role = true,
				permissions = EXCLUDED.permissions,
				updated_at = now()`,
			tenantID,
			role.Name,
			role.Slug,
			role.Description,
			permsJSON,
		); err != nil {
			return fmt.Errorf("seed role %s: %w", role.Slug, err)
		}
	}
	return nil
}
