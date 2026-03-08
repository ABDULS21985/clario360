package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/integration/bot/commands"
	bottypes "github.com/clario360/platform/internal/integration/bot/types"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

type Router struct {
	api    *intsvc.ClarioAPIClient
	logger zerolog.Logger
}

func NewRouter(api *intsvc.ClarioAPIClient, logger zerolog.Logger) *Router {
	return &Router{
		api:    api,
		logger: logger.With().Str("component", "integration_bot_router").Logger(),
	}
}

func (r *Router) Route(ctx context.Context, cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	subcommand := strings.ToLower(strings.TrimSpace(cmd.Subcommand))
	if subcommand == "" {
		subcommand = "help"
	}

	switch subcommand {
	case "help":
		return commands.ExecuteHelp(cmd)
	case "status":
		return commands.ExecuteStatus(ctx, r.api, cmd)
	case "alerts":
		return commands.ExecuteAlerts(ctx, r.api, cmd)
	case "risk":
		return commands.ExecuteRisk(ctx, r.api, cmd)
	case "investigate":
		return commands.ExecuteInvestigate(ctx, r.api, cmd)
	case "ack":
		return commands.ExecuteAck(ctx, r.api, cmd)
	case "assign":
		return commands.ExecuteAssign(ctx, r.api, cmd)
	default:
		help, _ := commands.ExecuteHelp(cmd)
		help.Text = fmt.Sprintf("Unknown command `%s`.\n\n%s", subcommand, help.Text)
		return help, nil
	}
}
