package continuous

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm"
	"github.com/clario360/platform/internal/cyber/dspm/compliance"
	"github.com/clario360/platform/internal/cyber/dspm/continuous/watchers"
	"github.com/clario360/platform/internal/cyber/dspm/shadow"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// Engine orchestrates all continuous DSPM scanning watchers.
type Engine struct {
	config     Config
	scheduler  *Scheduler
	pipeline   *watchers.PipelineWatcher
	transit    *watchers.TransitWatcher
	atRest     *watchers.AtRestWatcher
	shadowW    *watchers.ShadowWatcher
	logger     zerolog.Logger
}

// NewEngine creates a continuous DSPM scanning engine with all watchers.
func NewEngine(
	cyberDB *pgxpool.Pool,
	dataDB *pgxpool.Pool,
	dspmRepo *repository.DSPMRepository,
	alertRepo *repository.AlertRepository,
	classifier *dspm.DSPMClassifier,
	tagger *compliance.ComplianceTagger,
	shadowDetector *shadow.Detector,
	producer *events.Producer,
	config Config,
	logger zerolog.Logger,
) *Engine {
	log := logger.With().Str("component", "continuous-dspm").Logger()

	pipelineW := watchers.NewPipelineWatcher(cyberDB, dspmRepo, alertRepo, classifier, tagger, producer, log)
	transitW := watchers.NewTransitWatcher(cyberDB, dataDB, alertRepo, producer, config.TransitEncryptionRequired, config.PipelineApprovalRequired, log)
	atRestW := watchers.NewAtRestWatcher(cyberDB, dspmRepo, alertRepo, classifier, tagger, producer, config.AtRestScanInterval, log)
	shadowW := watchers.NewShadowWatcher(shadowDetector, alertRepo, producer, config.ShadowScanInterval, config.ShadowSimilarityThreshold, log)

	allWatchers := []watchers.Watcher{pipelineW, transitW, atRestW, shadowW}
	scheduler := NewScheduler(allWatchers, log)

	return &Engine{
		config:    config,
		scheduler: scheduler,
		pipeline:  pipelineW,
		transit:   transitW,
		atRest:    atRestW,
		shadowW:   shadowW,
		logger:    log,
	}
}

// Start begins all continuous scanning watchers.
func (e *Engine) Start(ctx context.Context) error {
	e.logger.Info().Msg("starting continuous DSPM scanning engine")
	return e.scheduler.Start(ctx)
}

// Stop gracefully stops all watchers.
func (e *Engine) Stop() error {
	e.logger.Info().Msg("stopping continuous DSPM scanning engine")
	return e.scheduler.Stop()
}

// PipelineWatcher returns the pipeline watcher for event handler registration.
func (e *Engine) PipelineWatcher() *watchers.PipelineWatcher { return e.pipeline }

// TransitWatcher returns the transit watcher for event handler registration.
func (e *Engine) TransitWatcher() *watchers.TransitWatcher { return e.transit }

// ShadowWatcher returns the shadow watcher for on-demand scans.
func (e *Engine) ShadowWatcher() *watchers.ShadowWatcher { return e.shadowW }

// HandlePipelineEvent routes pipeline events to the appropriate watcher.
func (e *Engine) HandlePipelineEvent(ctx context.Context, evt *events.Event) error {
	if evt == nil {
		return fmt.Errorf("nil event")
	}

	// Route based on pipeline status in event data
	// Both watchers handle their own status filtering
	var errs []error
	if err := e.pipeline.HandlePipelineCompleted(ctx, evt); err != nil {
		errs = append(errs, fmt.Errorf("pipeline watcher: %w", err))
	}
	if err := e.transit.HandlePipelineRunning(ctx, evt); err != nil {
		errs = append(errs, fmt.Errorf("transit watcher: %w", err))
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
