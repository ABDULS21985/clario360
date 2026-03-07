package ctem

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

type ScoreFactor struct {
	Factor      string  `json:"factor"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

type ScoringEngine struct {
	db           *pgxpool.Pool
	snapshotRepo *repository.CTEMSnapshotRepository
	logger       zerolog.Logger
}

func NewScoringEngine(db *pgxpool.Pool, snapshotRepo *repository.CTEMSnapshotRepository, logger zerolog.Logger) *ScoringEngine {
	return &ScoringEngine{db: db, snapshotRepo: snapshotRepo, logger: logger}
}

func (s *ScoringEngine) CalculateExposureScore(ctx context.Context, tenantID uuid.UUID) (*model.ExposureScore, error) {
	totalAssets, vulnScore, err := s.calculateVulnerabilityScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	surfaceScore, err := s.calculateAttackSurfaceScore(ctx, tenantID, totalAssets)
	if err != nil {
		return nil, err
	}
	threatScore, err := s.calculateThreatExposureScore(ctx, tenantID, totalAssets)
	if err != nil {
		return nil, err
	}
	velocityScore, err := s.calculateVelocityScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	finalScore := round2((vulnScore * 0.35) + (surfaceScore * 0.25) + (threatScore * 0.25) + (velocityScore * 0.15))
	breakdown := model.ScoreBreakdown{
		VulnerabilityScore:  vulnScore,
		AttackSurfaceScore:  surfaceScore,
		ThreatExposureScore: threatScore,
		RemediationVelocity: velocityScore,
	}

	var trend string
	var delta float64
	last, err := s.snapshotRepo.Last(ctx, tenantID)
	if err != nil && err != repository.ErrNotFound {
		return nil, err
	}
	if last != nil {
		delta = round2(finalScore - last.Score)
		trend = trendForDelta(delta)
	} else {
		trend = "stable"
	}

	return &model.ExposureScore{
		Score:        finalScore,
		Grade:        gradeForScore(finalScore),
		Breakdown:    breakdown,
		Trend:        trend,
		TrendDelta:   delta,
		CalculatedAt: time.Now().UTC(),
	}, nil
}

func (s *ScoringEngine) calculateVulnerabilityScore(ctx context.Context, tenantID uuid.UUID) (int, float64, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
			COALESCE(COUNT(DISTINCT a.id), 0) AS total_assets,
			COALESCE(SUM(
				CASE v.severity
					WHEN 'critical' THEN 10
					WHEN 'high' THEN 7
					WHEN 'medium' THEN 4
					WHEN 'low' THEN 1
					ELSE 0
				END *
				CASE a.criticality
					WHEN 'critical' THEN 4
					WHEN 'high' THEN 3
					WHEN 'medium' THEN 2
					WHEN 'low' THEN 1
					ELSE 1
				END
			), 0) AS weighted_sum
		FROM assets a
		LEFT JOIN vulnerabilities v
		       ON v.asset_id = a.id
		      AND v.tenant_id = a.tenant_id
		      AND v.status IN ('open','in_progress')
		      AND v.deleted_at IS NULL
		WHERE a.tenant_id = $1
		  AND a.deleted_at IS NULL
		  AND a.status = 'active'`,
		tenantID,
	)

	var totalAssets int
	var weightedSum float64
	if err := row.Scan(&totalAssets, &weightedSum); err != nil {
		return 0, 0, fmt.Errorf("calculate vulnerability score: %w", err)
	}
	if totalAssets == 0 {
		return 0, 0, nil
	}
	maxPossible := float64(totalAssets * 10 * 4)
	if maxPossible == 0 {
		return totalAssets, 0, nil
	}
	return totalAssets, math.Min((weightedSum/maxPossible)*100, 100), nil
}

