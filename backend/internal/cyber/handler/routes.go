package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
	cyberhealth "github.com/clario360/platform/internal/cyber/health"
	cybermw "github.com/clario360/platform/internal/cyber/middleware"
	uebahandler "github.com/clario360/platform/internal/cyber/ueba/handler"
	"github.com/clario360/platform/internal/middleware"
)

// RegisterRoutes mounts all cyber service routes on the given router.
// All routes require a valid JWT; tenant isolation is enforced by the Auth middleware.
func RegisterRoutes(
	r chi.Router,
	assetHandler *AssetHandler,
	alertHandler *AlertHandler,
	ruleHandler *RuleHandler,
	threatHandler *ThreatHandler,
	threatFeedHandler *ThreatFeedHandler,
	mitreHandler *MITREHandler,
	ctemHandler *CTEMHandler,
	ctemReportHandler *CTEMReportHandler,
	riskHandler *RiskHandler,
	dashboardHandler *DashboardHandler,
	vulnerabilityHandler *VulnerabilityHandler,
	remediationHandler *RemediationHandler,
	dspmHandler *DSPMHandler,
	uebaHandler *uebahandler.UEBAHandler,
	eventHandler *EventHandler,
	analyticsHandler *AnalyticsHandler,
	jwtMgr *auth.JWTManager,
	rdb *redis.Client,
	extraRoutes ...func(chi.Router),
) {
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

		// ---- Activity ----
		r.Get("/assets/{id}/activity", assetHandler.ListActivity)

		// ---- Relationships ----
		r.Get("/assets/{id}/relationships", assetHandler.ListRelationships)
		r.Post("/assets/{id}/relationships", assetHandler.CreateRelationship)
		r.Delete("/assets/{id}/relationships/{relId}", assetHandler.DeleteRelationship)

		// ---- Vulnerabilities ----
		r.Get("/assets/{id}/vulnerabilities", assetHandler.ListVulnerabilities)
		r.Post("/assets/{id}/vulnerabilities", assetHandler.CreateVulnerability)
		r.Put("/assets/{id}/vulnerabilities/{vid}", assetHandler.UpdateVulnerability)

		// ---- Alerts ----
		r.Get("/alerts/stats", alertHandler.Stats)
		r.Get("/alerts/count", alertHandler.Count)
		r.Get("/alerts/{id}/comments", alertHandler.ListComments)
		r.Get("/alerts/{id}/timeline", alertHandler.ListTimeline)
		r.Get("/alerts/{id}/related", alertHandler.Related)
		r.Get("/alerts/{id}", alertHandler.GetAlert)
		r.Get("/alerts", alertHandler.ListAlerts)
		r.Put("/alerts/{id}/status", alertHandler.UpdateStatus)
		r.Put("/alerts/{id}/false-positive", alertHandler.MarkFalsePositive)
		r.Put("/alerts/{id}/assign", alertHandler.Assign)
		r.Post("/alerts/{id}/escalate", alertHandler.Escalate)
		r.Post("/alerts/{id}/comment", alertHandler.AddComment)
		r.Post("/alerts/{id}/comments", alertHandler.AddComment)
		r.Post("/alerts/{id}/merge", alertHandler.Merge)

		// ---- Detection Rules ----
		r.Get("/rules/stats", ruleHandler.Stats)
		r.Get("/rules/templates", ruleHandler.ListTemplates)
		r.Get("/rules/{id}/performance", ruleHandler.Performance)
		r.Get("/rules/{id}", ruleHandler.GetRule)
		r.Get("/rules", ruleHandler.ListRules)
		r.Post("/rules", ruleHandler.CreateRule)
		r.Put("/rules/{id}", ruleHandler.UpdateRule)
		r.Delete("/rules/{id}", ruleHandler.DeleteRule)
		r.Put("/rules/{id}/toggle", ruleHandler.Toggle)
		r.Post("/rules/{id}/test", ruleHandler.TestRule)
		r.Post("/rules/{id}/feedback", ruleHandler.Feedback)

		// ---- Threats & Indicators ----
		r.Get("/threats/stats", threatHandler.Stats)
		r.Get("/threats/stats/trend", threatHandler.Trend)
		r.Post("/threats", threatHandler.CreateThreat)
		r.Get("/threats/{id}/indicators", threatHandler.ListIndicatorsForThreat)
		r.Post("/threats/{id}/indicators", threatHandler.AddIndicatorToThreat)
		r.Put("/indicators/{indicatorId}/status", threatHandler.UpdateIndicatorStatus)
		r.Get("/threats/{id}/alerts", threatHandler.RelatedAlerts)
		r.Get("/threats/{id}/timeline", threatHandler.Timeline)
		r.Get("/threats/{id}", threatHandler.GetThreat)
		r.Put("/threats/{id}", threatHandler.UpdateThreat)
		r.Delete("/threats/{id}", threatHandler.DeleteThreat)
		r.Get("/threats", threatHandler.ListThreats)
		r.Put("/threats/{id}/status", threatHandler.UpdateStatus)
		r.Post("/indicators/check", threatHandler.CheckIndicators)
		r.Post("/indicators/bulk", threatHandler.BulkImportIndicators)
		r.Get("/indicators/stats", threatHandler.IndicatorStats)
		r.Post("/indicators", threatHandler.CreateIndicator)
		r.Get("/indicators", threatHandler.ListIndicators)
		r.Get("/indicators/{indicatorId}", threatHandler.GetIndicator)
		r.Put("/indicators/{indicatorId}", threatHandler.UpdateIndicator)
		r.Delete("/indicators/{indicatorId}", threatHandler.DeleteIndicator)
		r.Get("/indicators/{indicatorId}/enrichment", threatHandler.IndicatorEnrichment)
		r.Get("/indicators/{indicatorId}/matches", threatHandler.IndicatorMatches)

		// ---- Threat Feeds ----
		if threatFeedHandler != nil {
			r.Get("/threat-feeds", threatFeedHandler.List)
			r.Post("/threat-feeds", threatFeedHandler.Create)
			r.Put("/threat-feeds/{feedId}", threatFeedHandler.Update)
			r.Post("/threat-feeds/{feedId}/sync", threatFeedHandler.Sync)
			r.Get("/threat-feeds/{feedId}/history", threatFeedHandler.History)
		}

		// ---- MITRE ATT&CK ----
		r.Get("/mitre/tactics", mitreHandler.ListTactics)
		r.Get("/mitre/techniques/{id}", mitreHandler.GetTechnique)
		r.Get("/mitre/techniques", mitreHandler.ListTechniques)
		r.Get("/mitre/coverage", mitreHandler.Coverage)

		if vulnerabilityHandler != nil {
			r.Get("/vulnerabilities/stats", vulnerabilityHandler.Stats)
			r.Get("/vulnerabilities/aging", vulnerabilityHandler.Aging)
			r.Get("/vulnerabilities/top-cves", vulnerabilityHandler.TopCVEs)
			r.Get("/vulnerabilities/{id}", vulnerabilityHandler.Get)
			r.Get("/vulnerabilities", vulnerabilityHandler.List)
			r.Put("/vulnerabilities/{id}/status", vulnerabilityHandler.UpdateStatus)
		}

		if riskHandler != nil {
			r.Get("/risk/score", riskHandler.GetScore)
			r.Get("/risk/score/trend", riskHandler.GetTrend)
			r.Get("/risk/score/recalculate", riskHandler.Recalculate)
			r.Get("/risk/heatmap", riskHandler.GetHeatmap)
			r.Get("/risk/top-risks", riskHandler.GetTopRisks)
			r.Get("/risk/recommendations", riskHandler.GetRecommendations)
		}

		if dashboardHandler != nil {
			r.Get("/dashboard", dashboardHandler.GetDashboard)
			r.Get("/dashboard/kpis", dashboardHandler.GetKPIs)
			r.Get("/dashboard/metrics", dashboardHandler.GetMetrics)
			r.Get("/dashboard/alerts-timeline", dashboardHandler.GetAlertsTimeline)
			r.Get("/dashboard/severity-distribution", dashboardHandler.GetSeverityDistribution)
			r.Get("/dashboard/mttr", dashboardHandler.GetMTTR)
			r.Get("/dashboard/analyst-workload", dashboardHandler.GetAnalystWorkload)
			r.Get("/dashboard/top-attacked-assets", dashboardHandler.GetTopAttackedAssets)
			r.Get("/dashboard/mitre-heatmap", dashboardHandler.GetMITREHeatmap)
			r.Get("/dashboard/trends", dashboardHandler.GetTrends)
		}

		if remediationHandler != nil {
			r.Get("/remediation/stats", remediationHandler.Stats)
			r.Post("/remediation", remediationHandler.Create)
			r.Get("/remediation", remediationHandler.List)
			r.Get("/remediation/{id}", remediationHandler.Get)
			r.Put("/remediation/{id}", remediationHandler.Update)
			r.Delete("/remediation/{id}", remediationHandler.Delete)
			r.Post("/remediation/{id}/submit", remediationHandler.Submit)
			r.Post("/remediation/{id}/approve", remediationHandler.Approve)
			r.Post("/remediation/{id}/reject", remediationHandler.Reject)
			r.Post("/remediation/{id}/request-revision", remediationHandler.RequestRevision)
			r.Post("/remediation/{id}/dry-run", remediationHandler.DryRun)
			r.Get("/remediation/{id}/dry-run", remediationHandler.GetDryRun)
			r.Post("/remediation/{id}/execute", remediationHandler.Execute)
			r.Post("/remediation/{id}/verify", remediationHandler.Verify)
			r.Post("/remediation/{id}/rollback", remediationHandler.Rollback)
			r.Post("/remediation/{id}/close", remediationHandler.Close)
			r.Get("/remediation/{id}/audit-trail", remediationHandler.AuditTrail)
		}

		if dspmHandler != nil {
			r.Get("/dspm/data-assets", dspmHandler.ListDataAssets)
			r.Get("/dspm/data-assets/{id}", dspmHandler.GetDataAsset)
			r.Post("/dspm/scan", dspmHandler.TriggerScan)
			r.Get("/dspm/scans", dspmHandler.ListScans)
			r.Get("/dspm/scans/{id}", dspmHandler.GetScan)
			r.Get("/dspm/classification", dspmHandler.Classification)
			r.Get("/dspm/exposure", dspmHandler.Exposure)
			r.Get("/dspm/dependencies", dspmHandler.Dependencies)
			r.Get("/dspm/dashboard", dspmHandler.Dashboard)
			r.Get("/dspm/shadow-copies", dspmHandler.DetectShadowCopies)
		}

		uebahandler.RegisterRoutes(r, uebaHandler)

		// ---- Security Events ----
		if eventHandler != nil {
			r.Get("/events/stats", eventHandler.GetEventStats)
			r.Get("/events/{id}", eventHandler.GetEvent)
			r.Get("/events", eventHandler.ListEvents)
		}

		// ---- Analytics ----
		if analyticsHandler != nil {
			r.Get("/analytics/threat-forecast", analyticsHandler.ThreatForecast)
			r.Get("/analytics/alert-forecast", analyticsHandler.AlertForecast)
			r.Get("/analytics/technique-trends", analyticsHandler.TechniqueTrends)
			r.Get("/analytics/campaigns", analyticsHandler.Campaigns)
			r.Get("/analytics/landscape", analyticsHandler.Landscape)
		}

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

		// ---- Extra sub-module routes (DSPM intelligence, access, remediation) ----
		for _, fn := range extraRoutes {
			fn(r)
		}
	})
}
