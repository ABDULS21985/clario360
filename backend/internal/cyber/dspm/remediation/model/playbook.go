package model

import (
	"time"

	"github.com/google/uuid"
)

// StepAction enumerates the types of actions a playbook step can perform.
type StepAction string

const (
	StepActionEncryptAtRest    StepAction = "encrypt_at_rest"
	StepActionEncryptInTransit StepAction = "encrypt_in_transit"
	StepActionRevokeAccess     StepAction = "revoke_access"
	StepActionDowngradeAccess  StepAction = "downgrade_access"
	StepActionRestrictNetwork  StepAction = "restrict_network"
	StepActionEnableAuditLog   StepAction = "enable_audit_logging"
	StepActionConfigureBackup  StepAction = "configure_backup"
	StepActionCreateTicket     StepAction = "create_itsm_ticket"
	StepActionNotifyOwner      StepAction = "notify_asset_owner"
	StepActionQuarantine       StepAction = "quarantine_asset"
	StepActionReclassify       StepAction = "reclassify_data"
	StepActionScheduleReview   StepAction = "schedule_access_review"
	StepActionArchiveData      StepAction = "archive_data"
	StepActionDeleteData       StepAction = "delete_data"
)

// StepStatus is the status of a playbook step.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// FailureHandling defines what to do when a step fails.
type FailureHandling string

const (
	FailureHandlingSkip  FailureHandling = "skip"
	FailureHandlingAbort FailureHandling = "abort"
	FailureHandlingRetry FailureHandling = "retry(3)"
)

// Playbook defines a sequence of remediation steps for a specific finding type.
type Playbook struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	FindingType      FindingType    `json:"finding_type"`
	Steps            []PlaybookStep `json:"steps"`
	EstimatedMinutes int            `json:"estimated_minutes"`
	RequiresApproval bool           `json:"requires_approval"`
	AutoRollback     bool           `json:"auto_rollback"`
}

// PlaybookStep defines a single step within a playbook.
type PlaybookStep struct {
	ID              string          `json:"id"`
	Order           int             `json:"order"`
	Action          StepAction      `json:"action"`
	Description     string          `json:"description"`
	Params          map[string]any  `json:"params,omitempty"`
	Guidance        string          `json:"guidance"`
	Timeout         time.Duration   `json:"timeout"`
	SuccessCriteria string          `json:"success_criteria"`
	FailureHandling FailureHandling `json:"failure_handling"`
	IsManual        bool            `json:"is_manual"`
}

// StepResult records the outcome of executing a playbook step.
type StepResult struct {
	StepID      string                 `json:"step_id"`
	Action      string                 `json:"action"`
	Status      StepStatus             `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	DurationMs  int64                  `json:"duration_ms"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// DryRunResult is the outcome of validating a playbook against a remediation target.
type DryRunResult struct {
	Valid                bool             `json:"valid"`
	Issues               []string         `json:"issues,omitempty"`
	AssetsAffected       int              `json:"assets_affected"`
	IdentitiesAffected   int              `json:"identities_affected"`
	EstimatedRiskReduction float64        `json:"estimated_risk_reduction"`
	ConflictingRemediations []uuid.UUID   `json:"conflicting_remediations,omitempty"`
}
