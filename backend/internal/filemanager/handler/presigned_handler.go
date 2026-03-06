package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/filemanager/dto"
	"github.com/clario360/platform/internal/filemanager/service"
)

// PresignedHandler handles presigned URL endpoints.
type PresignedHandler struct {
	fileSvc *service.FileService
	logger  zerolog.Logger
}

// NewPresignedHandler creates a new presigned handler.
func NewPresignedHandler(fileSvc *service.FileService, logger zerolog.Logger) *PresignedHandler {
	return &PresignedHandler{fileSvc: fileSvc, logger: logger}
}

// GenerateUploadURL handles POST /api/v1/files/upload/presigned
func (h *PresignedHandler) GenerateUploadURL(w http.ResponseWriter, r *http.Request) {
	var req dto.PresignedUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", r)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	tenantID := auth.TenantFromContext(r.Context())
	user := auth.UserFromContext(r.Context())
	if tenantID == "" || user == nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "authentication required", r)
		return
	}

	resp, err := h.fileSvc.GeneratePresignedUpload(r.Context(), &req, tenantID, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// ConfirmUpload handles POST /api/v1/files/upload/confirm
func (h *PresignedHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	var req dto.PresignedConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", r)
		return
	}

	if req.FileID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file_id is required", r)
		return
	}

	tenantID := auth.TenantFromContext(r.Context())
	user := auth.UserFromContext(r.Context())
	if tenantID == "" || user == nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "authentication required", r)
		return
	}

	record, err := h.fileSvc.ConfirmPresignedUpload(r.Context(), req.FileID, tenantID, user.ID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, dto.FileResponseFromModel(record))
}

// GenerateDownloadURL handles GET /api/v1/files/:id/presigned
func (h *PresignedHandler) GenerateDownloadURL(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id is required", r)
		return
	}

	tenantID := auth.TenantFromContext(r.Context())
	user := auth.UserFromContext(r.Context())
	if tenantID == "" || user == nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "authentication required", r)
		return
	}

	resp, err := h.fileSvc.GeneratePresignedDownload(r.Context(), tenantID, fileID, user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
