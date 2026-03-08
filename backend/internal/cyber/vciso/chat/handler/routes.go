package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	cybermw "github.com/clario360/platform/internal/cyber/middleware"
	"github.com/clario360/platform/internal/middleware"
)

type RouteDeps struct {
	ChatHandler *ChatHandler
	WSHandler   *WebSocketHandler
	JWTManager  *auth.JWTManager
	Redis       *redis.Client
	Logger      zerolog.Logger
}

func RegisterRoutes(r chi.Router, deps RouteDeps) {
	if deps.ChatHandler != nil && deps.JWTManager != nil {
		r.Route("/api/v1/cyber/vciso", func(r chi.Router) {
			r.Use(middleware.Auth(deps.JWTManager))
			r.Use(middleware.Tenant)
			if deps.Redis != nil {
				r.Use(cybermw.RateLimiter(deps.Redis, 1200, deps.Logger))
			}
			r.Post("/chat", deps.ChatHandler.Chat)
			r.Get("/conversations", deps.ChatHandler.ListConversations)
			r.Get("/conversations/{id}", deps.ChatHandler.GetConversation)
			r.Delete("/conversations/{id}", deps.ChatHandler.DeleteConversation)
			r.Get("/suggestions", deps.ChatHandler.Suggestions)
		})
	}
	if deps.WSHandler != nil {
		r.Get("/ws/v1/cyber/vciso/chat", deps.WSHandler.HandleWebSocket)
	}
}
