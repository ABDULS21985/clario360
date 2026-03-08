package remediation

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// remediationRepo abstracts the remediation repository methods used by the executor.
type remediationRepo interface {
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.RemediationStatus, fields map[string]interface{}) error
}

// alertRepo abstracts the alert repository methods used by the executor.
type alertRepo interface {
	GetByID(ctx context.Context, tenantID, alertID uuid.UUID) (*model.Alert, error)
	UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, status model.AlertStatus, notes, reason *string) (*model.Alert, error)
	Create(ctx context.Context, alert *model.Alert) (*model.Alert, error)
}

// vulnerabilityRepo abstracts the vulnerability repository methods used by the executor.
type vulnerabilityRepo interface {
	GetByID(ctx context.Context, tenantID, vulnID uuid.UUID) (*model.Vulnerability, error)
	UpdateStatusGlobal(ctx context.Context, tenantID, vulnID uuid.UUID, status string, notes *string) (*model.Vulnerability, error)
}

// auditRecorder abstracts the audit trail methods used by the executor.
type auditRecorder interface {
	RecordTransition(ctx context.Context, tenantID, remediationID uuid.UUID, action string, actorID *uuid.UUID, actorName string, oldStatus, newStatus model.RemediationStatus, details map[string]interface{})
	RecordStep(ctx context.Context, tenantID, remediationID uuid.UUID, stepNum int, stepAction, stepResult string, durationMs int64, errMsg string, details map[string]interface{})
	RecordAction(ctx context.Context, tenantID, remediationID uuid.UUID, action string, actorID *uuid.UUID, actorName string, details map[string]interface{})
}
