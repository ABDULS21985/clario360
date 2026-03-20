package vciso

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	cyberrisk "github.com/clario360/platform/internal/cyber/risk"
)

// RecommendationAggregator enriches Prompt 19 risk recommendations with remediation and DSPM context.
type RecommendationAggregator struct {
	db     *pgxpool.Pool
	base   *cyberrisk.RecommendationEngine
	logger zerolog.Logger
}

// NewRecommendationAggregator creates a VCISO recommendation aggregator.
func NewRecommendationAggregator(db *pgxpool.Pool, base *cyberrisk.RecommendationEngine, logger zerolog.Logger) *RecommendationAggregator {
	return &RecommendationAggregator{
		db:     db,
		base:   base,
		logger: logger.With().Str("component", "vciso-recommendations").Logger(),
	}
}

// Generate returns executive-ready recommendations after deduplicating work already underway.
func (a *RecommendationAggregator) Generate(ctx context.Context, tenantID uuid.UUID, score *model.OrganizationRiskScore) ([]model.RiskRecommendation, error) {
	recommendations, err := a.base.Generate(ctx, tenantID, score)
	if err != nil {
		return nil, err
	}

	activeRemediations, err := a.loadActiveRemediations(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	filtered := make([]model.RiskRecommendation, 0, len(recommendations)+2)
	for _, recommendation := range recommendations {
		if recommendationCoveredByActiveRemediation(recommendation, activeRemediations) {
			continue
		}
		filtered = append(filtered, recommendation)
	}

	var avgDSPMPosture float64
	var highRiskDataAssets int
	if err := a.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(posture_score), 0)::float8,
		       COUNT(*) FILTER (WHERE risk_score >= 70)::int
		FROM dspm_data_assets
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&avgDSPMPosture, &highRiskDataAssets); err == nil && avgDSPMPosture < 70 {
		filtered = append(filtered, model.RiskRecommendation{
			Title:           "Strengthen data security posture controls",
			Description:     fmt.Sprintf("%d high-risk data assets are dragging the average DSPM posture score down to %.1f.", highRiskDataAssets, avgDSPMPosture),
			Component:       "dspm_posture",
			EstimatedImpact: impactFromDSPM(avgDSPMPosture, highRiskDataAssets),
			Effort:          recommendationEffortFromCount(highRiskDataAssets),
			Category:        "configure",
			Actions: []string{
				"Enable encryption, backup, and audit logging on sensitive data stores",
				"Complete overdue access reviews for restricted and confidential datasets",
			},
		})
	}

	var acceptedCriticalRisks int
	if err := a.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM ctem_findings
		WHERE tenant_id = $1
		  AND status = 'accepted_risk'
		  AND severity IN ('critical', 'high')`,
		tenantID,
	).Scan(&acceptedCriticalRisks); err == nil && acceptedCriticalRisks > 0 {
		filtered = append(filtered, model.RiskRecommendation{
			Title:           "Review accepted high-risk findings",
			Description:     fmt.Sprintf("%d high or critical findings are currently accepted risk and should be revalidated with business ownership.", acceptedCriticalRisks),
			Component:       "governance",
			EstimatedImpact: float64(minInt(acceptedCriticalRisks, 5)),
			Effort:          "medium",
			Category:        "process",
			Actions: []string{
				"Reconfirm business justification and expiry dates for accepted high-risk findings",
				"Convert outdated risk acceptances into governed remediation plans where possible",
			},
		})
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].EstimatedImpact == filtered[j].EstimatedImpact {
			return filtered[i].Title < filtered[j].Title
		}
		return filtered[i].EstimatedImpact > filtered[j].EstimatedImpact
	})
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}
	for idx := range filtered {
		filtered[idx].Priority = idx + 1
	}
	return filtered, nil
}

type activeRemediation struct {
	Type  string
	Title string
}

func (a *RecommendationAggregator) loadActiveRemediations(ctx context.Context, tenantID uuid.UUID) ([]activeRemediation, error) {
	rows, err := a.db.Query(ctx, `
		SELECT type, title
		FROM remediation_actions
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND status IN (
			  'pending_approval', 'approved', 'dry_run_running', 'dry_run_completed',
			  'execution_pending', 'executing', 'executed', 'verification_pending',
			  'verified', 'rollback_pending', 'rolling_back'
		  )`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active remediations: %w", err)
	}
	defer rows.Close()

	out := make([]activeRemediation, 0)
	for rows.Next() {
		var item activeRemediation
		if err := rows.Scan(&item.Type, &item.Title); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func recommendationCoveredByActiveRemediation(recommendation model.RiskRecommendation, remediations []activeRemediation) bool {
	title := strings.ToLower(recommendation.Title)
	for _, remediation := range remediations {
		remediationType := strings.ToLower(remediation.Type)
		remediationTitle := strings.ToLower(remediation.Title)
		switch recommendation.Category {
		case "patch":
			if remediationType == "patch" || strings.Contains(remediationTitle, "patch") {
				return true
			}
		case "configure":
			if remediationType == "config_change" || remediationType == "firewall_rule" || remediationType == "block_ip" || remediationType == "isolate_asset" {
				if strings.Contains(title, "surface") || strings.Contains(title, "misconfiguration") || strings.Contains(remediationTitle, "config") {
					return true
				}
			}
		case "investigate":
			if strings.Contains(remediationTitle, "alert") || remediationType == "custom" {
				return true
			}
		case "detect":
			if strings.Contains(remediationTitle, "rule") || strings.Contains(remediationTitle, "detection") {
				return true
			}
		}
	}
	return false
}

func impactFromDSPM(avgPosture float64, highRiskAssets int) float64 {
	if highRiskAssets == 0 {
		return 0
	}
	return float64(minInt(highRiskAssets, 10)) * ((100 - avgPosture) / 100)
}

func recommendationEffortFromCount(count int) string {
	switch {
	case count <= 4:
		return "low"
	case count <= 15:
		return "medium"
	default:
		return "high"
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
