package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// eventRepository defines the data-access surface used by EventHandler.
type eventRepository interface {
	QuerySecurityEvents(ctx context.Context, tenantID uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error)
	GetSecurityEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*model.SecurityEvent, error)
	GetSecurityEventStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*model.EventStats, error)
	InsertSecurityEvents(ctx context.Context, events []model.SecurityEvent) error
}

// EventHandler handles security event query endpoints.
type EventHandler struct {
	repo   eventRepository
	logger zerolog.Logger
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(repo eventRepository, logger zerolog.Logger) *EventHandler {
	return &EventHandler{repo: repo, logger: logger}
}

// ListEvents handles GET /api/v1/cyber/events — paginated, filtered event list.
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseEventQueryParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	events, total, err := h.repo.QuerySecurityEvents(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("list security events failed")
		writeError(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}
	totalPages := (total + params.PerPage - 1) / params.PerPage
	writeJSON(w, http.StatusOK, envelope{
		"data": events,
		"meta": envelope{
			"total":       total,
			"page":        params.Page,
			"per_page":    params.PerPage,
			"total_pages": totalPages,
		},
	})
}

// GetEvent handles GET /api/v1/cyber/events/{id} — single event detail.
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	eventID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	event, err := h.repo.GetSecurityEvent(r.Context(), tenantID, eventID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": event})
}

// GetEventStats handles GET /api/v1/cyber/events/stats — volume stats.
func (h *EventHandler) GetEventStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	params := &model.EventQueryParams{}
	if v := q.Get("from"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid from: "+err.Error(), nil)
			return
		}
		params.From = &ts
	}
	if v := q.Get("to"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid to: "+err.Error(), nil)
			return
		}
		params.To = &ts
	}

	stats, err := h.repo.GetSecurityEventStats(r.Context(), tenantID, params.From, params.To)
	if err != nil {
		h.logger.Error().Err(err).Msg("get event stats failed")
		writeError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

// exportMaxRows is the hard cap on rows returned in a single export.
const exportMaxRows = 10000

// allEventExportColumns is the ordered set of columns available for CSV export.
var allEventExportColumns = []string{
	"id", "timestamp", "source", "type", "severity",
	"source_ip", "dest_ip", "dest_port", "protocol",
	"username", "process", "parent_process", "command_line",
	"file_path", "file_hash", "asset_id", "matched_rules", "processed_at",
}

// ExportEvents handles GET /api/v1/cyber/events/export — streaming CSV or NDJSON.
func (h *EventHandler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseEventQueryParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if params.From == nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "from is required for export", nil)
		return
	}

	// Override pagination for export: fetch up to exportMaxRows.
	params.Page = 1
	params.PerPage = exportMaxRows

	events, total, err := h.repo.QuerySecurityEvents(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("export security events failed")
		writeError(w, http.StatusInternalServerError, "EXPORT_FAILED", err.Error(), nil)
		return
	}

	format := r.URL.Query().Get("format")
	now := time.Now().Format("2006-01-02")

	if format == "ndjson" {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"events_export_%s.ndjson\"", now))
		w.Header().Set("X-Export-Total", strconv.Itoa(total))
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		for i := range events {
			_ = enc.Encode(events[i])
		}
		return
	}

	// Default: CSV
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"events_export_%s.csv\"", now))
	w.Header().Set("X-Export-Total", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)

	csvW := csv.NewWriter(w)
	_ = csvW.Write(allEventExportColumns)

	for i := range events {
		e := &events[i]
		record := make([]string, len(allEventExportColumns))
		record[0] = e.ID.String()
		record[1] = e.Timestamp.Format(time.RFC3339)
		record[2] = e.Source
		record[3] = e.Type
		record[4] = string(e.Severity)
		record[5] = derefStr(e.SourceIP)
		record[6] = derefStr(e.DestIP)
		if e.DestPort != nil {
			record[7] = strconv.Itoa(*e.DestPort)
		}
		record[8] = derefStr(e.Protocol)
		record[9] = derefStr(e.Username)
		record[10] = derefStr(e.Process)
		record[11] = derefStr(e.ParentProcess)
		record[12] = derefStr(e.CommandLine)
		record[13] = derefStr(e.FilePath)
		record[14] = derefStr(e.FileHash)
		if e.AssetID != nil {
			record[15] = e.AssetID.String()
		}
		ruleStrs := make([]string, len(e.MatchedRules))
		for j, r := range e.MatchedRules {
			ruleStrs[j] = r.String()
		}
		record[16] = strings.Join(ruleStrs, ";")
		record[17] = e.ProcessedAt.Format(time.RFC3339)
		_ = csvW.Write(record)

		if (i+1)%1000 == 0 {
			csvW.Flush()
		}
	}
	csvW.Flush()
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// IngestEvents handles POST /api/v1/cyber/events — batch ingest from detection engine.
func (h *EventHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}

	var events []model.SecurityEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON body: "+err.Error(), nil)
		return
	}
	if len(events) == 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "events array must not be empty", nil)
		return
	}
	if len(events) > 1000 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "batch size must not exceed 1000", nil)
		return
	}

	// Set tenant ID and defaults for each event.
	now := time.Now().UTC()
	for i := range events {
		events[i].TenantID = tenantID
		if events[i].ID == uuid.Nil {
			events[i].ID = uuid.New()
		}
		if events[i].Timestamp.IsZero() {
			events[i].Timestamp = now
		}
		if events[i].ProcessedAt.IsZero() {
			events[i].ProcessedAt = now
		}
		if events[i].Severity == "" {
			events[i].Severity = model.SeverityInfo
		}
		if events[i].RawEvent == nil {
			events[i].RawEvent = json.RawMessage("{}")
		}
		if events[i].MatchedRules == nil {
			events[i].MatchedRules = []uuid.UUID{}
		}
	}

	if err := h.repo.InsertSecurityEvents(r.Context(), events); err != nil {
		h.logger.Error().Err(err).Int("count", len(events)).Msg("ingest security events failed")
		writeError(w, http.StatusInternalServerError, "INGEST_FAILED", err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		"inserted": len(events),
		"total":    len(events),
	})
}

// parseEventQueryParams extracts filter/sort/pagination from the HTTP request.
func parseEventQueryParams(r *http.Request) (*model.EventQueryParams, error) {
	q := r.URL.Query()
	params := &model.EventQueryParams{
		Source:      q.Get("source"),
		Type:        q.Get("type"),
		Severities:  splitQueryValues(q, "severity"),
		SourceIP:    q.Get("source_ip"),
		DestIP:      q.Get("dest_ip"),
		Protocols:   splitQueryValues(q, "protocol"),
		Username:    q.Get("username"),
		Process:     q.Get("process"),
		CmdContains: q.Get("cmd_contains"),
		FileHash:    q.Get("file_hash"),
		Search:      q.Get("search"),
		MatchedRule: q.Get("matched_rule"),
		Sort:        q.Get("sort"),
		Order:       q.Get("order"),
	}
	if v := q.Get("from"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			return nil, err
		}
		params.From = &ts
	}
	if v := q.Get("to"); v != "" {
		ts, err := parseFlexibleTime(v)
		if err != nil {
			return nil, err
		}
		params.To = &ts
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()
	return params, nil
}
