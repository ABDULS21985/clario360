package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/filemanager/dto"
	"github.com/clario360/platform/internal/filemanager/service"
)

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	fileSvc *service.FileService
	logger  zerolog.Logger
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(fileSvc *service.FileService, logger zerolog.Logger) *AdminHandler {
	return &AdminHandler{fileSvc: fileSvc, logger: logger}
}

// ListQuarantined handles GET /api/v1/files/quarantine
func (h *AdminHandler) ListQuarantined(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	page := 1
	perPage := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("per_page")); err == nil && v > 0 && v <= 100 {
		perPage = v
	}

	entries, total, err := h.fileSvc.ListQuarantined(r.Context(), page, perPage)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list quarantined files")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list quarantined files", r)
		return
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, dto.ListResponse{
		Data:       entries,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// ResolveQuarantine handles POST /api/v1/files/quarantine/:id/resolve
func (h *AdminHandler) ResolveQuarantine(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	quarantineID := chi.URLParam(r, "id")
	if quarantineID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id is required", r)
		return
	}

	var req dto.QuarantineResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", r)
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	user := auth.UserFromContext(r.Context())

	if err := h.fileSvc.ResolveQuarantine(r.Context(), quarantineID, user.ID, req.Action); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"quarantine_id": quarantineID,
		"action":        req.Action,
		"resolved_by":   user.ID,
		"status":        "resolved",
	})
}

// GetStats handles GET /api/v1/files/stats
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	stats, err := h.fileSvc.GetStorageStats(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get storage stats")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get storage stats", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"storage_stats": stats,
	})
}

// Rescan handles POST /api/v1/files/:id/rescan
func (h *AdminHandler) Rescan(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	fileID := chi.URLParam(r, "id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id is required", r)
		return
	}

	if err := h.fileSvc.RescanFile(r.Context(), fileID); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"file_id": fileID,
		"status":  "rescan_queued",
	})
}

func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", r)
		return false
	}

	for _, role := range user.Roles {
		if role == "super_admin" || role == "security-manager" {
			return true
		}
	}

	writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required", r)
	return false
}
