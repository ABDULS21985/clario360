package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/middleware"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

// AssetHandler handles all /api/v1/cyber/assets endpoints.
type AssetHandler struct {
	svc    *service.AssetService
	logger zerolog.Logger
}

// NewAssetHandler creates a new AssetHandler.
func NewAssetHandler(svc *service.AssetService, logger zerolog.Logger) *AssetHandler {
	return &AssetHandler{svc: svc, logger: logger}
}

// ---- Asset CRUD ----

// CreateAsset handles POST /api/v1/cyber/assets
func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}

	var req dto.CreateAssetRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}

	asset, err := h.svc.CreateAsset(r.Context(), tenantID, userID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "an asset with the same IP address already exists", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": asset})
}

// ListAssets handles GET /api/v1/cyber/assets
func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}

	params, err := parseAssetListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	result, err := h.svc.ListAssets(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetAsset handles GET /api/v1/cyber/assets/:id
func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	asset, err := h.svc.GetAsset(r.Context(), tenantID, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "asset not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "GET_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": asset})
}

// UpdateAsset handles PUT /api/v1/cyber/assets/:id
func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	var req dto.UpdateAssetRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}

	asset, err := h.svc.UpdateAsset(r.Context(), tenantID, assetID, userID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "asset not found", nil)
			return
		}
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "an asset with the same IP address already exists", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": asset})
}

// DeleteAsset handles DELETE /api/v1/cyber/assets/:id
func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	if err := h.svc.DeleteAsset(r.Context(), tenantID, assetID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "asset not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PatchTags handles PATCH /api/v1/cyber/assets/:id/tags
func (h *AssetHandler) PatchTags(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	var req dto.TagPatchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}

	asset, err := h.svc.PatchTags(r.Context(), tenantID, assetID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "asset not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "PATCH_TAGS_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": asset})
}

// ---- Bulk Operations ----

// BulkCreate handles POST /api/v1/cyber/assets/bulk
func (h *AssetHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}

	contentType := r.Header.Get("Content-Type")
	var result *dto.BulkCreateResult
	var err error

	if strings.Contains(contentType, "multipart/form-data") {
		if err = r.ParseMultipartForm(10 << 20); err != nil { // 10MB
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "failed to parse multipart form", nil)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "multipart field 'file' is required", nil)
			return
		}
		defer file.Close()
		result, err = h.svc.BulkCreateFromCSV(r.Context(), tenantID, userID, file)
	} else {
		var reqs []dto.CreateAssetRequest
		if !decodeJSON(w, r, &reqs) {
			return
		}
		result, err = h.svc.BulkCreate(r.Context(), tenantID, userID, reqs)
	}

	if err != nil {
		var bulkErr *service.BulkValidationError
		if errors.As(err, &bulkErr) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"code":    bulkErr.Code,
					"message": bulkErr.Message,
					"details": map[string]any{"rows": bulkErr.Rows},
					"request_id": w.Header().Get(middleware.RequestIDHeader),
				},
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "BULK_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

