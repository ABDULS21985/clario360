package tools

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"

	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type AssetLookupTool struct {
	baseTool
}

func NewAssetLookupTool(deps *Dependencies) *AssetLookupTool {
	return &AssetLookupTool{baseTool: newBaseTool(deps)}
}

func (t *AssetLookupTool) Name() string { return "asset_lookup" }

func (t *AssetLookupTool) Description() string { return "look up details about an asset, server, host, or device" }

func (t *AssetLookupTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *AssetLookupTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.AssetService == nil || t.deps.AlertService == nil {
		return nil, fmt.Errorf("%w: asset service", errToolUnavailable)
	}

	var (
		asset any
		err   error
	)
	if rawID := strings.TrimSpace(params["asset_name"]); rawID != "" {
		if parsed, parseErr := uuid.Parse(rawID); parseErr == nil {
			asset, err = t.deps.AssetService.GetAsset(ctx, tenantID, parsed)
		}
	}
	if asset == nil && err == nil {
		search := strings.TrimSpace(params["asset_name"])
		if ip := strings.TrimSpace(params["asset_ip"]); ip != "" && net.ParseIP(ip) != nil {
			search = ip
		}
		result, listErr := t.deps.AssetService.ListAssets(ctx, tenantID, assetSearchParams(search))
		if listErr != nil {
			return nil, listErr
		}
		if result.Total == 0 || len(result.Data) == 0 {
			return makeListResult(fmt.Sprintf("I couldn't find an asset matching **%s**.", search), map[string]any{"assets": []any{}}, []chatmodel.SuggestedAction{}, nil), nil
		}
		asset = &result.Data[0]
	}
	if err != nil {
		return nil, err
	}

	assetID := asset.(*cybermodel.Asset).ID
	vulns, vulnTotal, vulnErr := t.deps.AssetService.ListVulnerabilities(ctx, tenantID, assetID, &cyberdto.VulnerabilityListParams{Page: 1, PerPage: 5})
	if vulnErr != nil {
		return nil, vulnErr
	}
	actor := t.actorFromContext(ctx, userID)
	alerts, alertErr := t.deps.AlertService.ListAlerts(ctx, tenantID, &cyberdto.AlertListParams{AssetID: &assetID, Page: 1, PerPage: 5, Sort: "created_at", Order: "desc"}, actor)
	if alertErr != nil {
		return nil, alertErr
	}

	typed := asset.(*cybermodel.Asset)
	lines := []string{
		fmt.Sprintf("## %s", typed.Name),
		"",
		fmt.Sprintf("- Type: **%s**", typed.Type),
		fmt.Sprintf("- Criticality: **%s**", typed.Criticality),
		fmt.Sprintf("- Status: **%s**", typed.Status),
	}
	if typed.IPAddress != nil {
		lines = append(lines, fmt.Sprintf("- IP: **%s**", *typed.IPAddress))
	}
	if typed.Hostname != nil {
		lines = append(lines, fmt.Sprintf("- Hostname: **%s**", *typed.Hostname))
	}
	lines = append(lines, "", fmt.Sprintf("Open vulnerabilities shown: **%d**. Recent alerts shown: **%d**.", vulnTotal, len(alerts.Data)))
	actions := []chatmodel.SuggestedAction{
		navigateAction("View asset page", "/cyber/assets/"+typed.ID.String()),
		messageAction("Show vulnerabilities", fmt.Sprintf("Top vulnerabilities for %s", typed.Name)),
	}
	if len(alerts.Data) > 0 {
		actions = append(actions, messageAction("Investigate first alert", fmt.Sprintf("Investigate alert %s", alerts.Data[0].ID.String())))
	}
	return &ToolResult{
		Text: strings.Join(lines, "\n"),
		Data: map[string]any{
			"asset":           typed,
			"vulnerabilities": vulns,
			"alerts":          alerts.Data,
		},
		DataType: "table",
		Actions:  actions,
		Entities: []chatmodel.EntityReference{entityRef("asset", typed.ID.String(), typed.Name, 0)},
	}, nil
}
