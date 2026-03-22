package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/dto"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
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

	writeJSON(w, http.StatusOK, toAuditLogDetail(entry))
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

	// Parse optional timeline filters
	var filter *repository.TimelineFilter
	q := r.URL.Query()
	action := q.Get("action")
	dateFromStr := q.Get("date_from")
	dateToStr := q.Get("date_to")
	if action != "" || dateFromStr != "" || dateToStr != "" {
		filter = &repository.TimelineFilter{Action: action}
		if dateFromStr != "" {
			if t, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
				filter.DateFrom = t
			}
		}
		if dateToStr != "" {
			if t, err := time.Parse(time.RFC3339, dateToStr); err == nil {
				filter.DateTo = t
			}
		}
	}

	roles := getRoles(r)

	timeline, err := h.querySvc.GetTimeline(r.Context(), tenantID, resourceID, page, perPage, roles, filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get timeline")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get timeline", r)
		return
	}

	writeJSON(w, http.StatusOK, timeline)
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
		"error": map[string]interface{}{
			"code":       code,
			"message":    message,
			"request_id": requestID,
		},
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

// ── Enriched detail response ────────────────────────────────────────────────

// auditLogDetailResponse matches the frontend AuditLogDetail interface.
// It wraps AuditEntry fields plus computed/extracted detail fields.
type auditLogDetailResponse struct {
	// Core AuditEntry fields
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	UserID        *string         `json:"user_id,omitempty"`
	UserEmail     string          `json:"user_email"`
	Service       string          `json:"service"`
	Action        string          `json:"action"`
	Severity      string          `json:"severity"`
	ResourceType  string          `json:"resource_type"`
	ResourceID    string          `json:"resource_id"`
	OldValue      json.RawMessage `json:"old_value,omitempty"`
	NewValue      json.RawMessage `json:"new_value,omitempty"`
	IPAddress     string          `json:"ip_address"`
	UserAgent     string          `json:"user_agent"`
	Metadata      json.RawMessage `json:"metadata"`
	EventID       string          `json:"event_id"`
	CorrelationID string          `json:"correlation_id"`
	PreviousHash  string          `json:"previous_hash"`
	EntryHash     string          `json:"entry_hash"`
	CreatedAt     string          `json:"created_at"`

	// Enriched detail fields (computed or extracted from metadata)
	RequestBody    interface{}         `json:"request_body"`
	ResponseStatus *int                `json:"response_status"`
	ResponseBody   interface{}         `json:"response_body"`
	GeoLocation    interface{}         `json:"geo_location"`
	SessionID      *string             `json:"session_id"`
	DurationMs     *float64            `json:"duration_ms"`
	Changes        []auditChangeRecord `json:"changes"`
}

type auditChangeRecord struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// toAuditLogDetail converts an AuditEntry into an enriched detail response.
func toAuditLogDetail(entry *model.AuditEntry) auditLogDetailResponse {
	resp := auditLogDetailResponse{
		ID:            entry.ID,
		TenantID:      entry.TenantID,
		UserID:        entry.UserID,
		UserEmail:     entry.UserEmail,
		Service:       entry.Service,
		Action:        entry.Action,
		Severity:      entry.Severity,
		ResourceType:  entry.ResourceType,
		ResourceID:    entry.ResourceID,
		OldValue:      entry.OldValue,
		NewValue:      entry.NewValue,
		IPAddress:     entry.IPAddress,
		UserAgent:     entry.UserAgent,
		Metadata:      entry.Metadata,
		EventID:       entry.EventID,
		CorrelationID: entry.CorrelationID,
		PreviousHash:  entry.PreviousHash,
		EntryHash:     entry.EntryHash,
		CreatedAt:     entry.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Changes:       computeChanges(entry.OldValue, entry.NewValue),
	}

	// Extract optional fields from metadata bag.
	if len(entry.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(entry.Metadata, &meta); err == nil {
			if v, ok := meta["request_body"]; ok {
				resp.RequestBody = v
			}
			if v, ok := meta["response_status"]; ok {
				if n, ok := v.(float64); ok {
					intN := int(n)
					resp.ResponseStatus = &intN
				}
			}
			if v, ok := meta["response_body"]; ok {
				resp.ResponseBody = v
			}
			if v, ok := meta["geo_location"]; ok {
				resp.GeoLocation = v
			}
			if v, ok := meta["session_id"].(string); ok && v != "" {
				resp.SessionID = &v
			}
			if v, ok := meta["duration_ms"]; ok {
				if n, ok := v.(float64); ok {
					resp.DurationMs = &n
				}
			}
		}
	}

	return resp
}

// computeChanges builds a list of field-level changes by diffing old_value and new_value JSON objects.
func computeChanges(oldJSON, newJSON json.RawMessage) []auditChangeRecord {
	var changes []auditChangeRecord

	if len(oldJSON) == 0 && len(newJSON) == 0 {
		return changes
	}

	oldMap := map[string]interface{}{}
	newMap := map[string]interface{}{}

	if len(oldJSON) > 0 {
		_ = json.Unmarshal(oldJSON, &oldMap)
	}
	if len(newJSON) > 0 {
		_ = json.Unmarshal(newJSON, &newMap)
	}

	// Fields that changed or were added.
	for key, newVal := range newMap {
		oldVal, existed := oldMap[key]
		if !existed {
			changes = append(changes, auditChangeRecord{Field: key, OldValue: nil, NewValue: newVal})
		} else {
			oldBytes, _ := json.Marshal(oldVal)
			newBytes, _ := json.Marshal(newVal)
			if string(oldBytes) != string(newBytes) {
				changes = append(changes, auditChangeRecord{Field: key, OldValue: oldVal, NewValue: newVal})
			}
		}
	}

	// Fields that were removed.
	for key, oldVal := range oldMap {
		if _, exists := newMap[key]; !exists {
			changes = append(changes, auditChangeRecord{Field: key, OldValue: oldVal, NewValue: nil})
		}
	}

	return changes
}
