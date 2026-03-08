package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
)

type DashboardHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewDashboardHandler(services Services, logger zerolog.Logger) *DashboardHandler {
	return &DashboardHandler{services: services, logger: logger.With().Str("handler", "ai_dashboard").Logger()}
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	item, err := h.services.Dashboard.Get(r.Context(), tenantID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *DashboardHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	return r
}
