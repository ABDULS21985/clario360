package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

type Engine struct {
	pipelineRepo    *repository.PipelineRepository
	runRepo         *repository.PipelineRunRepository
	logRepo         *repository.PipelineRunLogRepository
	sourceRepo      *repository.SourceRepository
	modelRepo       *repository.ModelRepository
	extractor       *Extractor
	transformer     *Transformer
	loader          *Loader
	qualityGates    *QualityGateEvaluator
	producer        *events.Producer
	logger          zerolog.Logger
	maxConcurrent   int
	mu              sync.Mutex
	semaphores      map[string]chan struct{}
}

func NewEngine(
	pipelineRepo *repository.PipelineRepository,
	runRepo *repository.PipelineRunRepository,
	logRepo *repository.PipelineRunLogRepository,
	sourceRepo *repository.SourceRepository,
	modelRepo *repository.ModelRepository,
	extractor *Extractor,
	transformer *Transformer,
	loader *Loader,
	qualityGates *QualityGateEvaluator,
	producer *events.Producer,
	logger zerolog.Logger,
	maxConcurrent int,
) *Engine {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	return &Engine{
		pipelineRepo:  pipelineRepo,
		runRepo:       runRepo,
		logRepo:       logRepo,
		sourceRepo:    sourceRepo,
		modelRepo:     modelRepo,
		extractor:     extractor,
		transformer:   transformer,
		loader:        loader,
		qualityGates:  qualityGates,
		producer:      producer,
		logger:        logger,
		maxConcurrent: maxConcurrent,
		semaphores:    make(map[string]chan struct{}),
	}
}

