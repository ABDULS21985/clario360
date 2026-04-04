package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
)

type ExplanationHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewExplanationHandler(services Services, logger zerolog.Logger) *ExplanationHandler {
	return &ExplanationHandler{services: services, logger: logger.With().Str("handler", "ai_explanations").Logger()}
}

func (h *ExplanationHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	predictionID, err := suiteapi.UUIDParam(r, "predictionId")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	logEntry, err := h.services.Predictions.Get(r.Context(), tenantID, predictionID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	explanation, err := h.services.Explanations.FromPrediction(logEntry)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, explanation)
}

func (h *ExplanationHandler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.services.Predictions.SearchExplanations(r.Context(), tenantID, r.URL.Query().Get("q"), limit)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *ExplanationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/search", h.Search)
	r.Get("/{predictionId}", h.Get)
	return r
}
