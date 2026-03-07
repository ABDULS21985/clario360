package ctem

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func (e *CTEMEngine) runMobilization(ctx context.Context, assessment *model.CTEMAssessment) error {
	findings, err := e.findingRepo.ListAllByAssessment(ctx, assessment.TenantID, assessment.ID)
	if err != nil {
		return err
	}
	existingGroups, err := e.remGroupRepo.ListByAssessment(ctx, assessment.TenantID, assessment.ID)
	if err != nil {
		return err
	}
	existingWorkflowBySignature := make(map[string]*string, len(existingGroups))
	for _, group := range existingGroups {
		signature := remediationGroupIdentity(group)
		if group.WorkflowInstanceID != nil && *group.WorkflowInstanceID != "" {
			existingWorkflowBySignature[signature] = group.WorkflowInstanceID
		}
	}
	assets, err := e.assetRepo.GetMany(ctx, assessment.TenantID, assessment.ResolvedAssetIDs)
	if err != nil {
		return err
	}
	assetIndex := make(map[uuid.UUID]*model.Asset, len(assets))
	for _, asset := range assets {
		assetIndex[asset.ID] = asset
	}

	if err := e.findingRepo.ClearRemediationAssignments(ctx, assessment.TenantID, assessment.ID); err != nil {
		return err
	}

	type groupState struct {
		Group       *model.CTEMRemediationGroup
		FindingRefs []*model.CTEMFinding
		AssetSet    map[uuid.UUID]bool
	}

	grouped := make(map[string]*groupState)
	now := time.Now().UTC()
	for _, finding := range findings {
		signature, group := buildRemediationGroup(assessment, finding, assetIndex, now)
		state, ok := grouped[signature]
		if !ok {
			state = &groupState{
				Group:    group,
				AssetSet: make(map[uuid.UUID]bool),
			}
			if workflowID, exists := existingWorkflowBySignature[remediationGroupIdentity(group)]; exists {
				state.Group.WorkflowInstanceID = workflowID
			}
			grouped[signature] = state
		}
		state.FindingRefs = append(state.FindingRefs, finding)
		if finding.PriorityScore > state.Group.MaxPriorityScore {
			state.Group.MaxPriorityScore = finding.PriorityScore
			state.Group.PriorityGroup = finding.PriorityGroup
		}
		for _, assetID := range finding.AffectedAssetIDs {
			state.AssetSet[assetID] = true
		}
		for _, cveID := range finding.CVEIDs {
			if !containsAny(state.Group.CVEIDs, cveID) {
				state.Group.CVEIDs = append(state.Group.CVEIDs, cveID)
			}
		}
	}

	groups := make([]*model.CTEMRemediationGroup, 0, len(grouped))
	for _, state := range grouped {
		state.Group.FindingCount = len(state.FindingRefs)
		state.Group.AffectedAssetCount = len(state.AssetSet)
		effort, estimatedDays := remediationEffortForGroup(state.Group.Type, state.Group.AffectedAssetCount)
		state.Group.Effort = effort
		state.Group.EstimatedDays = &estimatedDays
		scoreReduction := projectedScoreReduction(state.FindingRefs)
		state.Group.ScoreReduction = &scoreReduction
		state.Group.TargetDate = remediationTargetDate(now, state.Group.PriorityGroup)
		state.Group.Status = model.CTEMRemediationGroupPlanned

		description := remediationDescription(state.Group, state.FindingRefs, assetIndex)
		state.Group.Description = description
		for _, finding := range state.FindingRefs {
			finding.RemediationGroupID = &state.Group.ID
			remType := state.Group.Type
			remEffort := state.Group.Effort
			finding.RemediationType = &remType
			finding.RemediationDescription = &description
			finding.RemediationEffort = &remEffort
			finding.EstimatedDays = state.Group.EstimatedDays
		}
		groups = append(groups, state.Group)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].MaxPriorityScore == groups[j].MaxPriorityScore {
			return groups[i].Title < groups[j].Title
		}
		return groups[i].MaxPriorityScore > groups[j].MaxPriorityScore
	})

	if err := e.remGroupRepo.ReplaceForAssessment(ctx, assessment.TenantID, assessment.ID, groups); err != nil {
		return err
	}
	if err := e.autoTriggerPriorityOneGroups(ctx, assessment, groups); err != nil {
		return err
	}
	if err := e.findingRepo.SaveAnalysis(ctx, assessment.TenantID, assessment.ID, findings); err != nil {
		return err
	}
	progress := assessment.Phases["mobilizing"]
	progress.ItemsProcessed = len(groups)
	progress.ItemsTotal = len(groups)
	progress.Result = marshalMap(map[string]int{"groups": len(groups)})
	assessment.Phases["mobilizing"] = progress
	return e.assessmentRepo.SaveState(ctx, assessment)
}

func (e *CTEMEngine) autoTriggerPriorityOneGroups(ctx context.Context, assessment *model.CTEMAssessment, groups []*model.CTEMRemediationGroup) error {
	if e.workflow == nil || assessment.CreatedBy == nil {
		return nil
	}
	for _, group := range groups {
		if group.PriorityGroup != 1 || group.WorkflowInstanceID != nil {
			continue
		}
		instanceID, err := e.workflow.StartRemediation(ctx, assessment.TenantID, *assessment.CreatedBy, group, assessment)
		if err != nil {
			return err
		}
		group.WorkflowInstanceID = &instanceID
		if err := e.remGroupRepo.UpdateWorkflowInstance(ctx, assessment.TenantID, group.ID, instanceID); err != nil {
			return err
		}
		e.publishEvent(ctx, "cyber.ctem.remediation.triggered", assessment.TenantID.String(), map[string]any{
			"group_id":             group.ID.String(),
			"workflow_instance_id": instanceID,
		})
	}
	return nil
}

