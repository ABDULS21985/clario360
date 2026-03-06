package dto

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AllowedSortFields are the only fields that can be used for sorting.
var AllowedSortFields = map[string]bool{
	"created_at": true,
	"action":     true,
	"severity":   true,
	"service":    true,
}

// QueryParams holds validated query parameters for audit log searches.
type QueryParams struct {
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	Service      string    `json:"service"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	DateFrom     time.Time `json:"date_from"`
	DateTo       time.Time `json:"date_to"`
	Search       string    `json:"search"`
	Severity     string    `json:"severity"`
	Sort         string    `json:"sort"`
	Order        string    `json:"order"`
	Page         int       `json:"page"`
	PerPage      int       `json:"per_page"`
}

const (
	defaultPerPage = 50
	maxPerPage     = 200
	maxDateRange   = 93 * 24 * time.Hour // 93 days = ~1 quarter
)

// ParseQueryParams extracts and validates query parameters from an HTTP request.
func ParseQueryParams(r *http.Request) (*QueryParams, error) {
	q := r.URL.Query()

	qp := &QueryParams{
		TenantID:     q.Get("tenant_id"),
		UserID:       q.Get("user_id"),
		Service:      q.Get("service"),
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		ResourceID:   q.Get("resource_id"),
		Search:       q.Get("search"),
		Severity:     q.Get("severity"),
		Sort:         q.Get("sort"),
		Order:        q.Get("order"),
	}

	// Parse date_from (required)
	dateFromStr := q.Get("date_from")
	if dateFromStr == "" {
		return nil, fmt.Errorf("date_from is required for audit log queries")
	}
	dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date_from format, expected RFC3339: %w", err)
	}
	qp.DateFrom = dateFrom

	// Parse date_to (defaults to now)
	dateToStr := q.Get("date_to")
	if dateToStr != "" {
		dateTo, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date_to format, expected RFC3339: %w", err)
		}
		qp.DateTo = dateTo
	} else {
		qp.DateTo = time.Now().UTC()
	}

	// Parse page
	if pageStr := q.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			qp.Page = 1
		} else {
			qp.Page = page
		}
	} else {
		qp.Page = 1
	}

	// Parse per_page
	if ppStr := q.Get("per_page"); ppStr != "" {
		pp, err := strconv.Atoi(ppStr)
		if err != nil || pp < 1 {
			qp.PerPage = defaultPerPage
		} else {
			qp.PerPage = pp
		}
	} else {
		qp.PerPage = defaultPerPage
	}

	return qp, qp.Validate()
}

// Validate checks that all query parameters are within acceptable bounds.
func (qp *QueryParams) Validate() error {
	// Date range must not exceed maxDateRange
	if qp.DateTo.Sub(qp.DateFrom) > maxDateRange {
		return fmt.Errorf("date range must not exceed 93 days")
	}
	if qp.DateTo.Before(qp.DateFrom) {
		return fmt.Errorf("date_to must be after date_from")
	}

	// Sort field validation
	if qp.Sort == "" {
		qp.Sort = "created_at"
	}
	if !AllowedSortFields[qp.Sort] {
		return fmt.Errorf("invalid sort field %q; allowed: created_at, action, severity, service", qp.Sort)
	}

	// Order validation
	if qp.Order == "" {
		qp.Order = "desc"
	}
	qp.Order = strings.ToLower(qp.Order)
	if qp.Order != "asc" && qp.Order != "desc" {
		return fmt.Errorf("invalid order %q; allowed: asc, desc", qp.Order)
	}

	// Clamp per_page
	if qp.PerPage > maxPerPage {
		qp.PerPage = maxPerPage
	}
	if qp.PerPage < 1 {
		qp.PerPage = defaultPerPage
	}

	// Validate action wildcard: only trailing * allowed
	if qp.Action != "" && strings.Contains(qp.Action, "*") {
		if !strings.HasSuffix(qp.Action, "*") || strings.Count(qp.Action, "*") > 1 {
			return fmt.Errorf("action wildcard only supports trailing *")
		}
	}

	// Validate severity
	if qp.Severity != "" {
		validSeverities := map[string]bool{"info": true, "warning": true, "high": true, "critical": true}
		if !validSeverities[qp.Severity] {
			return fmt.Errorf("invalid severity %q; allowed: info, warning, high, critical", qp.Severity)
		}
	}

	// Sanitize search: strip SQL-unsafe characters
	if qp.Search != "" {
		qp.Search = sanitizeSearch(qp.Search)
	}

	return nil
}

// Offset computes the SQL OFFSET from page and per_page.
func (qp *QueryParams) Offset() int {
	return (qp.Page - 1) * qp.PerPage
}

// sanitizeSearch removes characters that could be dangerous in full-text queries.
func sanitizeSearch(s string) string {
	// Remove characters that are special in tsquery
	replacer := strings.NewReplacer(
		"'", "",
		"\"", "",
		"\\", "",
		"(", "",
		")", "",
		":", "",
		"!", "",
		"&", "",
		"|", "",
		"<", "",
		">", "",
		";", "",
		"--", "",
	)
	return strings.TrimSpace(replacer.Replace(s))
}

// PaginatedResult wraps query results with pagination metadata.
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination holds pagination metadata.
type Pagination struct {
	Page     int `json:"page"`
	PerPage  int `json:"per_page"`
	Total    int `json:"total"`
	LastPage int `json:"last_page"`
}

// NewPagination creates pagination metadata from query params and total count.
func NewPagination(page, perPage, total int) Pagination {
	lastPage := total / perPage
	if total%perPage > 0 {
		lastPage++
	}
	if lastPage < 1 {
		lastPage = 1
	}
	return Pagination{
		Page:     page,
		PerPage:  perPage,
		Total:    total,
		LastPage: lastPage,
	}
}
