package ctem

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func (e *CTEMEngine) runPrioritization(ctx context.Context, assessment *model.CTEMAssessment) error {
	findings, err := e.findingRepo.ListAllByAssessment(ctx, assessment.TenantID, assessment.ID)
	if err != nil {
		return err
	}
	if len(findings) == 0 {
		progress := assessment.Phases["prioritizing"]
		progress.Result = json.RawMessage(`{"priority_group":{}}`)
		assessment.Phases["prioritizing"] = progress
		return e.assessmentRepo.SaveState(ctx, assessment)
	}

	assets, err := e.assetRepo.GetMany(ctx, assessment.TenantID, assessment.ResolvedAssetIDs)
	if err != nil {
		return err
	}
	assetIndex := make(map[uuid.UUID]*model.Asset, len(assets))
	for _, asset := range assets {
		assetIndex[asset.ID] = asset
	}

	relationships, err := e.loadScopedRelationships(ctx, assessment.TenantID, assessment.ResolvedAssetIDs)
	if err != nil {
		return err
	}
	incomingDependsOn := make(map[uuid.UUID]int)
	for _, rel := range relationships {
		if rel.RelationshipType == model.RelationshipDependsOn {
			incomingDependsOn[rel.TargetAssetID]++
		}
	}

	networkAccessibleAssets := make(map[uuid.UUID]bool)
	for _, finding := range findings {
		if finding.Type != model.CTEMFindingTypeAttackPath {
			continue
		}
		for _, id := range finding.AffectedAssetIDs {
			networkAccessibleAssets[id] = true
		}
	}
	activeThreatAssets, err := e.loadActiveThreatAssetSet(ctx, assessment.TenantID)
	if err != nil {
		return err
	}

	for index, finding := range findings {
		if err := ctx.Err(); err != nil {
			return err
		}
		var primaryAsset *model.Asset
		if finding.PrimaryAssetID != nil {
			primaryAsset = assetIndex[*finding.PrimaryAssetID]
		}
		if primaryAsset == nil && len(finding.AffectedAssetIDs) > 0 {
			primaryAsset = assetIndex[finding.AffectedAssetIDs[0]]
		}
		if primaryAsset == nil {
			continue
		}

		impact, impactFactors := CalculateBusinessImpact(primaryAsset, incomingDependsOn[primaryAsset.ID])
		knownExploited, err := e.hasKnownExploitation(ctx, finding.CVEIDs)
		if err != nil {
			return err
		}
		exploitability, exploitFactors := CalculateExploitability(
			finding,
			primaryAsset,
			networkAccessibleAssets[primaryAsset.ID] || containsAny(primaryAsset.Tags, "internet-facing", "dmz", "public"),
			activeThreatAssets[primaryAsset.ID],
			knownExploited,
		)
		finding.BusinessImpactScore = impact
		finding.BusinessImpactFactors = marshalFactors(impactFactors)
		finding.ExploitabilityScore = exploitability
		finding.ExploitabilityFactors = marshalFactors(exploitFactors)
		finding.PriorityScore = CalculatePriorityScore(impact, exploitability)
		finding.PriorityGroup = PriorityGroupForScore(finding.PriorityScore)

		if (index+1)%50 == 0 || index == len(findings)-1 {
			if err := e.UpdatePhaseProgress(ctx, assessment, "prioritizing", index+1, len(findings)); err != nil {
				return err
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].PriorityScore == findings[j].PriorityScore {
			return severityWeight(findings[i].Severity) > severityWeight(findings[j].Severity)
		}
		return findings[i].PriorityScore > findings[j].PriorityScore
	})
	for index, finding := range findings {
		rank := index + 1
		finding.PriorityRank = &rank
	}

	if err := e.findingRepo.SaveAnalysis(ctx, assessment.TenantID, assessment.ID, findings); err != nil {
		return err
	}

	groupCounts := map[string]int{}
	for _, finding := range findings {
		groupCounts[fmt.Sprintf("%d", finding.PriorityGroup)]++
	}
	progress := assessment.Phases["prioritizing"]
	progress.Result = marshalMap(groupCounts)
	assessment.Phases["prioritizing"] = progress
	e.recordPrioritizationPrediction(ctx, assessment, findings)
	return e.assessmentRepo.SaveState(ctx, assessment)
}

func (e *CTEMEngine) loadActiveThreatAssetSet(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := e.db.Query(ctx, `
		SELECT DISTINCT asset_ref
		FROM (
			SELECT COALESCE(asset_id, affected_asset) AS asset_ref
			FROM (
				SELECT a.asset_id,
				       unnest(CASE WHEN cardinality(a.asset_ids) = 0 THEN ARRAY[a.asset_id]::uuid[] ELSE a.asset_ids END) AS affected_asset
				FROM alerts a
				WHERE a.tenant_id = $1
				  AND a.deleted_at IS NULL
				  AND a.status IN ('new','acknowledged','investigating')
			) expanded
		) refs
		WHERE asset_ref IS NOT NULL`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assetSet := make(map[uuid.UUID]bool)
	for rows.Next() {
		var assetID uuid.UUID
		if err := rows.Scan(&assetID); err != nil {
			return nil, err
		}
		assetSet[assetID] = true
	}
	return assetSet, rows.Err()
}

func (e *CTEMEngine) hasKnownExploitation(ctx context.Context, cveIDs []string) (bool, error) {
	if len(cveIDs) == 0 {
		return false, nil
	}
	row := e.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM cve_database
			WHERE cve_id = ANY($1)
			  AND (
				lower(references::text) LIKE '%known-exploited%'
				OR lower(references::text) LIKE '%kev%'
				OR lower(description) LIKE '%actively exploited%'
			  )
		)`,
		cveIDs,
	)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func marshalFactors(factors []ScoreFactor) json.RawMessage {
	payload, _ := json.Marshal(factors)
	return payload
}

func marshalMap(value map[string]int) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func findingIdentity(finding *model.CTEMFinding) string {
	assetID := ""
	if finding.PrimaryAssetID != nil {
		assetID = finding.PrimaryAssetID.String()
	}
	cveID := ""
	if len(finding.CVEIDs) > 0 {
		cveID = finding.CVEIDs[0]
	}
	if cveID == "" {
		cveID = strings.ToLower(finding.Title)
	}
	return strings.Join([]string{string(finding.Type), assetID, cveID}, "|")
}
