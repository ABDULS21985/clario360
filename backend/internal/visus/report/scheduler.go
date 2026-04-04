package report

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/repository"
)

type Scheduler struct {
	reports   *repository.ReportRepository
	generator *Generator
	interval  time.Duration
	logger    zerolog.Logger
}

func NewScheduler(reports *repository.ReportRepository, generator *Generator, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{
		reports:   reports,
		generator: generator,
		interval:  interval,
		logger:    logger.With().Str("component", "visus_report_scheduler").Logger(),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if err := s.RunOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error().Err(err).Msg("report scheduler iteration failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) error {
	due, err := s.reports.ListDue(ctx, time.Now().UTC(), 25)
	if err != nil {
		return err
	}
	for _, reportDef := range due {
		if _, err := s.generator.Generate(ctx, reportDef.ID, nil); err != nil {
			return err
		}
	}
	return nil
}

func NextRun(expr string, after time.Time) (time.Time, error) {
	schedule, err := parseCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	next := after.UTC().Add(time.Minute).Truncate(time.Minute)
	limit := next.AddDate(1, 0, 0)
	for !next.After(limit) {
		if schedule.matches(next) {
			return next, nil
		}
		next = next.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no next run found within one year for %q", expr)
}

type cronSchedule struct {
	minutes map[int]struct{}
	hours   map[int]struct{}
	days    map[int]struct{}
	months  map[int]struct{}
	weekday map[int]struct{}
}

func (c cronSchedule) matches(value time.Time) bool {
	_, okMinute := c.minutes[value.Minute()]
	_, okHour := c.hours[value.Hour()]
	_, okDay := c.days[value.Day()]
	_, okMonth := c.months[int(value.Month())]
	_, okWeekday := c.weekday[int(value.Weekday())]
	return okMinute && okHour && okDay && okMonth && okWeekday
}

func parseCron(expr string) (*cronSchedule, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields")
	}
	minutes, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("parse cron minutes: %w", err)
	}
	hours, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("parse cron hours: %w", err)
	}
	days, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("parse cron days: %w", err)
	}
	months, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("parse cron months: %w", err)
	}
	weekday, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("parse cron weekdays: %w", err)
	}
	return &cronSchedule{
		minutes: minutes,
		hours:   hours,
		days:    days,
		months:  months,
		weekday: weekday,
	}, nil
}

func parseCronField(raw string, min, max int) (map[int]struct{}, error) {
	raw = strings.TrimSpace(raw)
	out := map[int]struct{}{}
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "*" {
			for value := min; value <= max; value++ {
				out[value] = struct{}{}
			}
			continue
		}
		step := 1
		if strings.Contains(part, "/") {
			split := strings.SplitN(part, "/", 2)
			part = split[0]
			parsed, err := strconv.Atoi(split[1])
			if err != nil || parsed < 1 {
				return nil, fmt.Errorf("invalid step %q", split[1])
			}
			step = parsed
		}
		rangeMin, rangeMax := min, max
		if part != "*" {
			if strings.Contains(part, "-") {
				split := strings.SplitN(part, "-", 2)
				var err error
				rangeMin, err = strconv.Atoi(split[0])
				if err != nil {
					return nil, err
				}
				rangeMax, err = strconv.Atoi(split[1])
				if err != nil {
					return nil, err
				}
			} else {
				value, err := strconv.Atoi(part)
				if err != nil {
					return nil, err
				}
				rangeMin = value
				rangeMax = value
			}
		}
		if rangeMin < min || rangeMax > max || rangeMin > rangeMax {
			return nil, fmt.Errorf("field %q outside allowed range", raw)
		}
		for value := rangeMin; value <= rangeMax; value += step {
			out[value] = struct{}{}
		}
	}
	return out, nil
}

func NextRunForReport(schedule *string, after time.Time) (*time.Time, error) {
	if schedule == nil || strings.TrimSpace(*schedule) == "" {
		return nil, nil
	}
	next, err := NextRun(*schedule, after)
	if err != nil {
		return nil, err
	}
	return &next, nil
}

func TriggeredBy(triggeredBy *uuid.UUID) *uuid.UUID {
	return triggeredBy
}
