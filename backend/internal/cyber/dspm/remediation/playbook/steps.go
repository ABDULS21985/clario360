package playbook

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// StepExecutor defines the interface for executing a single playbook step action.
type StepExecutor interface {
	Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error)
}

// baseExecutor provides shared timing and result construction logic for all step executors.
type baseExecutor struct {
	logger zerolog.Logger
}

func (b *baseExecutor) buildResult(step *model.PlaybookStep, start time.Time, resultMap map[string]interface{}) *model.StepResult {
	now := time.Now()
	return &model.StepResult{
		StepID:      step.ID,
		Action:      string(step.Action),
		Status:      model.StepStatusCompleted,
		StartedAt:   start,
		CompletedAt: &now,
		DurationMs:  now.Sub(start).Milliseconds(),
		Result:      resultMap,
	}
}

func (b *baseExecutor) buildError(step *model.PlaybookStep, start time.Time, err error) *model.StepResult {
	now := time.Now()
	return &model.StepResult{
		StepID:      step.ID,
		Action:      string(step.Action),
		Status:      model.StepStatusFailed,
		StartedAt:   start,
		CompletedAt: &now,
		DurationMs:  now.Sub(start).Milliseconds(),
		Error:       err.Error(),
	}
}

// EncryptAtRestExecutor simulates applying at-rest encryption to a data asset.
type EncryptAtRestExecutor struct {
	baseExecutor
}

func NewEncryptAtRestExecutor(logger zerolog.Logger) *EncryptAtRestExecutor {
	return &EncryptAtRestExecutor{baseExecutor{logger: logger.With().Str("executor", "encrypt_at_rest").Logger()}}
}

func (e *EncryptAtRestExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("applying at-rest encryption")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	algorithm := "AES-256-GCM"
	if v, ok := step.Params["algorithm"]; ok {
		if s, ok := v.(string); ok && s != "" {
			algorithm = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"encryption_applied": true,
		"algorithm":          algorithm,
		"key_management":     "platform-managed-kms",
		"scope":              "all_columns",
		"previous_state":     "unencrypted",
		"new_state":          "encrypted_at_rest",
		"action_summary":     fmt.Sprintf("Applied %s encryption at rest via platform KMS", algorithm),
	}), nil
}

// EncryptInTransitExecutor simulates enabling in-transit encryption for a data asset.
type EncryptInTransitExecutor struct {
	baseExecutor
}

func NewEncryptInTransitExecutor(logger zerolog.Logger) *EncryptInTransitExecutor {
	return &EncryptInTransitExecutor{baseExecutor{logger: logger.With().Str("executor", "encrypt_in_transit").Logger()}}
}

func (e *EncryptInTransitExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("enabling in-transit encryption")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	protocol := "TLS-1.3"
	if v, ok := step.Params["protocol"]; ok {
		if s, ok := v.(string); ok && s != "" {
			protocol = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"encryption_applied": true,
		"protocol":           protocol,
		"certificate_type":   "auto-provisioned",
		"min_tls_version":    "1.2",
		"enforced_tls":       protocol,
		"previous_state":     "plaintext_transport",
		"new_state":          "encrypted_in_transit",
		"action_summary":     fmt.Sprintf("Enforced %s for all data-in-transit connections", protocol),
	}), nil
}

// RevokeAccessExecutor simulates revoking access permissions from an identity or group.
type RevokeAccessExecutor struct {
	baseExecutor
}

func NewRevokeAccessExecutor(logger zerolog.Logger) *RevokeAccessExecutor {
	return &RevokeAccessExecutor{baseExecutor{logger: logger.With().Str("executor", "revoke_access").Logger()}}
}

func (e *RevokeAccessExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("revoking access permissions")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	scope := "all_permissions"
	if v, ok := step.Params["scope"]; ok {
		if s, ok := v.(string); ok && s != "" {
			scope = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"access_revoked":    true,
		"scope":             scope,
		"permissions_removed": 12,
		"sessions_terminated": 3,
		"previous_state":    "active_access",
		"new_state":         "access_revoked",
		"rollback_token":    fmt.Sprintf("rbk-%s-%d", step.ID, time.Now().UnixMilli()),
		"action_summary":    fmt.Sprintf("Revoked %s; terminated 3 active sessions", scope),
	}), nil
}

