package rca

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PipelineFailureAnalyzer performs RCA for pipeline failures by walking data lineage upstream.
type PipelineFailureAnalyzer struct {
	dataDB      *pgxpool.Pool
	cyberDB     *pgxpool.Pool
	timeline    *TimelineBuilder
	chain       *ChainBuilder
	impact      *ImpactAssessor
	recommender *Recommender
	logger      zerolog.Logger
}

// NewPipelineFailureAnalyzer creates a pipeline failure RCA analyzer.
func NewPipelineFailureAnalyzer(
	dataDB *pgxpool.Pool,
	cyberDB *pgxpool.Pool,
	timeline *TimelineBuilder,
	chain *ChainBuilder,
	impact *ImpactAssessor,
	recommender *Recommender,
	logger zerolog.Logger,
) *PipelineFailureAnalyzer {
	return &PipelineFailureAnalyzer{
		dataDB:      dataDB,
		cyberDB:     cyberDB,
		timeline:    timeline,
		chain:       chain,
		impact:      impact,
		recommender: recommender,
		logger:      logger.With().Str("analyzer", "pipeline_failure").Logger(),
	}
}

// Analyze performs RCA on a pipeline failure by:
// 1. Loading the failed run with error details
// 2. Checking immediate causes (timeout, permissions, schema, OOM, quality)
// 3. Walking upstream via data lineage to find cascading failures
// 4. Checking for recent schema or configuration changes
// 5. Generating recommendations
func (a *PipelineFailureAnalyzer) Analyze(ctx context.Context, tenantID, runID uuid.UUID) (*RootCauseAnalysis, error) {
	run, err := a.loadFailedRun(ctx, tenantID, runID)
	if err != nil {
		return nil, fmt.Errorf("load failed run: %w", err)
	}

	var causalSteps []CausalStep
	var rootCauseType string
	order := 1

	// 2. Check immediate causes
	immediateCause, immediateType := a.classifyImmediateCause(run)
	causalSteps = append(causalSteps, CausalStep{
		Order:       order,
		EventID:     runID.String(),
		EventType:   "pipeline_run",
		Description: immediateCause,
		Timestamp:   run.failedAt,
		Evidence: []Evidence{
			{Label: "Pipeline", Field: "pipeline_name", Value: run.pipelineName, Description: fmt.Sprintf("Pipeline %s failed in phase %s: %s", run.pipelineName, run.errorPhase, run.errorMessage)},
		},
		Metadata: map[string]interface{}{
			"pipeline_id":   run.pipelineID.String(),
			"pipeline_name": run.pipelineName,
			"error_phase":   run.errorPhase,
			"error_message": run.errorMessage,
		},
	})
	order++
	rootCauseType = immediateType

	// 3. Walk upstream for cascade failures
	if a.dataDB != nil {
		upstreamCause, upstreamSteps := a.walkUpstream(ctx, tenantID, run.pipelineID, run.failedAt, order)
		if upstreamCause != "" {
			rootCauseType = "upstream_failure"
			for i := range upstreamSteps {
				upstreamSteps[i].Order = order
				order++
			}
			causalSteps = append(causalSteps, upstreamSteps...)
		}
	}

	// 4. Check for recent schema changes
	if rootCauseType == "schema_drift" || rootCauseType == "unknown" {
		schemaSteps := a.checkSchemaChanges(ctx, tenantID, run, order)
		if len(schemaSteps) > 0 {
			rootCauseType = "schema_drift"
			causalSteps = append(causalSteps, schemaSteps...)
		}
	}

	// Mark root cause
	if len(causalSteps) > 0 {
		causalSteps[len(causalSteps)-1].IsRootCause = true
	}

	recommendations := a.recommender.ForPipelineFailure(rootCauseType)

	timelineEvents, _ := a.timeline.BuildForPipeline(ctx, tenantID, runID, 24*time.Hour)

	var impactResult *ImpactAssessment
	if run.sourceID != nil || run.targetID != nil {
		var assetIDs []uuid.UUID
		if run.sourceID != nil {
			assetIDs = append(assetIDs, *run.sourceID)
		}
		if run.targetID != nil {
			assetIDs = append(assetIDs, *run.targetID)
		}
		impactResult, _ = a.impact.AssessForPipeline(ctx, tenantID, assetIDs)
	}

	confidence := 0.6
	if rootCauseType != "unknown" {
		confidence = 0.75
	}
	if rootCauseType == "upstream_failure" || rootCauseType == "connection_timeout" {
		confidence = 0.85
	}

	summary := buildPipelineSummary(run, rootCauseType, len(causalSteps))

	var rootCause *CausalStep
	for i := range causalSteps {
		if causalSteps[i].IsRootCause {
			rootCause = &causalSteps[i]
			break
		}
	}

	return &RootCauseAnalysis{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Type:            AnalysisTypePipeline,
		IncidentID:      runID,
		Status:          "completed",
		RootCause:       rootCause,
		CausalChain:     causalSteps,
		Timeline:        timelineEvents,
		Impact:          impactResult,
		Recommendations: recommendations,
		Confidence:      confidence,
		Summary:         summary,
		AnalyzedAt:      time.Now().UTC(),
	}, nil
}

