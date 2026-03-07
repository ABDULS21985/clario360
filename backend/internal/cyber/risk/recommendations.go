package risk

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
)

var recommendationCriticalTechniques = []string{
	"T1059", "T1078", "T1110", "T1190", "T1021", "T1486", "T1071", "T1055", "T1003", "T1047",
	"T1053", "T1036", "T1070", "T1082", "T1105", "T1027", "T1569", "T1543", "T1140", "T1068",
}

type RecommendationEngine struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewRecommendationEngine(db *pgxpool.Pool, logger zerolog.Logger) *RecommendationEngine {
	return &RecommendationEngine{db: db, logger: logger.With().Str("component", "risk-recommendations").Logger()}
}

func (r *RecommendationEngine) Generate(ctx context.Context, tenantID uuid.UUID, score *model.OrganizationRiskScore) ([]model.RiskRecommendation, error) {
	recommendations := make([]model.RiskRecommendation, 0, 10)

	if score.Components.VulnerabilityRisk.Score > 60 {
		criticalOnCritical, err := r.countCriticalOnCriticalVulns(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		impact := estimateReduction(score.Components.VulnerabilityRisk.Weighted, float64(criticalOnCritical), float64(maxInt(1, score.Context.TotalOpenVulns)))
		recommendations = append(recommendations, model.RiskRecommendation{
			Title:           "Patch critical vulnerabilities on critical assets",
			Description:     fmt.Sprintf("%d critical vulnerabilities remain on critical assets and are the largest driver of organizational risk.", criticalOnCritical),
			Component:       "vulnerability_risk",
			EstimatedImpact: impact,
			Effort:          effortFromCount(criticalOnCritical),
			Category:        "patch",
			Actions: []string{
				fmt.Sprintf("Prioritize remediation windows for %d critical vulnerabilities on business-critical systems", criticalOnCritical),
				"Deploy validated patches to production servers first, then high-criticality support systems",
			},
		})
	}

	if score.Components.ThreatExposure.Score > 50 {
		slaBreaches := intFromDetails(score.Components.ThreatExposure.Details, "critical_sla_breached") + intFromDetails(score.Components.ThreatExposure.Details, "high_sla_breached")
		if slaBreaches > 0 {
			recommendations = append(recommendations, model.RiskRecommendation{
				Title:           "Resolve SLA-breached critical alerts",
				Description:     fmt.Sprintf("%d unresolved alerts are already outside response SLA and are inflating the threat exposure score.", slaBreaches),
				Component:       "threat_exposure",
				EstimatedImpact: estimateReduction(score.Components.ThreatExposure.Weighted, float64(slaBreaches), float64(maxInt(1, score.Context.TotalOpenAlerts))),
				Effort:          effortFromCount(slaBreaches),
				Category:        "investigate",
				Actions: []string{
					fmt.Sprintf("Assign analysts to %d SLA-breached alerts immediately", slaBreaches),
					"Escalate any unowned critical alert to the security manager queue",
				},
			})
		}
	}

	if score.Components.AttackSurfaceRisk.Score > 50 {
		exposedAssets := intFromDetails(score.Components.AttackSurfaceRisk.Details, "internet_facing_vulnerable")
		internetFacing := intFromDetails(score.Components.AttackSurfaceRisk.Details, "internet_facing")
		recommendations = append(recommendations, model.RiskRecommendation{
			Title:           "Reduce internet-facing attack surface",
			Description:     fmt.Sprintf("%d internet-facing assets are carrying critical or high vulnerabilities across %d exposed systems.", exposedAssets, internetFacing),
			Component:       "attack_surface_risk",
			EstimatedImpact: estimateReduction(score.Components.AttackSurfaceRisk.Weighted, float64(exposedAssets), float64(maxInt(1, internetFacing))),
			Effort:          effortFromCount(exposedAssets),
			Category:        "configure",
			Actions: []string{
				fmt.Sprintf("Review %d internet-facing assets with open high-severity vulnerabilities", exposedAssets),
				"Close unnecessary remote administration ports on DMZ systems",
				"Segment exposed databases and restrict east-west reachability",
			},
		})
	}

	if score.Components.ComplianceGapRisk.Score > 50 {
		missingCritical := stringSliceFromDetails(score.Components.ComplianceGapRisk.Details, "missing_critical_techniques")
		techniqueNames := make([]string, 0, minInt(3, len(missingCritical)))
		for _, techniqueID := range missingCritical {
			if technique, ok := mitre.TechniqueByID(techniqueID); ok {
				techniqueNames = append(techniqueNames, technique.Name)
			}
			if len(techniqueNames) == 3 {
				break
			}
		}
		recommendations = append(recommendations, model.RiskRecommendation{
			Title:           "Improve MITRE ATT&CK detection coverage",
			Description:     fmt.Sprintf("%d critical ATT&CK techniques lack enabled detections and are increasing compliance and monitoring risk.", len(missingCritical)),
			Component:       "compliance_gap_risk",
			EstimatedImpact: estimateReduction(score.Components.ComplianceGapRisk.Weighted, float64(len(missingCritical)), float64(len(recommendationCriticalTechniques))),
			Effort:          effortFromCount(len(missingCritical)),
			Category:        "detect",
			Actions: []string{
				fmt.Sprintf("Deploy coverage for high-value techniques such as %v", techniqueNames),
				"Enable pre-built detection rule templates for Initial Access and Credential Access tactics",
			},
		})
	}

	if score.Components.ConfigurationRisk.Score > 40 {
		totalMisconfigs := intFromDetails(score.Components.ConfigurationRisk.Details, "total_misconfigs")
		recommendations = append(recommendations, model.RiskRecommendation{
			Title:           "Remediate critical misconfigurations",
			Description:     fmt.Sprintf("%d open misconfigurations from CTEM remain unresolved and are a direct source of preventable exposure.", totalMisconfigs),
			Component:       "configuration_risk",
			EstimatedImpact: estimateReduction(score.Components.ConfigurationRisk.Weighted, float64(totalMisconfigs), float64(maxInt(1, totalMisconfigs))),
			Effort:          effortFromCount(totalMisconfigs),
			Category:        "configure",
			Actions: []string{
				"Close exposed management ports on internet-facing systems",
				"Rotate expired certificates and remove insecure protocol support",
				"Enforce strong authentication controls on database and admin assets",
			},
		})
	}

	lastAssessment, err := r.lastCompletedAssessment(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if lastAssessment == nil || lastAssessment.Before(time.Now().UTC().AddDate(0, 0, -30)) {
		recommendations = append(recommendations, model.RiskRecommendation{
			Title:           "Run a CTEM assessment",
			Description:     "CTEM findings are stale or missing, reducing confidence in configuration and attack-path exposure data.",
			Component:       "configuration_risk",
			EstimatedImpact: 3,
			Effort:          "medium",
			Category:        "process",
			Actions: []string{
				"Execute a tenant-wide CTEM assessment covering production assets",
				"Review and approve priority group 1 and 2 findings with security leadership",
			},
		})
	}

	fpRules, err := r.falsePositiveHeavyRules(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(fpRules) > 0 {
		recommendations = append(recommendations, model.RiskRecommendation{
			Title:           "Review detection rule performance",
			Description:     fmt.Sprintf("%d enabled detection rules exceed a 30%% false-positive rate and are adding analyst load without proportional value.", len(fpRules)),
			Component:       "threat_exposure",
			EstimatedImpact: 2,
			Effort:          "medium",
			Category:        "detect",
			Actions: []string{
				fmt.Sprintf("Tune or disable noisy rules such as %s", fpRules[0]),
				"Add suppression logic or threshold adjustments for repeat false positives",
			},
		})
	}

	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].EstimatedImpact == recommendations[j].EstimatedImpact {
			return recommendations[i].Title < recommendations[j].Title
		}
		return recommendations[i].EstimatedImpact > recommendations[j].EstimatedImpact
	})
	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}
	for i := range recommendations {
		recommendations[i].Priority = i + 1
		recommendations[i].EstimatedImpact = roundTo2(recommendations[i].EstimatedImpact)
	}
	return recommendations, nil
}

