package handler

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all DSPM Advanced Intelligence endpoints on the given router.
func RegisterRoutes(r chi.Router, h *IntelligenceHandler) {
	// Classification
	r.Get("/dspm/classification/enhanced", h.EnhancedClassification)
	r.Post("/dspm/classification/custom-rules", h.CreateCustomRule)
	r.Get("/dspm/classification/history/{assetId}", h.ClassificationHistory)

	// Lineage
	r.Get("/dspm/lineage/graph", h.GetLineageGraph)
	r.Get("/dspm/lineage/upstream/{assetId}", h.GetUpstream)
	r.Get("/dspm/lineage/downstream/{assetId}", h.GetDownstream)
	r.Get("/dspm/lineage/impact/{assetId}", h.GetImpactAnalysis)
	r.Get("/dspm/lineage/pii-flow", h.GetPIIFlow)

	// AI Security
	r.Get("/dspm/ai/usage", h.ListAIUsage)
	r.Get("/dspm/ai/usage/{assetId}", h.GetAIUsageByAsset)
	r.Get("/dspm/ai/models/{modelSlug}/data", h.GetModelDataGovernance)
	r.Get("/dspm/ai/risk-ranking", h.GetAIRiskRanking)
	r.Get("/dspm/ai/dashboard", h.GetAIDashboard)

	// Financial
	r.Get("/dspm/financial/impact", h.GetPortfolioRisk)
	r.Get("/dspm/financial/impact/{assetId}", h.GetAssetFinancialImpact)
	r.Get("/dspm/financial/top-risks", h.GetTopFinancialRisks)

	// Compliance
	r.Get("/dspm/compliance/posture", h.GetCompliancePosture)
	r.Get("/dspm/compliance/posture/{framework}", h.GetFrameworkPosture)
	r.Get("/dspm/compliance/gaps", h.GetComplianceGaps)
	r.Get("/dspm/compliance/residency", h.GetResidencyAnalysis)
	r.Post("/dspm/compliance/audit-report/{framework}", h.GenerateAuditReport)

	// Proliferation
	r.Get("/dspm/proliferation/overview", h.GetProliferationOverview)
	r.Get("/dspm/proliferation/{assetId}", h.GetAssetProliferation)
}
