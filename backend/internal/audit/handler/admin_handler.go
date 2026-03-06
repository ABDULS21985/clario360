package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/repository"
	"github.com/clario360/platform/internal/audit/service"
	"github.com/clario360/platform/internal/auth"
)

// AdminHandler handles partition management and integrity verification.
type AdminHandler struct {
	partitionMgr *repository.PartitionManager
	integritySvc *service.IntegrityService
	logger       zerolog.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	partitionMgr *repository.PartitionManager,
	integritySvc *service.IntegrityService,
	logger zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
		partitionMgr: partitionMgr,
		integritySvc: integritySvc,
		logger:       logger,
	}
}

// VerifyChain handles POST /api/v1/audit/verify — verify hash chain integrity.
func (h *AdminHandler) VerifyChain(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	if tenantID == "" {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "tenant context required", r)
		return
	}

	var req struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", r)
		return
	}

	if req.DateFrom == "" {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "date_from is required", r)
		return
	}

	dateFrom, err := time.Parse(time.RFC3339, req.DateFrom)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid date_from format", r)
		return
	}

	dateTo := time.Now().UTC()
	if req.DateTo != "" {
		dt, err := time.Parse(time.RFC3339, req.DateTo)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid date_to format", r)
			return
		}
		dateTo = dt
	}

	result, err := h.integritySvc.VerifyChain(r.Context(), tenantID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("hash chain verification failed")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "verification failed", r)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListPartitions handles GET /api/v1/audit/partitions — list partition info.
func (h *AdminHandler) ListPartitions(w http.ResponseWriter, r *http.Request) {
	partitions, err := h.partitionMgr.ListPartitions(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list partitions")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list partitions", r)
		return
	}

	if partitions == nil {
		partitions = []interface{}(nil)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"partitions": partitions,
	})
}

// CreatePartition handles POST /api/v1/audit/partitions/create — manually trigger partition creation.
func (h *AdminHandler) CreatePartition(w http.ResponseWriter, r *http.Request) {
	created, err := h.partitionMgr.EnsurePartitions(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create partitions")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "partition creation failed", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"created": created,
		"message": "partition maintenance completed",
	})
}
