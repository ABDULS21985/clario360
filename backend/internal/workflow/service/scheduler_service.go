package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/workflow/model"
)

const (
	// redisTimerKey is the sorted set key for pending workflow timers.
	// Scores are Unix milliseconds when the timer should fire.
	redisTimerKey = "workflow:timers"

	// timerBatchSize is the maximum number of timers to process per poll cycle.
	timerBatchSize = 100
)

// SchedulerService is a background service that polls for timer expirations
// and monitors SLA compliance for human tasks.
type SchedulerService struct {
	rdb           *redis.Client
	taskRepo      taskRepo
	engine        *EngineService
	producer      eventPublisher
	logger        zerolog.Logger
	timerInterval time.Duration
	slaInterval   time.Duration
	stopCh        chan struct{}
	done          chan struct{}
}

// NewSchedulerService creates a new SchedulerService with configurable poll intervals.
func NewSchedulerService(
	rdb *redis.Client,
	taskRepo taskRepo,
	engine *EngineService,
	producer eventPublisher,
	logger zerolog.Logger,
	timerIntervalSec, slaIntervalSec int,
) *SchedulerService {
	if timerIntervalSec <= 0 {
		timerIntervalSec = 5
	}
	if slaIntervalSec <= 0 {
		slaIntervalSec = 60
	}

	return &SchedulerService{
		rdb:           rdb,
		taskRepo:      taskRepo,
		engine:        engine,
		producer:      producer,
		logger:        logger.With().Str("service", "workflow-scheduler").Logger(),
		timerInterval: time.Duration(timerIntervalSec) * time.Second,
		slaInterval:   time.Duration(slaIntervalSec) * time.Second,
		stopCh:        make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// Start begins the scheduler's background loops for timer polling and SLA monitoring.
// This method returns immediately; the loops run in separate goroutines.
// Call Stop() to gracefully shut down.
func (s *SchedulerService) Start(ctx context.Context) {
	s.logger.Info().
		Dur("timer_interval", s.timerInterval).
		Dur("sla_interval", s.slaInterval).
		Msg("scheduler service starting")

	go s.timerLoop(ctx)
	go s.slaLoop(ctx)
}

// Stop signals the scheduler to stop and waits for both loops to finish.
func (s *SchedulerService) Stop() {
	s.logger.Info().Msg("scheduler service stopping")
	close(s.stopCh)

	// Wait for both goroutines to signal completion. Each sends one value to done.
	<-s.done
	<-s.done

	s.logger.Info().Msg("scheduler service stopped")
}

// RegisterTimer registers a timer in Redis for a workflow step.
// The timer will fire at the specified time, triggering AdvanceWorkflow.
func (s *SchedulerService) RegisterTimer(ctx context.Context, instanceID, stepID string, fireAt time.Time) error {
	member := instanceID + ":" + stepID
	score := float64(fireAt.UnixMilli())

	if err := s.rdb.ZAdd(ctx, redisTimerKey, redis.Z{
		Score:  score,
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("registering timer in Redis: %w", err)
	}

	s.logger.Debug().
		Str("instance_id", instanceID).
		Str("step_id", stepID).
		Time("fire_at", fireAt).
		Msg("timer registered")

	return nil
}

// RemoveTimer removes a timer from Redis (e.g., when an instance is cancelled).
func (s *SchedulerService) RemoveTimer(ctx context.Context, instanceID, stepID string) error {
	member := instanceID + ":" + stepID
	if err := s.rdb.ZRem(ctx, redisTimerKey, member).Err(); err != nil {
		return fmt.Errorf("removing timer from Redis: %w", err)
	}
	return nil
}

// timerLoop periodically polls Redis for expired timers and fires them.
func (s *SchedulerService) timerLoop(ctx context.Context) {
	defer func() {
		s.done <- struct{}{}
	}()

	ticker := time.NewTicker(s.timerInterval)
	defer ticker.Stop()

	s.logger.Info().Msg("timer poller started")

	for {
		select {
		case <-s.stopCh:
			s.logger.Info().Msg("timer poller stopping")
			return
		case <-ctx.Done():
			s.logger.Info().Msg("timer poller context cancelled")
			return
		case <-ticker.C:
			s.pollTimers(ctx)
		}
	}
}

// pollTimers queries Redis for all timers whose fire time has passed,
// removes them atomically, and dispatches AdvanceWorkflow for each.
func (s *SchedulerService) pollTimers(ctx context.Context) {
	nowMs := strconv.FormatInt(time.Now().UnixMilli(), 10)

	// ZRANGEBYSCORE "workflow:timers" -inf {now_ms} LIMIT 0 100
	members, err := s.rdb.ZRangeByScore(ctx, redisTimerKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    nowMs,
		Offset: 0,
		Count:  timerBatchSize,
	}).Result()
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to poll timers from Redis")
		return
	}

	if len(members) == 0 {
		return
	}

	s.logger.Debug().
		Int("count", len(members)).
		Msg("processing expired timers")

	for _, member := range members {
		// Atomically remove the timer to prevent duplicate processing.
		removed, err := s.rdb.ZRem(ctx, redisTimerKey, member).Result()
		if err != nil {
			s.logger.Error().Err(err).
				Str("member", member).
				Msg("failed to remove timer from Redis")
			continue
		}
		if removed == 0 {
			// Another instance already processed this timer.
			continue
		}

		// Parse instanceID:stepID.
		parts := strings.SplitN(member, ":", 2)
		if len(parts) != 2 {
			s.logger.Error().
				Str("member", member).
				Msg("invalid timer member format, expected instanceID:stepID")
			continue
		}

		instanceID := parts[0]
		stepID := parts[1]

		s.logger.Info().
			Str("instance_id", instanceID).
			Str("step_id", stepID).
			Msg("timer fired, advancing workflow")

		if err := s.engine.AdvanceWorkflow(ctx, instanceID, stepID); err != nil {
			s.logger.Error().Err(err).
				Str("instance_id", instanceID).
				Str("step_id", stepID).
				Msg("failed to advance workflow from timer")
		}
	}
}

// slaLoop periodically checks for overdue human tasks and processes SLA breaches.
func (s *SchedulerService) slaLoop(ctx context.Context) {
	defer func() {
		s.done <- struct{}{}
	}()

	ticker := time.NewTicker(s.slaInterval)
	defer ticker.Stop()

	s.logger.Info().Msg("SLA monitor started")

	for {
		select {
		case <-s.stopCh:
			s.logger.Info().Msg("SLA monitor stopping")
			return
		case <-ctx.Done():
			s.logger.Info().Msg("SLA monitor context cancelled")
			return
		case <-ticker.C:
			s.checkSLABreaches(ctx)
		}
	}
}

// checkSLABreaches queries for overdue tasks, marks them as SLA-breached,
// escalates if configured, and publishes breach events.
func (s *SchedulerService) checkSLABreaches(ctx context.Context) {
	overdueTasks, err := s.taskRepo.GetOverdueTasks(ctx, timerBatchSize)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to query overdue tasks")
		return
	}

	if len(overdueTasks) == 0 {
		return
	}

	s.logger.Info().
		Int("count", len(overdueTasks)).
		Msg("processing SLA breaches")

	for _, task := range overdueTasks {
		// Mark the task as SLA breached.
		if err := s.taskRepo.MarkSLABreached(ctx, task.ID); err != nil {
			s.logger.Error().Err(err).
				Str("task_id", task.ID).
				Msg("failed to mark task as SLA breached")
			continue
		}

		s.logger.Warn().
			Str("task_id", task.ID).
			Str("instance_id", task.InstanceID).
			Str("tenant_id", task.TenantID).
			Msg("SLA breached for task")

		// Escalate if an escalation role is configured.
		if task.EscalationRole != nil && *task.EscalationRole != "" {
			if err := s.taskRepo.EscalateTask(ctx, task.ID, *task.EscalationRole); err != nil {
				s.logger.Error().Err(err).
					Str("task_id", task.ID).
					Str("escalation_role", *task.EscalationRole).
					Msg("failed to escalate task")
			} else {
				s.logger.Info().
					Str("task_id", task.ID).
					Str("escalation_role", *task.EscalationRole).
					Msg("task escalated due to SLA breach")
			}
		}

		// Publish SLA breach event.
		s.publishSLAEvent(ctx, task)
	}
}

// publishSLAEvent publishes an event when a task's SLA is breached.
func (s *SchedulerService) publishSLAEvent(ctx context.Context, task *model.HumanTask) {
	if s.producer == nil {
		return
	}

	evt, err := events.NewEvent("workflow.task.sla_breached", "workflow-scheduler", task.TenantID, map[string]interface{}{
		"task_id":     task.ID,
		"instance_id": task.InstanceID,
		"step_id":     task.StepID,
		"task_name":   task.Name,
	})
	if err != nil {
		s.logger.Error().Err(err).
			Str("task_id", task.ID).
			Msg("failed to create SLA breach event")
		return
	}

	if err := s.producer.Publish(ctx, events.Topics.WorkflowEvents, evt); err != nil {
		s.logger.Error().Err(err).
			Str("task_id", task.ID).
			Msg("failed to publish SLA breach event")
	}
}
