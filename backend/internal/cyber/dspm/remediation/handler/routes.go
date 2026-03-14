package handler

import (
	"time"

	"github.com/go-chi/chi/v5"
)

// timeNow is a package-level function for testability.
var timeNow = func() time.Time { return time.Now().UTC() }

// RegisterRoutes mounts all DSPM remediation, policy, and exception routes onto the router.
// The router should already be scoped under /api/v1/cyber with auth middleware applied.
func RegisterRoutes(r chi.Router, h *DSPMRemediationHandler) {
	r.Route("/dspm/remediations", func(r chi.Router) {
		r.Get("/", h.ListRemediations)
		r.Get("/stats", h.GetStats)
		r.Get("/dashboard", h.GetDashboard)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetRemediation)
			r.Post("/execute", h.ExecuteStep)
			r.Post("/approve", h.ApproveRemediation)
			r.Post("/cancel", h.CancelRemediation)
			r.Post("/rollback", h.RollbackRemediation)
			r.Put("/assign", h.AssignRemediation)
			r.Get("/history", h.GetHistory)
		})
	})

	r.Route("/dspm/policies", func(r chi.Router) {
		r.Get("/", h.ListPolicies)
		r.Post("/", h.CreatePolicy)
		r.Get("/violations", h.GetViolations)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetPolicy)
			r.Put("/", h.UpdatePolicy)
			r.Delete("/", h.DeletePolicy)
			r.Post("/dry-run", h.DryRunPolicy)
			r.Post("/evaluate", h.EvaluatePolicy)
		})
	})

	r.Route("/dspm/exceptions", func(r chi.Router) {
		r.Get("/", h.ListExceptions)
		r.Post("/", h.CreateException)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetException)
			r.Post("/approve", h.ApproveException)
			r.Post("/reject", h.RejectException)
			r.Post("/review", h.ReviewException)
		})
	})
}
