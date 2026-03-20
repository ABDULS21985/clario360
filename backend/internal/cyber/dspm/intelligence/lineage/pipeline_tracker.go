package lineage

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// PipelineTracker creates lineage edges from ETL/pipeline completion events.
type PipelineTracker struct {
	logger zerolog.Logger
}

// NewPipelineTracker creates a PipelineTracker.
func NewPipelineTracker(logger zerolog.Logger) *PipelineTracker {
	return &PipelineTracker{
		logger: logger.With().Str("component", "pipeline_tracker").Logger(),
	}
}

// TrackPipelineEvent creates a lineage edge representing a data transfer
// between two assets via an ETL/data pipeline. The edge is populated with
// pipeline metadata and marked as active with high confidence.
func (t *PipelineTracker) TrackPipelineEvent(
	sourceAssetID, targetAssetID uuid.UUID,
	pipelineID, pipelineName string,
) *model.LineageEdge {
	now := time.Now().UTC()

	edge := &model.LineageEdge{
		ID:            uuid.New(),
		SourceAssetID: sourceAssetID,
		TargetAssetID: targetAssetID,
		EdgeType:      model.EdgeTypeETLPipeline,
		PipelineID:    pipelineID,
		PipelineName:  pipelineName,
		Confidence:    1.0,
		Status:        model.EdgeStatusActive,
		LastTransferAt: &now,
		Evidence: map[string]interface{}{
			"source":        "pipeline_event",
			"pipeline_id":   pipelineID,
			"pipeline_name": pipelineName,
			"tracked_at":    now.Format(time.RFC3339),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.logger.Info().
		Str("source_asset", sourceAssetID.String()).
		Str("target_asset", targetAssetID.String()).
		Str("pipeline_id", pipelineID).
		Str("pipeline_name", pipelineName).
		Msg("pipeline lineage edge created")

	return edge
}