// BulkUpdateTags handles PUT /api/v1/cyber/assets/bulk/tags
func (h *AssetHandler) BulkUpdateTags(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.BulkTagRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.BulkUpdateTags(r.Context(), tenantID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "BULK_TAG_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BulkDelete handles DELETE /api/v1/cyber/assets/bulk
func (h *AssetHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.BulkDeleteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	if err := h.svc.BulkDelete(r.Context(), tenantID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "BULK_DELETE_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Relationships ----

// ListRelationships handles GET /api/v1/cyber/assets/:id/relationships
func (h *AssetHandler) ListRelationships(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rels, err := h.svc.ListRelationships(r.Context(), tenantID, assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_REL_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": rels})
}

// CreateRelationship handles POST /api/v1/cyber/assets/:id/relationships
func (h *AssetHandler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateRelationshipRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	rel, err := h.svc.CreateRelationship(r.Context(), tenantID, assetID, userID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "relationship already exists", nil)
			return
		}
		if errors.Is(err, repository.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "CREATE_REL_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": rel})
}

// DeleteRelationship handles DELETE /api/v1/cyber/assets/:id/relationships/:relId
func (h *AssetHandler) DeleteRelationship(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	relID, ok := parseUUID(w, chi.URLParam(r, "relId"))
	if !ok {
		return
	}
	if err := h.svc.DeleteRelationship(r.Context(), tenantID, assetID, relID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "relationship not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "DELETE_REL_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Vulnerabilities ----

// ListVulnerabilities handles GET /api/v1/cyber/assets/:id/vulnerabilities
func (h *AssetHandler) ListVulnerabilities(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	params := parseVulnListParams(r)
	vulns, total, err := h.svc.ListVulnerabilities(r.Context(), tenantID, assetID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_VULNS_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     vulns,
		"total":    total,
		"page":     params.Page,
		"per_page": params.PerPage,
	})
}

// CreateVulnerability handles POST /api/v1/cyber/assets/:id/vulnerabilities
func (h *AssetHandler) CreateVulnerability(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.CreateVulnerabilityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	vuln, err := h.svc.CreateVulnerability(r.Context(), tenantID, assetID, userID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CREATE_VULN_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": vuln})
}

// UpdateVulnerability handles PUT /api/v1/cyber/assets/:id/vulnerabilities/:vid
func (h *AssetHandler) UpdateVulnerability(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	vulnID, ok := parseUUID(w, chi.URLParam(r, "vid"))
	if !ok {
		return
	}
	var req dto.UpdateVulnerabilityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	vuln, err := h.svc.UpdateVulnerability(r.Context(), tenantID, assetID, vulnID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "vulnerability not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "UPDATE_VULN_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": vuln})
}

// ---- Scans ----

// TriggerScan handles POST /api/v1/cyber/assets/scan
func (h *AssetHandler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.ScanTriggerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	scan, err := h.svc.TriggerScan(r.Context(), tenantID, userID, &req)
	if err != nil {
		status := http.StatusInternalServerError
		code := "SCAN_FAILED"
		if strings.Contains(err.Error(), "scan scope too large") {
			status = http.StatusUnprocessableEntity
			code = "SCAN_SCOPE_TOO_LARGE"
		}
		if strings.Contains(err.Error(), "public IP") {
			status = http.StatusUnprocessableEntity
			code = "SCAN_PUBLIC_IP_BLOCKED"
		}
		writeError(w, status, code, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"data": dto.ScanTriggerResponse{
		ScanID:  scan.ID.String(),
		Status:  scan.Status,
		Message: "scan started, poll GET /api/v1/cyber/assets/scans/" + scan.ID.String() + " for progress",
	}})
}

// ListScans handles GET /api/v1/cyber/assets/scans
func (h *AssetHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := parseScanListParams(r)
	scans, total, err := h.svc.ListScans(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_SCANS_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     scans,
		"total":    total,
		"page":     params.Page,
		"per_page": params.PerPage,
	})
}

// GetScan handles GET /api/v1/cyber/assets/scans/:id
func (h *AssetHandler) GetScan(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	scanID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	scan, err := h.svc.GetScan(r.Context(), tenantID, scanID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "scan not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "GET_SCAN_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": scan})
}

// CancelScan handles POST /api/v1/cyber/assets/scans/:id/cancel
func (h *AssetHandler) CancelScan(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	scanID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.CancelScan(r.Context(), tenantID, scanID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "scan not found or already completed", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "CANCEL_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Stats ----

// GetStats handles GET /api/v1/cyber/assets/stats
func (h *AssetHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.GetStats(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

// GetCount handles GET /api/v1/cyber/assets/count
func (h *AssetHandler) GetCount(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseAssetListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	count, err := h.svc.CountAssets(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "COUNT_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": dto.AssetCountResponse{Count: count}})
}

// ---- Helpers ----

type envelope map[string]any

func requireTenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tenantStr := auth.TenantFromContext(r.Context())
	if tenantStr == "" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "tenant context is required")
		return uuid.Nil, uuid.Nil, false
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "invalid tenant ID", nil)
		return uuid.Nil, uuid.Nil, false
	}

	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return uuid.Nil, uuid.Nil, false
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "invalid user ID", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, userID, true
}

func parseUUID(w http.ResponseWriter, s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("invalid UUID: %s", s), nil)
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "request body must be valid JSON", map[string]any{"cause": err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string, details ...any) {
	var detailValue any
	if len(details) > 0 {
		detailValue = details[0]
	}
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":       code,
			"message":    message,
			"details":    detailValue,
			"request_id": w.Header().Get(middleware.RequestIDHeader),
		},
	})
}

func writeValidationError(w http.ResponseWriter, fieldErrs map[string]string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"error": map[string]any{
			"code":    "VALIDATION_ERROR",
			"message": "request validation failed",
			"details": map[string]any{"fields": fieldErrs},
			"request_id": w.Header().Get(middleware.RequestIDHeader),
		},
	})
}

