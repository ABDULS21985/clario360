package rca

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// QualityIssueAnalyzer performs RCA for data quality issues by tracing upstream.
type QualityIssueAnalyzer struct {
	dataDB      *pgxpool.Pool
	timeline    *TimelineBuilder
	chain       *ChainBuilder
	recommender *Recommender
	logger      zerolog.Logger
}

// NewQualityIssueAnalyzer creates a quality issue RCA analyzer.
func NewQualityIssueAnalyzer(
	dataDB *pgxpool.Pool,
	timeline *TimelineBuilder,
	chain *ChainBuilder,
	recommender *Recommender,
	logger zerolog.Logger,
) *QualityIssueAnalyzer {
	return &QualityIssueAnalyzer{
		dataDB:      dataDB,
		timeline:    timeline,
		chain:       chain,
		recommender: recommender,
		logger:      logger.With().Str("analyzer", "quality_issue").Logger(),
	}
}

// Analyze performs RCA on a data quality issue.
func (a *QualityIssueAnalyzer) Analyze(ctx context.Context, tenantID, issueID uuid.UUID) (*RootCauseAnalysis, error) {
	issue, err := a.loadQualityIssue(ctx, tenantID, issueID)
	if err != nil {
		return nil, fmt.Errorf("load quality issue: %w", err)
	}

	var causalSteps []CausalStep
	rootCauseType := "unknown"
	order := 1

	// The failing quality rule itself
	causalSteps = append(causalSteps, CausalStep{
		Order:       order,
		EventID:     issueID.String(),
		EventType:   "quality_rule_failure",
		Description: fmt.Sprintf("Quality rule '%s' failed on %s.%s", issue.ruleName, issue.sourceName, issue.tableName),
		Timestamp:   issue.detectedAt,
		Evidence: []Evidence{
			{Label: "Rule", Field: "rule_name", Value: issue.ruleName, Description: fmt.Sprintf("Rule: %s, Column: %s, Expected: %s, Actual: %s", issue.ruleName, issue.columnName, issue.expectedValue, issue.actualValue)},
		},
		Metadata: map[string]interface{}{
			"rule_name":      issue.ruleName,
			"source_name":    issue.sourceName,
			"table_name":     issue.tableName,
			"column_name":    issue.columnName,
			"expected_value": issue.expectedValue,
			"actual_value":   issue.actualValue,
		},
	})
	order++

	// Trace upstream via lineage
	if a.dataDB != nil && issue.sourceID != nil {
		upstreamSteps := a.traceUpstream(ctx, tenantID, *issue.sourceID, issue.detectedAt, order)
		if len(upstreamSteps) > 0 {
			rootCauseType = "upstream_quality"
			causalSteps = append(causalSteps, upstreamSteps...)
			order += len(upstreamSteps)
		}
	}

	// Check for schema changes
	if rootCauseType == "unknown" && a.dataDB != nil && issue.sourceID != nil {
		schemaChanged := a.checkSchemaChange(ctx, tenantID, *issue.sourceID, issue.detectedAt)
		if schemaChanged {
			rootCauseType = "schema_change"
			causalSteps = append(causalSteps, CausalStep{
				Order:       order,
				EventID:     uuid.New().String(),
				EventType:   "schema_change",
				Description: "Schema change detected on source before quality failure",
				Timestamp:   issue.detectedAt.Add(-1 * time.Hour),
				Evidence: []Evidence{
					{Label: "Schema Change", Field: "schema_change", Value: "detected", Description: "Source schema was modified within 7 days of the quality issue."},
				},
				IsRootCause: true,
			})
		}
	}

	// Mark root cause
	if len(causalSteps) > 0 {
		causalSteps[len(causalSteps)-1].IsRootCause = true
	}

	recommendations := a.recommender.ForQualityIssue(rootCauseType)

	confidence := 0.5
	if rootCauseType != "unknown" {
		confidence = 0.7
	}

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
		Type:            AnalysisTypeQuality,
		IncidentID:      issueID,
		Status:          "completed",
		RootCause:       rootCause,
		CausalChain:     causalSteps,
		Recommendations: recommendations,
		Confidence:      confidence,
		Summary:         fmt.Sprintf("Quality issue in %s.%s: root cause is %s", issue.sourceName, issue.tableName, rootCauseType),
		AnalyzedAt:      time.Now().UTC(),
	}, nil
}

