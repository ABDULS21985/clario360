package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	intrepo "github.com/clario360/platform/internal/integration/repository"
)

type DeliveryWorker struct {
	deliveryService *DeliveryService
	deliveryRepo    *intrepo.DeliveryRepository
	logger          zerolog.Logger
	ticker          *time.Ticker
	batchSize       int
}

func NewDeliveryWorker(deliveryService *DeliveryService, deliveryRepo *intrepo.DeliveryRepository, logger zerolog.Logger) *DeliveryWorker {
	return &DeliveryWorker{
		deliveryService: deliveryService,
		deliveryRepo:    deliveryRepo,
		logger:          logger.With().Str("component", "integration_delivery_worker").Logger(),
		ticker:          time.NewTicker(10 * time.Second),
		batchSize:       50,
	}
}

func (w *DeliveryWorker) Run(ctx context.Context) error {
	defer w.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.ticker.C:
			records, err := w.deliveryRepo.ListDue(ctx, w.batchSize)
			if err != nil {
				w.logger.Warn().Err(err).Msg("failed to load due integration deliveries")
				continue
			}
			success := 0
			failed := 0
			for idx := range records {
				if err := w.deliveryService.Process(ctx, &records[idx]); err != nil {
					failed++
					w.logger.Warn().Err(err).Str("delivery_id", records[idx].ID).Msg("integration delivery attempt failed")
					continue
				}
				success++
			}
			if len(records) > 0 {
				w.logger.Info().
					Int("processed", len(records)).
					Int("success", success).
					Int("failed", failed).
					Msg("integration retry worker processed batch")
			}
		}
	}
}