func parseAssetListParams(r *http.Request) (*dto.AssetListParams, error) {
	q := r.URL.Query()
	params := &dto.AssetListParams{}

	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	params.Types = splitQueryValues(q, "type")
	params.Criticalities = splitQueryValues(q, "criticality")
	params.Statuses = splitQueryValues(q, "status")
	if v := q.Get("os"); v != "" {
		params.OS = &v
	}
	if v := q.Get("department"); v != "" {
		params.Department = &v
	}
	if v := q.Get("owner"); v != "" {
		params.Owner = &v
	}
	if v := q.Get("location"); v != "" {
		params.Location = &v
	}
	params.Tags = splitQueryValues(q, "tag")
	if v := q.Get("discovery_source"); v != "" {
		params.DiscoverySource = &v
	}
	if v := q.Get("discovered_after"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return nil, fmt.Errorf("invalid discovered_after: %w", err)
			}
		}
		params.DiscoveredAfter = &t
	}
	if v := q.Get("discovered_before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return nil, fmt.Errorf("invalid discovered_before: %w", err)
			}
		}
		params.DiscoveredBefore = &t
	}
	if v := q.Get("last_seen_after"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return nil, fmt.Errorf("invalid last_seen_after: %w", err)
			}
		}
		params.LastSeenAfter = &t
	}
	if v := q.Get("has_vulnerabilities"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid has_vulnerabilities: %w", err)
		}
		params.HasVulnerabilities = &b
	}
	if v := q.Get("vulnerability_severity"); v != "" {
		params.VulnerabilitySeverity = &v
	}
	if v := q.Get("min_vuln_count"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid min_vuln_count: %w", err)
		}
		params.MinVulnCount = &n
	}
	params.Sort = q.Get("sort")
	params.Order = q.Get("order")
	if v := q.Get("page"); v != "" {
		n, _ := strconv.Atoi(v)
		params.Page = n
	}
	if v := q.Get("per_page"); v != "" {
		n, _ := strconv.Atoi(v)
		params.PerPage = n
	}
	return params, nil
}

func splitQueryValues(values url.Values, key string) []string {
	raw := values[key]
	if len(raw) == 0 {
		return nil
	}
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		for _, split := range strings.Split(item, ",") {
			split = strings.TrimSpace(split)
			if split != "" {
				parts = append(parts, split)
			}
		}
	}
	return parts
}

func parseVulnListParams(r *http.Request) *dto.VulnerabilityListParams {
	q := r.URL.Query()
	params := &dto.VulnerabilityListParams{}
	if v := q.Get("status"); v != "" {
		params.Status = &v
	}
	if v := q.Get("severity"); v != "" {
		params.Severity = &v
	}
	if v := q.Get("page"); v != "" {
		n, _ := strconv.Atoi(v)
		params.Page = n
	}
	if v := q.Get("per_page"); v != "" {
		n, _ := strconv.Atoi(v)
		params.PerPage = n
	}
	params.SetDefaults()
	return params
}

func parseScanListParams(r *http.Request) *dto.ScanListParams {
	q := r.URL.Query()
	params := &dto.ScanListParams{}
	if v := q.Get("scan_type"); v != "" {
		params.ScanType = &v
	}
	if v := q.Get("status"); v != "" {
		params.Status = &v
	}
	if v := q.Get("page"); v != "" {
		n, _ := strconv.Atoi(v)
		params.Page = n
	}
	if v := q.Get("per_page"); v != "" {
		n, _ := strconv.Atoi(v)
		params.PerPage = n
	}
	params.SetDefaults()
	return params
}
