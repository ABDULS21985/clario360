package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/dto"
	"github.com/clario360/platform/internal/audit/service"
	"github.com/clario360/platform/internal/auth"
)

// ExportHandler handles audit log export requests.
type ExportHandler struct {
	exportSvc *service.ExportService
	logger    zerolog.Logger
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(exportSvc *service.ExportService, logger zerolog.Logger) *ExportHandler {
	return &ExportHandler{
		exportSvc: exportSvc,
		logger:    logger,
	}
}

// Export handles GET /api/v1/audit/logs/export — streaming export.
func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "tenant context required", r)
		return
	}

	// Parse export config from query params
	cfg := &dto.ExportConfig{
		TenantID:     tenantID,
		Format:       dto.ExportFormat(r.URL.Query().Get("format")),
		UserID:       r.URL.Query().Get("user_id"),
		Service:      r.URL.Query().Get("service"),
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		Severity:     r.URL.Query().Get("severity"),
		Search:       r.URL.Query().Get("search"),
	}

	// Parse dates
	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		t, err := time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid date_from format", r)
			return
		}
		cfg.DateFrom = t
	} else {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "date_from is required for export", r)
		return
	}

	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		t, err := time.Parse(time.RFC3339, dateTo)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid date_to format", r)
			return
		}
		cfg.DateTo = t
	}

	if err := cfg.Validate(); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	roles := getRoles(r)

	// Check if async export is needed
	shouldAsync, count, err := h.exportSvc.ShouldExportAsync(r.Context(), cfg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to check export size")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to prepare export", r)
		return
	}

	if shouldAsync {
		// For async exports, return 202 with job info
		writeJSON(w, http.StatusAccepted, dto.ExportJobStatus{
			Status:      "processing",
			RecordCount: count,
		})
		return
	}

	// Synchronous streaming export
	now := time.Now().Format("2006-01-02")

	switch cfg.Format {
	case dto.ExportFormatNDJSON:
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"audit_export_%s.ndjson\"", now))
		w.WriteHeader(http.StatusOK)
		_, err = h.exportSvc.ExportNDJSON(r.Context(), w, cfg, roles)

	default: // CSV
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"audit_export_%s.csv\"", now))
		w.WriteHeader(http.StatusOK)
		_, err = h.exportSvc.ExportCSV(r.Context(), w, cfg, roles)
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("export failed during streaming")
		// Can't change status code after headers are sent
	}
}
