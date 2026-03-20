package commands

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func ExecuteRisk(ctx context.Context, api *intsvc.ClarioAPIClient, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	if err := permissions.RequireLinkedUser(cmd); err != nil {
		return linkedError(), nil
	}
	if !permissions.UserHasPermission(cmd.User, "cyber:read") {
		return permissionError("cyber:read"), nil
	}

	var response struct {
		Data map[string]any `json:"data"`
	}
	if _, _, err := api.GatewayRequest(ctx, "GET", "/api/v1/cyber/risk/score", cmd.Token, nil, &response); err != nil {
		return nil, err
	}
	return &bottypes.BotResponse{
		Text:     fmt.Sprintf("📈 *Current Risk Score:* %v", response.Data["overall_score"]),
		DataType: "kpi",
		Data:     response.Data,
	}, nil
}
