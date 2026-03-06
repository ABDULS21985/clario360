package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/channel"
	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
)

// DeliveryResult describes the outcome of delivering to a single channel.
type DeliveryResult struct {
	Channel  string
	Success  bool
	Error    error
	Deferred bool
	Metadata map[string]interface{}
}

// DispatcherService fans out notification delivery to all enabled channels concurrently.
type DispatcherService struct {
	channels    map[string]channel.Channel
	deliveryRepo *repository.DeliveryRepository
	logger      zerolog.Logger
}

// NewDispatcherService creates a new DispatcherService.
func NewDispatcherService(channels map[string]channel.Channel, deliveryRepo *repository.DeliveryRepository, logger zerolog.Logger) *DispatcherService {
	return &DispatcherService{
		channels:     channels,
		deliveryRepo: deliveryRepo,
		logger:       logger.With().Str("component", "dispatcher").Logger(),
	}
}

// Dispatch delivers a notification to all specified channels concurrently.
func (d *DispatcherService) Dispatch(ctx context.Context, notif *model.Notification, deliveries []channel.ChannelDelivery) []DeliveryResult {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var mu sync.Mutex
	var results []DeliveryResult

	sem := make(chan struct{}, 4) // bounded concurrency
	var wg sync.WaitGroup

	for _, delivery := range deliveries {
		delivery := delivery
		wg.Add(1)

		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := d.deliverToChannel(ctx, notif, delivery)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

func (d *DispatcherService) deliverToChannel(ctx context.Context, notif *model.Notification, delivery channel.ChannelDelivery) DeliveryResult {
	start := time.Now()
	result := DeliveryResult{Channel: delivery.Channel}

	if delivery.Deferred {
		// Create delivery log with pending status.
		rec := &model.DeliveryRecord{
			NotificationID: notif.ID,
			Channel:        delivery.Channel,
			Status:         model.DeliveryPending,
			Attempt:        1,
		}
		if id, err := d.deliveryRepo.Insert(ctx, rec); err != nil {
			d.logger.Error().Err(err).Str("channel", delivery.Channel).Msg("failed to create deferred delivery log")
		} else {
			result.Metadata = map[string]interface{}{"delivery_log_id": id}
		}
		result.Deferred = true
		result.Success = true
		metrics.DeliveriesTotal.WithLabelValues(delivery.Channel, "deferred").Inc()
		return result
	}

	ch, ok := d.channels[delivery.Channel]
	if !ok {
		d.logger.Error().Str("channel", delivery.Channel).Msg("unknown channel")
		result.Error = fmt.Errorf("unknown channel: %s", delivery.Channel)
		return result
	}

	chResult := ch.Send(ctx, notif)

	duration := time.Since(start)
	metrics.DeliveryDuration.WithLabelValues(delivery.Channel).Observe(duration.Seconds())

	now := time.Now().UTC()
	rec := &model.DeliveryRecord{
		NotificationID: notif.ID,
		Channel:        delivery.Channel,
		Attempt:        1,
	}

	if chResult.Success {
		rec.Status = model.DeliveryDelivered
		rec.DeliveredAt = &now
		result.Success = true
		metrics.DeliveriesTotal.WithLabelValues(delivery.Channel, "delivered").Inc()
	} else {
		rec.Status = model.DeliveryFailed
		if chResult.Error != nil {
			errMsg := chResult.Error.Error()
			rec.ErrorMessage = &errMsg
			result.Error = chResult.Error
		}
		metrics.DeliveriesTotal.WithLabelValues(delivery.Channel, "failed").Inc()
	}

	result.Metadata = chResult.Metadata

	if id, err := d.deliveryRepo.Insert(ctx, rec); err != nil {
		d.logger.Error().Err(err).Str("channel", delivery.Channel).Msg("failed to create delivery log")
	} else if result.Metadata != nil {
		result.Metadata["delivery_log_id"] = id
	}

	return result
}