type qualityIssueInfo struct {
	id            uuid.UUID
	sourceID      *uuid.UUID
	sourceName    string
	tableName     string
	columnName    string
	ruleName      string
	expectedValue string
	actualValue   string
	detectedAt    time.Time
}

func (a *QualityIssueAnalyzer) loadQualityIssue(ctx context.Context, tenantID, issueID uuid.UUID) (*qualityIssueInfo, error) {
	if a.dataDB == nil {
		return nil, fmt.Errorf("data database not configured")
	}

	info := &qualityIssueInfo{id: issueID}
	err := a.dataDB.QueryRow(ctx, `
		SELECT r.source_id, COALESCE(ds.name, ''), r.table_name, COALESCE(r.column_name, ''),
		       r.rule_name, COALESCE(r.expected_value, ''), COALESCE(r.actual_value, ''),
		       r.created_at
		FROM quality_results r
		LEFT JOIN data_sources ds ON ds.id = r.source_id AND ds.tenant_id = r.tenant_id
		WHERE r.tenant_id = $1 AND r.id = $2
	`, tenantID, issueID).Scan(
		&info.sourceID, &info.sourceName, &info.tableName, &info.columnName,
		&info.ruleName, &info.expectedValue, &info.actualValue,
		&info.detectedAt,
	)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (a *QualityIssueAnalyzer) traceUpstream(ctx context.Context, tenantID uuid.UUID, sourceID uuid.UUID, detectedAt time.Time, startOrder int) []CausalStep {
	rows, err := a.dataDB.Query(ctx, `
		SELECT le.source_id, le.source_name, qr.id, qr.rule_name, qr.created_at
		FROM data_lineage_edges le
		JOIN quality_results qr ON qr.source_id = le.source_id::uuid AND qr.tenant_id = le.tenant_id
		WHERE le.tenant_id = $1
		  AND le.target_id = $2::text
		  AND le.active = true
		  AND qr.status = 'failed'
		  AND qr.created_at >= $3 - INTERVAL '24 hours'
		ORDER BY qr.created_at DESC
		LIMIT 5
	`, tenantID, sourceID, detectedAt)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var steps []CausalStep
	for rows.Next() {
		var upstreamSourceID, upstreamIssueID string
		var upstreamName, upstreamRuleName string
		var upstreamTime time.Time

		if err := rows.Scan(&upstreamSourceID, &upstreamName, &upstreamIssueID, &upstreamRuleName, &upstreamTime); err != nil {
			continue
		}

		steps = append(steps, CausalStep{
			Order:       startOrder,
			EventID:     upstreamIssueID,
			EventType:   "upstream_quality_failure",
			Description: fmt.Sprintf("Upstream quality failure: rule '%s' failed on %s", upstreamRuleName, upstreamName),
			Timestamp:   upstreamTime,
			Evidence: []Evidence{
				{Label: "Upstream Rule", Field: "upstream_rule", Value: upstreamRuleName, Description: fmt.Sprintf("Quality rule %s failed on upstream source %s before the downstream issue.", upstreamRuleName, upstreamName)},
			},
		})
		startOrder++
	}

	return steps
}

func (a *QualityIssueAnalyzer) checkSchemaChange(ctx context.Context, tenantID uuid.UUID, sourceID uuid.UUID, detectedAt time.Time) bool {
	var count int
	err := a.dataDB.QueryRow(ctx, `
		SELECT COUNT(*) FROM schema_change_log
		WHERE tenant_id = $1 AND source_id = $2
		  AND detected_at >= $3 - INTERVAL '7 days'
	`, tenantID, sourceID, detectedAt).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
