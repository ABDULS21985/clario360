package alert

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type dedupAlertRepository interface {
	FindDedupMatch(ctx context.Context, tenantID uuid.UUID, dedupKey string, window time.Duration) (*model.ExecutiveAlert, error)
	IncrementOccurrence(ctx context.Context, tenantID, id uuid.UUID) (*model.ExecutiveAlert, error)
}

type Deduplicator struct {
	alerts dedupAlertRepository
}

func NewDeduplicator(alerts dedupAlertRepository) *Deduplicator {
	return &Deduplicator{alerts: alerts}
}

func (d *Deduplicator) CheckAndUpdate(ctx context.Context, tenantID uuid.UUID, dedupKey string, window time.Duration) (*model.ExecutiveAlert, bool, error) {
	existing, err := d.alerts.FindDedupMatch(ctx, tenantID, dedupKey, window)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	updated, err := d.alerts.IncrementOccurrence(ctx, tenantID, existing.ID)
	if err != nil {
		return nil, false, err
	}
	return updated, true, nil
}
