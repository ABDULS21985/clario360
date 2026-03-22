package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/pipeline"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/sqlutil"
	"github.com/clario360/platform/internal/events"
)

const pipelineEventsTopic = "data.pipeline.events"

type PipelineService struct {
	pipelineRepo *repository.PipelineRepository
	runRepo      *repository.PipelineRunRepository
	logRepo      *repository.PipelineRunLogRepository
	sourceRepo   *repository.SourceRepository
	modelRepo    *repository.ModelRepository
	engine       *pipeline.Engine
	producer     *events.Producer
	logger       zerolog.Logger
}

func NewPipelineService(
	pipelineRepo *repository.PipelineRepository,
	runRepo *repository.PipelineRunRepository,
	logRepo *repository.PipelineRunLogRepository,
	sourceRepo *repository.SourceRepository,
	modelRepo *repository.ModelRepository,
	engine *pipeline.Engine,
	producer *events.Producer,
	logger zerolog.Logger,
) *PipelineService {
	return &PipelineService{
		pipelineRepo: pipelineRepo,
		runRepo:      runRepo,
		logRepo:      logRepo,
		sourceRepo:   sourceRepo,
		modelRepo:    modelRepo,
		engine:       engine,
		producer:     producer,
		logger:       logger,
	}
}

func (s *PipelineService) Create(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreatePipelineRequest) (*model.Pipeline, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	pipelineType := model.PipelineType(strings.TrimSpace(req.Type))
	if !pipelineType.IsValid() {
		return nil, fmt.Errorf("%w: invalid pipeline type", ErrValidation)
	}
	if _, err := s.sourceRepo.Get(ctx, tenantID, req.SourceID); err != nil {
		return nil, err
	}
	if req.TargetID != nil {
		if _, err := s.sourceRepo.Get(ctx, tenantID, *req.TargetID); err != nil {
			return nil, err
		}
	}
	if exists, err := s.pipelineRepo.ExistsByName(ctx, tenantID, req.Name, nil); err != nil {
		return nil, err
	} else if exists {
		return nil, fmt.Errorf("%w: a pipeline named %q already exists", ErrConflict, req.Name)
	}
	config, err := decodePipelineConfig(req.Config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := validatePipelineConfig(config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	nextRun, err := pipeline.NextRunTime(req.Schedule, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	now := time.Now().UTC()
	item := &model.Pipeline{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		Type:        pipelineType,
		SourceID:    req.SourceID,
		TargetID:    req.TargetID,
		Config:      config,
		Schedule:    req.Schedule,
		Status:      model.PipelineStatusActive,
		NextRunAt:   nextRun,
		Tags:        req.Tags,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.pipelineRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.publish(ctx, "data.pipeline.created", tenantID, map[string]any{
		"id":        item.ID,
		"name":      item.Name,
		"type":      item.Type,
		"source_id": item.SourceID,
	})
	return item, nil
}

func (s *PipelineService) List(ctx context.Context, tenantID uuid.UUID, params dto.ListPipelinesParams) ([]*model.Pipeline, int, error) {
	return s.pipelineRepo.List(ctx, tenantID, params)
}

func (s *PipelineService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.Pipeline, error) {
	return s.pipelineRepo.Get(ctx, tenantID, id)
}

func (s *PipelineService) Update(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdatePipelineRequest) (*model.Pipeline, error) {
	item, err := s.pipelineRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" && !strings.EqualFold(item.Name, *req.Name) {
		if exists, err := s.pipelineRepo.ExistsByName(ctx, tenantID, strings.TrimSpace(*req.Name), &id); err != nil {
			return nil, err
		} else if exists {
			return nil, fmt.Errorf("%w: a pipeline named %q already exists", ErrConflict, *req.Name)
		}
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Type != nil {
		pipelineType := model.PipelineType(strings.TrimSpace(*req.Type))
		if !pipelineType.IsValid() {
			return nil, fmt.Errorf("%w: invalid pipeline type", ErrValidation)
		}
		item.Type = pipelineType
	}
	if req.TargetID != nil {
		if _, err := s.sourceRepo.Get(ctx, tenantID, *req.TargetID); err != nil {
			return nil, err
		}
		item.TargetID = req.TargetID
	}
	if len(req.Config) > 0 {
		config, err := decodePipelineConfig(req.Config)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		if err := validatePipelineConfig(config); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		item.Config = config
	}
	if req.Schedule != nil {
		nextRun, err := pipeline.NextRunTime(req.Schedule, time.Now().UTC())
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		item.Schedule = req.Schedule
		item.NextRunAt = nextRun
	}
	if req.Status != nil {
		status := model.PipelineStatus(strings.TrimSpace(*req.Status))
		if !status.IsValid() {
			return nil, fmt.Errorf("%w: invalid pipeline status", ErrValidation)
		}
		item.Status = status
	}
	if req.Tags != nil {
		item.Tags = req.Tags
	}
	item.UpdatedAt = time.Now().UTC()
	if err := s.pipelineRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	s.publish(ctx, "data.pipeline.updated", tenantID, map[string]any{
		"id":   item.ID,
		"name": item.Name,
	})
	return item, nil
}

func (s *PipelineService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	item, err := s.pipelineRepo.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := s.pipelineRepo.SoftDelete(ctx, tenantID, id, time.Now().UTC()); err != nil {
		return err
	}
	s.publish(ctx, "data.pipeline.deleted", tenantID, map[string]any{"id": item.ID, "name": item.Name})
	return nil
}

func (s *PipelineService) Run(ctx context.Context, tenantID, id uuid.UUID, userID *uuid.UUID) (*model.PipelineRun, error) {
	if _, err := s.pipelineRepo.Get(ctx, tenantID, id); err != nil {
		return nil, err
	}
	run, err := s.engine.ExecutePipeline(ctx, id, string(model.PipelineTriggerAPI), userID)
	if err != nil {
		return nil, translatePipelineExecutionError(err)
	}
	return run, nil
}

func (s *PipelineService) Pause(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.pipelineRepo.UpdateStatus(ctx, tenantID, id, model.PipelineStatusPaused); err != nil {
		return err
	}
	s.publish(ctx, "data.pipeline.paused", tenantID, map[string]any{"id": id})
	return nil
}

func (s *PipelineService) Resume(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.pipelineRepo.UpdateStatus(ctx, tenantID, id, model.PipelineStatusActive); err != nil {
		return err
	}
	s.publish(ctx, "data.pipeline.resumed", tenantID, map[string]any{"id": id})
	return nil
}

func (s *PipelineService) ListRuns(ctx context.Context, tenantID, pipelineID uuid.UUID, params dto.ListPipelineRunsParams) ([]*model.PipelineRun, int, error) {
	if _, err := s.pipelineRepo.Get(ctx, tenantID, pipelineID); err != nil {
		return nil, 0, err
	}
	return s.runRepo.ListByPipeline(ctx, tenantID, pipelineID, params)
}

func (s *PipelineService) GetRun(ctx context.Context, tenantID, pipelineID, runID uuid.UUID) (*model.PipelineRun, error) {
	return s.runRepo.Get(ctx, tenantID, pipelineID, runID)
}

func (s *PipelineService) GetRunLogs(ctx context.Context, tenantID, pipelineID, runID uuid.UUID) ([]*model.PipelineRunLog, error) {
	if _, err := s.runRepo.Get(ctx, tenantID, pipelineID, runID); err != nil {
		return nil, err
	}
	return s.logRepo.ListByRun(ctx, tenantID, runID, 1000)
}

func (s *PipelineService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.PipelineStats, error) {
	return s.pipelineRepo.Stats(ctx, tenantID)
}

// DailyFailedRunCounts returns a zero-filled daily count of failed pipeline
// runs for the last N days, suitable for KPI sparklines.
func (s *PipelineService) DailyFailedRunCounts(ctx context.Context, tenantID uuid.UUID, days int) ([]int, error) {
	return s.pipelineRepo.DailyFailedRunCounts(ctx, tenantID, days)
}

func (s *PipelineService) Active(ctx context.Context, tenantID uuid.UUID) ([]*model.PipelineRun, error) {
	return s.runRepo.ListActive(ctx, tenantID)
}

func (s *PipelineService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return
	}
	_ = s.producer.Publish(ctx, pipelineEventsTopic, event)
}

func decodePipelineConfig(raw json.RawMessage) (model.PipelineConfig, error) {
	var cfg model.PipelineConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	if cfg.LoadStrategy == "" {
		cfg.LoadStrategy = model.LoadStrategyAppend
	}
	return cfg, nil
}

func validatePipelineConfig(cfg model.PipelineConfig) error {
	if strings.TrimSpace(cfg.SourceQuery) == "" && strings.TrimSpace(cfg.SourceTable) == "" {
		return fmt.Errorf("config.source_table or config.source_query is required")
	}
	if strings.TrimSpace(cfg.SourceQuery) != "" {
		if err := sqlutil.ValidateReadOnlySQL(cfg.SourceQuery); err != nil {
			return err
		}
	}
	return nil
}

func translatePipelineExecutionError(err error) error {
	msg := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return err
	case strings.Contains(msg, "already running"):
		return fmt.Errorf("%w: %v", ErrConflict, err)
	case strings.Contains(msg, "maximum concurrent"):
		return fmt.Errorf("%w: %v", ErrTooManyRequests, err)
	case strings.Contains(msg, "disabled"):
		return fmt.Errorf("%w: %v", ErrForbiddenOperation, err)
	case strings.Contains(msg, "decrypt source config"),
		strings.Contains(msg, "create source connector"),
		strings.Contains(msg, "fetch source batch"),
		strings.Contains(msg, "does not exist"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "failed to connect"):
		return fmt.Errorf("%w: %v", ErrPipelineExecution, err)
	default:
		return err
	}
}
