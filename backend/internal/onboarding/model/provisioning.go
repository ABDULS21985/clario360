package model

import (
	"time"

	"github.com/google/uuid"
)

type ProvisioningStepStatus string

const (
	ProvisioningStepPending   ProvisioningStepStatus = "pending"
	ProvisioningStepRunning   ProvisioningStepStatus = "running"
	ProvisioningStepCompleted ProvisioningStepStatus = "completed"
	ProvisioningStepFailed    ProvisioningStepStatus = "failed"
	ProvisioningStepSkipped   ProvisioningStepStatus = "skipped"
)

type ProvisioningStep struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	OnboardingID   uuid.UUID              `json:"onboarding_id"`
	StepNumber     int                    `json:"step_number"`
	StepName       string                 `json:"step_name"`
	Status         ProvisioningStepStatus `json:"status"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	DurationMS     *int64                 `json:"duration_ms,omitempty"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
	RetryCount     int                    `json:"retry_count"`
	IdempotencyKey *string                `json:"idempotency_key,omitempty"`
	Metadata       map[string]any         `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

type ProvisioningStatus struct {
	TenantID      uuid.UUID                    `json:"tenant_id"`
	Status        OnboardingProvisioningStatus `json:"status"`
	StartedAt     *time.Time                   `json:"started_at,omitempty"`
	CompletedAt   *time.Time                   `json:"completed_at,omitempty"`
	Error         *string                      `json:"error,omitempty"`
	Steps         []ProvisioningStep           `json:"steps"`
	CurrentStep   *ProvisioningStep            `json:"current_step,omitempty"`
	ProgressPct   int                          `json:"progress_pct"`
	CompletedStep int                          `json:"completed_steps"`
	TotalSteps    int                          `json:"total_steps"`
}
