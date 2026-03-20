package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/clario360/platform/internal/events"
)

type AuditActor struct {
	UserID    string
	UserEmail string
	IPAddress string
	UserAgent string
}

func publishIntegrationEvent(ctx context.Context, producer *events.Producer, eventType, tenantID, userID string, data any) {
	if producer == nil || tenantID == "" {
		return
	}
	event, err := events.NewEvent(eventType, "notification-service", tenantID, data)
	if err != nil {
		return
	}
	event.UserID = userID
	_ = producer.Publish(ctx, events.Topics.IntegrationEvents, event)
}

func publishIntegrationAudit(ctx context.Context, producer *events.Producer, eventType, tenantID string, actor *AuditActor, data any) {
	if producer == nil || tenantID == "" {
		return
	}
	event, err := events.NewEvent(eventType, "notification-service", tenantID, data)
	if err != nil {
		return
	}
	if actor != nil {
		event.UserID = actor.UserID
		if event.Metadata == nil {
			event.Metadata = map[string]string{}
		}
		if actor.UserEmail != "" {
			event.Metadata["user_email"] = actor.UserEmail
		}
		if actor.IPAddress != "" {
			event.Metadata["ip_address"] = actor.IPAddress
		}
		if actor.UserAgent != "" {
			event.Metadata["user_agent"] = actor.UserAgent
		}
	}
	_ = producer.Publish(ctx, events.Topics.AuditEvents, event)
}

type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("http status %d: %s", e.StatusCode, e.Body)
}

func statusError(code int, body string) error {
	return &HTTPStatusError{StatusCode: code, Body: body}
}

func isHTTPStatus(err error, codes ...int) bool {
	httpErr, ok := err.(*HTTPStatusError)
	if !ok {
		return false
	}
	for _, code := range codes {
		if httpErr.StatusCode == code {
			return true
		}
	}
	return false
}

func successfulResponse(code int) bool {
	return code >= http.StatusOK && code < http.StatusMultipleChoices
}
