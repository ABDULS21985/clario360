package commands

import (
	"context"

	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func ExecuteAck(ctx context.Context, api *intsvc.ClarioAPIClient, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	if err := permissions.RequireLinkedUser(cmd); err != nil {
		return linkedError(), nil
	}
	if !permissions.UserHasPermission(cmd.User, "cyber:write") && !permissions.UserHasPermission(cmd.User, "cyber:alerts:write") {
		return permissionError("cyber:write"), nil
	}
	if len(cmd.Args) == 0 {
		return &bottypes.BotResponse{Text: "Usage: `/clario ack <alert-id>`", Ephemeral: true}, nil
	}

	if err := api.UpdateAlertStatus(ctx, cmd.Token, cmd.Args[0], "acknowledged", nil, nil); err != nil {
		return nil, err
	}
	return &bottypes.BotResponse{
		Text:     "✅ Alert acknowledged.",
		DataType: "text",
	}, nil
}
