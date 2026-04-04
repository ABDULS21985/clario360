package cti

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Aggregator periodically refreshes CTI dashboard summary tables.
type Aggregator struct {
	repo     Repository
	tenantID uuid.UUID
	interval time.Duration
	logger   zerolog.Logger
}

func NewAggregator(repo Repository, tenantID uuid.UUID, interval time.Duration, logger zerolog.Logger) *Aggregator {
	return &Aggregator{repo: repo, tenantID: tenantID, interval: interval, logger: logger}
}

// Run starts the periodic aggregation loop until the context is cancelled.
func (a *Aggregator) Run(ctx context.Context) error {
	a.logger.Info().Dur("interval", a.interval).Msg("CTI aggregator started")
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	// Run once immediately
	a.refresh(ctx)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info().Msg("CTI aggregator stopped")
			return ctx.Err()
		case <-ticker.C:
			a.refresh(ctx)
		}
	}
}

func (a *Aggregator) refresh(ctx context.Context) {
	now := time.Now().UTC()
	periods := []struct{ start, end time.Time }{
		{now.Add(-24 * time.Hour), now},
		{now.Add(-7 * 24 * time.Hour), now},
		{now.Add(-30 * 24 * time.Hour), now},
	}
	for _, p := range periods {
		if err := a.repo.RefreshGeoThreatSummary(ctx, a.tenantID, p.start, p.end); err != nil {
			a.logger.Error().Err(err).Msg("refresh geo summary")
		}
		if err := a.repo.RefreshSectorThreatSummary(ctx, a.tenantID, p.start, p.end); err != nil {
			a.logger.Error().Err(err).Msg("refresh sector summary")
		}
	}
	if err := a.repo.RefreshExecutiveSnapshot(ctx, a.tenantID); err != nil {
		a.logger.Error().Err(err).Msg("refresh executive snapshot")
	}
	a.logger.Debug().Msg("CTI aggregation refresh completed")
}
