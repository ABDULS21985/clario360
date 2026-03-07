package health

import (
	"github.com/go-chi/chi/v5"

	obshealth "github.com/clario360/platform/internal/observability/health"
)

func Register(r chi.Router, checker *obshealth.CompositeHealthChecker, serviceName, version string) {
	if r == nil || checker == nil {
		return
	}
	handler := obshealth.NewHandler(checker, serviceName, version)
	r.Get("/healthz", handler.Healthz())
	r.Get("/readyz", handler.Readyz())
	r.Get("/health", handler.Health())
}
