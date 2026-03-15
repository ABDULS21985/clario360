package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// EventHandler handles security event query endpoints.
type EventHandler struct {
	ruleRepo *repository.RuleRepository
	logger   zerolog.Logger
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(ruleRepo *repository.RuleRepository, logger zerolog.Logger) *EventHandler {
	return &EventHandler{ruleRepo: ruleRepo, logger: logger}
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
	events, total, err := h.ruleRepo.QuerySecurityEvents(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("list security events failed")
		writeError(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{
		"data": events,
		"meta": envelope{
			"total":    total,
			"page":     params.Page,
			"per_page": params.PerPage,
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
	event, err := h.ruleRepo.GetSecurityEvent(r.Context(), tenantID, eventID)
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

	stats, err := h.ruleRepo.GetSecurityEventStats(r.Context(), tenantID, params.From, params.To)
	if err != nil {
		h.logger.Error().Err(err).Msg("get event stats failed")
		writeError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": stats})
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
