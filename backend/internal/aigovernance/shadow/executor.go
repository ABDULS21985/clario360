package shadow

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type ExplanationService interface {
	Explain(ctx context.Context, version *aigovmodel.ModelVersion, input any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error)
}

type ExecutionTask struct {
	TenantID          uuid.UUID
	ShadowVersion     *aigovmodel.ModelVersion
	ProductionVersion *aigovmodel.ModelVersion
	Params            aigovernance.PredictParams
	ProductionResult  *aigovernance.ModelOutput
	InputHash         string
	InputSummary      map[string]any
}

type ExecutionResult struct {
	Log        *aigovmodel.PredictionLog
	Divergence *aigovmodel.ShadowDivergence
	Agreement  bool
}

type Executor struct {
	explanationSvc ExplanationService
	logger         zerolog.Logger
}

func NewExecutor(explanationSvc ExplanationService, logger zerolog.Logger) *Executor {
	return &Executor{
		explanationSvc: explanationSvc,
		logger:         logger.With().Str("component", "ai_shadow_executor").Logger(),
	}
}

func (e *Executor) Execute(ctx context.Context, task *ExecutionTask) (*ExecutionResult, error) {
	start := time.Now()
	output, err := task.Params.ShadowModelFunc(ctx, task.Params.Input)
	if err != nil {
		return nil, err
	}
	explanation, err := e.explanationSvc.Explain(ctx, task.ShadowVersion, task.Params.Input, output)
	if err != nil {
		return nil, err
	}
	logEntry := &aigovmodel.PredictionLog{
		ID:                        uuid.New(),
		TenantID:                  task.TenantID,
		ModelID:                   task.ShadowVersion.ModelID,
		ModelVersionID:            task.ShadowVersion.ID,
		ModelSlug:                 task.ShadowVersion.ModelSlug,
		ModelVersionNumber:        task.ShadowVersion.VersionNumber,
		InputHash:                 task.InputHash,
		InputSummary:              mustJSON(task.InputSummary),
		Prediction:                mustJSON(output.Output),
		Confidence:                confidencePtr(output.Confidence),
		ExplanationStructured:     mustJSON(explanation.Structured),
		ExplanationText:           explanation.HumanReadable,
		ExplanationFactors:        mustJSON(explanation.Factors),
		Suite:                     string(task.ShadowVersion.ModelSuite),
		UseCase:                   task.Params.UseCase,
		EntityType:                task.Params.EntityType,
		EntityID:                  task.Params.EntityID,
		IsShadow:                  true,
		ShadowProductionVersionID: &task.ProductionVersion.ID,
		LatencyMS:                 int(time.Since(start).Milliseconds()),
		CreatedAt:                 time.Now().UTC(),
	}
	agreement, divergence := ComparePredictionLogs(
		&aigovmodel.PredictionLog{
			ID:                 uuid.New(),
			InputHash:          task.InputHash,
			Prediction:         mustJSON(task.ProductionResult.Output),
			Confidence:         confidencePtr(task.ProductionResult.Confidence),
			UseCase:            task.Params.UseCase,
			EntityID:           task.Params.EntityID,
		},
		logEntry,
	)
	if divergence != nil {
		logEntry.ShadowDivergence = mustJSON(divergence)
	}
	return &ExecutionResult{
		Log:        logEntry,
		Divergence: divergence,
		Agreement:  agreement,
	}, nil
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func confidencePtr(value float64) *float64 {
	return &value
}
