package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/middleware"
)

type envelope map[string]any

func requireTenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tenantStr := auth.TenantFromContext(r.Context())
	if tenantStr == "" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "tenant context is required")
		return uuid.Nil, uuid.Nil, false
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "invalid tenant ID")
		return uuid.Nil, uuid.Nil, false
	}
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return uuid.Nil, uuid.Nil, false
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "invalid user ID")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, userID, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "request body must be valid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(normalizePaginatedResponse(value))
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	if status >= http.StatusInternalServerError {
		code = "INTERNAL_ERROR"
		message = "internal server error"
	}
	writeJSON(w, status, map[string]any{
		"code":       code,
		"message":    message,
		"request_id": w.Header().Get(middleware.RequestIDHeader),
	})
}

func parseUUID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("invalid UUID: %s", raw))
		return uuid.Nil, false
	}
	return id, true
}

func parsePageParams(r *http.Request, defaultPerPage int) (int, int) {
	page := 1
	perPage := defaultPerPage
	if raw := r.URL.Query().Get("page"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			page = value
		}
	}
	if raw := r.URL.Query().Get("per_page"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			perPage = value
		}
	}
	return page, perPage
}

func normalizePaginatedResponse(value any) any {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return value
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return value
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return value
	}

	dataField := rv.FieldByName("Data")
	pageField := rv.FieldByName("Page")
	perPageField := rv.FieldByName("PerPage")
	totalField := rv.FieldByName("Total")
	totalPagesField := rv.FieldByName("TotalPages")
	if !dataField.IsValid() || !pageField.IsValid() || !perPageField.IsValid() || !totalField.IsValid() || !totalPagesField.IsValid() {
		return value
	}
	if pageField.Kind() != reflect.Int || perPageField.Kind() != reflect.Int || totalField.Kind() != reflect.Int || totalPagesField.Kind() != reflect.Int {
		return value
	}

	totalPages := int(totalPagesField.Int())
	if totalPages < 1 {
		totalPages = 1
	}

	return map[string]any{
		"data": dataField.Interface(),
		"meta": map[string]any{
			"page":        int(pageField.Int()),
			"per_page":    int(perPageField.Int()),
			"total":       int(totalField.Int()),
			"total_pages": totalPages,
		},
	}
}
