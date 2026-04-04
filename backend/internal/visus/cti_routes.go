package visus

import "github.com/go-chi/chi/v5"

func RegisterCTIWidgetRoutes(r chi.Router, handler *CTIWidgetHandler) {
	if handler == nil {
		return
	}
	r.Route("/cti", func(r chi.Router) {
		r.Get("/overview", handler.GetCTIOverview)
		r.Get("/threat-map", handler.GetCTIThreatMap)
		r.Get("/sectors", handler.GetCTISectorOverview)
		r.Get("/campaigns", handler.GetCTICampaigns)
		r.Get("/brand-abuse", handler.GetCTIBrandAbuse)
		r.Get("/actors", handler.GetCTIActors)
		r.Get("/risk-score", handler.GetCTIRiskScore)
	})
}
