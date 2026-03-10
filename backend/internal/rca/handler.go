package rca

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// Handler provides HTTP endpoints for root cause analysis.
type Handler struct {
	engine *Engine
	logger zerolog.Logger
}

// NewHandler creates an RCA HTTP handler.
func NewHandler(engine *Engine, logger zerolog.Logger) *Handler {
	return &Handler{
		engine: engine,
		logger: logger.With().Str("component", "rca-handler").Logger(),
	}
}

// RegisterRoutes mounts RCA routes on the given router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/rca", func(r chi.Router) {
		r.Post("/analyze", h.Analyze)
		r.Get("/{type}/{incidentId}", h.GetResult)
		r.Get("/{type}/{incidentId}/timeline", h.GetTimeline)
	})
}

// Analyze triggers root cause analysis.
// POST /api/v1/rca/analyze
func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Type == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "type is required")
		return
	}
	if req.IncidentID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "incident_id is required")
		return
	}

	result, err := h.engine.Analyze(r.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Incident not found")
			return
		}
		h.logger.Error().Err(err).Msg("RCA failed")
		h.writeError(w, http.StatusInternalServerError, "RCA_FAILED", "Root cause analysis failed: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"data": result})
}

// GetResult returns a cached or re-computed RCA result.
// GET /api/v1/rca/{type}/{incidentId}
func (h *Handler) GetResult(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}

	analysisType := AnalysisType(chi.URLParam(r, "type"))
	incidentID, err := uuid.Parse(chi.URLParam(r, "incidentId"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid incident ID")
		return
	}

	result, err := h.engine.GetCachedResult(r.Context(), tenantID, analysisType, incidentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Incident not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "RCA_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"data": result})
}

// GetTimeline returns just the event timeline for an incident.
// GET /api/v1/rca/{type}/{incidentId}/timeline
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}

	analysisType := AnalysisType(chi.URLParam(r, "type"))
	incidentID, err := uuid.Parse(chi.URLParam(r, "incidentId"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid incident ID")
		return
	}

	timeline, err := h.engine.GetTimeline(r.Context(), tenantID, analysisType, incidentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Incident not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "TIMELINE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"data": timeline})
}

func (h *Handler) requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenant := auth.TenantFromContext(r.Context())
	if tenant == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant context required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(tenant)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_TENANT", "Invalid tenant ID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}
