package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RemediationType enumerates supported remediation action types.
type RemediationType string

const (
	RemediationTypePatch           RemediationType = "patch"
	RemediationTypeConfigChange    RemediationType = "config_change"
	RemediationTypeBlockIP         RemediationType = "block_ip"
	RemediationTypeIsolateAsset    RemediationType = "isolate_asset"
	RemediationTypeFirewallRule    RemediationType = "firewall_rule"
	RemediationTypeAccessRevoke    RemediationType = "access_revoke"
	RemediationTypeCertRenew       RemediationType = "certificate_renew"
	RemediationTypeCustom          RemediationType = "custom"
)

// RemediationStatus enumerates all valid lifecycle states.
type RemediationStatus string

const (
	StatusDraft               RemediationStatus = "draft"
	StatusPendingApproval     RemediationStatus = "pending_approval"
	StatusApproved            RemediationStatus = "approved"
	StatusRejected            RemediationStatus = "rejected"
	StatusRevisionRequested   RemediationStatus = "revision_requested"
	StatusDryRunRunning       RemediationStatus = "dry_run_running"
	StatusDryRunCompleted     RemediationStatus = "dry_run_completed"
	StatusDryRunFailed        RemediationStatus = "dry_run_failed"
	StatusExecutionPending    RemediationStatus = "execution_pending"
	StatusExecuting           RemediationStatus = "executing"
	StatusExecuted            RemediationStatus = "executed"
	StatusExecutionFailed     RemediationStatus = "execution_failed"
	StatusVerificationPending RemediationStatus = "verification_pending"
	StatusVerified            RemediationStatus = "verified"
	StatusVerificationFailed  RemediationStatus = "verification_failed"
	StatusRollbackPending     RemediationStatus = "rollback_pending"
	StatusRollingBack         RemediationStatus = "rolling_back"
	StatusRolledBack          RemediationStatus = "rolled_back"
	StatusRollbackFailed      RemediationStatus = "rollback_failed"
	StatusClosed              RemediationStatus = "closed"
)

// RemediationPlan describes the steps and properties of the remediation.
type RemediationPlan struct {
	Steps          []RemediationStep `json:"steps"`
	Reversible     bool              `json:"reversible"`
	RequiresReboot bool              `json:"requires_reboot"`
	EstimatedDowntime string        `json:"estimated_downtime"`
	RiskLevel      string            `json:"risk_level"`
	TargetVersion  string            `json:"target_version,omitempty"`
	TargetConfig   map[string]interface{} `json:"target_config,omitempty"`
	BlockTargets   []string          `json:"block_targets,omitempty"` // IPs/CIDRs for block_ip
	IsolateConfig  map[string]interface{} `json:"isolate_config,omitempty"`
}

// RemediationStep is one step within the plan.
type RemediationStep struct {
	Number      int    `json:"number"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Target      string `json:"target,omitempty"`
	Expected    string `json:"expected,omitempty"`
}

// RemediationAction is the core governance record for a remediation lifecycle.
type RemediationAction struct {
	ID                    uuid.UUID         `json:"id"`
	TenantID              uuid.UUID         `json:"tenant_id"`
	AlertID               *uuid.UUID        `json:"alert_id,omitempty"`
	VulnerabilityID       *uuid.UUID        `json:"vulnerability_id,omitempty"`
	AssessmentID          *uuid.UUID        `json:"assessment_id,omitempty"`
	CTEMFindingID         *uuid.UUID        `json:"ctem_finding_id,omitempty"`
	RemediationGroupID    *uuid.UUID        `json:"remediation_group_id,omitempty"`
	Type                  RemediationType   `json:"type"`
	Severity              string            `json:"severity"`
	Title                 string            `json:"title"`
	Description           string            `json:"description"`
	Plan                  RemediationPlan   `json:"plan"`
	AffectedAssetIDs      []uuid.UUID       `json:"affected_asset_ids"`
	AffectedAssetCount    int               `json:"affected_asset_count"`
	ExecutionMode         string            `json:"execution_mode"`
	Status                RemediationStatus `json:"status"`

	// Approval chain
	SubmittedBy          *uuid.UUID `json:"submitted_by,omitempty"`
	SubmittedAt          *time.Time `json:"submitted_at,omitempty"`
	ApprovedBy           *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	RejectedBy           *uuid.UUID `json:"rejected_by,omitempty"`
	RejectedAt           *time.Time `json:"rejected_at,omitempty"`
	RejectionReason      *string    `json:"rejection_reason,omitempty"`
	ApprovalNotes        *string    `json:"approval_notes,omitempty"`
	RequiresApprovalFrom string     `json:"requires_approval_from"`

	// Dry-run
	DryRunResult     *DryRunResult `json:"dry_run_result,omitempty"`
	DryRunAt         *time.Time    `json:"dry_run_at,omitempty"`
	DryRunDurationMs *int64        `json:"dry_run_duration_ms,omitempty"`

	// Execution
	PreExecutionState     json.RawMessage   `json:"pre_execution_state,omitempty"`
	ExecutionResult       *ExecutionResult  `json:"execution_result,omitempty"`
	ExecutedBy            *uuid.UUID        `json:"executed_by,omitempty"`
	ExecutionStartedAt    *time.Time        `json:"execution_started_at,omitempty"`
	ExecutionCompletedAt  *time.Time        `json:"execution_completed_at,omitempty"`
	ExecutionDurationMs   *int64            `json:"execution_duration_ms,omitempty"`

	// Verification
	VerificationResult *VerificationResult `json:"verification_result,omitempty"`
	VerifiedBy         *uuid.UUID          `json:"verified_by,omitempty"`
	VerifiedAt         *time.Time          `json:"verified_at,omitempty"`

	// Rollback
	RollbackResult    *RollbackResult `json:"rollback_result,omitempty"`
	RollbackReason    *string         `json:"rollback_reason,omitempty"`
	RollbackApprovedBy *uuid.UUID     `json:"rollback_approved_by,omitempty"`
	RolledBackAt      *time.Time      `json:"rolled_back_at,omitempty"`
	RollbackDeadline  *time.Time      `json:"rollback_deadline,omitempty"`

	WorkflowInstanceID *uuid.UUID `json:"workflow_instance_id,omitempty"`
	Tags               []string   `json:"tags"`
	Metadata           map[string]interface{} `json:"metadata"`
	CreatedBy          uuid.UUID  `json:"created_by"`
	CreatedByName      *string    `json:"created_by_name,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Populated on demand
	AuditTrail []RemediationAuditEntry `json:"audit_trail,omitempty"`
}

