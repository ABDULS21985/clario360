package channel

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/websocket"
)

// InAppChannel delivers notifications via in-app storage + WebSocket push.
type InAppChannel struct {
	hub    *websocket.Hub
	logger zerolog.Logger
}

// NewInAppChannel creates a new InAppChannel.
func NewInAppChannel(hub *websocket.Hub, logger zerolog.Logger) *InAppChannel {
	return &InAppChannel{hub: hub, logger: logger.With().Str("channel", "in_app").Logger()}
}

// Name returns the channel name.
func (c *InAppChannel) Name() string { return model.ChannelInApp }

// Send pushes a real-time notification to the user via WebSocket.
// The notification is already persisted in DB by the notification service.
// In-app delivery cannot fail — the record is already in DB.
func (c *InAppChannel) Send(_ context.Context, notif *model.Notification) *ChannelResult {
	msg, err := websocket.NewWSMessage(websocket.MsgTypeNotificationNew, notif)
	if err != nil {
		c.logger.Warn().Err(err).Str("notification_id", notif.ID).Msg("failed to marshal ws message")
		return &ChannelResult{Success: true, Metadata: map[string]interface{}{"ws_push": false}}
	}

	sent := c.hub.SendToUser(notif.TenantID, notif.UserID, msg)

	return &ChannelResult{
		Success:  true,
		Metadata: map[string]interface{}{"ws_sessions_sent": sent},
	}
}
