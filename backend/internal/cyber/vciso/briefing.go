package vciso

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	cyberdash "github.com/clario360/platform/internal/cyber/dashboard"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
	cyberrisk "github.com/clario360/platform/internal/cyber/risk"
)

// BriefingGenerator assembles structured executive intelligence from the cyber suite.
type BriefingGenerator struct {
	db           *pgxpool.Pool
	riskScorer   *cyberrisk.RiskScorer
	mttr         *cyberdash.MTTRCalculator
	recommender  *RecommendationAggregator
	logger       zerolog.Logger
}

// NewBriefingGenerator creates a briefing generator.
func NewBriefingGenerator(
	db *pgxpool.Pool,
	riskScorer *cyberrisk.RiskScorer,
	mttr *cyberdash.MTTRCalculator,
	recommender *RecommendationAggregator,
	logger zerolog.Logger,
) *BriefingGenerator {
	return &BriefingGenerator{
		db:          db,
		riskScorer:  riskScorer,
		mttr:        mttr,
		recommender: recommender,
		logger:      logger.With().Str("component", "vciso-briefing").Logger(),
	}
}

// GenerateExecutiveBriefing builds an executive-level briefing for the requested period.
func (b *BriefingGenerator) GenerateExecutiveBriefing(ctx context.Context, tenantID uuid.UUID, periodDays int) (*model.ExecutiveBriefing, error) {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -periodDays)
	previousStart := start.AddDate(0, 0, -periodDays)

	score, err := b.riskScorer.CalculateOrganizationRisk(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	recommendations, err := b.recommender.Generate(ctx, tenantID, score)
	if err != nil {
		return nil, err
	}

	riskPosture, err := b.buildRiskPosture(ctx, tenantID, score, start)
	if err != nil {
		return nil, err
	}
	criticalIssues, err := b.buildCriticalIssues(ctx, tenantID, start)
	if err != nil {
		return nil, err
	}
	threatLandscape, err := b.buildThreatLandscape(ctx, tenantID, start, end, previousStart, start)
	if err != nil {
		return nil, err
	}
	remediationSummary, err := b.buildRemediationSummary(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	keyMetrics, err := b.buildKeyMetrics(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	compliance, err := b.buildComplianceSummary(ctx, tenantID, keyMetrics)
	if err != nil {
		return nil, err
	}
	comparison, err := b.buildComparison(ctx, tenantID, start, end, previousStart)
	if err != nil {
		return nil, err
	}

	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	return &model.ExecutiveBriefing{
		GeneratedAt:       end,
		Period:            model.DateRange{Start: start, End: end, Days: periodDays},
		RiskPosture:       *riskPosture,
		CriticalIssues:    criticalIssues,
		ThreatLandscape:   *threatLandscape,
		RemediationStatus: *remediationSummary,
		KeyMetrics:        *keyMetrics,
		Recommendations:   recommendations,
		ComplianceStatus:  *compliance,
		Comparison:        comparison,
	}, nil
}

func (b *BriefingGenerator) buildRiskPosture(ctx context.Context, tenantID uuid.UUID, current *model.OrganizationRiskScore, periodStart time.Time) (*model.RiskPostureSummary, error) {
	summary := &model.RiskPostureSummary{
		CurrentScore:    current.OverallScore,
		Trend:           current.Trend,
		TrendDelta:      current.TrendDelta,
		Grade:           current.Grade,
		Components:      map[string]float64{},
		ComponentTrends: map[string]string{},
	}
	if err := b.db.QueryRow(ctx, `
		SELECT overall_score, grade
		FROM risk_score_history
		WHERE tenant_id = $1 AND calculated_at < $2
		ORDER BY calculated_at DESC
		LIMIT 1`,
		tenantID, periodStart,
	).Scan(&summary.PreviousScore, &summary.GradeChange); err == nil {
		summary.TrendDelta = current.OverallScore - summary.PreviousScore
		summary.Trend = current.Trend
		if summary.GradeChange == current.Grade {
			summary.GradeChange = "unchanged"
		} else {
			summary.GradeChange = fmt.Sprintf("%s -> %s", summary.GradeChange, current.Grade)
		}
	} else {
		summary.GradeChange = "unchanged"
	}

	summary.Components["vulnerability_risk"] = current.Components.VulnerabilityRisk.Score
	summary.Components["threat_exposure"] = current.Components.ThreatExposure.Score
	summary.Components["configuration_risk"] = current.Components.ConfigurationRisk.Score
	summary.Components["attack_surface_risk"] = current.Components.AttackSurfaceRisk.Score
	summary.Components["compliance_gap_risk"] = current.Components.ComplianceGapRisk.Score
	summary.ComponentTrends["vulnerability_risk"] = current.Components.VulnerabilityRisk.Trend
	summary.ComponentTrends["threat_exposure"] = current.Components.ThreatExposure.Trend
	summary.ComponentTrends["configuration_risk"] = current.Components.ConfigurationRisk.Trend
	summary.ComponentTrends["attack_surface_risk"] = current.Components.AttackSurfaceRisk.Trend
	summary.ComponentTrends["compliance_gap_risk"] = current.Components.ComplianceGapRisk.Trend
	return summary, nil
}

func (b *BriefingGenerator) buildCriticalIssues(ctx context.Context, tenantID uuid.UUID, start time.Time) ([]model.CriticalIssue, error) {
	issues := make([]model.CriticalIssue, 0, 8)

	rows, err := b.db.Query(ctx, `
		SELECT id, title, EXTRACT(EPOCH FROM (now() - created_at)) / 86400
		FROM alerts
		WHERE tenant_id = $1
		  AND severity = 'critical'
		  AND status = 'new'
		  AND created_at < now() - interval '4 hours'
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 2`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			id      uuid.UUID
			title   string
			daysOpen float64
		)
		if err := rows.Scan(&id, &title, &daysOpen); err != nil {
			rows.Close()
			return nil, err
		}
		issues = append(issues, model.CriticalIssue{
			Type:        "alert",
			Title:       "Critical alert past SLA",
			Description: title,
			Severity:    "critical",
			Impact:      "Detection response is outside SLA for a critical issue.",
			Action:      "Assign an analyst immediately and escalate to the security manager.",
			LinkID:      id.String(),
			LinkType:    "alert",
			DaysOpen:    int(daysOpen),
		})
	}
	rows.Close()

	threatRows, err := b.db.Query(ctx, `
		SELECT id, name, severity::text, affected_asset_count
		FROM threats
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND deleted_at IS NULL
		ORDER BY severity_order(severity::text) DESC, affected_asset_count DESC
		LIMIT 2`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	for threatRows.Next() {
		var (
			id                 uuid.UUID
			name, severity     string
			affectedAssetCount int
		)
		if err := threatRows.Scan(&id, &name, &severity, &affectedAssetCount); err != nil {
			threatRows.Close()
			return nil, err
		}
		issues = append(issues, model.CriticalIssue{
			Type:        "threat",
			Title:       name,
			Description: fmt.Sprintf("Active threat affecting %d assets.", affectedAssetCount),
			Severity:    severity,
			Impact:      "Uncontained malicious activity remains active in the tenant environment.",
			Action:      "Contain the threat and validate eradication across impacted assets.",
			LinkID:      id.String(),
			LinkType:    "threat",
		})
	}
	threatRows.Close()

	vulnRows, err := b.db.Query(ctx, `
		SELECT v.id, v.title, COALESCE(v.cve_id, ''), EXTRACT(EPOCH FROM (now() - v.discovered_at)) / 86400
		FROM vulnerabilities v
		JOIN assets a ON a.id = v.asset_id AND a.deleted_at IS NULL
		WHERE v.tenant_id = $1
		  AND v.deleted_at IS NULL
		  AND v.status IN ('open', 'in_progress')
		  AND v.severity = 'critical'
		  AND a.criticality = 'critical'
		  AND v.discovered_at <= now() - interval '7 days'
		ORDER BY v.discovered_at ASC
		LIMIT 2`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	for vulnRows.Next() {
		var (
			id      uuid.UUID
			title   string
			cveID   string
			daysOpen float64
		)
		if err := vulnRows.Scan(&id, &title, &cveID, &daysOpen); err != nil {
			vulnRows.Close()
			return nil, err
		}
		issues = append(issues, model.CriticalIssue{
			Type:        "vulnerability",
			Title:       firstNonEmpty(cveID, title),
			Description: title,
			Severity:    "critical",
			Impact:      "Critical vulnerability remains unpatched on a critical asset beyond the expected remediation window.",
			Action:      "Create or accelerate a governed patch remediation immediately.",
			LinkID:      id.String(),
			LinkType:    "vulnerability",
			DaysOpen:    int(daysOpen),
		})
	}
	vulnRows.Close()

	findingRows, err := b.db.Query(ctx, `
		SELECT id, title, priority_score
		FROM ctem_findings
		WHERE tenant_id = $1
		  AND priority_group = 1
		  AND status = 'open'
		ORDER BY priority_score DESC
		LIMIT 2`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	for findingRows.Next() {
		var (
			id            uuid.UUID
			title         string
			priorityScore float64
		)
		if err := findingRows.Scan(&id, &title, &priorityScore); err != nil {
			findingRows.Close()
			return nil, err
		}
		issues = append(issues, model.CriticalIssue{
			Type:        "ctem_finding",
			Title:       title,
			Description: fmt.Sprintf("Immediate CTEM finding with priority score %.1f.", priorityScore),
			Severity:    "high",
			Impact:      "Validated exposure exists on a critical attack path or exploitable control weakness.",
			Action:      "Convert the finding into a governed remediation and track completion.",
			LinkID:      id.String(),
			LinkType:    "ctem_finding",
		})
	}
	findingRows.Close()

	sort.SliceStable(issues, func(i, j int) bool {
		severityRank := func(severity string) int {
			switch severity {
			case "critical":
				return 4
			case "high":
				return 3
			case "medium":
				return 2
			default:
				return 1
			}
		}
		if severityRank(issues[i].Severity) == severityRank(issues[j].Severity) {
			return issues[i].DaysOpen > issues[j].DaysOpen
		}
		return severityRank(issues[i].Severity) > severityRank(issues[j].Severity)
	})
	if len(issues) > 5 {
		issues = issues[:5]
	}
	for idx := range issues {
		issues[idx].Rank = idx + 1
	}
	return issues, nil
}

func (b *BriefingGenerator) buildThreatLandscape(ctx context.Context, tenantID uuid.UUID, start, end, previousStart, previousEnd time.Time) (*model.ThreatLandscapeSummary, error) {
	summary := &model.ThreatLandscapeSummary{TopMITRETactics: []string{}, TopThreatTypes: []model.ThreatTypeSummary{}}
	var previousAlerts int
	if err := b.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3 AND deleted_at IS NULL`, tenantID, start, end).Scan(&summary.AlertVolume); err != nil {
		return nil, err
	}
	if err := b.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3 AND deleted_at IS NULL`, tenantID, previousStart, previousEnd).Scan(&previousAlerts); err != nil {
		return nil, err
	}
	if previousAlerts > 0 {
		summary.AlertVolumeChange = ((float64(summary.AlertVolume-previousAlerts) / float64(previousAlerts)) * 100)
	}
	if err := b.db.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COALESCE(100.0 * AVG(CASE WHEN contained_at IS NOT NULL THEN 1 ELSE 0 END), 0)::float8
		FROM threats
		WHERE tenant_id = $1
		  AND detected_at >= $2
		  AND detected_at < $3
		  AND deleted_at IS NULL`,
		tenantID, start, end,
	).Scan(&summary.NewThreats, &summary.ContainmentRate); err != nil {
		return nil, err
	}

	tacticRows, err := b.db.Query(ctx, `
		SELECT COALESCE(mitre_tactic_name, mitre_tactic_id), COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1
		  AND created_at >= $2
		  AND created_at < $3
		  AND deleted_at IS NULL
		  AND (mitre_tactic_name IS NOT NULL OR mitre_tactic_id IS NOT NULL)
		GROUP BY COALESCE(mitre_tactic_name, mitre_tactic_id)
		ORDER BY COUNT(*) DESC
		LIMIT 3`,
		tenantID, start, end,
	)
	if err != nil {
		return nil, err
	}
	for tacticRows.Next() {
		var tactic string
		var count int
		if err := tacticRows.Scan(&tactic, &count); err != nil {
			tacticRows.Close()
			return nil, err
		}
		summary.TopMITRETactics = append(summary.TopMITRETactics, tactic)
	}
	tacticRows.Close()

	typeRows, err := b.db.Query(ctx, `
		SELECT type, COUNT(*)::int
		FROM threats
		WHERE tenant_id = $1
		  AND detected_at >= $2
		  AND detected_at < $3
		  AND deleted_at IS NULL
		GROUP BY type
		ORDER BY COUNT(*) DESC
		LIMIT 3`,
		tenantID, start, end,
	)
	if err != nil {
		return nil, err
	}
	for typeRows.Next() {
		var entry model.ThreatTypeSummary
		if err := typeRows.Scan(&entry.Type, &entry.Count); err != nil {
			typeRows.Close()
			return nil, err
		}
		summary.TopThreatTypes = append(summary.TopThreatTypes, entry)
	}
	if err := typeRows.Err(); err != nil {
		typeRows.Close()
		return nil, err
	}
	typeRows.Close()
	return summary, nil
}

func (b *BriefingGenerator) buildRemediationSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*model.RemediationSummary, error) {
	summary := &model.RemediationSummary{}
	if err := b.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'closed' AND updated_at >= $2 AND updated_at < $3)::int,
			COUNT(*) FILTER (WHERE status = 'pending_approval')::int,
			COUNT(*) FILTER (WHERE status IN ('approved','dry_run_running','dry_run_completed','execution_pending','executing','verification_pending','rollback_pending','rolling_back'))::int,
			COUNT(*) FILTER (WHERE status IN ('dry_run_failed','execution_failed','verification_failed','rollback_failed') AND updated_at >= $2 AND updated_at < $3)::int,
			COUNT(*) FILTER (WHERE status = 'rolled_back' AND updated_at >= $2 AND updated_at < $3)::int,
			COALESCE(100.0 * AVG(CASE WHEN status IN ('verified','closed') THEN 1 ELSE 0 END)
				FILTER (WHERE status IN ('verified','closed','verification_failed') AND updated_at >= $2 AND updated_at < $3), 0)::float8,
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(execution_started_at, execution_completed_at) - created_at)) / 3600)
				FILTER (WHERE execution_started_at IS NOT NULL AND created_at >= $2 AND created_at < $3), 0)::float8
		FROM remediation_actions
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID, start, end,
	).Scan(
		&summary.CompletedInPeriod,
		&summary.PendingApproval,
		&summary.InProgress,
		&summary.FailedInPeriod,
		&summary.RollbackCount,
		&summary.VerificationSuccessRate,
		&summary.AvgTimeToExecuteHours,
	); err != nil {
		return nil, err
	}
	return summary, nil
}

func (b *BriefingGenerator) buildKeyMetrics(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*model.BriefingMetrics, error) {
	metrics := &model.BriefingMetrics{AlertsBySeverity: map[string]int{}}
	mttr, err := b.mttr.Calculate(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	metrics.MTTR = mttr.Overall.AvgResponseHours
	metrics.SLAComplianceRate = mttr.Overall.SLACompliance

	if err := b.db.QueryRow(ctx, `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (created_at - first_event_at)) / 3600), 0)::float8,
			COUNT(*)::int,
			COALESCE(100.0 * AVG(CASE WHEN status = 'false_positive' THEN 1 ELSE 0 END), 0)::float8
		FROM alerts
		WHERE tenant_id = $1
		  AND created_at >= $2
		  AND created_at < $3
		  AND deleted_at IS NULL`,
		tenantID, start, end,
	).Scan(&metrics.MTTD, &metrics.AlertVolumeTotal, &metrics.FalsePositiveRate); err != nil {
		return nil, err
	}
	if err := b.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (contained_at - first_seen_at)) / 3600), 0)::float8
		FROM threats
		WHERE tenant_id = $1
		  AND contained_at IS NOT NULL
		  AND contained_at >= $2
		  AND contained_at < $3
		  AND deleted_at IS NULL`,
		tenantID, start, end,
	).Scan(&metrics.MTTC); err != nil {
		return nil, err
	}

	rows, err := b.db.Query(ctx, `
		SELECT severity::text, COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3 AND deleted_at IS NULL
		GROUP BY severity`,
		tenantID, start, end,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			rows.Close()
			return nil, err
		}
		metrics.AlertsBySeverity[severity] = count
	}
	rows.Close()

	if err := b.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(score), 0)::float8
		FROM exposure_score_snapshots
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&metrics.CTEMExposureScore); err != nil {
		return nil, err
	}
	if err := b.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(posture_score), 0)::float8
		FROM dspm_data_assets
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&metrics.DSPMComplianceScore); err != nil {
		return nil, err
	}
	return metrics, nil
}