func (r *RecommendationEngine) countCriticalOnCriticalVulns(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM vulnerabilities v
		JOIN assets a ON a.id = v.asset_id AND a.deleted_at IS NULL
		WHERE v.tenant_id = $1
		  AND v.status IN ('open', 'in_progress')
		  AND v.severity = 'critical'
		  AND a.criticality = 'critical'
		  AND v.deleted_at IS NULL`,
		tenantID,
	).Scan(&count)
	return count, err
}

func (r *RecommendationEngine) lastCompletedAssessment(ctx context.Context, tenantID uuid.UUID) (*time.Time, error) {
	var completedAt *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT MAX(completed_at)
		FROM ctem_assessments
		WHERE tenant_id = $1 AND status = 'completed' AND deleted_at IS NULL`,
		tenantID,
	).Scan(&completedAt)
	return completedAt, err
}

func (r *RecommendationEngine) falsePositiveHeavyRules(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT name
		FROM detection_rules
		WHERE tenant_id = $1
		  AND enabled = true
		  AND deleted_at IS NULL
		  AND (false_positive_count + true_positive_count) > 0
		  AND (false_positive_count::float / (false_positive_count + true_positive_count)) > 0.30
		ORDER BY (false_positive_count::float / NULLIF(false_positive_count + true_positive_count, 0)) DESC,
		         false_positive_count DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func estimateReduction(weightedScore, portion, total float64) float64 {
	if total <= 0 || weightedScore <= 0 {
		return 0
	}
	fraction := portion / total
	if fraction > 1 {
		fraction = 1
	}
	return weightedScore * fraction
}

func effortFromCount(count int) string {
	switch {
	case count <= 3:
		return "low"
	case count <= 15:
		return "medium"
	default:
		return "high"
	}
}

func intFromDetails(details map[string]interface{}, key string) int {
	if details == nil {
		return 0
	}
	switch value := details[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringSliceFromDetails(details map[string]interface{}, key string) []string {
	if details == nil {
		return nil
	}
	raw, ok := details[key]
	if !ok {
		return nil
	}
	switch value := raw.(type) {
	case []string:
		return value
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