// DowngradeAccessExecutor simulates reducing access privileges to the minimum required level.
type DowngradeAccessExecutor struct {
	baseExecutor
}

func NewDowngradeAccessExecutor(logger zerolog.Logger) *DowngradeAccessExecutor {
	return &DowngradeAccessExecutor{baseExecutor{logger: logger.With().Str("executor", "downgrade_access").Logger()}}
}

func (e *DowngradeAccessExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("downgrading access privileges")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	targetLevel := "read_only"
	if v, ok := step.Params["target_level"]; ok {
		if s, ok := v.(string); ok && s != "" {
			targetLevel = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"access_downgraded":     true,
		"previous_level":        "read_write_admin",
		"new_level":             targetLevel,
		"permissions_removed":   8,
		"permissions_retained":  4,
		"previous_state":        "elevated_privileges",
		"new_state":             "least_privilege",
		"rollback_token":        fmt.Sprintf("rbk-%s-%d", step.ID, time.Now().UnixMilli()),
		"action_summary":        fmt.Sprintf("Downgraded access from read_write_admin to %s; removed 8 excess permissions", targetLevel),
	}), nil
}

// RestrictNetworkExecutor simulates applying network access restrictions to a data asset.
type RestrictNetworkExecutor struct {
	baseExecutor
}

func NewRestrictNetworkExecutor(logger zerolog.Logger) *RestrictNetworkExecutor {
	return &RestrictNetworkExecutor{baseExecutor{logger: logger.With().Str("executor", "restrict_network").Logger()}}
}

func (e *RestrictNetworkExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("restricting network access")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	targetExposure := "internal_only"
	if v, ok := step.Params["target_exposure"]; ok {
		if s, ok := v.(string); ok && s != "" {
			targetExposure = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"network_restricted":   true,
		"previous_exposure":    "internet_facing",
		"new_exposure":         targetExposure,
		"firewall_rules_added": 3,
		"endpoints_blocked":    5,
		"allowed_cidrs":        []string{"10.0.0.0/8", "172.16.0.0/12"},
		"previous_state":       "publicly_accessible",
		"new_state":            "network_restricted",
		"action_summary":       fmt.Sprintf("Restricted network exposure to %s; blocked 5 public endpoints", targetExposure),
	}), nil
}

// EnableAuditLogExecutor simulates enabling comprehensive audit logging for a data asset.
type EnableAuditLogExecutor struct {
	baseExecutor
}

func NewEnableAuditLogExecutor(logger zerolog.Logger) *EnableAuditLogExecutor {
	return &EnableAuditLogExecutor{baseExecutor{logger: logger.With().Str("executor", "enable_audit_logging").Logger()}}
}

func (e *EnableAuditLogExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("enabling audit logging")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	retentionDays := 90
	if v, ok := step.Params["retention_days"]; ok {
		if n, ok := v.(float64); ok {
			retentionDays = int(n)
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"audit_logging_enabled": true,
		"log_level":             "detailed",
		"events_captured":       []string{"read", "write", "delete", "schema_change", "permission_change"},
		"retention_days":        retentionDays,
		"destination":           "centralized-siem",
		"previous_state":        "minimal_logging",
		"new_state":             "comprehensive_audit_logging",
		"action_summary":        fmt.Sprintf("Enabled comprehensive audit logging with %d-day retention to centralized SIEM", retentionDays),
	}), nil
}

// ConfigureBackupExecutor simulates configuring automated backups for a data asset.
type ConfigureBackupExecutor struct {
	baseExecutor
}

func NewConfigureBackupExecutor(logger zerolog.Logger) *ConfigureBackupExecutor {
	return &ConfigureBackupExecutor{baseExecutor{logger: logger.With().Str("executor", "configure_backup").Logger()}}
}

func (e *ConfigureBackupExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("configuring backup policy")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	schedule := "daily"
	if v, ok := step.Params["schedule"]; ok {
		if s, ok := v.(string); ok && s != "" {
			schedule = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"backup_configured":  true,
		"schedule":           schedule,
		"retention_policy":   "30-day-rolling",
		"backup_type":        "incremental",
		"encryption_enabled": true,
		"cross_region":       true,
		"previous_state":     "no_backup_policy",
		"new_state":          "automated_backup_configured",
		"action_summary":     fmt.Sprintf("Configured %s incremental backups with 30-day rolling retention and cross-region replication", schedule),
	}), nil
}

