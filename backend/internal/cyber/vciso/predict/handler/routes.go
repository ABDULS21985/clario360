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
	Handler    *PredictionHandler
	JWTManager *auth.JWTManager
	Redis      *redis.Client
	Logger     zerolog.Logger
}

func RegisterRoutes(r chi.Router, deps RouteDeps) {
	if deps.Handler == nil || deps.JWTManager == nil {
		return
	}
	r.Route("/api/v1/cyber/vciso/predict", func(r chi.Router) {
		r.Use(middleware.Auth(deps.JWTManager))
		r.Use(middleware.Tenant)
		if deps.Redis != nil {
			r.Use(cybermw.RateLimiter(deps.Redis, 600, deps.Logger))
		}
		r.Get("/forecast", deps.Handler.Forecast)
		r.Get("/assets", deps.Handler.Assets)
		r.Get("/vulnerabilities", deps.Handler.Vulnerabilities)
		r.Get("/techniques", deps.Handler.Techniques)
		r.Get("/insider-threats", deps.Handler.InsiderThreats)
		r.Get("/campaigns", deps.Handler.Campaigns)
		r.Get("/accuracy", deps.Handler.Accuracy)
		r.Post("/retrain/{model_type}", deps.Handler.Retrain)
	})
}
