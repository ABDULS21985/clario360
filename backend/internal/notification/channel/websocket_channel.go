package channel

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/websocket"
)

// WebSocketChannel pushes real-time notifications to connected WebSocket clients.
type WebSocketChannel struct {
	hub    *websocket.Hub
	logger zerolog.Logger
}

// NewWebSocketChannel creates a new WebSocketChannel.
func NewWebSocketChannel(hub *websocket.Hub, logger zerolog.Logger) *WebSocketChannel {
	return &WebSocketChannel{hub: hub, logger: logger.With().Str("channel", "websocket").Logger()}
}

// Name returns the channel name.
func (c *WebSocketChannel) Name() string { return model.ChannelWebSocket }

// Send pushes a notification message to all connected sessions for the user.
// WebSocket delivery is best-effort — if the user is not connected, it's not a failure.
func (c *WebSocketChannel) Send(_ context.Context, notif *model.Notification) *ChannelResult {
	msg, err := websocket.NewWSMessage(websocket.MsgTypeNotificationNew, notif)
	if err != nil {
		c.logger.Warn().Err(err).Str("notification_id", notif.ID).Msg("failed to marshal ws message")
		return &ChannelResult{Success: true, Metadata: map[string]interface{}{"ws_push": false}}
	}

	sent := c.hub.SendToUser(notif.TenantID, notif.UserID, msg)

	return &ChannelResult{
		Success:  true,
		Metadata: map[string]interface{}{"sessions_sent": sent},
	}
}
