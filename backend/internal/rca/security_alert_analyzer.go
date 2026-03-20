package rca

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// SecurityAlertAnalyzer performs RCA for security alerts by walking the MITRE kill chain.
type SecurityAlertAnalyzer struct {
	cyberDB     *pgxpool.Pool
	timeline    *TimelineBuilder
	chain       *ChainBuilder
	impact      *ImpactAssessor
	recommender *Recommender
	logger      zerolog.Logger
}

// NewSecurityAlertAnalyzer creates a security alert RCA analyzer.
func NewSecurityAlertAnalyzer(
	cyberDB *pgxpool.Pool,
	timeline *TimelineBuilder,
	chain *ChainBuilder,
	impact *ImpactAssessor,
	recommender *Recommender,
	logger zerolog.Logger,
) *SecurityAlertAnalyzer {
	return &SecurityAlertAnalyzer{
		cyberDB:     cyberDB,
		timeline:    timeline,
		chain:       chain,
		impact:      impact,
		recommender: recommender,
		logger:      logger.With().Str("analyzer", "security_alert").Logger(),
	}
}

// Analyze performs RCA on a security alert by:
// 1. Loading the alert with explanation and MITRE technique
// 2. Building a timeline from multiple event sources
// 3. Walking the MITRE kill chain to identify the earliest phase
// 4. Correlating events by shared attributes (IP, user, asset, technique)
// 5. Assessing blast radius impact
// 6. Generating recommendations based on root cause type
func (a *SecurityAlertAnalyzer) Analyze(ctx context.Context, tenantID, alertID uuid.UUID) (*RootCauseAnalysis, error) {
	// 1. Load the alert
	alert, err := a.loadAlert(ctx, tenantID, alertID)
	if err != nil {
		return nil, fmt.Errorf("load alert: %w", err)
	}

	// 2. Build timeline: alert time -2h to +1h
	timelineEvents, err := a.timeline.BuildForAlert(ctx, tenantID, alertID, 2*time.Hour, 1*time.Hour)
	if err != nil {
		a.logger.Warn().Err(err).Msg("build timeline failed, continuing with alert only")
		timelineEvents = []TimelineEvent{
			{
				ID:               alertID.String(),
				Timestamp:        alert.createdAt,
				Source:           "alert",
				Type:             "cyber_alert",
				Summary:          alert.title,
				Severity:         alert.severity,
				MITRETechID: ptrToString(alert.mitreTechniqueID),
			},
		}
	}

	// 3. Walk MITRE kill chain — build causal chain
	causalChain := a.chain.BuildFromTimeline(timelineEvents, AnalysisTypeSecurity)

	// 4. Assess impact
	var impactResult *ImpactAssessment
	if len(alert.assetIDs) > 0 {
		impactResult, err = a.impact.AssessForAlert(ctx, tenantID, alert.assetIDs)
		if err != nil {
			a.logger.Warn().Err(err).Msg("impact assessment failed")
		}
	}

	// 5. Classify root cause and generate recommendations
	rootCauseType := ClassifySecurityRootCause(causalChain)
	recommendations := a.recommender.ForSecurityAlert(rootCauseType, causalChain)

	// 6. Build summary
	confidence := calculateConfidence(len(causalChain), len(timelineEvents))
	summary := buildSecuritySummary(alert, rootCauseType, causalChain)

	// Find the root cause step
	var rootCause *CausalStep
	for i := range causalChain {
		if causalChain[i].IsRootCause {
			rootCause = &causalChain[i]
			break
		}
	}

	return &RootCauseAnalysis{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Type:            AnalysisTypeSecurity,
		IncidentID:      alertID,
		Status:          "completed",
		RootCause:       rootCause,
		CausalChain:     causalChain,
		Timeline:        timelineEvents,
		Impact:          impactResult,
		Recommendations: recommendations,
		Confidence:      confidence,
		Summary:         summary,
		AnalyzedAt:      time.Now().UTC(),
	}, nil
}

type alertInfo struct {
	id               uuid.UUID
	title            string
	severity         string
	source           string
	assetIDs         []uuid.UUID
	mitreTechniqueID *string
	mitreTacticID    *string
	createdAt        time.Time
}

func (a *SecurityAlertAnalyzer) loadAlert(ctx context.Context, tenantID, alertID uuid.UUID) (*alertInfo, error) {
	info := &alertInfo{id: alertID}
	err := a.cyberDB.QueryRow(ctx, `
		SELECT title, severity::text, source, asset_ids,
		       mitre_technique_id, mitre_tactic_id, created_at
		FROM alerts
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`, tenantID, alertID).Scan(
		&info.title, &info.severity, &info.source, &info.assetIDs,
		&info.mitreTechniqueID, &info.mitreTacticID, &info.createdAt,
	)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func calculateConfidence(chainLen, timelineLen int) float64 {
	if chainLen == 0 {
		return 0.1
	}
	conf := 0.3
	if chainLen >= 2 {
		conf = 0.5
	}
	if chainLen >= 3 {
		conf = 0.65
	}
	if chainLen >= 5 {
		conf = 0.8
	}
	if timelineLen > chainLen*2 {
		conf += 0.1
	}
	if conf > 0.95 {
		conf = 0.95
	}
	return conf
}

func buildSecuritySummary(alert *alertInfo, rootCauseType string, chain []CausalStep) string {
	switch rootCauseType {
	case "exposed_service":
		return fmt.Sprintf("Root cause: Initial compromise via exposed service. Alert '%s' traced back to external access through %d correlated events.", alert.title, len(chain))
	case "credential_compromise":
		return fmt.Sprintf("Root cause: Compromised credentials. Alert '%s' originated from credential access activity across %d events.", alert.title, len(chain))
	case "insider_threat":
		return fmt.Sprintf("Root cause: Possible insider threat. Alert '%s' linked to authorized user activity across %d events.", alert.title, len(chain))
	case "lateral_movement":
		return fmt.Sprintf("Root cause: Lateral movement from previously compromised system. Alert '%s' traced through %d events.", alert.title, len(chain))
	default:
		return fmt.Sprintf("Root cause analysis for alert '%s': %d events analyzed, %d causal steps identified.", alert.title, len(chain), len(chain))
	}
}
