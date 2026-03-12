package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
	llmrepo "github.com/clario360/platform/internal/cyber/vciso/llm/repository"
)

type RateLimiter struct {
	repo     *llmrepo.LLMAuditRepository
	defaults llmcfg.RateLimitDefaults
	logger   zerolog.Logger
	mu       sync.Mutex
	userHits map[string][]time.Time
}

type RateLimitError struct {
	LimitType string
	Message   string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

func NewRateLimiter(repo *llmrepo.LLMAuditRepository, defaults llmcfg.RateLimitDefaults, logger zerolog.Logger) *RateLimiter {
	return &RateLimiter{
		repo:     repo,
		defaults: defaults,
		logger:   logger.With().Str("component", "vciso_llm_rate_limiter").Logger(),
		userHits: map[string][]time.Time{},
	}
}

func (r *RateLimiter) Check(ctx context.Context, tenantID, userID uuid.UUID) error {
	if r == nil || r.repo == nil {
		return nil
	}
	record, err := r.repo.GetRateLimit(ctx, tenantID, r.defaults)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	normalizeResets(record, now)
	if record.CurrentCallsMinute >= record.MaxCallsPerMinute {
		return &RateLimitError{LimitType: "tenant_minute", Message: "tenant minute rate limit exceeded"}
	}
	if record.CurrentCallsHour >= record.MaxCallsPerHour {
		return &RateLimitError{LimitType: "tenant_hour", Message: "tenant hourly rate limit exceeded"}
	}
	if record.CurrentCallsDay >= record.MaxCallsPerDay {
		return &RateLimitError{LimitType: "tenant_day", Message: "tenant daily rate limit exceeded"}
	}
	if record.CurrentTokensDay >= record.MaxTokensPerDay {
		return &RateLimitError{LimitType: "tenant_tokens", Message: "tenant token budget exceeded"}
	}
	if record.CurrentCostDayUSD >= record.MaxCostPerDayUSD {
		return &RateLimitError{LimitType: "tenant_cost", Message: "tenant cost budget exceeded"}
	}
	if err := r.checkPerUser(userID, now); err != nil {
		return err
	}
	if err := r.repo.SaveRateLimit(ctx, record); err != nil {
		return err
	}
	return nil
}

func (r *RateLimiter) Consume(ctx context.Context, tenantID uuid.UUID, tokens int, costUSD float64) error {
	if r == nil || r.repo == nil {
		return nil
	}
	record, err := r.repo.GetRateLimit(ctx, tenantID, r.defaults)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	normalizeResets(record, now)
	record.CurrentCallsMinute++
	record.CurrentCallsHour++
	record.CurrentCallsDay++
	record.CurrentTokensDay += max(tokens, 0)
	record.CurrentCostDayUSD += maxFloat(costUSD, 0)
	return r.repo.SaveRateLimit(ctx, record)
}

func (r *RateLimiter) checkPerUser(userID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := userID.String()
	window := now.Add(-1 * time.Minute)
	values := r.userHits[key]
	filtered := values[:0]
	for _, ts := range values {
		if ts.After(window) {
			filtered = append(filtered, ts)
		}
	}
	if len(filtered) >= max(r.defaults.MaxCallsPerMinute/2, 1) {
		r.userHits[key] = filtered
		return &RateLimitError{LimitType: "user_minute", Message: "user minute rate limit exceeded"}
	}
	r.userHits[key] = append(filtered, now)
	return nil
}

func normalizeResets(record *llmmodel.RateLimitRecord, now time.Time) {
	if record == nil {
		return
	}
	if !record.MinuteResetAt.After(now) {
		record.CurrentCallsMinute = 0
		record.MinuteResetAt = now.Add(time.Minute)
	}
	if !record.HourResetAt.After(now) {
		record.CurrentCallsHour = 0
		record.HourResetAt = now.Add(time.Hour)
	}
	if !record.DayResetAt.After(now) {
		record.CurrentCallsDay = 0
		record.CurrentTokensDay = 0
		record.CurrentCostDayUSD = 0
		record.DayResetAt = now.Add(24 * time.Hour)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (r *RateLimiter) String() string {
	return fmt.Sprintf("RateLimiter(default_per_min=%d)", r.defaults.MaxCallsPerMinute)
}
