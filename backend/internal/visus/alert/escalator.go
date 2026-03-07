package alert

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type Escalator struct {
	alerts *repository.AlertRepository
}

func NewEscalator(alerts *repository.AlertRepository) *Escalator {
	return &Escalator{alerts: alerts}
}

func (e *Escalator) EscalateStaleCritical(ctx context.Context, tenantID uuid.UUID, actorID *uuid.UUID) error {
	alerts, _, err := e.alerts.List(ctx, tenantID, repository.AlertListFilters{
		Status:   []string{"new", "viewed", "acknowledged"},
		Severity: []string{string(model.AlertSeverityCritical)},
	}, 1, 200)
	if err != nil {
		return err
	}
	for _, item := range alerts {
		if time.Since(item.CreatedAt) < 24*time.Hour {
			continue
		}
		if _, err := e.alerts.UpdateStatus(ctx, tenantID, item.ID, model.AlertStatusEscalated, actorID, nil, nil); err != nil {
			return err
		}
	}
	return nil
}
