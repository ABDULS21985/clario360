package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FindingType enumerates the types of findings that can trigger a remediation.
type FindingType string

const (
	FindingPostureGap           FindingType = "posture_gap"
	FindingOverprivilegedAccess FindingType = "overprivileged_access"
	FindingStaleAccess          FindingType = "stale_access"
	FindingClassificationDrift  FindingType = "classification_drift"
	FindingShadowCopy           FindingType = "shadow_copy"
	FindingPolicyViolation      FindingType = "policy_violation"
	FindingEncryptionMissing    FindingType = "encryption_missing"
	FindingExposureRisk         FindingType = "exposure_risk"
	FindingPIIUnprotected       FindingType = "pii_unprotected"
	FindingRetentionExpired     FindingType = "retention_expired"
	FindingBlastRadiusExcessive FindingType = "blast_radius_excessive"
)

// ValidFindingTypes returns all valid finding type values.
func ValidFindingTypes() []FindingType {
	return []FindingType{
		FindingPostureGap, FindingOverprivilegedAccess, FindingStaleAccess,
		FindingClassificationDrift, FindingShadowCopy, FindingPolicyViolation,
		FindingEncryptionMissing, FindingExposureRisk, FindingPIIUnprotected,
		FindingRetentionExpired, FindingBlastRadiusExcessive,
	}
}

// IsValid returns true if the finding type is a known value.
func (f FindingType) IsValid() bool {
	for _, v := range ValidFindingTypes() {
		if f == v {
			return true
		}
	}
	return false
}

// RemediationStatus tracks the lifecycle state of a remediation item.
type RemediationStatus string

const (
	StatusOpen              RemediationStatus = "open"
	StatusInProgress        RemediationStatus = "in_progress"
	StatusAwaitingApproval  RemediationStatus = "awaiting_approval"
	StatusCompleted         RemediationStatus = "completed"
	StatusFailed            RemediationStatus = "failed"
	StatusCancelled         RemediationStatus = "cancelled"
	StatusRolledBack        RemediationStatus = "rolled_back"
	StatusExceptionGranted  RemediationStatus = "exception_granted"
)

// ValidStatuses returns all valid remediation statuses.
func ValidStatuses() []RemediationStatus {
	return []RemediationStatus{
		StatusOpen, StatusInProgress, StatusAwaitingApproval,
		StatusCompleted, StatusFailed, StatusCancelled,
		StatusRolledBack, StatusExceptionGranted,
	}
}

// IsValid returns true if the status is a known value.
func (s RemediationStatus) IsValid() bool {
	for _, v := range ValidStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the status is a terminal state.
func (s RemediationStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusCancelled, StatusRolledBack, StatusExceptionGranted:
		return true
	}
	return false
}

// Remediation is the primary work item for tracking DSPM remediation actions.
type Remediation struct {
	ID            uuid.UUID         `json:"id" db:"id"`
	TenantID      uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	FindingType   FindingType       `json:"finding_type" db:"finding_type"`
	FindingID     *uuid.UUID        `json:"finding_id,omitempty" db:"finding_id"`
	DataAssetID   *uuid.UUID        `json:"data_asset_id,omitempty" db:"data_asset_id"`
	DataAssetName string            `json:"data_asset_name,omitempty" db:"data_asset_name"`
	IdentityID    string            `json:"identity_id,omitempty" db:"identity_id"`
	PlaybookID    string            `json:"playbook_id" db:"playbook_id"`
	Title         string            `json:"title" db:"title"`
	Description   string            `json:"description" db:"description"`
	Severity      string            `json:"severity" db:"severity"`
	Steps         json.RawMessage   `json:"steps" db:"steps"`
	CurrentStep   int               `json:"current_step" db:"current_step"`
	TotalSteps    int               `json:"total_steps" db:"total_steps"`
	AssignedTo    *uuid.UUID        `json:"assigned_to,omitempty" db:"assigned_to"`
	AssignedTeam  string            `json:"assigned_team,omitempty" db:"assigned_team"`
	SLADueAt      *time.Time        `json:"sla_due_at,omitempty" db:"sla_due_at"`
	SLABreached   bool              `json:"sla_breached" db:"sla_breached"`
	RiskScoreBefore *float64        `json:"risk_score_before,omitempty" db:"risk_score_before"`
	RiskScoreAfter  *float64        `json:"risk_score_after,omitempty" db:"risk_score_after"`
	RiskReduction   *float64        `json:"risk_reduction,omitempty" db:"risk_reduction"`
	PreActionState  json.RawMessage `json:"pre_action_state,omitempty" db:"pre_action_state"`
	RollbackAvailable bool          `json:"rollback_available" db:"rollback_available"`
	RolledBack      bool            `json:"rolled_back" db:"rolled_back"`
	Status          RemediationStatus `json:"status" db:"status"`
	CyberAlertID    *uuid.UUID      `json:"cyber_alert_id,omitempty" db:"cyber_alert_id"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	ComplianceTags  json.RawMessage `json:"compliance_tags" db:"compliance_tags"`
}

// RemediationStep describes one step within a remediation playbook execution.
type RemediationStep struct {
	StepID      string                 `json:"step_id"`
	Order       int                    `json:"order"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Status      string                 `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// RemediationStats aggregates remediation metrics for the dashboard.
type RemediationStats struct {
	TotalOpen            int            `json:"total_open"`
	TotalCriticalOpen    int            `json:"total_critical_open"`
	TotalInProgress      int            `json:"total_in_progress"`
	CompletedLast7Days   int            `json:"completed_last_7_days"`
	SLABreaches          int            `json:"sla_breaches"`
	AvgResolutionHours   float64        `json:"avg_resolution_hours"`
	ByStatus             map[string]int `json:"by_status"`
	BySeverity           map[string]int `json:"by_severity"`
	ByFindingType        map[string]int `json:"by_finding_type"`
	TotalRiskReduction   float64        `json:"total_risk_reduction"`
}

// RemediationDashboard aggregates all dashboard KPIs for the remediation module.
type RemediationDashboard struct {
	Stats              RemediationStats      `json:"stats"`
	RecentRemediations []Remediation         `json:"recent_remediations"`
	BurndownData       []BurndownDataPoint   `json:"burndown_data"`
}

// BurndownDataPoint is one data point in the remediation burndown chart.
type BurndownDataPoint struct {
	Date   string `json:"date"`
	Open   int    `json:"open"`
	Closed int    `json:"closed"`
}

// SLAConfig defines severity-to-SLA-hours mappings.
type SLAConfig struct {
	Critical int `json:"critical"` // hours
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// DefaultSLAConfig returns enterprise-standard SLA configuration.
func DefaultSLAConfig() SLAConfig {
	return SLAConfig{
		Critical: 4,
		High:     24,
		Medium:   72,
		Low:      168, // 7 days
	}
}

// SLAHoursForSeverity returns the SLA deadline in hours for a given severity.
func (c SLAConfig) SLAHoursForSeverity(severity string) int {
	switch severity {
	case "critical":
		return c.Critical
	case "high":
		return c.High
	case "medium":
		return c.Medium
	case "low":
		return c.Low
	default:
		return c.Medium
	}
}
