package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/mitre"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type InvestigationTool struct {
	baseTool
}

func NewInvestigationTool(deps *Dependencies) *InvestigationTool {
	return &InvestigationTool{baseTool: newBaseTool(deps)}
}

func (t *InvestigationTool) Name() string { return "investigation" }

func (t *InvestigationTool) Description() string { return "run a comprehensive investigation on a specific alert" }

func (t *InvestigationTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *InvestigationTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.AlertService == nil {
		return nil, fmt.Errorf("%w: alert service", errToolUnavailable)
	}
	alertID, err := t.requireCyberAlertID(ctx, tenantID, params["alert_id"])
	if err != nil {
		return nil, err
	}
	actor := t.actorFromContext(ctx, userID)
	alert, err := t.deps.AlertService.GetAlert(ctx, tenantID, alertID, actor)
	if err != nil {
		return nil, err
	}
	related, _ := t.deps.AlertService.Related(ctx, tenantID, alertID, actor)
	var ruleName string
	if alert.RuleID != nil && t.deps.RuleService != nil {
		if rule, ruleErr := t.deps.RuleService.GetRule(ctx, tenantID, *alert.RuleID, actor); ruleErr == nil {
			ruleName = rule.Name
		}
	}
	assetIDs := make([]uuid.UUID, 0, 10)
	if alert.AssetID != nil {
		assetIDs = append(assetIDs, *alert.AssetID)
	}
	for _, assetID := range alert.AssetIDs {
		if len(assetIDs) == 10 {
			break
		}
		duplicate := false
		for _, existing := range assetIDs {
			if existing == assetID {
				duplicate = true
				break
			}
		}
		if !duplicate {
			assetIDs = append(assetIDs, assetID)
		}
	}
	assets := make([]*cybermodel.Asset, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		if t.deps.AssetService == nil {
			break
		}
		if asset, assetErr := t.deps.AssetService.GetAsset(ctx, tenantID, assetID); assetErr == nil {
			assets = append(assets, asset)
		}
	}
	var (
		uebaData    any
		entityID    string
		entityName  string
		entityField map[string]any
	)
	if len(alert.Metadata) > 0 {
		_ = json.Unmarshal(alert.Metadata, &entityField)
		if raw, ok := entityField["entity_id"].(string); ok {
			entityID = raw
		}
		if raw, ok := entityField["entity_name"].(string); ok {
			entityName = raw
		}
	}
	if entityID != "" && t.deps.UEBAService != nil {
		if profile, profileErr := t.deps.UEBAService.GetProfile(ctx, tenantID, entityID); profileErr == nil {
			uebaData = profile
		}
	}
	var mitreData map[string]any
	if alert.MITRETechniqueID != nil {
		if technique, ok := mitre.TechniqueByID(*alert.MITRETechniqueID); ok {
			mitreData = map[string]any{
				"technique_id":   technique.ID,
				"technique_name": technique.Name,
				"tactic_ids":     technique.TacticIDs,
			}
		}
	}
	lines := []string{
		fmt.Sprintf("## 🔍 Investigation: %s", alert.Title),
		"",
		fmt.Sprintf("**Severity:** %s | **Confidence:** %s | **Status:** %s", alert.Severity, confidencePercent(alert.ConfidenceScore), alert.Status),
		fmt.Sprintf("**Detected:** %s", alert.CreatedAt.Format(time.RFC3339)),
	}
	if ruleName != "" {
		lines = append(lines, fmt.Sprintf("**Rule:** %s", ruleName))
	}
	lines = append(lines, "", "### What Happened", alert.Explanation.Summary)
	if len(alert.Explanation.ConfidenceFactors) > 0 {
		lines = append(lines, "", "### Confidence Analysis")
		for _, factor := range alert.Explanation.ConfidenceFactors {
			lines = append(lines, fmt.Sprintf("- %s (%+.1f): %s", factor.Factor, factor.Impact, factor.Description))
		}
	}
	if len(alert.Explanation.MatchedConditions) > 0 {
		lines = append(lines, "", "### Matched Conditions")
		for _, condition := range alert.Explanation.MatchedConditions {
			lines = append(lines, "- "+condition)
		}
	}
	if len(assets) > 0 {
		lines = append(lines, "", fmt.Sprintf("### Affected Assets (%d)", len(assets)))
		for _, asset := range assets {
			lines = append(lines, fmt.Sprintf("- %s (%s, %s)", asset.Name, asset.Type, asset.Criticality))
		}
	}
	if len(related) > 0 {
		lines = append(lines, "", fmt.Sprintf("### Related Alerts (%d)", len(related)))
		for _, item := range related {
			lines = append(lines, fmt.Sprintf("- %s %s (%s)", formatSeverityIcon(string(item.Severity)), item.Title, item.Status))
		}
	}
	if mitreData != nil {
		lines = append(lines, "", "### MITRE ATT&CK", fmt.Sprintf("Technique: **%s** (%s)", mitreData["technique_name"], mitreData["technique_id"]))
	}
	if uebaData != nil {
		lines = append(lines, "", "### Behavioral Context")
		if entityName == "" {
			entityName = entityID
		}
		lines = append(lines, fmt.Sprintf("Entity: **%s**", entityName))
	}
	if len(alert.Explanation.RecommendedActions) > 0 {
		lines = append(lines, "", "### Recommended Actions")
		for idx, action := range alert.Explanation.RecommendedActions {
			lines = append(lines, fmt.Sprintf("%d. %s", idx+1, action))
		}
	}
	if len(alert.Explanation.FalsePositiveIndicators) > 0 {
		lines = append(lines, "", "### False Positive Indicators")
		for _, indicator := range alert.Explanation.FalsePositiveIndicators {
			lines = append(lines, "- "+indicator)
		}
	}
	entities := []chatmodel.EntityReference{entityRef("alert", alert.ID.String(), alert.Title, 0)}
	for idx, asset := range assets {
		entities = append(entities, entityRef("asset", asset.ID.String(), asset.Name, idx))
	}
	return &ToolResult{
		Text: strings.Join(lines, "\n"),
		Data: map[string]any{
			"alert":          alert,
			"assets":         assets,
			"related_alerts": related,
			"mitre":          mitreData,
			"ueba":           uebaData,
			"rule_name":      ruleName,
		},
		DataType: "investigation",
		Actions: []chatmodel.SuggestedAction{
			navigateAction("View in dashboard", "/cyber/alerts/"+alert.ID.String()),
			confirmMessageAction("Start remediation", "Start remediation for alert "+alert.ID.String(), "This will create a remediation action that may require approval."),
			navigateAction("Open alert queue", "/cyber/alerts"),
		},
		Entities: entities,
	}, nil
}