// CreateTicketExecutor simulates creating an ITSM ticket for tracking the remediation.
type CreateTicketExecutor struct {
	baseExecutor
}

func NewCreateTicketExecutor(logger zerolog.Logger) *CreateTicketExecutor {
	return &CreateTicketExecutor{baseExecutor{logger: logger.With().Str("executor", "create_itsm_ticket").Logger()}}
}

func (e *CreateTicketExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("creating ITSM ticket")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	priority := "high"
	if v, ok := step.Params["priority"]; ok {
		if s, ok := v.(string); ok && s != "" {
			priority = s
		}
	}

	ticketID := fmt.Sprintf("DSPM-%d", time.Now().UnixMilli()%100000)

	return e.buildResult(step, start, map[string]interface{}{
		"ticket_created":    true,
		"ticket_id":         ticketID,
		"ticket_system":     "integrated-itsm",
		"priority":          priority,
		"assigned_queue":    "security-operations",
		"sla_hours":         24,
		"escalation_policy": "auto-escalate-on-breach",
		"action_summary":    fmt.Sprintf("Created %s priority ITSM ticket %s assigned to security-operations queue", priority, ticketID),
	}), nil
}

// NotifyOwnerExecutor simulates sending a notification to the data asset owner.
type NotifyOwnerExecutor struct {
	baseExecutor
}

func NewNotifyOwnerExecutor(logger zerolog.Logger) *NotifyOwnerExecutor {
	return &NotifyOwnerExecutor{baseExecutor{logger: logger.With().Str("executor", "notify_asset_owner").Logger()}}
}

func (e *NotifyOwnerExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("notifying asset owner")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	channels := []string{"email", "in-app"}
	if v, ok := step.Params["channels"]; ok {
		if arr, ok := v.([]interface{}); ok {
			channels = make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					channels = append(channels, s)
				}
			}
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"notification_sent":  true,
		"channels":           channels,
		"recipients":         1,
		"delivery_status":    "delivered",
		"notification_type":  "remediation_required",
		"includes_playbook":  true,
		"response_requested": true,
		"action_summary":     fmt.Sprintf("Notified asset owner via %v; response requested within SLA window", channels),
	}), nil
}

// QuarantineExecutor simulates isolating a data asset from normal access paths.
type QuarantineExecutor struct {
	baseExecutor
}

func NewQuarantineExecutor(logger zerolog.Logger) *QuarantineExecutor {
	return &QuarantineExecutor{baseExecutor{logger: logger.With().Str("executor", "quarantine_asset").Logger()}}
}

func (e *QuarantineExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("quarantining asset")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	return e.buildResult(step, start, map[string]interface{}{
		"quarantine_applied":    true,
		"access_paths_blocked":  6,
		"network_isolated":      true,
		"read_access_preserved": true,
		"admin_bypass_enabled":  true,
		"quarantine_zone":       "security-hold",
		"previous_state":        "normal_access",
		"new_state":             "quarantined",
		"rollback_token":        fmt.Sprintf("rbk-%s-%d", step.ID, time.Now().UnixMilli()),
		"action_summary":        "Quarantined asset: blocked 6 access paths, isolated network, preserved read-only admin access",
	}), nil
}

// ReclassifyExecutor simulates reclassifying a data asset to the correct classification level.
type ReclassifyExecutor struct {
	baseExecutor
}

func NewReclassifyExecutor(logger zerolog.Logger) *ReclassifyExecutor {
	return &ReclassifyExecutor{baseExecutor{logger: logger.With().Str("executor", "reclassify_data").Logger()}}
}

func (e *ReclassifyExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("reclassifying data asset")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	targetClassification := "confidential"
	if v, ok := step.Params["target_classification"]; ok {
		if s, ok := v.(string); ok && s != "" {
			targetClassification = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"reclassification_applied": true,
		"previous_classification":  "internal",
		"new_classification":       targetClassification,
		"pii_detected":             true,
		"phi_detected":             false,
		"labels_updated":           3,
		"dependent_policies":       2,
		"previous_state":           "misclassified",
		"new_state":                "correctly_classified",
		"action_summary":           fmt.Sprintf("Reclassified data asset from internal to %s; updated 3 labels and triggered 2 dependent policies", targetClassification),
	}), nil
}

