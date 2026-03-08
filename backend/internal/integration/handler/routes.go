package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/middleware"
	notifmw "github.com/clario360/platform/internal/notification/middleware"
)

type RouteDependencies struct {
	JWTManager          *auth.JWTManager
	Redis               *redis.Client
	RateLimitPerMinute  int
	Integration         *IntegrationHandler
	Slack               *SlackHandler
	Teams               *TeamsHandler
	Jira                *JiraHandler
	ServiceNow          *ServiceNowHandler
	Webhook             *WebhookHandler
	Logger              zerolog.Logger
}

func RegisterRoutes(r chi.Router, deps RouteDependencies) {
	r.Route("/api/v1/integrations", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(deps.JWTManager))
			r.Use(notifmw.TenantGuard)
			r.Use(notifmw.RateLimiter(deps.Redis, deps.RateLimitPerMinute, deps.Logger))

			r.Get("/", deps.Integration.List)
			r.Post("/", deps.Integration.Create)
			r.Get("/{id}", deps.Integration.Get)
			r.Put("/{id}", deps.Integration.Update)
			r.Delete("/{id}", deps.Integration.Delete)
			r.Post("/{id}/test", deps.Integration.Test)
			r.Get("/{id}/deliveries", deps.Integration.Deliveries)
			r.Post("/{id}/retry-failed", deps.Integration.RetryFailed)
			r.Put("/{id}/status", deps.Integration.UpdateStatus)

			r.Get("/ticket-links", deps.Integration.ListTicketLinks)
			r.Get("/ticket-links/{id}/sync", deps.Integration.SyncTicketLink)

			r.Get("/slack/oauth/start", deps.Slack.OAuthStart)
			r.Get("/jira/oauth/start", deps.Jira.OAuthStart)
			r.Post("/jira/create-ticket", deps.Jira.CreateTicket)
			r.Post("/servicenow/create-incident", deps.ServiceNow.CreateIncident)
		})

		r.Get("/slack/oauth/callback", deps.Slack.OAuthCallback)
		r.Post("/slack/events", deps.Slack.Events)
		r.Post("/slack/commands", deps.Slack.Commands)
		r.Post("/slack/interactions", deps.Slack.Interactions)

		r.Post("/teams/messages", deps.Teams.Messages)

		r.Get("/jira/oauth/callback", deps.Jira.OAuthCallback)
		r.Post("/jira/webhook", deps.Jira.Webhook)

		r.Post("/servicenow/webhook", deps.ServiceNow.Webhook)
		r.MethodFunc(http.MethodPost, "/webhook/test-receiver", deps.Webhook.TestReceiver)
	})
}