func buildRemediationGroup(assessment *model.CTEMAssessment, finding *model.CTEMFinding, assetIndex map[uuid.UUID]*model.Asset, now time.Time) (string, *model.CTEMRemediationGroup) {
	groupType := model.CTEMRemediationConfiguration
	title := "Configuration remediation"
	signature := findingIdentity(finding)

	switch {
	case len(finding.CVEIDs) > 0:
		groupType = model.CTEMRemediationPatch
		cveID := finding.CVEIDs[0]
		title = fmt.Sprintf("Apply patch for %s", cveID)
		signature = "patch:" + cveID
	case finding.Type == model.CTEMFindingTypeAttackPath:
		groupType = model.CTEMRemediationArchitecture
		title = "Implement network segmentation for validated attack paths"
		signature = "arch:attack_path"
	case finding.Type == model.CTEMFindingTypeMissingPatch:
		groupType = model.CTEMRemediationUpgrade
		title = "Upgrade outdated operating systems and runtimes"
		signature = "upgrade:" + remediationOSSignature(finding, assetIndex)
	case finding.PriorityGroup == 4:
		groupType = model.CTEMRemediationAcceptRisk
		title = "Accept and monitor low-priority findings"
		signature = "accept_risk"
	default:
		asset := findingPrimaryAsset(finding, assetIndex)
		if asset != nil && asset.Status == model.AssetStatusInactive {
			groupType = model.CTEMRemediationDecommission
			title = "Decommission inactive vulnerable assets"
			signature = "decommission"
		} else {
			groupType = model.CTEMRemediationConfiguration
			title = remediationConfigurationTitle(finding)
			signature = "config:" + string(finding.Type) + ":" + title
		}
	}

	return signature, &model.CTEMRemediationGroup{
		ID:               uuid.New(),
		TenantID:         assessment.TenantID,
		AssessmentID:     assessment.ID,
		Title:            title,
		Type:             groupType,
		MaxPriorityScore: finding.PriorityScore,
		PriorityGroup:    finding.PriorityGroup,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func remediationEffortForGroup(groupType model.CTEMRemediationType, assetCount int) (model.CTEMRemediationEffort, int) {
	if groupType == model.CTEMRemediationArchitecture {
		return model.CTEMRemediationEffortHigh, 21
	}
	switch {
	case assetCount < 5:
		return model.CTEMRemediationEffortLow, 1
	case assetCount <= 20:
		return model.CTEMRemediationEffortMedium, 3
	default:
		return model.CTEMRemediationEffortHigh, 7
	}
}

func projectedScoreReduction(findings []*model.CTEMFinding) float64 {
	total := 0.0
	for _, finding := range findings {
		total += finding.PriorityScore
	}
	return round2(total / 100)
}

func remediationTargetDate(now time.Time, priorityGroup int) *time.Time {
	switch priorityGroup {
	case 1:
		target := now.AddDate(0, 0, 3)
		return &target
	case 2:
		target := now.AddDate(0, 0, 14)
		return &target
	case 3:
		target := now.AddDate(0, 0, 90)
		return &target
	default:
		return nil
	}
}

func remediationDescription(group *model.CTEMRemediationGroup, findings []*model.CTEMFinding, assetIndex map[uuid.UUID]*model.Asset) string {
	switch group.Type {
	case model.CTEMRemediationPatch:
		return fmt.Sprintf("Deploy the vendor patch or fixed package version for %s across %d findings.", strings.Join(group.CVEIDs, ", "), len(findings))
	case model.CTEMRemediationArchitecture:
		return "Reduce lateral movement by segmenting internet-reachable systems from critical workloads and tightening trust paths."
	case model.CTEMRemediationUpgrade:
		return "Upgrade affected platforms to a currently supported version with the latest security fixes."
	case model.CTEMRemediationDecommission:
		return "Retire inactive assets from service and remove network access, credentials, and DNS records."
	case model.CTEMRemediationAcceptRisk:
		return "Document business acceptance, compensating controls, and quarterly review for low-priority findings."
	default:
		names := make([]string, 0)
		for _, finding := range findings {
			asset := findingPrimaryAsset(finding, assetIndex)
			if asset != nil && !containsAny(names, asset.Name) {
				names = append(names, asset.Name)
			}
		}
		return fmt.Sprintf("Apply a configuration correction across %d affected assets: %s.", len(names), strings.Join(names, ", "))
	}
}

func remediationConfigurationTitle(finding *model.CTEMFinding) string {
	var evidence map[string]any
	_ = json.Unmarshal(finding.Evidence, &evidence)
	if port, ok := evidence["port"]; ok {
		return fmt.Sprintf("Close exposed management port %v", port)
	}
	if protocol, ok := evidence["protocol"]; ok {
		return fmt.Sprintf("Disable insecure protocol %v", protocol)
	}
	return "Apply shared configuration hardening"
}

func remediationOSSignature(finding *model.CTEMFinding, assetIndex map[uuid.UUID]*model.Asset) string {
	asset := findingPrimaryAsset(finding, assetIndex)
	if asset == nil || asset.OSVersion == nil {
		return "general"
	}
	return *asset.OSVersion
}

func remediationGroupIdentity(group *model.CTEMRemediationGroup) string {
	cveIDs := append([]string(nil), group.CVEIDs...)
	sort.Strings(cveIDs)
	return strings.Join([]string{string(group.Type), group.Title, strings.Join(cveIDs, ",")}, "|")
}
