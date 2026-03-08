package seeder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	visuskpi "github.com/clario360/platform/internal/visus/kpi"
	visusmodel "github.com/clario360/platform/internal/visus/model"
)

type KPISeeder struct {
	pool *pgxpool.Pool
}

func NewKPISeeder(pool *pgxpool.Pool) *KPISeeder {
	return &KPISeeder{pool: pool}
}

func (s *KPISeeder) Seed(ctx context.Context, tenantID, adminUserID uuid.UUID) error {
	definitions := visuskpi.DefaultDefinitions(tenantID, adminUserID)
	for idx := range definitions {
		definitions[idx].SnapshotFrequency = visusmodel.KPIFrequencyHour
		if definitions[idx].Name == "Mean Time to Respond" {
			definitions[idx].Name = "MTTR"
			definitions[idx].Description = "Mean time to respond"
		}
		if definitions[idx].Name == "MITRE ATT&CK Coverage" {
			definitions[idx].Name = "MITRE Coverage"
			definitions[idx].Description = "MITRE ATT&CK coverage"
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO visus_kpi_definitions (
				tenant_id, name, description, category, suite, icon, query_endpoint, query_params,
				value_path, unit, format_pattern, target_value, warning_threshold, critical_threshold,
				direction, calculation_type, calculation_window, snapshot_frequency, enabled, is_default,
				tags, created_by
			)
			SELECT
				$1,$2,$3,$4,$5,$6,$7,$8::jsonb,
				$9,$10,$11,$12,$13,$14,
				$15,$16,$17,$18,$19,$20,
				$21,$22
			WHERE NOT EXISTS (
				SELECT 1
				FROM visus_kpi_definitions
				WHERE tenant_id = $1
				  AND name = $2
				  AND deleted_at IS NULL
			)`,
			definitions[idx].TenantID,
			definitions[idx].Name,
			definitions[idx].Description,
			string(definitions[idx].Category),
			string(definitions[idx].Suite),
			definitions[idx].Icon,
			definitions[idx].QueryEndpoint,
			marshalJSON(definitions[idx].QueryParams),
			definitions[idx].ValuePath,
			string(definitions[idx].Unit),
			definitions[idx].FormatPattern,
			definitions[idx].TargetValue,
			definitions[idx].WarningThreshold,
			definitions[idx].CriticalThreshold,
			string(definitions[idx].Direction),
			string(definitions[idx].CalculationType),
			definitions[idx].CalculationWindow,
			string(definitions[idx].SnapshotFrequency),
			definitions[idx].Enabled,
			definitions[idx].IsDefault,
			stringArray(definitions[idx].Tags),
			definitions[idx].CreatedBy,
		); err != nil {
			return fmt.Errorf("seed kpi %s: %w", definitions[idx].Name, err)
		}
	}
	return nil
}
