package commands

import bottypes "github.com/clario360/platform/internal/integration/bot/types"

func ExecuteHelp(cmd bottypes.BotCommand) (*bottypes.BotResponse, error) {
	return &bottypes.BotResponse{
		Text: "🤖 *Clario 360 Bot Commands*\n\n" +
			"`/clario status` — Platform health summary\n" +
			"`/clario alerts [severity]` — Recent alerts\n" +
			"`/clario risk` — Current risk score\n" +
			"`/clario investigate <alert-id>` — Alert detail\n" +
			"`/clario ack <alert-id>` — Acknowledge an alert\n" +
			"`/clario assign <alert-id> <user-id|email>` — Assign an alert\n" +
			"`/clario help` — Show this help message",
		DataType:  "text",
		Ephemeral: true,
	}, nil
}
