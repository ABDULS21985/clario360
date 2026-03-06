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
	PageSize int    `json:"page_size"`
	SortBy   string `json:"sort_by,omitempty"`
	SortDir  string `json:"sort_dir,omitempty"` // "asc" or "desc"
}

// Offset returns the SQL offset for the current page.
func (p PaginationRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the SQL limit.
func (p PaginationRequest) Limit() int {
	return p.PageSize
}

// PaginationResponse holds pagination metadata in the response.
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResult wraps a list of items with pagination metadata.
type PaginatedResult[T any] struct {
	Items      []T                `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

// NewPaginatedResult creates a paginated result from items and total count.
func NewPaginatedResult[T any](items []T, total int64, req PaginationRequest) PaginatedResult[T] {
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}
	if items == nil {
		items = []T{}
	}
	return PaginatedResult[T]{
		Items: items,
		Pagination: PaginationResponse{
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalItems: total,
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

	pageSize := queryInt(r, "page_size", DefaultPageSize)
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	sortDir := r.URL.Query().Get("sort_dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	return PaginationRequest{
		Page:     page,
		PageSize: pageSize,
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
