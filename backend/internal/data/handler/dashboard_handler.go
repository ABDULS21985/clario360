package handler

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/suiteapi"
)

type DashboardHandler struct {
	baseHandler
	service *service.DashboardService
}

func NewDashboardHandler(service *service.DashboardService, logger zerolog.Logger) *DashboardHandler {
	return &DashboardHandler{
		baseHandler: baseHandler{logger: logger},
		service:     service,
	}
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return
	}
	item, err := h.service.Get(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}