func (e *Engine) ExecutePipeline(ctx context.Context, pipelineID uuid.UUID, triggeredBy string, userID *uuid.UUID) (*model.PipelineRun, error) {
	pipelineItem, err := e.pipelineRepo.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if pipelineItem.Status == model.PipelineStatusDisabled {
		return nil, fmt.Errorf("pipeline is disabled")
	}
	running, err := e.runRepo.HasRunningRun(ctx, pipelineItem.TenantID, pipelineID)
	if err != nil {
		return nil, err
	}
	if running {
		return nil, fmt.Errorf("pipeline already running")
	}
	if !e.acquireSlot(pipelineItem.TenantID) {
		return nil, fmt.Errorf("maximum concurrent pipelines reached for this tenant")
	}
	defer e.releaseSlot(pipelineItem.TenantID)

	run := &model.PipelineRun{
		ID:          uuid.New(),
		TenantID:    pipelineItem.TenantID,
		PipelineID:  pipelineItem.ID,
		Status:      model.PipelineRunStatusRunning,
		TriggeredBy: model.PipelineTrigger(triggeredBy),
		TriggeredByUser: userID,
		StartedAt:   time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}
	if triggeredBy == "" {
		run.TriggeredBy = model.PipelineTriggerManual
	}
	if err := e.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}
	logger := NewRunLogger(e.logRepo)
	logger.Info(ctx, run.TenantID, run.ID, "run", "Pipeline run started.", map[string]any{"pipeline_id": pipelineItem.ID})
	e.publish(ctx, "data.pipeline.run.started", pipelineItem.TenantID, map[string]any{
		"id":          run.ID,
		"pipeline_id": pipelineItem.ID,
		"triggered_by": run.TriggeredBy,
	})

	sourceRecord, err := e.sourceRepo.Get(ctx, pipelineItem.TenantID, pipelineItem.SourceID)
	if err != nil {
		return e.failRun(ctx, pipelineItem, run, "extracting", err, logger)
	}

	phase := string(model.PipelinePhaseExtracting)
	run.CurrentPhase = &phase
	now := time.Now().UTC()
	run.ExtractStartedAt = &now
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	logger.Info(ctx, run.TenantID, run.ID, phase, "Extraction started.", map[string]any{"source_id": sourceRecord.Source.ID})
	extracted, err := e.extractor.Extract(ctx, sourceRecord, pipelineItem.Config)
	if err != nil {
		return e.failRun(ctx, pipelineItem, run, phase, err, logger)
	}
	run.RecordsExtracted = extracted.RecordsExtracted
	run.BytesRead = extracted.BytesRead
	run.IncrementalFrom = extracted.IncrementalFrom
	run.IncrementalTo = extracted.IncrementalTo
	complete := time.Now().UTC()
	run.ExtractCompletedAt = &complete
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	logger.Info(ctx, run.TenantID, run.ID, phase, "Extraction completed.", map[string]any{"records": run.RecordsExtracted})

	phase = string(model.PipelinePhaseTransforming)
	run.CurrentPhase = &phase
	now = time.Now().UTC()
	run.TransformStartedAt = &now
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	transformed, summary, err := e.transformer.Apply(extracted.Rows, pipelineItem.Config.Transformations)
	if err != nil {
		return e.failRun(ctx, pipelineItem, run, phase, err, logger)
	}
	run.RecordsTransformed = int64(len(transformed))
	run.RecordsFiltered = int64(summary.FilteredRows)
	run.RecordsDeduplicated = int64(summary.DedupedRows)
	complete = time.Now().UTC()
	run.TransformCompletedAt = &complete
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	logger.Info(ctx, run.TenantID, run.ID, phase, "Transformation completed.", summary)

	phase = string(model.PipelinePhaseQualityGate)
	run.CurrentPhase = &phase
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	previousRun, _ := e.runRepo.LatestCompleted(ctx, run.TenantID, run.PipelineID)
	gateResults, err := e.qualityGates.Evaluate(transformed, pipelineItem.Config.QualityGates, previousRun)
	if err != nil {
		return e.failRun(ctx, pipelineItem, run, phase, err, logger)
	}
	for _, result := range gateResults {
		switch result.Status {
		case "passed":
			run.QualityGatesPassed++
		case "warned":
			run.QualityGatesWarned++
		default:
			run.QualityGatesFailed++
		}
	}
	run.QualityGateResults = gateResults
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	if pipelineItem.Config.FailOnQualityGate && run.QualityGatesFailed > 0 {
		return e.failRun(ctx, pipelineItem, run, phase, fmt.Errorf("one or more quality gates failed"), logger)
	}

	phase = string(model.PipelinePhaseLoading)
	run.CurrentPhase = &phase
	now = time.Now().UTC()
	run.LoadStartedAt = &now
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}
	loadResult, err := e.loader.Load(ctx, pipelineItem.TenantID, pipelineItem, transformed)
	if err != nil {
		return e.failRun(ctx, pipelineItem, run, phase, err, logger)
	}
	run.RecordsLoaded = loadResult.RecordsLoaded
	run.RecordsFailed += loadResult.RecordsFailed
	run.BytesWritten = loadResult.BytesWritten
	complete = time.Now().UTC()
	run.LoadCompletedAt = &complete

	run.Status = model.PipelineRunStatusCompleted
	run.CompletedAt = &complete
	durationMs := complete.Sub(run.StartedAt).Milliseconds()
	run.DurationMs = &durationMs
	if err := e.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	totalRuns := pipelineItem.TotalRuns + 1
	successfulRuns := pipelineItem.SuccessfulRuns + 1
	totalRecordsProcessed := pipelineItem.TotalRecordsProcessed + run.RecordsLoaded
	avgDuration := durationMs
	if pipelineItem.AvgDurationMs != nil && pipelineItem.TotalRuns > 0 {
		avgDuration = ((*pipelineItem.AvgDurationMs * int64(pipelineItem.TotalRuns)) + durationMs) / int64(totalRuns)
	}
	nextRun, _ := NextRunTime(pipelineItem.Schedule, complete)
	configPatch := pipelineItem.Config
	configPatch.IncrementalValue = run.IncrementalTo
	completedStatus := string(model.PipelineRunStatusCompleted)
	if err := e.pipelineRepo.UpdateRunState(ctx, pipelineItem.TenantID, pipelineItem.ID, repository.PipelineRunStatePatch{
		LastRunID:             &run.ID,
		LastRunAt:             run.CompletedAt,
		LastRunStatus:         &completedStatus,
		LastRunError:          nil,
		NextRunAt:             nextRun,
		TotalRuns:             &totalRuns,
		SuccessfulRuns:        &successfulRuns,
		FailedRuns:            &pipelineItem.FailedRuns,
		TotalRecordsProcessed: &totalRecordsProcessed,
		AvgDurationMs:         &avgDuration,
		Config:                &configPatch,
	}); err != nil {
		return nil, err
	}

	logger.Info(ctx, run.TenantID, run.ID, "run", "Pipeline completed.", map[string]any{
		"records_extracted": run.RecordsExtracted,
		"records_loaded":    run.RecordsLoaded,
		"duration_ms":       durationMs,
	})
	e.publish(ctx, "data.pipeline.run.completed", pipelineItem.TenantID, map[string]any{
		"id":                run.ID,
		"pipeline_id":       pipelineItem.ID,
		"pipeline_name":     pipelineItem.Name,
		"tenant_id":         pipelineItem.TenantID,
		"status":            model.PipelineRunStatusCompleted,
		"records_extracted": run.RecordsExtracted,
		"records_loaded":    run.RecordsLoaded,
		"duration_ms":       durationMs,
	})
	return run, nil
}

