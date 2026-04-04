package cti

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all CTI endpoints on the given router under /cti.
// If wsHub is provided, a WebSocket endpoint is also registered.
func RegisterRoutes(r chi.Router, h *Handler, wsHub ...*WSHub) {
	r.Route("/cti", func(r chi.Router) {
		// Reference data (read-only)
		r.Get("/severity-levels", h.ListSeverityLevels)
		r.Get("/categories", h.ListCategories)
		r.Get("/regions", h.ListRegions)
		r.Get("/sectors", h.ListSectors)
		r.Get("/data-sources", h.ListDataSources)

		// Threat events
		r.Route("/events", func(r chi.Router) {
			r.Post("/", h.CreateThreatEvent)
			r.Get("/", h.ListThreatEvents)
			r.Route("/{eventID}", func(r chi.Router) {
				r.Get("/", h.GetThreatEvent)
				r.Put("/", h.UpdateThreatEvent)
				r.Delete("/", h.DeleteThreatEvent)
				r.Post("/false-positive", h.MarkEventFalsePositive)
				r.Post("/resolve", h.ResolveEvent)
				r.Get("/tags", h.GetEventTags)
				r.Post("/tags", h.AddEventTags)
				r.Delete("/tags/{tag}", h.RemoveEventTag)
			})
		})

		// Threat actors
		r.Route("/actors", func(r chi.Router) {
			r.Post("/", h.CreateThreatActor)
			r.Get("/", h.ListThreatActors)
			r.Route("/{actorID}", func(r chi.Router) {
				r.Get("/", h.GetThreatActor)
				r.Put("/", h.UpdateThreatActor)
				r.Delete("/", h.DeleteThreatActor)
			})
		})

		// Campaigns
		r.Route("/campaigns", func(r chi.Router) {
			r.Post("/", h.CreateCampaign)
			r.Get("/", h.ListCampaigns)
			r.Route("/{campaignID}", func(r chi.Router) {
				r.Get("/", h.GetCampaign)
				r.Put("/", h.UpdateCampaign)
				r.Delete("/", h.DeleteCampaign)
				r.Patch("/status", h.UpdateCampaignStatus)
				r.Get("/events", h.ListCampaignEvents)
				r.Post("/events/{eventID}", h.LinkEventToCampaign)
				r.Delete("/events/{eventID}", h.UnlinkEventFromCampaign)
				r.Get("/iocs", h.ListCampaignIOCs)
				r.Post("/iocs", h.CreateCampaignIOC)
				r.Delete("/iocs/{iocID}", h.DeleteCampaignIOC)
			})
		})

		// Brand monitoring
		r.Route("/brands", func(r chi.Router) {
			r.Post("/", h.CreateMonitoredBrand)
			r.Get("/", h.ListMonitoredBrands)
			r.Route("/{brandID}", func(r chi.Router) {
				r.Put("/", h.UpdateMonitoredBrand)
				r.Delete("/", h.DeleteMonitoredBrand)
			})
		})
		r.Route("/brand-abuse", func(r chi.Router) {
			r.Post("/", h.CreateBrandAbuseIncident)
			r.Get("/", h.ListBrandAbuseIncidents)
			r.Route("/{incidentID}", func(r chi.Router) {
				r.Get("/", h.GetBrandAbuseIncident)
				r.Put("/", h.UpdateBrandAbuseIncident)
				r.Patch("/takedown-status", h.UpdateTakedownStatus)
			})
		})

		// Dashboard & analytics
		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/threat-map", h.GetGlobalThreatMap)
			r.Get("/sectors", h.GetSectorThreatOverview)
			r.Get("/executive", h.GetExecutiveDashboard)
		})

		// Admin
		r.Post("/admin/refresh-aggregations", h.RefreshAggregations)

		// WebSocket for real-time CTI events
		if len(wsHub) > 0 && wsHub[0] != nil {
			r.Get("/ws", wsHub[0].HandleWebSocket)
		}
	})
}
