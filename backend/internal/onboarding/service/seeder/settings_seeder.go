package seeder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsSeeder struct {
	pool *pgxpool.Pool
}

func NewSettingsSeeder(pool *pgxpool.Pool) *SettingsSeeder {
	return &SettingsSeeder{pool: pool}
}

func (s *SettingsSeeder) Seed(ctx context.Context, tenantID, adminUserID uuid.UUID) error {
	settings := []struct {
		Key         string
		Value       any
		Description string
	}{
		{"general.timezone", map[string]any{"value": "Asia/Riyadh"}, "Default tenant timezone."},
		{"general.date_format", map[string]any{"value": "DD/MM/YYYY"}, "Default date format."},
		{"general.language", map[string]any{"value": "en"}, "Default language."},
		{"security.password_policy", map[string]any{"min_length": 12, "require_uppercase": true, "require_lowercase": true, "require_digit": true, "require_special": true}, "Default password policy."},
		{"security.session_timeout_minutes", map[string]any{"value": 30}, "Session timeout in minutes."},
		{"security.mfa_required", map[string]any{"value": false}, "Whether MFA is mandatory."},
		{"notification.email_enabled", map[string]any{"value": true}, "Email notifications enabled."},
		{"notification.in_app_enabled", map[string]any{"value": true}, "In-app notifications enabled."},
		{"data.pii_detection_enabled", map[string]any{"value": true}, "PII detection enabled."},
		{"cyber.auto_remediation_enabled", map[string]any{"value": false}, "Automatic remediation disabled by default."},
	}
	for _, item := range settings {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO system_settings (tenant_id, key, value, description, updated_by)
			VALUES ($1, $2, $3::jsonb, $4, $5)
			ON CONFLICT (tenant_id, key) DO NOTHING`,
			tenantID,
			item.Key,
			marshalJSON(item.Value),
			item.Description,
			adminUserID,
		); err != nil {
			return fmt.Errorf("seed system setting %s: %w", item.Key, err)
		}
	}
	return nil
}

func marshalJSON(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return payload
}
