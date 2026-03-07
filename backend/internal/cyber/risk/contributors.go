package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type ContributorAnalyzer struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewContributorAnalyzer(db *pgxpool.Pool, logger zerolog.Logger) *ContributorAnalyzer {
	return &ContributorAnalyzer{db: db, logger: logger.With().Str("component", "risk-contributors").Logger()}
}

func (c *ContributorAnalyzer) Analyze(ctx context.Context, tenantID uuid.UUID) ([]model.RiskContributor, error) {
	contributors := make([]model.RiskContributor, 0, 25)

	vulnRows, err := c.db.Query(ctx, `
		SELECT
			v.id, v.title, COALESCE(v.cve_id, ''), v.severity::text, COALESCE(v.cvss_score, 0)::float8,
			a.id, a.name, a.criticality::text,
			COALESCE(EXTRACT(EPOCH FROM (now() - v.discovered_at)) / 86400.0, 0)::float8
		FROM vulnerabilities v
		JOIN assets a ON a.id = v.asset_id
		WHERE v.tenant_id = $1
		  AND v.status IN ('open', 'in_progress')
		  AND v.severity IN ('critical', 'high')
		  AND a.criticality IN ('critical', 'high')
		  AND v.deleted_at IS NULL
		  AND a.deleted_at IS NULL
		ORDER BY severity_order(v.severity) DESC, COALESCE(v.cvss_score, 0) DESC, v.discovered_at ASC
		LIMIT 10`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("top vulnerability contributors: %w", err)
	}
	defer vulnRows.Close()
	for vulnRows.Next() {
		var (
			item      model.RiskContributor
			cveID     string
			cvssScore float64
			assetID   uuid.UUID
			assetName string
			assetCrit string
			ageDays   float64
		)
		if err := vulnRows.Scan(&item.ID, &item.Title, &cveID, &item.Severity, &cvssScore, &assetID, &assetName, &assetCrit, &ageDays); err != nil {
			return nil, err
		}
		item.Type = "vulnerability"
		item.AssetID = &assetID
		item.AssetName = assetName
		item.Score = roundTo2(severityWeight(item.Severity) * assetCriticalityWeight(assetCrit) * (1 + math.Min(ageDays/90, 1)))
		item.Remediation = fmt.Sprintf("Patch %s on %s", firstNonEmpty(cveID, item.Title), assetName)
		item.Link = "/cyber/vulnerabilities/" + item.ID.String()
		contributors = append(contributors, item)
	}
	if err := vulnRows.Err(); err != nil {
		return nil, err
	}

	alertRows, err := c.db.Query(ctx, `
		SELECT id, title, severity::text
		FROM alerts
		WHERE tenant_id = $1
		  AND severity = 'critical'
		  AND status = 'new'
		  AND created_at < now() - interval '4 hours'
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 5`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("top alert contributors: %w", err)
	}
	defer alertRows.Close()
	for alertRows.Next() {
		var item model.RiskContributor
		if err := alertRows.Scan(&item.ID, &item.Title, &item.Severity); err != nil {
			return nil, err
		}
		item.Type = "alert_sla_breach"
		item.Score = 15
		item.Remediation = fmt.Sprintf("Acknowledge and investigate alert %q immediately", item.Title)
		item.Link = "/cyber/alerts/" + item.ID.String()
		contributors = append(contributors, item)
	}
	if err := alertRows.Err(); err != nil {
		return nil, err
	}

	threatRows, err := c.db.Query(ctx, `
		SELECT id, name, severity::text, affected_asset_count
		FROM threats
		WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL
		ORDER BY severity_order(severity) DESC, affected_asset_count DESC, last_seen_at DESC
		LIMIT 5`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("top threat contributors: %w", err)
	}
	defer threatRows.Close()
	for threatRows.Next() {
		var (
			item               model.RiskContributor
			affectedAssetCount int
		)
		if err := threatRows.Scan(&item.ID, &item.Title, &item.Severity, &affectedAssetCount); err != nil {
			return nil, err
		}
		item.Type = "threat"
		item.Score = roundTo2(severityWeight(item.Severity) * 3)
		item.Remediation = fmt.Sprintf("Contain threat %q affecting %d assets", item.Title, affectedAssetCount)
		item.Link = "/cyber/threats/" + item.ID.String()
		contributors = append(contributors, item)
	}
	if err := threatRows.Err(); err != nil {
		return nil, err
	}

	pathRows, err := c.db.Query(ctx, `
		SELECT id, title, severity::text, priority_score, attack_path, attack_path_length
		FROM ctem_findings
		WHERE tenant_id = $1
		  AND type = 'attack_path'
		  AND status = 'open'
		ORDER BY priority_score DESC, created_at DESC
		LIMIT 5`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("top attack path contributors: %w", err)
	}
	defer pathRows.Close()
	for pathRows.Next() {
		var (
			item       model.RiskContributor
			priority   float64
			pathJSON   []byte
			pathLength *int
		)
		if err := pathRows.Scan(&item.ID, &item.Title, &item.Severity, &priority, &pathJSON, &pathLength); err != nil {
			return nil, err
		}
		item.Type = "attack_path"
		item.Score = roundTo2(priority / 5)
		item.Remediation = buildAttackPathRemediation(pathJSON, pathLength)
		item.Link = "/cyber/ctem/findings/" + item.ID.String()
		contributors = append(contributors, item)
	}
	if err := pathRows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(contributors, func(i, j int) bool {
		if contributors[i].Score == contributors[j].Score {
			return severityWeight(contributors[i].Severity) > severityWeight(contributors[j].Severity)
		}
		return contributors[i].Score > contributors[j].Score
	})
	if len(contributors) > 10 {
		contributors = contributors[:10]
	}

	totalScore := 0.0
	for _, item := range contributors {
		totalScore += item.Score
	}
	for i := range contributors {
		if totalScore > 0 {
			contributors[i].Impact = roundTo2((contributors[i].Score / totalScore) * 100)
		}
	}

	return contributors, nil
}

func buildAttackPathRemediation(pathJSON []byte, length *int) string {
	type hop struct {
		AssetName string `json:"asset_name"`
	}
	type path struct {
		Hops []hop `json:"hops"`
	}
	var parsed path
	if len(pathJSON) > 0 && json.Unmarshal(pathJSON, &parsed) == nil && len(parsed.Hops) > 1 {
		return fmt.Sprintf(
			"Remediate attack path from %s to %s (%d hops)",
			parsed.Hops[0].AssetName,
			parsed.Hops[len(parsed.Hops)-1].AssetName,
			maxInt(1, derefInt(length)),
		)
	}
	return fmt.Sprintf("Remediate attack path with %d hops", maxInt(1, derefInt(length)))
}

func severityWeight(severity string) float64 {
	switch severity {
	case "critical":
		return 10
	case "high":
		return 7
	case "medium":
		return 4
	case "low":
		return 1
	default:
		return 0
	}
}

func assetCriticalityWeight(criticality string) float64 {
	switch criticality {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func daysBetween(from, to time.Time) float64 {
	return to.Sub(from).Hours() / 24
}
