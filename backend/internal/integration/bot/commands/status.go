package commands

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func ExecuteStatus(ctx context.Context, api *intsvc.ClarioAPIClient, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	if err := permissions.RequireLinkedUser(cmd); err != nil {
		return linkedError(), nil
	}
	if !permissions.UserHasPermission(cmd.User, "cyber:read") {
		return permissionError("cyber:read"), nil
	}

	var riskResp struct {
		Data map[string]any `json:"data"`
	}
	var alertCount struct {
		Count int `json:"count"`
	}
	var pipelineResp struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_, _, _ = api.GatewayRequest(ctx, "GET", "/api/v1/cyber/risk/score", cmd.Token, nil, &riskResp)
	_, _, _ = api.GatewayRequest(ctx, "GET", "/api/v1/cyber/alerts/count?status=new,acknowledged", cmd.Token, nil, &alertCount)
	_, _, _ = api.GatewayRequest(ctx, "GET", "/api/v1/data/pipelines?status=failed&page=1&per_page=1", cmd.Token, nil, &pipelineResp)

	text := fmt.Sprintf("🏢 *Clario 360 Platform Status*\n\n*Risk Score:* %v\n*Open Alerts:* %d\n*Failing Pipelines:* %d",
		riskResp.Data["overall_score"], alertCount.Count, pipelineResp.Meta.Total)
	return &bottypes.BotResponse{
		Text:     text,
		DataType: "kpi",
		Data: map[string]any{
			"risk_score":       riskResp.Data["overall_score"],
			"open_alerts":      alertCount.Count,
			"failing_pipelines": pipelineResp.Meta.Total,
		},
	}, nil
}
