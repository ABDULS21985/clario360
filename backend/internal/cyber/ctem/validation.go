package ctem

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func (e *CTEMEngine) runValidation(ctx context.Context, assessment *model.CTEMAssessment) error {
	findings, err := e.findingRepo.ListAllByAssessment(ctx, assessment.TenantID, assessment.ID)
	if err != nil {
		return err
	}
	assets, err := e.assetRepo.GetMany(ctx, assessment.TenantID, assessment.ResolvedAssetIDs)
	if err != nil {
		return err
	}
	assetIndex := make(map[uuid.UUID]*model.Asset, len(assets))
	for _, asset := range assets {
		assetIndex[asset.ID] = asset
	}
	activeThreatAssets, err := e.loadActiveThreatAssetSet(ctx, assessment.TenantID)
	if err != nil {
		return err
	}
	reachableAssets := make(map[uuid.UUID]bool)
	for _, finding := range findings {
		if finding.Type != model.CTEMFindingTypeAttackPath {
			continue
		}
		for _, id := range finding.AffectedAssetIDs {
			reachableAssets[id] = true
		}
	}

	validated := 0
	for _, finding := range findings {
		if err := ctx.Err(); err != nil {
			return err
		}
		if finding.PriorityGroup > 2 {
			continue
		}
		asset := findingPrimaryAsset(finding, assetIndex)
		if asset == nil {
			continue
		}

		notes := []string{}
		controls := make([]string, 0, 4)
		adjustment := 1.0
		status := model.CTEMValidationValidated

		if asset.LastSeenAt.Before(time.Now().UTC().AddDate(0, 0, -30)) {
			status = model.CTEMValidationRequiresManual
			notes = append(notes, "Asset version information may be outdated")
		}

		if finding.Type == model.CTEMFindingTypeVulnerability {
			openPorts := metadataIntSlice(decodeMetadata(asset.Metadata)["open_ports"])
			if len(openPorts) == 0 && !reachableAssets[asset.ID] {
				status = model.CTEMValidationNotExploitable
				adjustment *= 0.5
				notes = append(notes, "No reachable service exposure was confirmed for this vulnerability")
			}
		}

		if hasProtection(asset, "waf") && (finding.Type == model.CTEMFindingTypeVulnerability || finding.Type == model.CTEMFindingTypeMisconfiguration || finding.Type == model.CTEMFindingTypeAttackPath) {
			controls = append(controls, "Web Application Firewall active")
		}
		if hasProtection(asset, "segmented") {
			controls = append(controls, "Network segmentation limits access")
		}
		if activeThreatAssets[asset.ID] {
			controls = append(controls, "Detection rules active on asset")
		}
		if strings.Contains(stringOrEmptyCVSS(finding), "PR:L") || strings.Contains(stringOrEmptyCVSS(finding), "PR:H") {
			if containsAny(asset.Tags, "mfa", "sso") {
				controls = append(controls, "Multi-factor authentication required")
			}
		}

		if len(controls) > 0 && status != model.CTEMValidationNotExploitable {
			status = model.CTEMValidationCompensated
			adjustment *= 0.8
		}

		finding.PriorityScore = round2(finding.PriorityScore * adjustment)
		finding.PriorityGroup = PriorityGroupForScore(finding.PriorityScore)
		finding.ValidationStatus = status
		finding.CompensatingControls = controls
		finding.ValidatedAt = timePtr(time.Now().UTC())
		if len(notes) > 0 {
			note := strings.Join(notes, "; ")
			finding.ValidationNotes = &note
		}
		validated++
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
	progress := assessment.Phases["validating"]
	progress.ItemsProcessed = validated
	progress.ItemsTotal = countPriorityGroups(findings, 1, 2)
	progress.Result = json.RawMessage(`{"validated":true}`)
	assessment.Phases["validating"] = progress
	return e.assessmentRepo.SaveState(ctx, assessment)
}

func findingPrimaryAsset(finding *model.CTEMFinding, assetIndex map[uuid.UUID]*model.Asset) *model.Asset {
	if finding.PrimaryAssetID != nil {
		if asset := assetIndex[*finding.PrimaryAssetID]; asset != nil {
			return asset
		}
	}
	if len(finding.AffectedAssetIDs) > 0 {
		return assetIndex[finding.AffectedAssetIDs[0]]
	}
	return nil
}

func hasProtection(asset *model.Asset, name string) bool {
	if asset == nil {
		return false
	}
	if containsAny(asset.Tags, name) {
		return true
	}
	meta := decodeMetadata(asset.Metadata)
	for _, key := range []string{name, name + "_enabled", "behind_" + name} {
		if metadataBool(meta[key]) {
			return true
		}
	}
	if name == "segmented" {
		return meta["vlan_segment"] != nil
	}
	return false
}

func stringOrEmptyCVSS(finding *model.CTEMFinding) string {
	var evidence struct {
		CVSSVector *string `json:"cvss_vector"`
	}
	if err := json.Unmarshal(finding.Evidence, &evidence); err != nil || evidence.CVSSVector == nil {
		return ""
	}
	return *evidence.CVSSVector
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func countPriorityGroups(findings []*model.CTEMFinding, groups ...int) int {
	set := make(map[int]bool, len(groups))
	for _, group := range groups {
		set[group] = true
	}
	count := 0
	for _, finding := range findings {
		if set[finding.PriorityGroup] {
			count++
		}
	}
	return count
}
