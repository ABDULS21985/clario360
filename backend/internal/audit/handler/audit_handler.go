package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/dto"
	"github.com/clario360/platform/internal/audit/service"
	"github.com/clario360/platform/internal/auth"
)

// AuditHandler handles REST API requests for querying audit logs.
type AuditHandler struct {
	querySvc *service.QueryService
	logger   zerolog.Logger
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(querySvc *service.QueryService, logger zerolog.Logger) *AuditHandler {
	return &AuditHandler{
		querySvc: querySvc,
		logger:   logger,
	}
}

// ListLogs handles GET /api/v1/audit/logs — query audit logs with filters and pagination.
func (h *AuditHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	params, err := dto.ParseQueryParams(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	// Enforce tenant from JWT context
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "tenant context required", r)
		return
	}
	params.TenantID = tenantID

	roles := getRoles(r)

	result, err := h.querySvc.Query(r.Context(), params, roles)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to query audit logs")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query audit logs", r)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetLog handles GET /api/v1/audit/logs/{id} — get a single audit log entry.
func (h *AuditHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "id is required", r)
		return
	}

	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "tenant context required", r)
		return
	}

	roles := getRoles(r)

	entry, err := h.querySvc.GetByID(r.Context(), tenantID, id, roles)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to get audit log")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get audit log", r)
		return
	}

	if entry == nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "audit log entry not found", r)
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// GetStats handles GET /api/v1/audit/logs/stats — aggregated statistics.
func (h *AuditHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "tenant context required", r)
		return
	}

	params, err := dto.ParseQueryParams(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	stats, err := h.querySvc.GetStats(r.Context(), tenantID, params.DateFrom, params.DateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get audit stats")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get audit stats", r)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GetTimeline handles GET /api/v1/audit/logs/timeline/{resourceId} — activity timeline.
func (h *AuditHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceId")
	if resourceID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "resourceId is required", r)
		return
	}

	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "tenant context required", r)
		return
	}

	page := 1
	perPage := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := parseInt(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := parseInt(pp); err == nil && v > 0 {
			perPage = v
		}
	}

	roles := getRoles(r)

	result, err := h.querySvc.GetTimeline(r.Context(), tenantID, resourceID, page, perPage, roles)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get timeline")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get timeline", r)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// helper functions

func getRoles(r *http.Request) []string {
	user := auth.UserFromContext(r.Context())
	if user != nil {
		return user.Roles
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, status int, code, message string, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	requestID := r.Header.Get("X-Request-ID")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":       code,
		"message":    message,
		"request_id": requestID,
	})
}

func parseInt(s string) (int, error) {
	var v int
	_, err := json.Number(s).Int64()
	if err != nil {
		return 0, err
	}
	n, _ := json.Number(s).Int64()
	v = int(n)
	return v, nil
}
