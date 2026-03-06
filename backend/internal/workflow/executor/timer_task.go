package executor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/workflow/expression"
	"github.com/clario360/platform/internal/workflow/model"
)

// TimerCreator is a narrow interface for persisting timer records in the database
// as a durable fallback alongside the Redis sorted set.
type TimerCreator interface {
	CreateTimer(ctx context.Context, instanceID, stepID string, fireAt time.Time) (string, error)
}

// TimerTaskExecutor schedules a future wake-up for the workflow. It stores the
// timer in a Redis sorted set (for fast polling) and in the database (for
// durability). The workflow is parked until the timer fires.
type TimerTaskExecutor struct {
	rdb       *redis.Client
	timerRepo TimerCreator
	resolver  *expression.VariableResolver
	logger    zerolog.Logger
}

// NewTimerTaskExecutor creates a TimerTaskExecutor.
func NewTimerTaskExecutor(rdb *redis.Client, timerRepo TimerCreator, logger zerolog.Logger) *TimerTaskExecutor {
	return &TimerTaskExecutor{
		rdb:       rdb,
		timerRepo: timerRepo,
		resolver:  expression.NewVariableResolver(),
		logger:    logger.With().Str("executor", "timer").Logger(),
	}
}

// Execute schedules a timer and parks the workflow.
//
// Expected step.Config keys (one of the following is required):
//   - duration (string): ISO 8601 duration, e.g., "PT4H", "PT30M", "PT1H30M", "PT90S"
//   - fire_at (string): absolute time or a ${...} variable reference resolving to an RFC3339 time
func (e *TimerTaskExecutor) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	var fireAt time.Time

	dataCtx := buildDataContext(instance)

	if durationStr, ok := step.Config["duration"]; ok {
		// Resolve if it's a variable reference.
		resolved, err := e.resolver.Resolve(durationStr, dataCtx)
		if err != nil {
			return nil, fmt.Errorf("timer %s: resolving duration: %w", step.ID, err)
		}

		durStr := fmt.Sprintf("%v", resolved)
		dur, err := parseDuration(durStr)
		if err != nil {
			return nil, fmt.Errorf("timer %s: parsing duration %q: %w", step.ID, durStr, err)
		}

		fireAt = time.Now().UTC().Add(dur)
	} else if fireAtStr, ok := step.Config["fire_at"]; ok {
		// Resolve variable references.
		resolved, err := e.resolver.Resolve(fireAtStr, dataCtx)
		if err != nil {
			return nil, fmt.Errorf("timer %s: resolving fire_at: %w", step.ID, err)
		}

		resolvedStr := fmt.Sprintf("%v", resolved)
		parsed, err := parseTime(resolvedStr)
		if err != nil {
			return nil, fmt.Errorf("timer %s: parsing fire_at %q: %w", step.ID, resolvedStr, err)
		}
		fireAt = parsed
	} else {
		return nil, fmt.Errorf("timer %s: config must specify either 'duration' or 'fire_at'", step.ID)
	}

	// Ensure the fire time is in the future.
	if fireAt.Before(time.Now().UTC()) {
		e.logger.Warn().
			Str("step_id", step.ID).
			Time("fire_at", fireAt).
			Msg("timer fire_at is in the past, firing immediately")
		fireAt = time.Now().UTC().Add(1 * time.Second)
	}

	// Register in Redis sorted set: ZADD "workflow:timers" with score=fireAt.UnixMilli().
	redisMember := fmt.Sprintf("%s:%s", instance.ID, step.ID)
	score := float64(fireAt.UnixMilli())

	if err := e.rdb.ZAdd(ctx, "workflow:timers", redis.Z{
		Score:  score,
		Member: redisMember,
	}).Err(); err != nil {
		return nil, fmt.Errorf("timer %s: adding to Redis sorted set: %w", step.ID, err)
	}

	// Persist in database as a durable fallback.
	timerID, err := e.timerRepo.CreateTimer(ctx, instance.ID, step.ID, fireAt)
	if err != nil {
		// Log the error but don't fail - Redis is the primary mechanism.
		e.logger.Error().
			Err(err).
			Str("step_id", step.ID).
			Str("instance_id", instance.ID).
			Time("fire_at", fireAt).
			Msg("failed to persist timer to database; Redis entry is primary")
		timerID = ""
	}

	e.logger.Info().
		Str("step_id", step.ID).
		Str("instance_id", instance.ID).
		Str("timer_id", timerID).
		Time("fire_at", fireAt).
		Str("redis_member", redisMember).
		Msg("timer scheduled, parking workflow")

	return &ExecutionResult{
		Output: map[string]interface{}{
			"timer_id": timerID,
			"fire_at":  fireAt.Format(time.RFC3339),
		},
		Parked: true,
	}, nil
}

// ---------- ISO 8601 Duration Parser ----------

// iso8601DurationRegex matches ISO 8601 duration strings like PT4H, PT30M, PT1H30M, PT90S, PT1H30M45S.
var iso8601DurationRegex = regexp.MustCompile(`^P(?:(\d+)D)?T?(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseDuration parses an ISO 8601 duration string into a Go time.Duration.
// Supported format: P[nD]T[nH][nM][nS]
// Examples: PT4H, PT30M, PT1H30M, PT90S, PT1H30M45S, P1DT2H
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	matches := iso8601DurationRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid ISO 8601 duration format: %q", s)
	}

	var dur time.Duration

	// Days
	if matches[1] != "" {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid days in duration: %w", err)
		}
		dur += time.Duration(days) * 24 * time.Hour
	}

	// Hours
	if matches[2] != "" {
		hours, err := strconv.Atoi(matches[2])
		if err != nil {
			return 0, fmt.Errorf("invalid hours in duration: %w", err)
		}
		dur += time.Duration(hours) * time.Hour
	}

	// Minutes
	if matches[3] != "" {
		minutes, err := strconv.Atoi(matches[3])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in duration: %w", err)
		}
		dur += time.Duration(minutes) * time.Minute
	}

	// Seconds
	if matches[4] != "" {
		seconds, err := strconv.Atoi(matches[4])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds in duration: %w", err)
		}
		dur += time.Duration(seconds) * time.Second
	}

	if dur == 0 {
		return 0, fmt.Errorf("duration is zero: %q", s)
	}

	return dur, nil
}

// parseTime attempts to parse a time string in several common formats.
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time %q (supported formats: RFC3339, datetime, date)", s)
}
