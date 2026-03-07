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
	pipelineHandler *PipelineHandler,
	qualityHandler *QualityHandler,
	contradictionHandler *ContradictionHandler,
	lineageHandler *LineageHandler,
	darkDataHandler *DarkDataHandler,
	analyticsHandler *AnalyticsHandler,
	dashboardHandler *DashboardHandler,
	jwtMgr *auth.JWTManager,
	rdb *redis.Client,
) {
	r.Route("/api/v1/data", func(r chi.Router) {
		r.Use(middleware.Auth(jwtMgr))
		r.Use(middleware.Tenant)
		r.Use(datamw.RateLimiter(rdb))

		r.Get("/sources/stats", sourceHandler.GetAggregateStats)
		r.Post("/sources/test-config", sourceHandler.TestConfig)
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

		r.Get("/pipelines/stats", pipelineHandler.Stats)
		r.Get("/pipelines/active", pipelineHandler.Active)
		r.Post("/pipelines", pipelineHandler.Create)
		r.Get("/pipelines", pipelineHandler.List)
		r.Get("/pipelines/{id}", pipelineHandler.Get)
		r.Put("/pipelines/{id}", pipelineHandler.Update)
		r.Delete("/pipelines/{id}", pipelineHandler.Delete)
		r.Post("/pipelines/{id}/run", pipelineHandler.Run)
		r.Post("/pipelines/{id}/pause", pipelineHandler.Pause)
		r.Post("/pipelines/{id}/resume", pipelineHandler.Resume)
		r.Get("/pipelines/{id}/runs", pipelineHandler.ListRuns)
		r.Get("/pipelines/{id}/runs/{runId}", pipelineHandler.GetRun)
		r.Get("/pipelines/{id}/runs/{runId}/logs", pipelineHandler.GetRunLogs)

		r.Get("/quality/score/trend", qualityHandler.Trend)
		r.Get("/quality/score", qualityHandler.Score)
		r.Get("/quality/dashboard", qualityHandler.Dashboard)
		r.Post("/quality/rules", qualityHandler.CreateRule)
		r.Get("/quality/rules", qualityHandler.ListRules)
		r.Get("/quality/rules/{id}", qualityHandler.GetRule)
		r.Put("/quality/rules/{id}", qualityHandler.UpdateRule)
		r.Delete("/quality/rules/{id}", qualityHandler.DeleteRule)
		r.Post("/quality/rules/{id}/run", qualityHandler.RunRule)
		r.Get("/quality/results", qualityHandler.ListResults)
		r.Get("/quality/results/{id}", qualityHandler.GetResult)

		r.Post("/contradictions/scan", contradictionHandler.Scan)
		r.Get("/contradictions/scans", contradictionHandler.ListScans)
		r.Get("/contradictions/scans/{id}", contradictionHandler.GetScan)
		r.Get("/contradictions/stats", contradictionHandler.Stats)
		r.Get("/contradictions/dashboard", contradictionHandler.Dashboard)
		r.Get("/contradictions", contradictionHandler.List)
		r.Get("/contradictions/{id}", contradictionHandler.Get)
		r.Put("/contradictions/{id}/status", contradictionHandler.UpdateStatus)
		r.Post("/contradictions/{id}/resolve", contradictionHandler.Resolve)

		r.Get("/lineage/graph", lineageHandler.FullGraph)
		r.Get("/lineage/graph/{entityType}/{entityId}", lineageHandler.EntityGraph)
		r.Get("/lineage/upstream/{entityType}/{entityId}", lineageHandler.Upstream)
		r.Get("/lineage/downstream/{entityType}/{entityId}", lineageHandler.Downstream)
		r.Get("/lineage/impact/{entityType}/{entityId}", lineageHandler.Impact)
		r.Post("/lineage/record", lineageHandler.Record)
		r.Delete("/lineage/edges/{id}", lineageHandler.DeleteEdge)
		r.Get("/lineage/search", lineageHandler.Search)
		r.Get("/lineage/stats", lineageHandler.Stats)

		r.Post("/dark-data/scan", darkDataHandler.Scan)
		r.Get("/dark-data/scans", darkDataHandler.ListScans)
		r.Get("/dark-data/scans/{id}", darkDataHandler.GetScan)
		r.Get("/dark-data/stats", darkDataHandler.Stats)
		r.Get("/dark-data/dashboard", darkDataHandler.Dashboard)
		r.Get("/dark-data", darkDataHandler.ListAssets)
		r.Get("/dark-data/{id}", darkDataHandler.GetAsset)
		r.Put("/dark-data/{id}/status", darkDataHandler.UpdateStatus)
		r.Post("/dark-data/{id}/govern", darkDataHandler.Govern)

		r.Post("/analytics/query", analyticsHandler.Execute)
		r.Post("/analytics/explore/{modelId}", analyticsHandler.Explore)
		r.Post("/analytics/explain", analyticsHandler.Explain)
		r.Get("/analytics/saved", analyticsHandler.ListSaved)
		r.Post("/analytics/saved", analyticsHandler.CreateSaved)
		r.Get("/analytics/saved/{id}", analyticsHandler.GetSaved)
		r.Put("/analytics/saved/{id}", analyticsHandler.UpdateSaved)
		r.Delete("/analytics/saved/{id}", analyticsHandler.DeleteSaved)
		r.Post("/analytics/saved/{id}/run", analyticsHandler.RunSaved)
		r.Get("/analytics/audit", analyticsHandler.Audit)

		r.Get("/dashboard", dashboardHandler.Get)
	})
}
