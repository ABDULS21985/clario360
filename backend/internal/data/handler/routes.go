package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
	datamw "github.com/clario360/platform/internal/data/middleware"
	"github.com/clario360/platform/internal/middleware"
)

func RegisterRoutes(
	r chi.Router,
	sourceHandler *SourceHandler,
	modelHandler *ModelHandler,
	jwtMgr *auth.JWTManager,
	rdb *redis.Client,
) {
	r.Route("/api/v1/data", func(r chi.Router) {
		r.Use(middleware.Auth(jwtMgr))
		r.Use(middleware.Tenant)
		r.Use(datamw.RateLimiter(rdb))

		r.Get("/sources/stats", sourceHandler.GetAggregateStats)
		r.Post("/sources", sourceHandler.Create)
		r.Get("/sources", sourceHandler.List)
		r.Get("/sources/{id}", sourceHandler.Get)
		r.Put("/sources/{id}", sourceHandler.Update)
		r.Delete("/sources/{id}", sourceHandler.Delete)
		r.Patch("/sources/{id}/status", sourceHandler.ChangeStatus)
		r.Post("/sources/{id}/test", sourceHandler.TestConnection)
		r.Post("/sources/{id}/discover", sourceHandler.Discover)
		r.Get("/sources/{id}/schema", sourceHandler.GetSchema)
		r.Post("/sources/{id}/sync", sourceHandler.TriggerSync)
		r.Get("/sources/{id}/sync-history", sourceHandler.ListSyncHistory)
		r.Get("/sources/{id}/stats", sourceHandler.GetStats)

		r.Post("/models", modelHandler.Create)
		r.Get("/models", modelHandler.List)
		r.Post("/models/derive", modelHandler.Derive)
		r.Get("/models/{id}", modelHandler.Get)
		r.Put("/models/{id}", modelHandler.Update)
		r.Delete("/models/{id}", modelHandler.Delete)
		r.Post("/models/{id}/validate", modelHandler.Validate)
		r.Get("/models/{id}/versions", modelHandler.Versions)
		r.Get("/models/{id}/lineage", modelHandler.Lineage)
	})
}
