package dto

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// QueryParams holds validated list query parameters.
type QueryParams struct {
	TenantID string
	UserID   string
	Category string
	Type     string
	Priority string
	Read     *bool // nil=all, true=read only, false=unread only
	DateFrom *time.Time
	DateTo   *time.Time
	Sort     string
	Order    string
	Page     int
	PerPage  int
}

// Offset returns the SQL offset for pagination.
func (q *QueryParams) Offset() int {
	return (q.Page - 1) * q.PerPage
}

var validSortFields = map[string]bool{
	"created_at": true,
	"priority":   true,
}

var validCategories = map[string]bool{
	"security": true, "data": true, "governance": true,
	"legal": true, "system": true, "workflow": true,
}

var validPriorities = map[string]bool{
	"critical": true, "high": true, "medium": true, "low": true,
}

// ParseQueryParams extracts and validates query parameters from an HTTP request.
func ParseQueryParams(r *http.Request) (*QueryParams, error) {
	q := r.URL.Query()

	qp := &QueryParams{
		Sort:    "created_at",
		Order:   "desc",
		Page:    1,
		PerPage: 20,
	}

	if v := q.Get("category"); v != "" {
		if !validCategories[v] {
			return nil, fmt.Errorf("invalid category %q; allowed: security, data, governance, legal, system, workflow", v)
		}
		qp.Category = v
	}

	if v := q.Get("type"); v != "" {
		qp.Type = v
	}

	if v := q.Get("priority"); v != "" {
		if !validPriorities[v] {
			return nil, fmt.Errorf("invalid priority %q; allowed: critical, high, medium, low", v)
		}
		qp.Priority = v
	}

	if v := q.Get("read"); v != "" {
		switch strings.ToLower(v) {
		case "true":
			b := true
			qp.Read = &b
		case "false":
			b := false
			qp.Read = &b
		default:
			return nil, fmt.Errorf("invalid read filter %q; allowed: true, false", v)
		}
	}

	if v := q.Get("date_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, fmt.Errorf("invalid date_from: %w", err)
		}
		qp.DateFrom = &t
	}

	if v := q.Get("date_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, fmt.Errorf("invalid date_to: %w", err)
		}
		qp.DateTo = &t
	}

	if qp.DateFrom != nil && qp.DateTo != nil && qp.DateTo.Before(*qp.DateFrom) {
		return nil, fmt.Errorf("date_to must not be before date_from")
	}

	if v := q.Get("sort"); v != "" {
		if !validSortFields[v] {
			return nil, fmt.Errorf("invalid sort field %q; allowed: created_at, priority", v)
		}
		qp.Sort = v
	}

	if v := q.Get("order"); v != "" {
		lower := strings.ToLower(v)
		if lower != "asc" && lower != "desc" {
			return nil, fmt.Errorf("invalid order %q; allowed: asc, desc", v)
		}
		qp.Order = lower
	}

	if v := q.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			return nil, fmt.Errorf("page must be a positive integer")
		}
		qp.Page = p
	}

	if v := q.Get("per_page"); v != "" {
		pp, err := strconv.Atoi(v)
		if err != nil || pp < 1 {
			return nil, fmt.Errorf("per_page must be a positive integer")
		}
		if pp > 100 {
			pp = 100
		}
		qp.PerPage = pp
	}

	return qp, nil
}

// Pagination holds pagination metadata for list responses.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPagination creates pagination metadata from query and total count.
func NewPagination(page, perPage, total int) Pagination {
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}
	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// MarkReadRequest represents a bulk mark-as-read request body.
type MarkReadRequest struct {
	IDs []string `json:"ids,omitempty"` // if empty, mark all as read
}