type pipelineRunInfo struct {
	id           uuid.UUID
	pipelineID   uuid.UUID
	pipelineName string
	status       string
	errorPhase   string
	errorMessage string
	sourceID     *uuid.UUID
	targetID     *uuid.UUID
	failedAt     time.Time
}

func (a *PipelineFailureAnalyzer) loadFailedRun(ctx context.Context, tenantID, runID uuid.UUID) (*pipelineRunInfo, error) {
	if a.dataDB == nil {
		return nil, fmt.Errorf("data database not configured")
	}

	info := &pipelineRunInfo{id: runID}
	err := a.dataDB.QueryRow(ctx, `
		SELECT r.pipeline_id, p.name, r.status::text,
		       COALESCE(r.error_phase, r.current_phase, ''),
		       COALESCE(r.error_message, ''),
		       p.source_id, p.target_id,
		       COALESCE(r.completed_at, r.created_at)
		FROM pipeline_runs r
		JOIN pipelines p ON p.id = r.pipeline_id AND p.tenant_id = r.tenant_id
		WHERE r.tenant_id = $1 AND r.id = $2
	`, tenantID, runID).Scan(
		&info.pipelineID, &info.pipelineName, &info.status,
		&info.errorPhase, &info.errorMessage,
		&info.sourceID, &info.targetID,
		&info.failedAt,
	)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (a *PipelineFailureAnalyzer) classifyImmediateCause(run *pipelineRunInfo) (string, string) {
	msg := strings.ToLower(run.errorMessage)

	if strings.Contains(msg, "timeout") || strings.Contains(msg, "connection refused") || strings.Contains(msg, "unreachable") {
		return fmt.Sprintf("Source connection timeout: %s unreachable", run.pipelineName), "connection_timeout"
	}
	if strings.Contains(msg, "permission denied") || strings.Contains(msg, "access denied") || strings.Contains(msg, "unauthorized") {
		return fmt.Sprintf("Permission denied accessing data source for pipeline %s", run.pipelineName), "credential_expiry"
	}
	if strings.Contains(msg, "schema") || strings.Contains(msg, "column") || strings.Contains(msg, "type mismatch") {
		return fmt.Sprintf("Schema mismatch detected in pipeline %s", run.pipelineName), "schema_drift"
	}
	if strings.Contains(msg, "oom") || strings.Contains(msg, "memory") || strings.Contains(msg, "resource") {
		return fmt.Sprintf("Resource exhaustion in pipeline %s", run.pipelineName), "resource_exhaustion"
	}
	if strings.Contains(msg, "quality") || strings.Contains(msg, "validation") || strings.Contains(msg, "gate") {
		return fmt.Sprintf("Quality gate failed in pipeline %s", run.pipelineName), "quality_gate"
	}
	if strings.Contains(msg, "expired") || strings.Contains(msg, "token") || strings.Contains(msg, "credential") {
		return fmt.Sprintf("Credential expiry for pipeline %s", run.pipelineName), "credential_expiry"
	}

	return fmt.Sprintf("Pipeline %s failed: %s", run.pipelineName, run.errorMessage), "unknown"
}

func (a *PipelineFailureAnalyzer) walkUpstream(ctx context.Context, tenantID, pipelineID uuid.UUID, failTime time.Time, startOrder int) (string, []CausalStep) {
	if a.dataDB == nil {
		return "", nil
	}

	rows, err := a.dataDB.Query(ctx, `
		SELECT DISTINCT p.id, p.name, r.id, r.status::text, r.error_message, r.created_at
		FROM lineage_edges le
		JOIN pipelines p ON p.id = le.source_id::uuid AND p.tenant_id = le.tenant_id
		JOIN pipeline_runs r ON r.pipeline_id = p.id AND r.tenant_id = p.tenant_id
		WHERE le.tenant_id = $1
		  AND le.target_id = $2::text
		  AND le.active = true
		  AND r.status = 'failed'
		  AND r.created_at >= $3 - INTERVAL '24 hours'
		  AND r.created_at <= $3
		ORDER BY r.created_at DESC
		LIMIT 5
	`, tenantID, pipelineID, failTime)
	if err != nil {
		return "", nil
	}
	defer rows.Close()

	var steps []CausalStep
	for rows.Next() {
		var upstreamPipelineID, upstreamRunID uuid.UUID
		var upstreamName, upstreamStatus string
		var upstreamError *string
		var upstreamTime time.Time

		if err := rows.Scan(&upstreamPipelineID, &upstreamName, &upstreamRunID, &upstreamStatus, &upstreamError, &upstreamTime); err != nil {
			continue
		}

		errMsg := ""
		if upstreamError != nil {
			errMsg = *upstreamError
		}

		steps = append(steps, CausalStep{
			Order:       startOrder,
			EventID:     upstreamRunID.String(),
			EventType:   "upstream_pipeline_failure",
			Description: fmt.Sprintf("Upstream pipeline %s failed at %s: %s", upstreamName, upstreamTime.Format(time.RFC3339), errMsg),
			Timestamp:   upstreamTime,
			Evidence: []Evidence{
				{Label: "Upstream Pipeline", Field: "upstream_pipeline", Value: upstreamName, Description: fmt.Sprintf("Pipeline %s (run %s) failed with status %s before dependent pipeline.", upstreamName, upstreamRunID.String(), upstreamStatus)},
			},
			Metadata: map[string]interface{}{
				"upstream_pipeline_id": upstreamPipelineID.String(),
				"upstream_run_id":      upstreamRunID.String(),
				"upstream_error":       errMsg,
			},
		})
		startOrder++
	}

	if len(steps) > 0 {
		return "Upstream pipeline failure detected", steps
	}
	return "", nil
}

func (a *PipelineFailureAnalyzer) checkSchemaChanges(ctx context.Context, tenantID uuid.UUID, run *pipelineRunInfo, startOrder int) []CausalStep {
	if a.dataDB == nil || run.sourceID == nil {
		return nil
	}

	var changeCount int
	err := a.dataDB.QueryRow(ctx, `
		SELECT COUNT(*) FROM schema_change_log
		WHERE tenant_id = $1 AND source_id = $2
		  AND detected_at >= $3 - INTERVAL '7 days'
	`, tenantID, *run.sourceID, run.failedAt).Scan(&changeCount)
	if err != nil || changeCount == 0 {
		return nil
	}

	return []CausalStep{
		{
			Order:       startOrder,
			EventID:     uuid.New().String(),
			EventType:   "schema_change",
			Description: fmt.Sprintf("Schema change detected on source: %d changes in the last 7 days", changeCount),
			Timestamp:   run.failedAt.Add(-1 * time.Hour),
			Evidence: []Evidence{
				{Label: "Schema Changes", Field: "change_count", Value: changeCount, Description: fmt.Sprintf("Source %s had %d schema changes detected before the pipeline failure.", run.sourceID.String(), changeCount)},
			},
			IsRootCause: true,
			Metadata: map[string]interface{}{
				"source_id":    run.sourceID.String(),
				"change_count": changeCount,
			},
		},
	}
}

func buildPipelineSummary(run *pipelineRunInfo, rootCauseType string, chainLen int) string {
	switch rootCauseType {
	case "connection_timeout":
		return fmt.Sprintf("Root cause: Source connection timeout. Pipeline '%s' failed because the data source was unreachable.", run.pipelineName)
	case "schema_drift":
		return fmt.Sprintf("Root cause: Schema drift. Pipeline '%s' failed because the source schema changed.", run.pipelineName)
	case "upstream_failure":
		return fmt.Sprintf("Root cause: Upstream pipeline failure. Pipeline '%s' depends on output from another pipeline that failed first.", run.pipelineName)
	case "resource_exhaustion":
		return fmt.Sprintf("Root cause: Resource exhaustion. Pipeline '%s' exceeded memory or compute limits.", run.pipelineName)
	case "credential_expiry":
		return fmt.Sprintf("Root cause: Credential expiry. API key or credentials used by pipeline '%s' have expired.", run.pipelineName)
	default:
		return fmt.Sprintf("Root cause analysis for pipeline '%s': %d causal steps identified. Error: %s", run.pipelineName, chainLen, run.errorMessage)
	}
}
