package commands

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func ExecuteInvestigate(ctx context.Context, api *intsvc.ClarioAPIClient, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	if err := permissions.RequireLinkedUser(cmd); err != nil {
		return linkedError(), nil
	}
	if !permissions.UserHasPermission(cmd.User, "cyber:read") {
		return permissionError("cyber:read"), nil
	}
	if len(cmd.Args) == 0 {
		return &bottypes.BotResponse{Text: "Usage: `/clario investigate <alert-id>`", Ephemeral: true}, nil
	}

	entity, err := api.FetchEntity(ctx, cmd.Token, "alert", cmd.Args[0])
	if err != nil {
		return nil, err
	}
	text := fmt.Sprintf("🔍 *Investigation: %s*\n\n*Severity:* %s\n*Status:* %s\n*Summary:* %s",
		firstNonEmpty(stringValue(entity["title"]), "Alert"),
		firstNonEmpty(stringValue(entity["severity"]), "info"),
		firstNonEmpty(stringValue(entity["status"]), "new"),
		firstNonEmpty(extractNestedString(entity, "explanation", "summary"), stringValue(entity["description"])),
	)
	return &bottypes.BotResponse{
		Text:     text,
		DataType: "detail",
		Data:     entity,
		InThread: true,
	}, nil
}
