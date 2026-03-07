package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

const lineageEventsTopic = "data.lineage.events"

type LineageRecorder struct {
	repo         *repository.LineageRepository
	sourceRepo    *repository.SourceRepository
	modelRepo     *repository.ModelRepository
	producer      *events.Producer
	logger        zerolog.Logger
}

func NewLineageRecorder(repo *repository.LineageRepository, sourceRepo *repository.SourceRepository, modelRepo *repository.ModelRepository, producer *events.Producer, logger zerolog.Logger) *LineageRecorder {
	return &LineageRecorder{
		repo:      repo,
		sourceRepo: sourceRepo,
		modelRepo:  modelRepo,
		producer:   producer,
		logger:     logger,
	}
}

func (r *LineageRecorder) RecordPipelineLineage(ctx context.Context, pipeline *model.Pipeline, run *model.PipelineRun) error {
	sourceRecord, err := r.sourceRepo.Get(ctx, pipeline.TenantID, pipeline.SourceID)
	if err != nil {
		return err
	}
	columns := transformationColumns(pipeline.Config.Transformations)
	desc := transformationSummary(pipeline.Config.Transformations)

	sourceToPipeline := &model.LineageEdgeRecord{
		TenantID:        pipeline.TenantID,
		SourceType:      model.LineageEntityDataSource,
		SourceID:        sourceRecord.Source.ID,
		SourceName:      sourceRecord.Source.Name,
		TargetType:      model.LineageEntityPipeline,
		TargetID:        pipeline.ID,
		TargetName:      pipeline.Name,
		Relationship:    model.LineageRelationshipFeeds,
		ColumnsAffected: columns,
		PipelineID:      &pipeline.ID,
		RecordedBy:      model.LineageRecordedByPipeline,
	}
	if err := r.repo.Upsert(ctx, sourceToPipeline); err != nil {
		return err
	}
	if err := r.publish(ctx, "data.lineage.edge_created", pipeline.TenantID, map[string]any{
		"id":           sourceToPipeline.ID,
		"source_type":  sourceToPipeline.SourceType,
		"source_id":    sourceToPipeline.SourceID,
		"target_type":  sourceToPipeline.TargetType,
		"target_id":    sourceToPipeline.TargetID,
		"relationship": sourceToPipeline.Relationship,
	}); err != nil {
		r.logger.Warn().Err(err).Msg("failed to publish lineage edge event")
	}

	targetType := model.LineageEntityDataSource
	targetID := uuid.Nil
	targetName := ""
	if pipeline.Config.TargetModelID != nil {
		targetType = model.LineageEntityDataModel
		targetID = *pipeline.Config.TargetModelID
		if modelItem, err := r.modelRepo.Get(ctx, pipeline.TenantID, *pipeline.Config.TargetModelID); err == nil {
			targetName = modelItem.DisplayName
		}
	} else if pipeline.TargetID != nil {
		targetID = *pipeline.TargetID
		if targetRecord, err := r.sourceRepo.Get(ctx, pipeline.TenantID, *pipeline.TargetID); err == nil {
			targetName = targetRecord.Source.Name
		}
	}
	if targetID == uuid.Nil {
		return nil
	}
	pipelineToTarget := &model.LineageEdgeRecord{
		TenantID:           pipeline.TenantID,
		SourceType:         model.LineageEntityPipeline,
		SourceID:           pipeline.ID,
		SourceName:         pipeline.Name,
		TargetType:         targetType,
		TargetID:           targetID,
		TargetName:         targetName,
		Relationship:       model.LineageRelationshipTransformsInto,
		TransformationDesc: pointerString(desc),
		TransformationType: pointerString(strings.ToLower(string(pipeline.Type))),
		ColumnsAffected:    columns,
		PipelineID:         &pipeline.ID,
		PipelineRunID:      &run.ID,
		RecordedBy:         model.LineageRecordedByPipeline,
	}
	if err := r.repo.Upsert(ctx, pipelineToTarget); err != nil {
		return err
	}
	return r.publish(ctx, "data.lineage.edge_created", pipeline.TenantID, map[string]any{
		"id":           pipelineToTarget.ID,
		"source_type":  pipelineToTarget.SourceType,
		"source_id":    pipelineToTarget.SourceID,
		"target_type":  pipelineToTarget.TargetType,
		"target_id":    pipelineToTarget.TargetID,
		"relationship": pipelineToTarget.Relationship,
	})
}

func (r *LineageRecorder) RecordQueryLineage(ctx context.Context, tenantID, userID, modelID uuid.UUID, query model.AnalyticsQuery) error {
	modelItem, err := r.modelRepo.Get(ctx, tenantID, modelID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(query)
	queryID := uuid.NewSHA1(uuid.NameSpaceURL, append([]byte(modelID.String()), payload...))
	queryName := fmt.Sprintf("Ad hoc query by %s", userID.String())
	columns := analyticsColumns(query)
	edge := &model.LineageEdgeRecord{
		TenantID:        tenantID,
		SourceType:      model.LineageEntityDataModel,
		SourceID:        modelItem.ID,
		SourceName:      modelItem.DisplayName,
		TargetType:      model.LineageEntityAnalyticsQuery,
		TargetID:        queryID,
		TargetName:      queryName,
		Relationship:    model.LineageRelationshipQueriedBy,
		ColumnsAffected: columns,
		RecordedBy:      model.LineageRecordedByQuery,
	}
	if err := r.repo.Upsert(ctx, edge); err != nil {
		return err
	}
	return r.publish(ctx, "data.lineage.edge_created", tenantID, map[string]any{
		"id":           edge.ID,
		"source_type":  edge.SourceType,
		"source_id":    edge.SourceID,
		"target_type":  edge.TargetType,
		"target_id":    edge.TargetID,
		"relationship": edge.Relationship,
	})
}

func (r *LineageRecorder) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if r.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return r.producer.Publish(ctx, lineageEventsTopic, event)
}

func transformationColumns(transforms []model.Transformation) []string {
	columns := make([]string, 0)
	for _, item := range transforms {
		var payload map[string]any
		if err := json.Unmarshal(item.Config, &payload); err != nil {
			continue
		}
		for _, key := range []string{"column", "from", "to", "field"} {
			if value, ok := payload[key].(string); ok {
				columns = append(columns, value)
			}
		}
	}
	return mergeStringSlices(columns)
}

func transformationSummary(transforms []model.Transformation) string {
	if len(transforms) == 0 {
		return "direct pass-through"
	}
	parts := make([]string, 0, len(transforms))
	for _, item := range transforms {
		parts = append(parts, string(item.Type))
	}
	return strings.Join(parts, ", ")
}

func analyticsColumns(query model.AnalyticsQuery) []string {
	columns := append([]string(nil), query.Columns...)
	columns = append(columns, query.GroupBy...)
	for _, order := range query.OrderBy {
		columns = append(columns, order.Column)
	}
	for _, agg := range query.Aggregations {
		columns = append(columns, agg.Column)
	}
	for _, filter := range query.Filters {
		columns = append(columns, filter.Column)
	}
	return mergeStringSlices(columns)
}

func pointerString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

