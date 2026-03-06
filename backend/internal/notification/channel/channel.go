package channel

import (
	"context"

	"github.com/clario360/platform/internal/notification/model"
)

// Channel is the interface every delivery channel implements.
type Channel interface {
	Name() string
	Send(ctx context.Context, notif *model.Notification) *ChannelResult
}

// ChannelResult describes the outcome of a delivery attempt.
type ChannelResult struct {
	Success  bool
	Error    error
	Metadata map[string]interface{}
}

// ChannelDelivery describes a pending delivery to a channel.
type ChannelDelivery struct {
	Channel  string
	Deferred bool // true if quiet hours deferred this channel
}

// EmailTemplateRenderer renders email templates for notifications.
// Implemented by service.TemplateService — defined here to avoid import cycles.
type EmailTemplateRenderer interface {
	RenderEmail(notif *model.Notification) (subject string, body string, err error)
}