func (e *Engine) failRun(ctx context.Context, pipelineItem *model.Pipeline, run *model.PipelineRun, phase string, cause error, logger *RunLogger) (*model.PipelineRun, error) {
	now := time.Now().UTC()
	run.Status = model.PipelineRunStatusFailed
	run.CompletedAt = &now
	durationMs := now.Sub(run.StartedAt).Milliseconds()
	run.DurationMs = &durationMs
	run.CurrentPhase = &phase
	message := cause.Error()
	run.ErrorPhase = &phase
	run.ErrorMessage = &message
	_ = e.runRepo.Update(ctx, run)

	totalRuns := pipelineItem.TotalRuns + 1
	failedRuns := pipelineItem.FailedRuns + 1
	failedStatus := string(model.PipelineRunStatusFailed)
	_ = e.pipelineRepo.UpdateRunState(ctx, pipelineItem.TenantID, pipelineItem.ID, repository.PipelineRunStatePatch{
		LastRunID:      &run.ID,
		LastRunAt:      run.CompletedAt,
		LastRunStatus:  &failedStatus,
		LastRunError:   &message,
		TotalRuns:      &totalRuns,
		SuccessfulRuns: &pipelineItem.SuccessfulRuns,
		FailedRuns:     &failedRuns,
	})

	logger.Error(ctx, run.TenantID, run.ID, phase, "Pipeline failed.", map[string]any{"error": message})
	e.publish(ctx, "data.pipeline.run.failed", pipelineItem.TenantID, map[string]any{
		"id":            run.ID,
		"pipeline_id":   pipelineItem.ID,
		"pipeline_name": pipelineItem.Name,
		"tenant_id":     pipelineItem.TenantID,
		"status":        model.PipelineRunStatusFailed,
		"error_phase":   phase,
		"error":         message,
		"error_message": message,
	})

	if consecutive, err := e.runRepo.ConsecutiveFailures(ctx, pipelineItem.TenantID, pipelineItem.ID, 5); err == nil && consecutive >= 3 {
		status := model.PipelineStatusError
		_ = e.pipelineRepo.UpdateRunState(ctx, pipelineItem.TenantID, pipelineItem.ID, repository.PipelineRunStatePatch{
			Status: &status,
		})
		e.publish(ctx, "data.pipeline.auto_disabled", pipelineItem.TenantID, map[string]any{
			"id":   pipelineItem.ID,
			"name": pipelineItem.Name,
		})
	}
	return run, cause
}

func (e *Engine) acquireSlot(tenantID uuid.UUID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := tenantSlotKey(tenantID)
	ch, ok := e.semaphores[key]
	if !ok {
		ch = make(chan struct{}, e.maxConcurrent)
		e.semaphores[key] = ch
	}
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (e *Engine) releaseSlot(tenantID uuid.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := tenantSlotKey(tenantID)
	ch, ok := e.semaphores[key]
	if !ok {
		return
	}
	select {
	case <-ch:
	default:
	}
}

func (e *Engine) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if e.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return
	}
	_ = e.producer.Publish(ctx, "data.pipeline.events", event)
}
