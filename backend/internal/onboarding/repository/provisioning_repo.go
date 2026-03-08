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

type ProvisioningRepository struct {
	pool *pgxpool.Pool
}

func NewProvisioningRepository(pool *pgxpool.Pool) *ProvisioningRepository {
	return &ProvisioningRepository{pool: pool}
}

func (r *ProvisioningRepository) Initialize(ctx context.Context, tenantID, onboardingID uuid.UUID, stepNames []string) error {
	for idx, name := range stepNames {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO provisioning_steps (
				tenant_id, onboarding_id, step_number, step_name, status, idempotency_key
			) VALUES ($1, $2, $3, $4, 'pending', $5)
			ON CONFLICT (onboarding_id, step_number) DO NOTHING`,
			tenantID,
			onboardingID,
			idx+1,
			name,
			fmt.Sprintf("%s:%d", tenantID.String(), idx+1),
		); err != nil {
			return err
		}
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE tenant_onboarding
		SET provisioning_status = 'provisioning',
		    provisioning_started_at = COALESCE(provisioning_started_at, now()),
		    provisioning_error = NULL,
		    updated_at = now()
		WHERE tenant_id = $1`,
		tenantID,
	)
	return err
}

func (r *ProvisioningRepository) ListSteps(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.ProvisioningStep, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, onboarding_id, step_number, step_name, status, started_at, completed_at,
		       duration_ms, error_message, retry_count, idempotency_key, metadata, created_at
		FROM provisioning_steps
		WHERE tenant_id = $1
		ORDER BY step_number`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]onboardingmodel.ProvisioningStep, 0)
	for rows.Next() {
		item, scanErr := scanProvisioningStep(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *ProvisioningRepository) StartStep(ctx context.Context, tenantID uuid.UUID, stepNumber int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE provisioning_steps
		SET status = 'running',
		    started_at = now(),
		    completed_at = NULL,
		    duration_ms = NULL,
		    error_message = NULL,
		    retry_count = CASE WHEN status = 'failed' THEN retry_count + 1 ELSE retry_count END
		WHERE tenant_id = $1 AND step_number = $2`,
		tenantID,
		stepNumber,
	)
	return err
}

func (r *ProvisioningRepository) CompleteStep(ctx context.Context, tenantID uuid.UUID, stepNumber int, metadata map[string]any) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE provisioning_steps
		SET status = 'completed',
		    completed_at = now(),
		    duration_ms = GREATEST(EXTRACT(EPOCH FROM (now() - COALESCE(started_at, now()))) * 1000, 0)::bigint,
		    error_message = NULL,
		    metadata = COALESCE($3::jsonb, '{}'::jsonb)
		WHERE tenant_id = $1 AND step_number = $2`,
		tenantID,
		stepNumber,
		marshalJSON(metadata),
	)
	return err
}

func (r *ProvisioningRepository) FailStep(ctx context.Context, tenantID uuid.UUID, stepNumber int, errMessage string, metadata map[string]any) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE provisioning_steps
		SET status = 'failed',
		    completed_at = now(),
		    duration_ms = GREATEST(EXTRACT(EPOCH FROM (now() - COALESCE(started_at, now()))) * 1000, 0)::bigint,
		    error_message = $3,
		    metadata = COALESCE($4::jsonb, '{}'::jsonb)
		WHERE tenant_id = $1 AND step_number = $2`,
		tenantID,
		stepNumber,
		errMessage,
		marshalJSON(metadata),
	)
	return err
}

func (r *ProvisioningRepository) MarkFailed(ctx context.Context, tenantID uuid.UUID, errMessage string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tenant_onboarding
		SET provisioning_status = 'failed',
		    provisioning_error = $2,
		    updated_at = now()
		WHERE tenant_id = $1`,
		tenantID,
		errMessage,
	)
	return err
}

func (r *ProvisioningRepository) MarkCompleted(ctx context.Context, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tenant_onboarding
		SET provisioning_status = 'completed',
		    provisioning_completed_at = now(),
		    provisioning_error = NULL,
		    updated_at = now()
		WHERE tenant_id = $1`,
		tenantID,
	)
	return err
}

func (r *ProvisioningRepository) GetStatus(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.ProvisioningStatus, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT tenant_id, provisioning_status, provisioning_started_at, provisioning_completed_at, provisioning_error
		FROM tenant_onboarding
		WHERE tenant_id = $1`,
		tenantID,
	)
	status := &onboardingmodel.ProvisioningStatus{}
	if err := row.Scan(&status.TenantID, &status.Status, &status.StartedAt, &status.CompletedAt, &status.Error); err != nil {
		return nil, err
	}
	steps, err := r.ListSteps(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	status.Steps = steps
	status.TotalSteps = len(steps)
	for idx := range steps {
		if steps[idx].Status == onboardingmodel.ProvisioningStepCompleted {
			status.CompletedStep++
		}
		if status.CurrentStep == nil && steps[idx].Status == onboardingmodel.ProvisioningStepRunning {
			current := steps[idx]
			status.CurrentStep = &current
		}
		if status.CurrentStep == nil && (steps[idx].Status == onboardingmodel.ProvisioningStepPending || steps[idx].Status == onboardingmodel.ProvisioningStepFailed) {
			current := steps[idx]
			status.CurrentStep = &current
		}
	}
	if status.TotalSteps > 0 {
		status.ProgressPct = int(float64(status.CompletedStep) / float64(status.TotalSteps) * 100)
	}
	return status, nil
}

func (r *ProvisioningRepository) GetOnboardingID(ctx context.Context, tenantID uuid.UUID) (uuid.UUID, error) {
	var onboardingID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id
		FROM tenant_onboarding
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&onboardingID)
	return onboardingID, err
}

func (r *ProvisioningRepository) SetTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tenants
		SET status = $2, updated_at = now()
		WHERE id = $1`,
		tenantID,
		status,
	)
	return err
}
