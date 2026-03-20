package handler

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all DSPM Access Intelligence routes on the given router.
// This should be called within the /api/v1/cyber route group (already authed + tenant).
func RegisterRoutes(r chi.Router, h *AccessIntelligenceHandler) {
	if h == nil {
		return
	}
	r.Route("/dspm/access", func(r chi.Router) {
		// Dashboard
		r.Get("/dashboard", h.Dashboard)

		// Identity profiles
		r.Get("/identities", h.ListIdentities)
		r.Get("/identities/{identityId}", h.GetIdentity)
		r.Get("/identities/{identityId}/mappings", h.GetIdentityMappings)
		r.Get("/identities/{identityId}/blast-radius", h.GetBlastRadius)
		r.Get("/identities/{identityId}/recommendations", h.GetRecommendations)

		// Data asset access
		r.Get("/assets/{assetId}/identities", h.GetAssetIdentities)
		r.Get("/assets/{assetId}/audit", h.GetAssetAudit)

		// Access mappings
		r.Get("/mappings", h.ListMappings)
		r.Get("/mappings/overprivileged", h.GetOverprivileged)
		r.Get("/mappings/stale", h.GetStaleAccess)

		// Analysis
		r.Get("/analysis/risk-ranking", h.GetRiskRanking)
		r.Get("/analysis/blast-radius-ranking", h.GetBlastRadiusRanking)
		r.Get("/analysis/escalation-paths", h.GetEscalationPaths)
		r.Get("/analysis/cross-asset", h.GetCrossAsset)

		// Governance
		r.Get("/policies", h.ListPolicies)
		r.Post("/policies", h.CreatePolicy)
		r.Put("/policies/{policyId}", h.UpdatePolicy)
		r.Delete("/policies/{policyId}", h.DeletePolicy)
		r.Get("/policies/violations", h.GetPolicyViolations)

		// Collection
		r.Post("/collect", h.TriggerCollection)
	})
}