func (b *BriefingGenerator) buildComplianceSummary(ctx context.Context, tenantID uuid.UUID, keyMetrics *model.BriefingMetrics) (*model.VCISOComplianceSummary, error) {
	summary := &model.VCISOComplianceSummary{
		DSPMPostureScore:  keyMetrics.DSPMComplianceScore,
		SLAComplianceRate: keyMetrics.SLAComplianceRate,
	}

	var coveredTechniques int
	if err := b.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT technique_id)::int
		FROM detection_rules dr
		CROSS JOIN LATERAL unnest(dr.mitre_technique_ids) AS technique_id
		WHERE tenant_id = $1
		  AND enabled = true
		  AND deleted_at IS NULL`,
		tenantID,
	).Scan(&coveredTechniques); err != nil {
		return nil, err
	}
	totalTechniques := len(mitre.AllTechniques())
	if totalTechniques > 0 {
		summary.MITRECoveragePercent = (float64(coveredTechniques) / float64(totalTechniques)) * 100
	}
	if err := b.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM remediation_actions
		WHERE tenant_id = $1
		  AND status IN ('execution_failed', 'verification_failed', 'rollback_failed')
		  AND deleted_at IS NULL`,
		tenantID,
	).Scan(&summary.OpenAuditFindings); err != nil {
		return nil, err
	}
	return summary, nil
}