// ScheduleReviewExecutor simulates scheduling a periodic access review for a data asset.
type ScheduleReviewExecutor struct {
	baseExecutor
}

func NewScheduleReviewExecutor(logger zerolog.Logger) *ScheduleReviewExecutor {
	return &ScheduleReviewExecutor{baseExecutor{logger: logger.With().Str("executor", "schedule_access_review").Logger()}}
}

func (e *ScheduleReviewExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("scheduling access review")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	intervalDays := 30
	if v, ok := step.Params["interval_days"]; ok {
		if n, ok := v.(float64); ok {
			intervalDays = int(n)
		}
	}

	nextReview := time.Now().AddDate(0, 0, intervalDays)

	return e.buildResult(step, start, map[string]interface{}{
		"review_scheduled":    true,
		"interval_days":       intervalDays,
		"next_review_date":    nextReview.Format("2006-01-02"),
		"reviewers_assigned":  2,
		"scope":               "all_access_grants",
		"auto_revoke_on_miss": true,
		"previous_state":      "no_review_scheduled",
		"new_state":           "periodic_review_active",
		"action_summary":      fmt.Sprintf("Scheduled access review every %d days; next review on %s; auto-revoke on missed review", intervalDays, nextReview.Format("2006-01-02")),
	}), nil
}

// ArchiveDataExecutor simulates archiving data that has exceeded retention requirements.
type ArchiveDataExecutor struct {
	baseExecutor
}

func NewArchiveDataExecutor(logger zerolog.Logger) *ArchiveDataExecutor {
	return &ArchiveDataExecutor{baseExecutor{logger: logger.With().Str("executor", "archive_data").Logger()}}
}

func (e *ArchiveDataExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("archiving data")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	archiveTier := "cold-storage"
	if v, ok := step.Params["archive_tier"]; ok {
		if s, ok := v.(string); ok && s != "" {
			archiveTier = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"data_archived":       true,
		"archive_tier":        archiveTier,
		"records_archived":    15420,
		"compressed_size_mb":  128,
		"encryption_applied":  true,
		"retrieval_sla_hours": 24,
		"retention_lock":      true,
		"previous_state":      "active_storage",
		"new_state":           "archived",
		"action_summary":      fmt.Sprintf("Archived 15,420 records to %s; compressed to 128 MB with encryption; 24-hour retrieval SLA", archiveTier),
	}), nil
}

// DeleteDataExecutor simulates securely deleting data that must be removed per policy.
type DeleteDataExecutor struct {
	baseExecutor
}

func NewDeleteDataExecutor(logger zerolog.Logger) *DeleteDataExecutor {
	return &DeleteDataExecutor{baseExecutor{logger: logger.With().Str("executor", "delete_data").Logger()}}
}

func (e *DeleteDataExecutor) Execute(ctx context.Context, step *model.PlaybookStep) (*model.StepResult, error) {
	start := time.Now()
	e.logger.Info().Str("step_id", step.ID).Msg("securely deleting data")

	if err := ctx.Err(); err != nil {
		return e.buildError(step, start, fmt.Errorf("context cancelled before execution: %w", err)), nil
	}

	method := "crypto-shred"
	if v, ok := step.Params["method"]; ok {
		if s, ok := v.(string); ok && s != "" {
			method = s
		}
	}

	return e.buildResult(step, start, map[string]interface{}{
		"data_deleted":           true,
		"deletion_method":        method,
		"records_deleted":        8230,
		"backups_purged":         true,
		"replicas_purged":        3,
		"deletion_certificate":   fmt.Sprintf("DEL-CERT-%d", time.Now().UnixMilli()%100000),
		"verification_performed": true,
		"previous_state":         "active",
		"new_state":              "securely_deleted",
		"action_summary":         fmt.Sprintf("Securely deleted 8,230 records using %s method; purged 3 replicas and backups; deletion certificate issued", method),
	}), nil
}
