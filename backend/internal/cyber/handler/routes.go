package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
	cyberhealth "github.com/clario360/platform/internal/cyber/health"
	cybermw "github.com/clario360/platform/internal/cyber/middleware"
	"github.com/clario360/platform/internal/middleware"
)

// RegisterRoutes mounts all cyber service routes on the given router.
// All routes require a valid JWT; tenant isolation is enforced by the Auth middleware.
func RegisterRoutes(r chi.Router, assetHandler *AssetHandler, ctemHandler *CTEMHandler, ctemReportHandler *CTEMReportHandler, jwtMgr *auth.JWTManager, rdb *redis.Client) {
	cyberhealth.Register(r)

	r.Route("/api/v1/cyber", func(r chi.Router) {
		r.Use(middleware.Auth(jwtMgr))
		r.Use(middleware.Tenant)
		r.Use(cybermw.RateLimiter(rdb, 1200, assetHandler.logger))

		// ---- Statistics ----
		r.Get("/assets/stats", assetHandler.GetStats)
		r.Get("/assets/count", assetHandler.GetCount)

		// ---- Scan endpoints (listed before /{id} to avoid ambiguity) ----
		r.Post("/assets/scan", assetHandler.TriggerScan)
		r.Get("/assets/scans", assetHandler.ListScans)
		r.Get("/assets/scans/{id}", assetHandler.GetScan)
		r.Post("/assets/scans/{id}/cancel", assetHandler.CancelScan)

		// ---- Bulk operations ----
		r.Post("/assets/bulk", assetHandler.BulkCreate)
		r.Put("/assets/bulk/tags", assetHandler.BulkUpdateTags)
		r.Delete("/assets/bulk", assetHandler.BulkDelete)

		// ---- Asset CRUD ----
		r.Post("/assets", assetHandler.CreateAsset)
		r.Get("/assets", assetHandler.ListAssets)
		r.Get("/assets/{id}", assetHandler.GetAsset)
		r.Put("/assets/{id}", assetHandler.UpdateAsset)
		r.Delete("/assets/{id}", assetHandler.DeleteAsset)
		r.Patch("/assets/{id}/tags", assetHandler.PatchTags)

		// ---- Relationships ----
		r.Get("/assets/{id}/relationships", assetHandler.ListRelationships)
		r.Post("/assets/{id}/relationships", assetHandler.CreateRelationship)
		r.Delete("/assets/{id}/relationships/{relId}", assetHandler.DeleteRelationship)

		// ---- Vulnerabilities ----
		r.Get("/assets/{id}/vulnerabilities", assetHandler.ListVulnerabilities)
		r.Post("/assets/{id}/vulnerabilities", assetHandler.CreateVulnerability)
		r.Put("/assets/{id}/vulnerabilities/{vid}", assetHandler.UpdateVulnerability)

		if ctemHandler != nil && ctemReportHandler != nil {
			r.Route("/ctem", func(r chi.Router) {
				r.Post("/assessments", ctemHandler.CreateAssessment)
				r.Get("/assessments", ctemHandler.ListAssessments)
				r.Get("/dashboard", ctemHandler.Dashboard)
				r.Get("/exposure-score", ctemHandler.GetExposureScore)
				r.Get("/exposure-score/history", ctemHandler.GetExposureScoreHistory)
				r.Post("/exposure-score/calculate", ctemHandler.ForceCalculateExposureScore)

				r.Get("/findings/{findingId}", ctemHandler.GetFinding)
				r.Put("/findings/{findingId}/status", ctemHandler.UpdateFindingStatus)

				r.Get("/remediation-groups/{groupId}", ctemHandler.GetRemediationGroup)
				r.Put("/remediation-groups/{groupId}/status", ctemHandler.UpdateRemediationGroupStatus)
				r.Post("/remediation-groups/{groupId}/execute", ctemHandler.ExecuteRemediationGroup)

				r.Get("/assessments/{id}", ctemHandler.GetAssessment)
				r.Put("/assessments/{id}", ctemHandler.UpdateAssessment)
				r.Post("/assessments/{id}/start", ctemHandler.StartAssessment)
				r.Post("/assessments/{id}/cancel", ctemHandler.CancelAssessment)
				r.Delete("/assessments/{id}", ctemHandler.DeleteAssessment)

				r.Get("/assessments/{id}/scope", ctemHandler.GetScope)
				r.Get("/assessments/{id}/discovery", ctemHandler.GetDiscovery)
				r.Get("/assessments/{id}/priorities", ctemHandler.GetPriorities)
				r.Post("/assessments/{id}/validate", ctemHandler.ValidateAssessment)
				r.Get("/assessments/{id}/validation", ctemHandler.GetValidation)
				r.Post("/assessments/{id}/mobilize", ctemHandler.MobilizeAssessment)
				r.Get("/assessments/{id}/mobilization", ctemHandler.GetMobilization)
				r.Get("/assessments/{id}/findings", ctemHandler.ListFindings)
				r.Get("/assessments/{id}/remediation-groups", ctemHandler.ListRemediationGroups)
				r.Get("/assessments/{id}/report", ctemReportHandler.GetReport)
				r.Get("/assessments/{id}/report/executive", ctemReportHandler.GetExecutiveSummary)
				r.Post("/assessments/{id}/report/export", ctemReportHandler.ExportReport)
				r.Get("/assessments/{id}/compare/{otherId}", ctemHandler.CompareAssessments)
			})
		}
	})
}
