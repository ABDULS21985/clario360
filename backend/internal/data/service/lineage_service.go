package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/lineage"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

type LineageService struct {
	repo      *repository.LineageRepository
	builder   *lineage.GraphBuilder
	analyzer  *lineage.ImpactAnalyzer
	recorder  *lineage.LineageRecorder
	producer  *events.Producer
	logger    zerolog.Logger
}

func NewLineageService(repo *repository.LineageRepository, builder *lineage.GraphBuilder, analyzer *lineage.ImpactAnalyzer, recorder *lineage.LineageRecorder, producer *events.Producer, logger zerolog.Logger) *LineageService {
	return &LineageService{
		repo:     repo,
		builder:  builder,
		analyzer: analyzer,
		recorder: recorder,
		producer: producer,
		logger:   logger,
	}
}

func (s *LineageService) FullGraph(ctx context.Context, tenantID uuid.UUID) (*model.LineageGraph, error) {
	return s.builder.BuildFullGraph(ctx, tenantID)
}

func (s *LineageService) EntityGraph(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	return s.builder.BuildEntityGraph(ctx, tenantID, entityType, entityID, depth)
}

func (s *LineageService) Upstream(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	return s.builder.BuildDirectionalGraph(ctx, tenantID, entityType, entityID, depth, "upstream")
}

func (s *LineageService) Downstream(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	return s.builder.BuildDirectionalGraph(ctx, tenantID, entityType, entityID, depth, "downstream")
}

func (s *LineageService) Impact(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID) (*model.ImpactAnalysis, error) {
	return s.analyzer.Analyze(ctx, tenantID, entityType, entityID)
}

func (s *LineageService) Record(ctx context.Context, tenantID uuid.UUID, req dto.RecordLineageEdgeRequest) (*model.LineageEdgeRecord, error) {
	sourceType := model.LineageEntityType(req.SourceType)
	targetType := model.LineageEntityType(req.TargetType)
	relationship := model.LineageRelationship(req.Relationship)
	recordedBy := model.LineageRecordedBy(req.RecordedBy)
	if !sourceType.IsValid() || !targetType.IsValid() {
		return nil, fmt.Errorf("%w: invalid lineage entity type", ErrValidation)
	}
	if !relationship.IsValid() {
		return nil, fmt.Errorf("%w: invalid lineage relationship", ErrValidation)
	}
	if recordedBy == "" {
		recordedBy = model.LineageRecordedByManual
	}
	if req.SourceID == uuid.Nil || req.TargetID == uuid.Nil {
		return nil, fmt.Errorf("%w: source_id and target_id are required", ErrValidation)
	}
	if req.SourceID == req.TargetID && sourceType == targetType {
		return nil, fmt.Errorf("%w: lineage edges cannot self-reference", ErrValidation)
	}
	edge := &model.LineageEdgeRecord{
		TenantID:           tenantID,
		SourceType:         sourceType,
		SourceID:           req.SourceID,
		SourceName:         req.SourceName,
		TargetType:         targetType,
		TargetID:           req.TargetID,
		TargetName:         req.TargetName,
		Relationship:       relationship,
		TransformationDesc: req.TransformationDesc,
		TransformationType: req.TransformationType,
		ColumnsAffected:    req.ColumnsAffected,
		PipelineID:         req.PipelineID,
		PipelineRunID:      req.PipelineRunID,
		RecordedBy:         recordedBy,
		Active:             true,
		FirstSeenAt:        time.Now().UTC(),
		LastSeenAt:         time.Now().UTC(),
	}
	if createsCycle, err := s.wouldCreateCycle(ctx, tenantID, edge); err != nil {
		return nil, err
	} else if createsCycle {
		return nil, fmt.Errorf("%w: lineage edge would create a cycle", ErrConflict)
	}
	if err := s.repo.Upsert(ctx, edge); err != nil {
		return nil, err
	}
	if err := s.publish(ctx, "data.lineage.edge_created", tenantID, map[string]any{
		"id":           edge.ID,
		"source_type":  edge.SourceType,
		"source_id":    edge.SourceID,
		"target_type":  edge.TargetType,
		"target_id":    edge.TargetID,
		"relationship": edge.Relationship,
	}); err != nil {
		s.logger.Warn().Err(err).Msg("failed to publish lineage edge created event")
	}
	return edge, nil
}

func (s *LineageService) DeleteEdge(ctx context.Context, tenantID, edgeID uuid.UUID) error {
	if err := s.repo.Deactivate(ctx, tenantID, edgeID); err != nil {
		return err
	}
	return s.publish(ctx, "data.lineage.edge_removed", tenantID, map[string]any{"id": edgeID})
}

func (s *LineageService) Search(ctx context.Context, tenantID uuid.UUID, params dto.SearchLineageParams) ([]model.LineageNode, error) {
	return s.builder.Search(ctx, tenantID, params.Query, params.Type, params.Limit)
}

func (s *LineageService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.LineageStatsSummary, error) {
	return s.builder.Stats(ctx, tenantID)
}

func (s *LineageService) RecordPipelineRun(ctx context.Context, pipeline *model.Pipeline, run *model.PipelineRun) error {
	return s.recorder.RecordPipelineLineage(ctx, pipeline, run)
}

func (s *LineageService) RecordQueryExecution(ctx context.Context, tenantID, userID, modelID uuid.UUID, query model.AnalyticsQuery) error {
	return s.recorder.RecordQueryLineage(ctx, tenantID, userID, modelID, query)
}

func (s *LineageService) wouldCreateCycle(ctx context.Context, tenantID uuid.UUID, edge *model.LineageEdgeRecord) (bool, error) {
	edges, err := s.repo.ListActive(ctx, tenantID)
	if err != nil {
		return false, err
	}
	targetKey := fmt.Sprintf("%s:%s", edge.TargetType, edge.TargetID)
	sourceKey := fmt.Sprintf("%s:%s", edge.SourceType, edge.SourceID)
	outgoing := make(map[string][]string)
	for _, item := range edges {
		from := fmt.Sprintf("%s:%s", item.SourceType, item.SourceID)
		to := fmt.Sprintf("%s:%s", item.TargetType, item.TargetID)
		outgoing[from] = append(outgoing[from], to)
	}
	queue := []string{targetKey}
	visited := map[string]struct{}{targetKey: {}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == sourceKey {
			return true, nil
		}
		for _, next := range outgoing[current] {
			if _, ok := visited[next]; ok {
				continue
			}
			visited[next] = struct{}{}
			queue = append(queue, next)
		}
	}
	return false, nil
}

func (s *LineageService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if s.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, "data.lineage.events", event)
}

