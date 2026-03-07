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
func RegisterRoutes(r chi.Router, assetHandler *AssetHandler, jwtMgr *auth.JWTManager, rdb *redis.Client) {
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
	})
}
