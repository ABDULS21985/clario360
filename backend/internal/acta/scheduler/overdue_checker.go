package scheduler

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/service"
)

type OverdueChecker struct {
	actionItems *service.ActionItemService
	interval    time.Duration
	logger      zerolog.Logger
}

func NewOverdueChecker(actionItems *service.ActionItemService, interval time.Duration, logger zerolog.Logger) *OverdueChecker {
	return &OverdueChecker{
		actionItems: actionItems,
		interval:    interval,
		logger:      logger.With().Str("component", "acta_overdue_checker").Logger(),
	}
}

func (c *OverdueChecker) Run(ctx context.Context) error {
	if err := c.runOnce(ctx); err != nil {
		c.logger.Error().Err(err).Msg("initial overdue check failed")
	}
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.runOnce(ctx); err != nil {
				c.logger.Error().Err(err).Msg("scheduled overdue check failed")
			}
		}
	}
}

func (c *OverdueChecker) runOnce(ctx context.Context) error {
	return c.actionItems.MarkOverdueItems(ctx)
}
