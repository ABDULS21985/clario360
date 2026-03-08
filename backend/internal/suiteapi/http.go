package suiteapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/middleware"
)

const (
	defaultPageSize = 25
	maxPageSize     = 200
)

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type DataEnvelope struct {
	Data any `json:"data"`
}

type PaginatedEnvelope struct {
	Data any        `json:"data"`
	Meta Pagination `json:"meta"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, DataEnvelope{Data: data})
}

func WritePaginated(w http.ResponseWriter, status int, data any, page, perPage, total int) {
	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}
	if totalPages < 1 {
		totalPages = 1
	}
	WriteJSON(w, status, PaginatedEnvelope{
		Data: data,
		Meta: Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	WriteJSON(w, status, ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: middleware.GetRequestID(r.Context()),
	})
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func ParsePagination(r *http.Request) (page, perPage int) {
	page = parsePositiveInt(r.URL.Query().Get("page"), 1)
	perPage = parsePositiveInt(r.URL.Query().Get("per_page"), defaultPageSize)
	if perPage > maxPageSize {
		perPage = maxPageSize
	}
	return page, perPage
}

func ParseCSVParam(r *http.Request, key string) []string {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func UUIDParam(r *http.Request, key string) (uuid.UUID, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return uuid.Nil, fmt.Errorf("missing %s parameter", key)
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func TenantID(r *http.Request) (uuid.UUID, error) {
	tenant := auth.TenantFromContext(r.Context())
	if tenant == "" {
		return uuid.Nil, errors.New("missing tenant context")
	}
	tenantID, err := uuid.Parse(tenant)
	if err != nil {
		return uuid.Nil, errors.New("invalid tenant context")
	}
	return tenantID, nil
}

func UserID(r *http.Request) (*uuid.UUID, error) {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.ID == "" {
		return nil, nil
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, errors.New("invalid user context")
	}
	return &userID, nil
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
