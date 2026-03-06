package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
)

const prefCacheTTL = 5 * time.Minute
const prefCachePrefix = "notif:pref:"

// PreferenceService handles preference CRUD and resolution.
type PreferenceService struct {
	repo   *repository.PreferenceRepository
	rdb    *redis.Client
	logger zerolog.Logger
}

// NewPreferenceService creates a new PreferenceService.
func NewPreferenceService(repo *repository.PreferenceRepository, rdb *redis.Client, logger zerolog.Logger) *PreferenceService {
	return &PreferenceService{
		repo:   repo,
		rdb:    rdb,
		logger: logger.With().Str("component", "preference_service").Logger(),
	}
}

// Get returns the user's preferences, with defaults merged if no record exists.
func (s *PreferenceService) Get(ctx context.Context, userID, tenantID string) (*model.NotificationPreference, error) {
	// Try cache first.
	cacheKey := prefCachePrefix + userID + ":" + tenantID
	cached, err := s.rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var pref model.NotificationPreference
		if json.Unmarshal(cached, &pref) == nil {
			return &pref, nil
		}
	}

	pref, err := s.repo.Get(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}

	if pref == nil {
		pref = &model.NotificationPreference{
			UserID:       userID,
			TenantID:     tenantID,
			GlobalPrefs:  model.DefaultPreferences,
			PerTypePrefs: make(map[model.NotificationType]model.ChannelPreference),
			DigestConfig: model.DigestConfig{Daily: false, Weekly: true},
			UpdatedAt:    time.Now(),
		}
	}

	// Cache.
	if data, err := json.Marshal(pref); err == nil {
		s.rdb.Set(ctx, cacheKey, data, prefCacheTTL)
	}

	return pref, nil
}

// Update merges the update request into the user's existing preferences.
func (s *PreferenceService) Update(ctx context.Context, userID, tenantID string, globalPrefs *model.ChannelPreference, perTypePrefs map[model.NotificationType]model.ChannelPreference, quietHours *model.QuietHours, digestConfig *model.DigestConfig) error {
	existing, err := s.Get(ctx, userID, tenantID)
	if err != nil {
		return err
	}

	// Merge.
	if globalPrefs != nil {
		existing.GlobalPrefs = *globalPrefs
	}
	if perTypePrefs != nil {
		if existing.PerTypePrefs == nil {
			existing.PerTypePrefs = make(map[model.NotificationType]model.ChannelPreference)
		}
		for k, v := range perTypePrefs {
			existing.PerTypePrefs[k] = v
		}
	}
	if quietHours != nil {
		existing.QuietHours = quietHours
	}
	if digestConfig != nil {
		existing.DigestConfig = *digestConfig
	}

	if err := s.repo.Upsert(ctx, existing); err != nil {
		return err
	}

	// Invalidate cache.
	cacheKey := prefCachePrefix + userID + ":" + tenantID
	s.rdb.Del(ctx, cacheKey)

	return nil
}

// ResolveChannels resolves which channels should be used for a given notification.
// Returns enabled channels based on user preferences (per-type override or global).
func (s *PreferenceService) ResolveChannels(ctx context.Context, userID, tenantID string, notifType model.NotificationType) (model.ChannelPreference, error) {
	pref, err := s.Get(ctx, userID, tenantID)
	if err != nil {
		return model.DefaultPreferences, nil
	}

	// Check per-type override first.
	if perType, ok := pref.PerTypePrefs[notifType]; ok {
		return perType, nil
	}

	return pref.GlobalPrefs, nil
}

// IsInQuietHours checks if the current time falls within the user's quiet hours.
func (s *PreferenceService) IsInQuietHours(ctx context.Context, userID, tenantID string) (bool, error) {
	pref, err := s.Get(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}

	if pref.QuietHours == nil || !pref.QuietHours.Enabled {
		return false, nil
	}

	loc, err := time.LoadLocation(pref.QuietHours.Timezone)
	if err != nil {
		return false, fmt.Errorf("invalid timezone %q: %w", pref.QuietHours.Timezone, err)
	}

	now := time.Now().In(loc)
	currentMinutes := now.Hour()*60 + now.Minute()

	start, err := parseHHMM(pref.QuietHours.StartTime)
	if err != nil {
		return false, err
	}
	end, err := parseHHMM(pref.QuietHours.EndTime)
	if err != nil {
		return false, err
	}

	if start < end {
		return currentMinutes >= start && currentMinutes < end, nil
	}
	// Overnight (e.g., 22:00 to 07:00).
	return currentMinutes >= start || currentMinutes < end, nil
}

func parseHHMM(s string) (int, error) {
	var h, m int
	_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil {
		return 0, fmt.Errorf("invalid time format %q: %w", s, err)
	}
	return h*60 + m, nil
}