// DryRunResult holds the outcome of a dry-run simulation.
type DryRunResult struct {
	Success          bool              `json:"success"`
	SimulatedChanges []SimulatedChange `json:"simulated_changes"`
	Warnings         []string          `json:"warnings"`
	Blockers         []string          `json:"blockers"`
	EstimatedImpact  ImpactEstimate    `json:"estimated_impact"`
	AffectedServices []string          `json:"affected_services"`
	DurationMs       int64             `json:"duration_ms"`
}

// SimulatedChange describes one change that would occur during execution.
type SimulatedChange struct {
	AssetID       string `json:"asset_id,omitempty"`
	AssetName     string `json:"asset_name,omitempty"`
	ChangeType    string `json:"change_type"`
	Description   string `json:"description"`
	BeforeValue   string `json:"before_value,omitempty"`
	AfterValue    string `json:"after_value,omitempty"`
}

// ImpactEstimate describes expected disruption during execution.
type ImpactEstimate struct {
	Downtime         string `json:"downtime"`
	ServicesAffected int    `json:"services_affected"`
	UsersAffected    int    `json:"users_affected"`
	RiskLevel        string `json:"risk_level"`
	RecommendWindow  string `json:"recommend_window"`
}

// ExecutionResult holds the outcome of remediation execution.
type ExecutionResult struct {
	Success        bool           `json:"success"`
	StepsExecuted  int            `json:"steps_executed"`
	StepsTotal     int            `json:"steps_total"`
	StepResults    []StepResult   `json:"step_results"`
	DurationMs     int64          `json:"duration_ms"`
	ChangesApplied []AppliedChange `json:"changes_applied"`
}

// StepResult describes the outcome of a single execution step.
type StepResult struct {
	StepNumber int    `json:"step_number"`
	Action     string `json:"action"`
	Status     string `json:"status"` // success, failure, skipped
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AppliedChange records what was actually changed during execution.
type AppliedChange struct {
	AssetID     string `json:"asset_id,omitempty"`
	ChangeType  string `json:"change_type"`
	Description string `json:"description"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
}

// VerificationResult holds the outcome of post-execution verification.
type VerificationResult struct {
	Verified      bool               `json:"verified"`
	Checks        []VerificationCheck `json:"checks"`
	FailureReason string             `json:"failure_reason,omitempty"`
	DurationMs    int64              `json:"duration_ms"`
}

// VerificationCheck is one check within verification.
type VerificationCheck struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Notes    string `json:"notes,omitempty"`
}

// RollbackResult holds the outcome of a rollback operation.
type RollbackResult struct {
	Success    bool   `json:"success"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
	StepsReverted int `json:"steps_reverted"`
}

// RemediationAuditEntry is an immutable log line in the remediation audit trail.
type RemediationAuditEntry struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	RemediationID uuid.UUID  `json:"remediation_id"`
	Action        string     `json:"action"`
	ActorID       *uuid.UUID `json:"actor_id,omitempty"`
	ActorName     string     `json:"actor_name,omitempty"`
	OldStatus     string     `json:"old_status,omitempty"`
	NewStatus     string     `json:"new_status,omitempty"`
	StepNumber    *int       `json:"step_number,omitempty"`
	StepAction    string     `json:"step_action,omitempty"`
	StepResult    string     `json:"step_result,omitempty"`
	Details       map[string]interface{} `json:"details"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	DurationMs    *int64     `json:"duration_ms,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// RemediationStats aggregates counts across statuses.
type RemediationStats struct {
	Total               int     `json:"total"`
	Draft               int     `json:"draft"`
	PendingApproval     int     `json:"pending_approval"`
	Approved            int     `json:"approved"`
	DryRunCompleted     int     `json:"dry_run_completed"`
	Executing           int     `json:"executing"`
	Executed            int     `json:"executed"`
	Verified            int     `json:"verified"`
	VerificationFailed  int     `json:"verification_failed"`
	RolledBack          int     `json:"rolled_back"`
	Failed              int     `json:"failed"`
	Closed              int     `json:"closed"`
	AvgExecutionHours   float64 `json:"avg_execution_hours"`
	VerificationSuccessRate float64 `json:"verification_success_rate"`
	RollbackRate        float64 `json:"rollback_rate"`
}
