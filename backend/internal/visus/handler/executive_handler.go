package handler

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
	"github.com/clario360/platform/internal/visus/service"
)

type ExecutiveHandler struct {
	baseHandler
	service *service.ExecutiveService
}

func NewExecutiveHandler(service *service.ExecutiveService, logger zerolog.Logger) *ExecutiveHandler {
	return &ExecutiveHandler{baseHandler: baseHandler{logger: logger}, service: service}
}

func (h *ExecutiveHandler) View(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.GetView(r.Context(), tenantID, userID)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ExecutiveHandler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.GetSummary(r.Context(), tenantID, userID)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ExecutiveHandler) Health(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Health(r.Context(), tenantID)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}
