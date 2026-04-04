package handler

import (
	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/middleware"
)

func RegisterRoutes(r chi.Router, handler *UEBAHandler) {
	if handler == nil {
		return
	}
	r.Route("/ueba", func(r chi.Router) {
		r.Get("/profiles", handler.ListProfiles)
		r.Get("/profiles/{entityId}", handler.GetProfile)
		r.Get("/profiles/{entityId}/timeline", handler.GetTimeline)
		r.Get("/profiles/{entityId}/heatmap", handler.GetHeatmap)
		r.Put("/profiles/{entityId}/status", handler.UpdateProfileStatus)

		r.Get("/alerts", handler.ListAlerts)
		r.Put("/alerts/bulk/status", handler.BulkUpdateAlertStatus)
		r.Get("/alerts/{id}", handler.GetAlert)
		r.Put("/alerts/{id}/status", handler.UpdateAlertStatus)
		r.Post("/alerts/{id}/false-positive", handler.MarkFalsePositive)

		r.Get("/dashboard", handler.GetDashboard)
		r.Get("/risk-ranking", handler.GetRiskRanking)

		r.Get("/config", handler.GetConfig)
		r.With(middleware.RequirePermission(auth.PermAdminAll)).Put("/config", handler.UpdateConfig)
	})
}
