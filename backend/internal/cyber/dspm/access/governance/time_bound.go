package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// TimeBoundRepository manages time-bound access grants.
type TimeBoundRepository interface {
	ExpireTimeBoundGrants(ctx context.Context, tenantID uuid.UUID, now time.Time) (int, error)
}

// TimeBoundManager runs periodically (daily) to expire time-bound access grants.
// It finds all active mappings where expires_at <= now() and marks them as
// status='expired', publishing a dspm.access.grant.expired event for each.
type TimeBoundManager struct {
	repo   TimeBoundRepository
	logger zerolog.Logger
}

// NewTimeBoundManager creates a new time-bound grant manager.
func NewTimeBoundManager(repo TimeBoundRepository, logger zerolog.Logger) *TimeBoundManager {
	return &TimeBoundManager{
		repo:   repo,
		logger: logger.With().Str("component", "time_bound_manager").Logger(),
	}
}

// ExpireGrants finds and expires all active mappings with expires_at <= now.
// Returns the count of expired grants.
func (t *TimeBoundManager) ExpireGrants(ctx context.Context, tenantID uuid.UUID) (int, error) {
	now := time.Now().UTC()
	count, err := t.repo.ExpireTimeBoundGrants(ctx, tenantID, now)
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to expire time-bound grants")
		return 0, err
	}
	if count > 0 {
		t.logger.Info().Int("expired_count", count).Msg("expired time-bound grants")
	}
	return count, nil
}
