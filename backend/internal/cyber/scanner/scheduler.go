package scanner

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// ScheduledScan represents a scan that runs on a recurring schedule.
type ScheduledScan struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// Scheduler runs periodic scans in the background.
type Scheduler struct {
	scans  []ScheduledScan
	logger zerolog.Logger
}

// NewScheduler creates a new scan scheduler.
func NewScheduler(logger zerolog.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

// Register adds a scheduled scan.
func (s *Scheduler) Register(scan ScheduledScan) {
	s.scans = append(s.scans, scan)
}

// Start launches all scheduled scans as background goroutines.
// It blocks until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	if len(s.scans) == 0 {
		<-ctx.Done()
		return nil
	}

	for _, scan := range s.scans {
		scan := scan // capture for goroutine
		go func() {
			ticker := time.NewTicker(scan.Interval)
			defer ticker.Stop()
			s.logger.Info().
				Str("scan", scan.Name).
				Dur("interval", scan.Interval).
				Msg("scheduled scan starting")

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.logger.Info().Str("scan", scan.Name).Msg("running scheduled scan")
					if err := scan.Run(ctx); err != nil {
						s.logger.Error().Err(err).Str("scan", scan.Name).Msg("scheduled scan failed")
					}
				}
			}
		}()
	}

	<-ctx.Done()
	return nil
}
