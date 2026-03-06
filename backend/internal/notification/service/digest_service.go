package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
)

// DigestService aggregates notifications and sends periodic digest emails.
type DigestService struct {
	notifRepo  *repository.NotificationRepository
	prefRepo   *repository.PreferenceRepository
	tmplSvc    *TemplateService
	dispatcher *DispatcherService
	logger     zerolog.Logger

	dailyUTCHour int
	weeklyDay    int // 0=Sunday, 1=Monday, ...
}

// NewDigestService creates a new DigestService.
func NewDigestService(
	notifRepo *repository.NotificationRepository,
	prefRepo *repository.PreferenceRepository,
	tmplSvc *TemplateService,
	dispatcher *DispatcherService,
	dailyUTCHour int,
	weeklyDay int,
	logger zerolog.Logger,
) *DigestService {
	return &DigestService{
		notifRepo:    notifRepo,
		prefRepo:     prefRepo,
		tmplSvc:      tmplSvc,
		dispatcher:   dispatcher,
		dailyUTCHour: dailyUTCHour,
		weeklyDay:    weeklyDay,
		logger:       logger.With().Str("component", "digest_service").Logger(),
	}
}

// RunScheduler runs the digest scheduling loop until context is cancelled.
func (s *DigestService) RunScheduler(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	var lastDailyRun, lastWeeklyRun time.Time

	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			utcNow := now.UTC()

			// Daily digest: run once at the configured hour.
			if utcNow.Hour() == s.dailyUTCHour && utcNow.Sub(lastDailyRun) > 23*time.Hour {
				s.logger.Info().Msg("triggering daily digest")
				if err := s.sendDigests(ctx, "daily", 24*time.Hour); err != nil {
					s.logger.Error().Err(err).Msg("daily digest failed")
				}
				lastDailyRun = utcNow
			}

			// Weekly digest: run once on the configured day at the configured hour.
			if int(utcNow.Weekday()) == s.weeklyDay && utcNow.Hour() == s.dailyUTCHour && utcNow.Sub(lastWeeklyRun) > 6*24*time.Hour {
				s.logger.Info().Msg("triggering weekly digest")
				if err := s.sendDigests(ctx, "weekly", 7*24*time.Hour); err != nil {
					s.logger.Error().Err(err).Msg("weekly digest failed")
				}
				lastWeeklyRun = utcNow
			}
		}
	}
}

func (s *DigestService) sendDigests(ctx context.Context, frequency string, lookback time.Duration) error {
	tenants, err := s.prefRepo.ListAllTenants(ctx)
	if err != nil {
		return err
	}

	since := time.Now().UTC().Add(-lookback)

	for _, tenantID := range tenants {
		subscribers, err := s.prefRepo.GetDigestSubscribers(ctx, tenantID, frequency)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("failed to get digest subscribers")
			continue
		}

		if len(subscribers) == 0 {
			continue
		}

		// Fetch unread notifications for this tenant.
		notifications, err := s.notifRepo.GetUnreadForDigest(ctx, tenantID, since)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("failed to get digest notifications")
			continue
		}

		// Group notifications by user.
		userNotifs := make(map[string][]model.Notification)
		for _, n := range notifications {
			userNotifs[n.UserID] = append(userNotifs[n.UserID], n)
		}

		// Send digest to each subscribed user who has unread notifications.
		subscriberSet := make(map[string]bool, len(subscribers))
		for _, uid := range subscribers {
			subscriberSet[uid] = true
		}

		for userID, notifs := range userNotifs {
			if !subscriberSet[userID] || len(notifs) == 0 {
				continue
			}

			if err := s.sendUserDigest(ctx, tenantID, userID, notifs, frequency); err != nil {
				s.logger.Warn().Err(err).
					Str("user_id", userID).
					Str("frequency", frequency).
					Msg("failed to send digest to user")
			} else {
				metrics.DigestsSent.WithLabelValues(frequency).Inc()
			}
		}
	}

	return nil
}

func (s *DigestService) sendUserDigest(ctx context.Context, tenantID, userID string, notifs []model.Notification, frequency string) error {
	// Build a synthetic notification for the digest email.
	digestNotif := &model.Notification{
		TenantID: tenantID,
		UserID:   userID,
		Type:     "digest." + model.NotificationType(frequency),
		Category: model.CategorySystem,
		Priority: model.PriorityLow,
		Title:    "Your " + frequency + " notification digest",
		Body:     "",
	}

	// Insert as a regular notification.
	id, err := s.notifRepo.Insert(ctx, digestNotif)
	if err != nil {
		return err
	}
	digestNotif.ID = id

	return nil
}
