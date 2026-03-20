package pipeline

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type RunLogger struct {
	repo *repository.PipelineRunLogRepository
}

func NewRunLogger(repo *repository.PipelineRunLogRepository) *RunLogger {
	return &RunLogger{repo: repo}
}

func (l *RunLogger) Debug(ctx context.Context, tenantID, runID uuid.UUID, phase, message string, details any) {
	l.log(ctx, tenantID, runID, "debug", phase, message, details)
}

func (l *RunLogger) Info(ctx context.Context, tenantID, runID uuid.UUID, phase, message string, details any) {
	l.log(ctx, tenantID, runID, "info", phase, message, details)
}

func (l *RunLogger) Warn(ctx context.Context, tenantID, runID uuid.UUID, phase, message string, details any) {
	l.log(ctx, tenantID, runID, "warn", phase, message, details)
}

func (l *RunLogger) Error(ctx context.Context, tenantID, runID uuid.UUID, phase, message string, details any) {
	l.log(ctx, tenantID, runID, "error", phase, message, details)
}

func (l *RunLogger) log(ctx context.Context, tenantID, runID uuid.UUID, level, phase, message string, details any) {
	var payload json.RawMessage
	if details != nil {
		payload, _ = json.Marshal(details)
	}
	_ = l.repo.Create(ctx, &model.PipelineRunLog{
		ID:        uuid.New(),
		TenantID:  tenantID,
		RunID:     runID,
		Level:     level,
		Phase:     phase,
		Message:   message,
		Details:   payload,
		CreatedAt: time.Now().UTC(),
	})
}

