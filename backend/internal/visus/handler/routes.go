package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
	sharedmw "github.com/clario360/platform/internal/middleware"
	visusmw "github.com/clario360/platform/internal/visus/middleware"
)

type RouteDependencies struct {
	Dashboard       *DashboardHandler
	Widget          *WidgetHandler
	KPI             *KPIHandler
	Alert           *AlertHandler
	Report          *ReportHandler
	Executive       *ExecutiveHandler
	RegisterExtra   func(chi.Router)
	JWTManager      *auth.JWTManager
	Redis           *redis.Client
	RateLimitPerMin int
}

func RegisterRoutes(r chi.Router, deps RouteDependencies) {
	r.Route("/api/v1/visus", func(r chi.Router) {
		r.Use(sharedmw.Auth(deps.JWTManager))
		r.Use(visusmw.TenantGuard)
		r.Use(visusmw.RateLimiter(deps.Redis, deps.RateLimitPerMin))

		r.Post("/dashboards", deps.Dashboard.Create)
		r.Get("/dashboards", deps.Dashboard.List)
		r.Get("/dashboards/{id}", deps.Dashboard.Get)
		r.Put("/dashboards/{id}", deps.Dashboard.Update)
		r.Delete("/dashboards/{id}", deps.Dashboard.Delete)
		r.Post("/dashboards/{id}/duplicate", deps.Dashboard.Duplicate)
		r.Put("/dashboards/{id}/share", deps.Dashboard.Share)

		r.Post("/dashboards/{id}/widgets", deps.Widget.Create)
		r.Get("/dashboards/{id}/widgets", deps.Widget.List)
		r.Put("/dashboards/{id}/widgets/{wid}", deps.Widget.Update)
		r.Delete("/dashboards/{id}/widgets/{wid}", deps.Widget.Delete)
		r.Get("/dashboards/{id}/widgets/{wid}/data", deps.Widget.Data)
		r.Put("/dashboards/{id}/widgets/layout", deps.Widget.UpdateLayout)
		r.Get("/widgets/stats", deps.Widget.Stats)
		r.Get("/widgets/types", deps.Widget.Types)

		r.Post("/kpis", deps.KPI.Create)
		r.Get("/kpis", deps.KPI.List)
		r.Get("/kpis/summary", deps.KPI.Summary)
		r.Post("/kpis/snapshot", deps.KPI.TriggerSnapshot)
		r.Get("/kpis/{id}", deps.KPI.Get)
		r.Put("/kpis/{id}", deps.KPI.Update)
		r.Delete("/kpis/{id}", deps.KPI.Delete)
		r.Get("/kpis/{id}/history", deps.KPI.History)

		r.Get("/alerts", deps.Alert.List)
		r.Get("/alerts/count", deps.Alert.Count)
		r.Get("/alerts/stats", deps.Alert.Stats)
		r.Get("/alerts/{id}", deps.Alert.Get)
		r.Put("/alerts/{id}/status", deps.Alert.UpdateStatus)

		r.Post("/reports", deps.Report.Create)
		r.Get("/reports", deps.Report.List)
		r.Get("/reports/{id}", deps.Report.Get)
		r.Put("/reports/{id}", deps.Report.Update)
		r.Delete("/reports/{id}", deps.Report.Delete)
		r.Post("/reports/{id}/generate", deps.Report.Generate)
		r.Get("/reports/{id}/snapshots", deps.Report.Snapshots)
		r.Get("/reports/{id}/snapshots/latest", deps.Report.LatestSnapshot)
		r.Get("/reports/{id}/snapshots/{snapId}", deps.Report.Snapshot)

		r.Get("/executive", deps.Executive.View)
		r.Get("/executive/summary", deps.Executive.Summary)
		r.Get("/executive/health", deps.Executive.Health)

		if deps.RegisterExtra != nil {
			deps.RegisterExtra(r)
		}
	})
}