func (b *BriefingGenerator) buildComparison(ctx context.Context, tenantID uuid.UUID, start, end, previousStart time.Time) (*model.PeriodComparison, error) {
	comparison := &model.PeriodComparison{}
	if err := b.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(overall_score), 0)::float8
		FROM risk_score_history
		WHERE tenant_id = $1
		  AND calculated_at >= $2
		  AND calculated_at < $3`,
		tenantID, start, end,
	).Scan(&comparison.RiskScoreDelta); err != nil {
		return nil, err
	}
	var previousRisk float64
	if err := b.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(overall_score), 0)::float8
		FROM risk_score_history
		WHERE tenant_id = $1
		  AND calculated_at >= $2
		  AND calculated_at < $3`,
		tenantID, previousStart, start,
	).Scan(&previousRisk); err != nil {
		return nil, err
	}
	comparison.RiskScoreDelta = comparison.RiskScoreDelta - previousRisk

	var currentAlerts, previousAlerts int
	if err := b.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3 AND deleted_at IS NULL`, tenantID, start, end).Scan(&currentAlerts); err != nil {
		return nil, err
	}
	if err := b.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3 AND deleted_at IS NULL`, tenantID, previousStart, start).Scan(&previousAlerts); err != nil {
		return nil, err
	}
	comparison.AlertVolumeDelta = currentAlerts - previousAlerts
	if previousAlerts > 0 {
		comparison.AlertVolumeChangePct = (float64(comparison.AlertVolumeDelta) / float64(previousAlerts)) * 100
	}
	if err := b.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE discovered_at >= $2 AND discovered_at < $3)::int,
			COUNT(*) FILTER (WHERE resolved_at >= $2 AND resolved_at < $3)::int
		FROM vulnerabilities
		WHERE tenant_id = $1
		  AND deleted_at IS NULL`,
		tenantID, start, end,
	).Scan(&comparison.NewVulnerabilities, &comparison.ResolvedVulnerabilities); err != nil {
		return nil, err
	}
	if err := b.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM remediation_actions
		WHERE tenant_id = $1
		  AND status = 'closed'
		  AND updated_at >= $2
		  AND updated_at < $3
		  AND deleted_at IS NULL`,
		tenantID, start, end,
	).Scan(&comparison.RemediationsCompleted); err != nil {
		return nil, err
	}
	return comparison, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
