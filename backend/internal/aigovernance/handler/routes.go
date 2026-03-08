package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func RegisterRoutes(r chi.Router, services Services, logger zerolog.Logger) {
	registry := NewRegistryHandler(services, logger)
	predictions := NewPredictionHandler(services, logger)
	explanations := NewExplanationHandler(services, logger)
	shadow := NewShadowHandler(services, logger)
	lifecycle := NewLifecycleHandler(services, logger)
	drift := NewDriftHandler(services, logger)
	dashboard := NewDashboardHandler(services, logger)

	r.Route("/ai", func(r chi.Router) {
		r.Route("/models", func(r chi.Router) {
			r.Post("/", registry.RegisterModel)
			r.Get("/", registry.ListModels)
			r.Get("/{id}", registry.GetModel)
			r.Put("/{id}", registry.UpdateModel)
			r.Post("/{id}/versions", registry.CreateVersion)
			r.Get("/{id}/versions", registry.ListVersions)
			r.Get("/{id}/versions/{vid}", registry.GetVersion)
			r.Post("/{id}/versions/{vid}/promote", lifecycle.Promote)
			r.Post("/{id}/versions/{vid}/retire", lifecycle.Retire)
			r.Post("/{id}/rollback", lifecycle.Rollback)
			r.Get("/{id}/lifecycle-history", lifecycle.History)
			r.Post("/{id}/shadow/start", shadow.Start)
			r.Post("/{id}/shadow/stop", shadow.Stop)
			r.Get("/{id}/shadow/comparison", shadow.LatestComparison)
			r.Get("/{id}/shadow/comparison/history", shadow.History)
			r.Get("/{id}/shadow/divergences", shadow.Divergences)
			r.Get("/{id}/drift", drift.Latest)
			r.Get("/{id}/drift/history", drift.History)
			r.Get("/{id}/performance", drift.Performance)
		})
		r.Mount("/predictions", predictions.Routes())
		r.Mount("/explanations", explanations.Routes())
		r.Mount("/dashboard", dashboard.Routes())
	})
}
