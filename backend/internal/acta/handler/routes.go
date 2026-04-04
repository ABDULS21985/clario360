package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/acta/middleware"
	"github.com/clario360/platform/internal/auth"
	sharedmw "github.com/clario360/platform/internal/middleware"
)

type RouteDependencies struct {
	Committee       *CommitteeHandler
	Meeting         *MeetingHandler
	Agenda          *AgendaHandler
	Minutes         *MinutesHandler
	ActionItem      *ActionItemHandler
	Compliance      *ComplianceHandler
	Dashboard       *DashboardHandler
	JWTManager      *auth.JWTManager
	Redis           *redis.Client
	RateLimitPerMin int
}

func RegisterRoutes(r chi.Router, deps RouteDependencies) {
	r.Route("/api/v1/acta", func(r chi.Router) {
		r.Use(sharedmw.Auth(deps.JWTManager))
		r.Use(middleware.TenantGuard)
		r.Use(middleware.RateLimiter(deps.Redis, deps.RateLimitPerMin))

		r.Post("/committees", deps.Committee.Create)
		r.Get("/committees", deps.Committee.List)
		r.Get("/committees/{id}", deps.Committee.Get)
		r.Put("/committees/{id}", deps.Committee.Update)
		r.Delete("/committees/{id}", deps.Committee.Delete)
		r.Post("/committees/{id}/members", deps.Committee.AddMember)
		r.Put("/committees/{id}/members/{userId}", deps.Committee.UpdateMember)
		r.Delete("/committees/{id}/members/{userId}", deps.Committee.RemoveMember)

		r.Get("/meetings/upcoming", deps.Meeting.Upcoming)
		r.Get("/meetings/calendar", deps.Meeting.Calendar)
		r.Post("/meetings", deps.Meeting.Create)
		r.Get("/meetings", deps.Meeting.List)
		r.Get("/meetings/{id}", deps.Meeting.Get)
		r.Put("/meetings/{id}", deps.Meeting.Update)
		r.Delete("/meetings/{id}", deps.Meeting.Delete)
		r.Post("/meetings/{id}/start", deps.Meeting.Start)
		r.Post("/meetings/{id}/end", deps.Meeting.End)
		r.Post("/meetings/{id}/postpone", deps.Meeting.Postpone)
		r.Get("/meetings/{id}/attendance", deps.Meeting.GetAttendance)
		r.Post("/meetings/{id}/attendance", deps.Meeting.RecordAttendance)
		r.Post("/meetings/{id}/attendance/bulk", deps.Meeting.BulkRecordAttendance)
		r.Post("/meetings/{id}/attachments", deps.Meeting.UploadAttachment)
		r.Get("/meetings/{id}/attachments", deps.Meeting.ListAttachments)
		r.Delete("/meetings/{id}/attachments/{fileId}", deps.Meeting.DeleteAttachment)

		r.Post("/meetings/{id}/agenda", deps.Agenda.Create)
		r.Get("/meetings/{id}/agenda", deps.Agenda.List)
		r.Put("/meetings/{id}/agenda/reorder", deps.Agenda.Reorder)
		r.Put("/meetings/{id}/agenda/{itemId}", deps.Agenda.Update)
		r.Delete("/meetings/{id}/agenda/{itemId}", deps.Agenda.Delete)
		r.Put("/meetings/{id}/agenda/{itemId}/notes", deps.Agenda.UpdateNotes)
		r.Post("/meetings/{id}/agenda/{itemId}/vote", deps.Agenda.Vote)

		r.Post("/meetings/{id}/minutes", deps.Minutes.Create)
		r.Get("/meetings/{id}/minutes", deps.Minutes.GetLatest)
		r.Get("/meetings/{id}/minutes/versions", deps.Minutes.ListVersions)
		r.Post("/meetings/{id}/minutes/generate", deps.Minutes.Generate)
		r.Put("/meetings/{id}/minutes", deps.Minutes.Update)
		r.Post("/meetings/{id}/minutes/submit", deps.Minutes.Submit)
		r.Post("/meetings/{id}/minutes/request-revision", deps.Minutes.RequestRevision)
		r.Post("/meetings/{id}/minutes/approve", deps.Minutes.Approve)
		r.Post("/meetings/{id}/minutes/publish", deps.Minutes.Publish)

		r.Get("/action-items/overdue", deps.ActionItem.Overdue)
		r.Get("/action-items/my", deps.ActionItem.My)
		r.Get("/action-items/stats", deps.ActionItem.Stats)
		r.Post("/action-items", deps.ActionItem.Create)
		r.Get("/action-items", deps.ActionItem.List)
		r.Get("/action-items/{id}", deps.ActionItem.Get)
		r.Put("/action-items/{id}", deps.ActionItem.Update)
		r.Put("/action-items/{id}/status", deps.ActionItem.UpdateStatus)
		r.Post("/action-items/{id}/extend", deps.ActionItem.Extend)

		r.Get("/compliance/run", deps.Compliance.Run)
		r.Get("/compliance/results", deps.Compliance.Results)
		r.Get("/compliance/report", deps.Compliance.Report)
		r.Get("/compliance/score", deps.Compliance.Score)

		r.Get("/dashboard", deps.Dashboard.Get)
	})
}
