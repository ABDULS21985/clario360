package commands

import (
	"context"
	"strings"

	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func ExecuteAssign(ctx context.Context, api *intsvc.ClarioAPIClient, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	if err := permissions.RequireLinkedUser(cmd); err != nil {
		return linkedError(), nil
	}
	if !permissions.UserHasPermission(cmd.User, "cyber:write") && !permissions.UserHasPermission(cmd.User, "cyber:alerts:write") {
		return permissionError("cyber:write"), nil
	}
	if len(cmd.Args) < 2 {
		return &bottypes.BotResponse{Text: "Usage: `/clario assign <alert-id> <user-id|email>`", Ephemeral: true}, nil
	}

	target := strings.TrimSpace(cmd.Args[1])
	var userID string
	if strings.Contains(target, "@") {
		user, err := api.LookupUserByEmail(ctx, cmd.TenantID, target)
		if err != nil {
			return nil, err
		}
		userID = user.ID
	} else {
		userID = target
	}
	if err := api.AssignAlert(ctx, cmd.Token, cmd.Args[0], userID); err != nil {
		return nil, err
	}
	return &bottypes.BotResponse{
		Text:     "✅ Alert assigned.",
		DataType: "text",
	}, nil
}
