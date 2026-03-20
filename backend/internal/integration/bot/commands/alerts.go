package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/integration/bot/permissions"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func ExecuteAlerts(ctx context.Context, api *intsvc.ClarioAPIClient, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	if err := permissions.RequireLinkedUser(cmd); err != nil {
		return linkedError(), nil
	}
	if !permissions.UserHasPermission(cmd.User, "cyber:read") {
		return permissionError("cyber:read"), nil
	}

	severity := ""
	if len(cmd.Args) > 0 {
		severity = strings.ToLower(cmd.Args[0])
	}
	path := "/api/v1/cyber/alerts?status=new,acknowledged&per_page=5&sort=created_at&order=desc"
	if severity != "" {
		path += "&severity=" + severity
	}
	var response struct {
		Data []map[string]any `json:"data"`
	}
	if _, _, err := api.GatewayRequest(ctx, "GET", path, cmd.Token, nil, &response); err != nil {
		return nil, err
	}

	lines := []string{"📋 *Recent Alerts*"}
	for _, alert := range response.Data {
		lines = append(lines, fmt.Sprintf("• %s — %s — %s",
			firstNonEmpty(stringValue(alert["title"]), "Alert"),
			firstNonEmpty(stringValue(alert["severity"]), "info"),
			firstNonEmpty(stringValue(alert["status"]), "new"),
		))
	}
	return &bottypes.BotResponse{
		Text:     strings.Join(lines, "\n"),
		DataType: "list",
		Data:     response.Data,
	}, nil
}
