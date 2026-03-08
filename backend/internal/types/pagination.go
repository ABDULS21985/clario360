package types

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PaginationRequest holds pagination parameters from the request.
type PaginationRequest struct {
	Page     int    `json:"page"`
	PerPage  int    `json:"per_page"`
	SortBy   string `json:"sort_by,omitempty"`
	SortDir  string `json:"sort_dir,omitempty"` // "asc" or "desc"
}

// Offset returns the SQL offset for the current page.
func (p PaginationRequest) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the SQL limit.
func (p PaginationRequest) Limit() int {
	return p.PerPage
}

// PaginationMeta holds pagination metadata in the response.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResult wraps a list of items with pagination metadata.
type PaginatedResult[T any] struct {
	Data []T            `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// NewPaginatedResult creates a paginated result from items and total count.
func NewPaginatedResult[T any](items []T, total int64, req PaginationRequest) PaginatedResult[T] {
	totalPages := int(total) / req.PerPage
	if int(total)%req.PerPage > 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	if items == nil {
		items = []T{}
	}
	return PaginatedResult[T]{
		Data: items,
		Meta: PaginationMeta{
			Page:       req.Page,
			PerPage:    req.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// PaginationFromRequest extracts pagination parameters from an HTTP request.
func PaginationFromRequest(r *http.Request) PaginationRequest {
	page := queryInt(r, "page", 1)
	if page < 1 {
		page = 1
	}

	perPage := queryInt(r, "per_page", 0)
	if perPage == 0 {
		perPage = queryInt(r, "page_size", DefaultPageSize)
	}
	if perPage < 1 {
		perPage = DefaultPageSize
	}
	if perPage > MaxPageSize {
		perPage = MaxPageSize
	}

	sortDir := r.URL.Query().Get("sort_dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	return PaginationRequest{
		Page:     page,
		PerPage:  perPage,
		SortBy:   r.URL.Query().Get("sort_by"),
		SortDir:  sortDir,
	}
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}