func (s *ScoringEngine) calculateAttackSurfaceScore(ctx context.Context, tenantID uuid.UUID, totalAssets int) (float64, error) {
	if totalAssets == 0 {
		return 0, nil
	}
	row := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (
				WHERE a.tags && ARRAY['internet-facing','dmz','public']
			) AS internet_facing,
			COUNT(*) FILTER (
				WHERE a.tags && ARRAY['internet-facing','dmz','public']
				  AND EXISTS (
					SELECT 1
					FROM jsonb_array_elements_text(COALESCE(a.metadata->'open_ports', '[]'::jsonb)) AS port
					WHERE port::int = ANY(ARRAY[22,3389,445,3306,5432,1433])
				)
			) AS high_risk_ports
		FROM assets a
		WHERE a.tenant_id = $1
		  AND a.deleted_at IS NULL
		  AND a.status = 'active'`,
		tenantID,
	)
	var internetFacing, highRiskPorts int
	if err := row.Scan(&internetFacing, &highRiskPorts); err != nil {
		return 0, fmt.Errorf("calculate surface score: %w", err)
	}
	internetRatio := float64(internetFacing) / float64(totalAssets)
	highRiskRatio := float64(highRiskPorts) / float64(totalAssets)
	return math.Min((internetRatio*60)+(highRiskRatio*40), 100), nil
}

func (s *ScoringEngine) calculateThreatExposureScore(ctx context.Context, tenantID uuid.UUID, totalAssets int) (float64, error) {
	if totalAssets == 0 {
		return 0, nil
	}
	row := s.db.QueryRow(ctx, `
		WITH alert_assets AS (
			SELECT DISTINCT COALESCE(asset_id, affected_asset) AS asset_ref
			FROM (
				SELECT a.asset_id,
				       unnest(CASE WHEN cardinality(a.affected_assets) = 0 THEN ARRAY[a.asset_id]::uuid[] ELSE a.affected_assets END) AS affected_asset
				FROM alerts a
				WHERE a.tenant_id = $1
				  AND a.deleted_at IS NULL
				  AND a.status IN ('new','acknowledged','investigating')
			) expanded
			WHERE COALESCE(asset_id, affected_asset) IS NOT NULL
		)
		SELECT
			(SELECT COUNT(*) FROM alert_assets) AS assets_with_alerts,
			COUNT(*) FILTER (WHERE severity = 'critical' AND status IN ('new','acknowledged')) AS active_critical_alerts
		FROM alerts
		WHERE tenant_id = $1
		  AND deleted_at IS NULL`,
		tenantID,
	)
	var assetsWithAlerts, activeCriticalAlerts int
	if err := row.Scan(&assetsWithAlerts, &activeCriticalAlerts); err != nil {
		return 0, fmt.Errorf("calculate threat score: %w", err)
	}
	threatRatio := float64(assetsWithAlerts) / float64(totalAssets)
	criticalWeight := math.Min(float64(activeCriticalAlerts*5), 50)
	return math.Min((threatRatio*50)+criticalWeight, 100), nil
}

func (s *ScoringEngine) calculateVelocityScore(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	since := time.Now().UTC().AddDate(0, 0, -90)
	row := s.db.QueryRow(ctx, `
		SELECT
			AVG(EXTRACT(EPOCH FROM (status_changed_at - created_at)) / 86400.0)
				FILTER (WHERE severity = 'critical' AND status = 'remediated' AND status_changed_at >= $2) AS critical_avg_days,
			AVG(EXTRACT(EPOCH FROM (status_changed_at - created_at)) / 86400.0)
				FILTER (WHERE severity = 'high' AND status = 'remediated' AND status_changed_at >= $2) AS high_avg_days
		FROM ctem_findings
		WHERE tenant_id = $1`,
		tenantID, since,
	)
	var criticalAvg *float64
	var highAvg *float64
	if err := row.Scan(&criticalAvg, &highAvg); err != nil {
		return 0, fmt.Errorf("calculate remediation velocity: %w", err)
	}
	criticalVelocity := velocityBucket(criticalAvg)
	highVelocity := velocityBucket(highAvg)
	return round2((criticalVelocity * 0.6) + (highVelocity * 0.4)), nil
}

func CalculateBusinessImpact(asset *model.Asset, incomingDependsOn int) (float64, []ScoreFactor) {
	score := 0.0
	factors := make([]ScoreFactor, 0, 4)

	base := 10.0
	switch asset.Criticality {
	case model.CriticalityCritical:
		base = 40
	case model.CriticalityHigh:
		base = 30
	case model.CriticalityMedium:
		base = 20
	case model.CriticalityLow:
		base = 10
	}
	score += base
	factors = append(factors, ScoreFactor{
		Factor:      "asset_criticality",
		Value:       base,
		Description: fmt.Sprintf("Asset criticality is %s", asset.Criticality),
	})

	sensitivity := 0.0
	switch {
	case containsAny(asset.Tags, "restricted", "pci", "phi", "pii"):
		sensitivity = 30
	case containsAny(asset.Tags, "confidential"):
		sensitivity = 20
	case containsAny(asset.Tags, "internal"):
		sensitivity = 10
	}
	if sensitivity > 0 {
		score += sensitivity
		factors = append(factors, ScoreFactor{
			Factor:      "data_sensitivity",
			Value:       sensitivity,
			Description: "Sensitive data classification increases business impact",
		})
	}

	blastRadius := math.Min(float64(incomingDependsOn*5), 20)
	if blastRadius > 0 {
		score += blastRadius
		factors = append(factors, ScoreFactor{
			Factor:      "blast_radius",
			Value:       blastRadius,
			Description: fmt.Sprintf("%d dependent assets increase blast radius", incomingDependsOn),
		})
	}

	departmentWeight := 0.0
	if asset.Department != nil && containsAny([]string{*asset.Department}, "finance", "executive", "security") {
		departmentWeight = 5
		score += departmentWeight
		factors = append(factors, ScoreFactor{
			Factor:      "department",
			Value:       departmentWeight,
			Description: fmt.Sprintf("%s systems carry higher business weight", *asset.Department),
		})
	}

	return math.Min(score, 100), factors
}

func CalculateExploitability(finding *model.CTEMFinding, primaryAsset *model.Asset, networkAccessible bool, activeThreatMatch bool, knownExploited bool) (float64, []ScoreFactor) {
	score := 0.0
	factors := make([]ScoreFactor, 0, 6)

	switch finding.Type {
	case model.CTEMFindingTypeAttackPath:
		base := pathBaseScore(finding)
		score += base
		factors = append(factors, ScoreFactor{Factor: "attack_path_base", Value: base, Description: "Attack-path traversal score"})
		if primaryAsset != nil && containsAny(primaryAsset.Tags, "internet-facing", "dmz", "public") {
			score += 15
			factors = append(factors, ScoreFactor{Factor: "internet_facing_entry", Value: 15, Description: "Attack path starts at an internet-facing asset"})
		}
		if strings.EqualFold(finding.Severity, "critical") {
			score += 10
			factors = append(factors, ScoreFactor{Factor: "critical_path_hop", Value: 10, Description: "Path includes a critical weakness"})
		}
	default:
		base := baseExploitabilityForFinding(finding)
		score += base
		factors = append(factors, ScoreFactor{Factor: "base", Value: base, Description: "Base exploitability from CVSS or severity"})
		if finding.Type == model.CTEMFindingTypeMisconfiguration || finding.Type == model.CTEMFindingTypeWeakCredential ||
			finding.Type == model.CTEMFindingTypeExpiredCertificate || finding.Type == model.CTEMFindingTypeInsecureProtocol ||
			finding.Type == model.CTEMFindingTypeMissingPatch {
			if primaryAsset != nil && containsAny(primaryAsset.Tags, "internet-facing", "dmz", "public") {
				score += 20
				factors = append(factors, ScoreFactor{Factor: "internet_facing", Value: 20, Description: "Misconfiguration is internet reachable"})
			}
			if finding.ValidationStatus == model.CTEMValidationPending {
				score += 10
				factors = append(factors, ScoreFactor{Factor: "no_compensating_controls", Value: 10, Description: "No compensating controls confirmed yet"})
			}
		}
	}

	if knownExploited {
		score += 20
		factors = append(factors, ScoreFactor{Factor: "known_active_exploitation", Value: 20, Description: "Known exploitation has been observed in the wild"})
	}

	if hasMetadataFlag(finding.Metadata, "public_exploit_available") {
		score += 15
		factors = append(factors, ScoreFactor{Factor: "public_exploit", Value: 15, Description: "Public exploit code is available"})
	}

	if networkAccessible {
		score += 10
		factors = append(factors, ScoreFactor{Factor: "network_accessible", Value: 10, Description: "Finding is reachable from an entry point"})
	}

	if hasNoAuthenticationRequired(finding) {
		score += 10
		factors = append(factors, ScoreFactor{Factor: "no_auth_required", Value: 10, Description: "Exploit does not require authentication"})
	}

	if activeThreatMatch {
		score += 10
		factors = append(factors, ScoreFactor{Factor: "threat_intel", Value: 10, Description: "Active threat activity overlaps with this finding"})
	}

	return math.Min(score, 100), factors
}

func CalculatePriorityScore(impact, exploitability float64) float64 {
	return round2((impact * 0.6) + (exploitability * 0.4))
}

func PriorityGroupForScore(score float64) int {
	switch {
	case score > 80:
		return 1
	case score >= 60:
		return 2
	case score >= 40:
		return 3
	default:
		return 4
	}
}

func gradeForScore(score float64) string {
	switch {
	case score <= 20:
		return "A"
	case score <= 40:
		return "B"
	case score <= 60:
		return "C"
	case score <= 80:
		return "D"
	default:
		return "F"
	}
}

func trendForDelta(delta float64) string {
	switch {
	case delta > 2:
		return "worsening"
	case delta < -2:
		return "improving"
	default:
		return "stable"
	}
}

func velocityBucket(avgDays *float64) float64 {
	if avgDays == nil {
		return 50
	}
	switch {
	case *avgDays < 7:
		return 10
	case *avgDays < 14:
		return 30
	case *avgDays < 30:
		return 50
	case *avgDays < 60:
		return 70
	case *avgDays < 90:
		return 85
	default:
		return 100
	}
}

func baseExploitabilityForFinding(finding *model.CTEMFinding) float64 {
	var evidence struct {
		CVSSScore  *float64 `json:"cvss_score"`
		CVSSVector *string  `json:"cvss_vector"`
	}
	_ = json.Unmarshal(finding.Evidence, &evidence)
	if evidence.CVSSScore != nil {
		return math.Min(*evidence.CVSSScore*10, 100)
	}
	switch strings.ToLower(finding.Severity) {
	case "critical":
		return 90
	case "high":
		return 70
	case "medium":
		return 50
	case "low":
		return 30
	default:
		return 0
	}
}

func pathBaseScore(finding *model.CTEMFinding) float64 {
	var hops []map[string]any
	if err := json.Unmarshal(finding.AttackPath, &hops); err != nil || len(hops) == 0 {
		return math.Min(finding.PriorityScore, 100)
	}
	total := 0.0
	for index, hop := range hops {
		severity, _ := hop["vuln_severity"].(string)
		total += severityWeight(severity) * hopWeight(index)
	}
	return math.Min(total, 100)
}

func severityWeight(severity string) float64 {
	switch strings.ToLower(severity) {
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

func hopWeight(index int) float64 {
	switch index {
	case 0:
		return 1.0
	case 1:
		return 0.8
	case 2:
		return 0.6
	case 3:
		return 0.4
	default:
		return 0.2
	}
}

func containsAny(values []string, candidates ...string) bool {
	for _, value := range values {
		for _, candidate := range candidates {
			if strings.EqualFold(value, candidate) {
				return true
			}
		}
	}
	return false
}

func hasMetadataFlag(metadata json.RawMessage, key string) bool {
	if len(metadata) == 0 {
		return false
	}
	var raw map[string]any
	if err := json.Unmarshal(metadata, &raw); err != nil {
		return false
	}
	value, ok := raw[key]
	if !ok {
		return false
	}
	boolean, ok := value.(bool)
	return ok && boolean
}

func hasNoAuthenticationRequired(finding *model.CTEMFinding) bool {
	var evidence struct {
		CVSSVector *string `json:"cvss_vector"`
	}
	if err := json.Unmarshal(finding.Evidence, &evidence); err != nil || evidence.CVSSVector == nil {
		return false
	}
	vector := *evidence.CVSSVector
	return strings.Contains(vector, "AV:N") && strings.Contains(vector, "PR:N")
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
