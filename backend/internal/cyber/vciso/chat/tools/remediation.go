package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type RemediationTool struct {
	baseTool
}

func NewRemediationTool(deps *Dependencies) *RemediationTool {
	return &RemediationTool{baseTool: newBaseTool(deps)}
}

func (t *RemediationTool) Name() string { return "remediation" }

func (t *RemediationTool) Description() string {
	return "start a governed remediation action for an alert or issue"
}

func (t *RemediationTool) RequiredPermissions() []string { return []string{"cyber:write"} }

func (t *RemediationTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.AlertService == nil || t.deps.RemediationService == nil {
		return nil, fmt.Errorf("%w: remediation dependencies", errToolUnavailable)
	}
	alertID, err := t.requireCyberAlertID(ctx, tenantID, params["alert_id"])
	if err != nil {
		return nil, err
	}
	alert, err := t.deps.AlertService.GetAlert(ctx, tenantID, alertID, t.actorFromContext(ctx, userID))
	if err != nil {
		return nil, err
	}
	remediationType := deriveRemediationType(alert)
	affected := make([]uuid.UUID, 0, len(alert.AssetIDs)+1)
	if alert.AssetID != nil {
		affected = append(affected, *alert.AssetID)
	}
	affected = append(affected, alert.AssetIDs...)
	req := &cyberdto.CreateRemediationRequest{
		AlertID:              &alert.ID,
		Type:                 string(remediationType),
		Severity:             string(alert.Severity),
		Title:                "Remediate alert: " + alert.Title,
		Description:          "Governed remediation action created by vCISO for alert " + alert.ID.String(),
		Plan:                 remediationPlanForAlert(alert, remediationType),
		AffectedAssetIDs:     affected,
		ExecutionMode:        "guided",
		RequiresApprovalFrom: "security_manager",
		Tags:                 []string{"vciso", "alert", strings.ToLower(string(alert.Severity))},
		Metadata: map[string]any{
			"source":   "vciso",
			"alert_id": alert.ID.String(),
		},
	}
	item, err := t.deps.RemediationService.Create(ctx, tenantID, userID, t.actorFromContext(ctx, userID), req)
	if err != nil {
		return nil, err
	}
	return &ToolResult{
		Text:     fmt.Sprintf("I created remediation **%s** for alert **%s**. Status: **%s**.", item.Title, alert.Title, item.Status),
		Data:     item,
		DataType: "list",
		Actions: []chatmodel.SuggestedAction{
			navigateAction("Open remediation", "/cyber/remediation/"+item.ID.String()),
			navigateAction("Open alert", "/cyber/alerts/"+alert.ID.String()),
		},
		Entities: []chatmodel.EntityReference{
			entityRef("remediation", item.ID.String(), item.Title, 0),
			entityRef("alert", alert.ID.String(), alert.Title, 1),
		},
	}, nil
}

func deriveRemediationType(alert *cybermodel.Alert) cybermodel.RemediationType {
	if alert == nil {
		return cybermodel.RemediationTypeCustom
	}
	title := strings.ToLower(alert.Title + " " + alert.Description)
	switch {
	case strings.Contains(title, "credential"), strings.Contains(title, "account"), strings.Contains(title, "valid accounts"):
		return cybermodel.RemediationTypeAccessRevoke
	case strings.Contains(title, "ip"), strings.Contains(title, "network"), strings.Contains(title, "scanner"):
		return cybermodel.RemediationTypeBlockIP
	case alert.Severity == cybermodel.SeverityCritical:
		return cybermodel.RemediationTypeIsolateAsset
	default:
		return cybermodel.RemediationTypeConfigChange
	}
}

func remediationPlanForAlert(alert *cybermodel.Alert, remediationType cybermodel.RemediationType) cybermodel.RemediationPlan {
	return cybermodel.RemediationPlan{
		Steps: []cybermodel.RemediationStep{
			{Number: 1, Action: "review", Description: "Validate the alert evidence and confirm affected assets."},
			{Number: 2, Action: string(remediationType), Description: "Apply the recommended control to contain or correct the issue."},
			{Number: 3, Action: "verify", Description: "Re-check telemetry and confirm the triggering condition no longer appears."},
		},
		Reversible:        remediationType != cybermodel.RemediationTypeBlockIP,
		RequiresReboot:    false,
		EstimatedDowntime: "low",
		RiskLevel:         strings.ToLower(string(alert.Severity)),
	}
}
