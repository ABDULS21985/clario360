package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// RemediationAuditRepository handles remediation_audit_trail table operations.
type RemediationAuditRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewRemediationAuditRepository creates a new RemediationAuditRepository.
func NewRemediationAuditRepository(db *pgxpool.Pool, logger zerolog.Logger) *RemediationAuditRepository {
	return &RemediationAuditRepository{db: db, logger: logger}
}

// RecordEntry appends an immutable audit entry for a remediation action.
func (r *RemediationAuditRepository) RecordEntry(ctx context.Context, entry *model.RemediationAuditEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	entry.CreatedAt = time.Now().UTC()

	detailsJSON, err := json.Marshal(entry.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}
	var stepAction any
	if entry.StepAction != "" {
		stepAction = entry.StepAction
	}
	var stepResult any
	if entry.StepResult != "" {
		stepResult = entry.StepResult
	}
	var oldStatus any
	if entry.OldStatus != "" {
		oldStatus = entry.OldStatus
	}
	var newStatus any
	if entry.NewStatus != "" {
		newStatus = entry.NewStatus
	}
	var errorMessage any
	if entry.ErrorMessage != "" {
		errorMessage = entry.ErrorMessage
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO remediation_audit_trail (
			id, tenant_id, remediation_id, action, actor_id, actor_name,
			old_status, new_status, step_number, step_action, step_result,
			details, error_message, duration_ms, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		entry.ID, entry.TenantID, entry.RemediationID, entry.Action,
		entry.ActorID, entry.ActorName, oldStatus, newStatus,
		entry.StepNumber, stepAction, stepResult,
		detailsJSON, errorMessage, entry.DurationMs, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record audit entry: %w", err)
	}
	return nil
}

// ListByRemediation retrieves the full audit trail for a remediation action.
func (r *RemediationAuditRepository) ListByRemediation(ctx context.Context, tenantID, remediationID uuid.UUID) ([]model.RemediationAuditEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, remediation_id, action, actor_id, actor_name,
		       old_status, new_status, step_number, step_action, step_result,
		       details, error_message, duration_ms, created_at
		FROM remediation_audit_trail
		WHERE remediation_id=$1 AND tenant_id=$2
		ORDER BY created_at ASC`,
		remediationID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit trail: %w", err)
	}
	defer rows.Close()

	var entries []model.RemediationAuditEntry
	for rows.Next() {
		var entry model.RemediationAuditEntry
		var detailsJSON []byte
		var actorName sql.NullString
		var oldStatus sql.NullString
		var newStatus sql.NullString
		var stepAction sql.NullString
		var stepResult sql.NullString
		var errorMessage sql.NullString
		err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.RemediationID, &entry.Action,
			&entry.ActorID, &actorName, &oldStatus, &newStatus,
			&entry.StepNumber, &stepAction, &stepResult,
			&detailsJSON, &errorMessage, &entry.DurationMs, &entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if actorName.Valid {
			entry.ActorName = actorName.String
		}
		if oldStatus.Valid {
			entry.OldStatus = oldStatus.String
		}
		if newStatus.Valid {
			entry.NewStatus = newStatus.String
		}
		if stepAction.Valid {
			entry.StepAction = stepAction.String
		}
		if stepResult.Valid {
			entry.StepResult = stepResult.String
		}
		if errorMessage.Valid {
			entry.ErrorMessage = errorMessage.String
		}
		if detailsJSON != nil {
			_ = json.Unmarshal(detailsJSON, &entry.Details)
		}
		if entry.Details == nil {
			entry.Details = map[string]interface{}{}
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
