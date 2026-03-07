package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/auth"
	lexmw "github.com/clario360/platform/internal/lex/middleware"
	sharedmw "github.com/clario360/platform/internal/middleware"
)

type RouteDependencies struct {
	Contract        *ContractHandler
	Clause          *ClauseHandler
	Document        *DocumentHandler
	Compliance      *ComplianceHandler
	Dashboard       *DashboardHandler
	JWTManager      *auth.JWTManager
	Redis           *redis.Client
	RateLimitPerMin int
}

func RegisterRoutes(r chi.Router, deps RouteDependencies) {
	r.Route("/api/v1/lex", func(r chi.Router) {
		r.Use(sharedmw.Auth(deps.JWTManager))
		r.Use(lexmw.TenantGuard)
		r.Use(lexmw.RateLimiter(deps.Redis, deps.RateLimitPerMin))

		r.Get("/contracts/expiring", deps.Contract.Expiring)
		r.Get("/contracts/stats", deps.Contract.Stats)
		r.Get("/contracts/search", deps.Contract.Search)
		r.Post("/contracts", deps.Contract.Create)
		r.Get("/contracts", deps.Contract.List)
		r.Get("/contracts/{id}/analysis", deps.Contract.Analysis)
		r.Post("/contracts/{id}/upload", deps.Contract.UploadDocument)
		r.Post("/contracts/{id}/analyze", deps.Contract.Analyze)
		r.Put("/contracts/{id}/status", deps.Contract.UpdateStatus)
		r.Get("/contracts/{id}/versions", deps.Contract.Versions)
		r.Post("/contracts/{id}/renew", deps.Contract.Renew)
		r.Post("/contracts/{id}/review", deps.Contract.StartReview)
		r.Get("/contracts/{id}/clauses/risks", deps.Clause.RiskSummary)
		r.Get("/contracts/{id}/clauses/{clauseId}", deps.Clause.Get)
		r.Put("/contracts/{id}/clauses/{clauseId}/review", deps.Clause.Review)
		r.Get("/contracts/{id}/clauses", deps.Clause.List)
		r.Get("/contracts/{id}", deps.Contract.Get)
		r.Put("/contracts/{id}", deps.Contract.Update)
		r.Delete("/contracts/{id}", deps.Contract.Delete)

		r.Post("/documents", deps.Document.Create)
		r.Get("/documents", deps.Document.List)
		r.Post("/documents/{id}/upload", deps.Document.UploadVersion)
		r.Get("/documents/{id}/versions", deps.Document.Versions)
		r.Get("/documents/{id}", deps.Document.Get)
		r.Put("/documents/{id}", deps.Document.Update)
		r.Delete("/documents/{id}", deps.Document.Delete)

		r.Get("/compliance/rules", deps.Compliance.ListRules)
		r.Post("/compliance/rules", deps.Compliance.CreateRule)
		r.Put("/compliance/rules/{id}", deps.Compliance.UpdateRule)
		r.Delete("/compliance/rules/{id}", deps.Compliance.DeleteRule)
		r.Post("/compliance/run", deps.Compliance.Run)
		r.Get("/compliance/alerts/{id}", deps.Compliance.GetAlert)
		r.Put("/compliance/alerts/{id}/status", deps.Compliance.UpdateAlertStatus)
		r.Get("/compliance/alerts", deps.Compliance.ListAlerts)
		r.Get("/compliance/dashboard", deps.Compliance.Dashboard)
		r.Get("/compliance/score", deps.Compliance.Score)

		r.Get("/workflows", deps.Contract.ListWorkflows)
		r.Get("/dashboard", deps.Dashboard.Get)
	})
}
