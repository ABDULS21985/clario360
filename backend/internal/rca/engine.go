package rca

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Engine orchestrates root cause analysis across different incident types.
type Engine struct {
	securityAnalyzer *SecurityAlertAnalyzer
	pipelineAnalyzer *PipelineFailureAnalyzer
	qualityAnalyzer  *QualityIssueAnalyzer
	logger           zerolog.Logger
}

// NewEngine creates an RCA engine with all analyzers.
func NewEngine(
	cyberDB *pgxpool.Pool,
	dataDB *pgxpool.Pool,
	logger zerolog.Logger,
) *Engine {
	log := logger.With().Str("component", "rca-engine").Logger()

	timeline := NewTimelineBuilder(cyberDB, dataDB, log)
	chain := NewChainBuilder()
	impact := NewImpactAssessor(cyberDB, log)
	recommender := NewRecommender()

	return &Engine{
		securityAnalyzer: NewSecurityAlertAnalyzer(cyberDB, timeline, chain, impact, recommender, log),
		pipelineAnalyzer: NewPipelineFailureAnalyzer(dataDB, cyberDB, timeline, chain, impact, recommender, log),
		qualityAnalyzer:  NewQualityIssueAnalyzer(dataDB, timeline, chain, recommender, log),
		logger:           log,
	}
}

// Analyze performs root cause analysis for the given incident.
func (e *Engine) Analyze(ctx context.Context, tenantID uuid.UUID, req AnalyzeRequest) (*RootCauseAnalysis, error) {
	start := time.Now()
	e.logger.Info().
		Str("type", string(req.Type)).
		Str("incident_id", req.IncidentID.String()).
		Msg("starting RCA")

	var result *RootCauseAnalysis
	var err error

	switch req.Type {
	case AnalysisTypeSecurity:
		result, err = e.securityAnalyzer.Analyze(ctx, tenantID, req.IncidentID)
	case AnalysisTypePipeline:
		result, err = e.pipelineAnalyzer.Analyze(ctx, tenantID, req.IncidentID)
	case AnalysisTypeQuality:
		result, err = e.qualityAnalyzer.Analyze(ctx, tenantID, req.IncidentID)
	default:
		return nil, fmt.Errorf("unsupported analysis type: %s", req.Type)
	}

	if err != nil {
		e.logger.Error().Err(err).Str("type", string(req.Type)).Msg("RCA failed")
		return nil, err
	}

	result.DurationMs = time.Since(start).Milliseconds()
	e.logger.Info().
		Str("type", string(req.Type)).
		Int64("duration_ms", result.DurationMs).
		Int("chain_length", len(result.CausalChain)).
		Float64("confidence", result.Confidence).
		Msg("RCA completed")

	return result, nil
}

// GetCachedResult retrieves a previously computed RCA result.
func (e *Engine) GetCachedResult(ctx context.Context, tenantID uuid.UUID, analysisType AnalysisType, incidentID uuid.UUID) (*RootCauseAnalysis, error) {
	// In a production system, this would query a database cache.
	// For now, we re-analyze on each request.
	return e.Analyze(ctx, tenantID, AnalyzeRequest{
		Type:       analysisType,
		IncidentID: incidentID,
	})
}

// GetTimeline retrieves just the event timeline for an incident.
func (e *Engine) GetTimeline(ctx context.Context, tenantID uuid.UUID, analysisType AnalysisType, incidentID uuid.UUID) ([]TimelineEvent, error) {
	result, err := e.Analyze(ctx, tenantID, AnalyzeRequest{
		Type:       analysisType,
		IncidentID: incidentID,
	})
	if err != nil {
		return nil, err
	}
	return result.Timeline, nil
}
