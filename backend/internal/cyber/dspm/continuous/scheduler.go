package continuous

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/continuous/watchers"
)

// Scheduler manages the lifecycle of all continuous DSPM watchers.
type Scheduler struct {
	watchers []watchers.Watcher
	logger   zerolog.Logger
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewScheduler creates a watcher lifecycle manager.
func NewScheduler(w []watchers.Watcher, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		watchers: w,
		logger:   logger.With().Str("component", "dspm-scheduler").Logger(),
	}
}

// Start launches all watchers in goroutines and blocks until ctx is done.
func (s *Scheduler) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

	for _, w := range s.watchers {
		w := w
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.logger.Info().Str("watcher", w.Name()).Msg("starting watcher")
			if err := w.Start(ctx); err != nil {
				s.logger.Error().Err(err).Str("watcher", w.Name()).Msg("watcher exited with error")
			} else {
				s.logger.Info().Str("watcher", w.Name()).Msg("watcher stopped")
			}
		}()
	}

	<-ctx.Done()
	return nil
}

// Stop gracefully stops all watchers and waits for them to finish.
func (s *Scheduler) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	for _, w := range s.watchers {
		if err := w.Stop(); err != nil {
			s.logger.Error().Err(err).Str("watcher", w.Name()).Msg("watcher stop error")
		}
	}

	s.wg.Wait()
	s.logger.Info().Msg("all watchers stopped")
	return nil
}

// WatcherNames returns the names of all managed watchers.
func (s *Scheduler) WatcherNames() []string {
	names := make([]string, len(s.watchers))
	for i, w := range s.watchers {
		names[i] = w.Name()
	}
	return names
}
