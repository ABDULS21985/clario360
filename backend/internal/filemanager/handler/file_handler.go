package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/filemanager/dto"
	"github.com/clario360/platform/internal/filemanager/service"
)

// FileHandler handles file CRUD REST endpoints.
type FileHandler struct {
	fileSvc *service.FileService
	logger  zerolog.Logger
}

// NewFileHandler creates a new file handler.
func NewFileHandler(fileSvc *service.FileService, logger zerolog.Logger) *FileHandler {
	return &FileHandler{fileSvc: fileSvc, logger: logger}
}

// Upload handles POST /api/v1/files/upload
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (32MB in memory, rest on disk)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid multipart form", r)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file field is required", r)
		return
	}
	defer file.Close()

	req := dto.ParseUploadForm(r)
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

	record, err := h.fileSvc.Upload(
		r.Context(), req, file, header.Size,
		header.Filename, header.Header.Get("Content-Type"),
		tenantID, user.ID,
		clientIP(r), r.UserAgent(),
	)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusCreated, dto.FileResponseFromModel(record))
}

// GetFile handles GET /api/v1/files/:id
func (h *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
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

	record, err := h.fileSvc.GetFile(r.Context(), tenantID, fileID, user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, dto.FileResponseFromModel(record))
}

// Download handles GET /api/v1/files/:id/download
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
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

	reader, record, err := h.fileSvc.Download(r.Context(), tenantID, fileID, user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}
	defer reader.Close()

	// Set response headers
	w.Header().Set("Content-Type", record.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, record.SanitizedName))
	w.Header().Set("Content-Length", strconv.FormatInt(record.SizeBytes, 10))
	w.Header().Set("X-Checksum-SHA256", record.ChecksumSHA256)
	if record.VirusScanStatus == "pending" {
		w.Header().Set("X-Virus-Scan-Status", "pending")
	}

	w.WriteHeader(http.StatusOK)

	// Stream content
	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
		}
		if readErr != nil {
			break
		}
	}
}

// ListFiles handles GET /api/v1/files
func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "authentication required", r)
		return
	}

	params := dto.ParseListParams(r)
	params.TenantID = tenantID

	files, total, err := h.fileSvc.ListFiles(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list files")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list files", r)
		return
	}

	responses := make([]*dto.FileResponse, 0, len(files))
	for _, f := range files {
		responses = append(responses, dto.FileResponseFromModel(f))
	}

	writeJSON(w, http.StatusOK, dto.NewListResponse(responses, total, params.Page, params.PerPage))
}

// DeleteFile handles DELETE /api/v1/files/:id
func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
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

	if err := h.fileSvc.DeleteFile(r.Context(), tenantID, fileID, user.ID, clientIP(r), r.UserAgent()); err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetVersions handles GET /api/v1/files/:id/versions
func (h *FileHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "authentication required", r)
		return
	}

	versions, err := h.fileSvc.GetVersions(r.Context(), tenantID, fileID)
	if err != nil {
		handleServiceError(w, r, err, h.logger)
		return
	}

	responses := make([]*dto.FileResponse, 0, len(versions))
	for _, v := range versions {
		responses = append(responses, dto.FileResponseFromModel(v))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"versions": responses,
		"total":    len(responses),
	})
}

// GetAccessLog handles GET /api/v1/files/:id/access-log
func (h *FileHandler) GetAccessLog(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "authentication required", r)
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

	logs, total, err := h.fileSvc.GetAccessLog(r.Context(), tenantID, fileID, page, perPage)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get access log")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get access log", r)
		return
	}

	writeJSON(w, http.StatusOK, dto.NewListResponse(logs, total, page, perPage))
}

// helpers

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":       code,
		"message":    message,
		"request_id": r.Header.Get("X-Request-ID"),
	})
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error, logger zerolog.Logger) {
	if svcErr, ok := err.(*service.ServiceError); ok {
		writeError(w, svcErr.Code, svcErr.ErrCode, svcErr.Message, r)
		return
	}
	logger.Error().Err(err).Msg("internal error")
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", r)
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
