package dto

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	intmodel "github.com/clario360/platform/internal/integration/model"
)

type CreateIntegrationRequest struct {
	Type         intmodel.IntegrationType `json:"type"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	Config       map[string]any           `json:"config"`
	EventFilters []intmodel.EventFilter   `json:"event_filters"`
}

func (r *CreateIntegrationRequest) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("type is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if r.Config == nil {
		r.Config = map[string]any{}
	}
	return nil
}

type UpdateIntegrationRequest struct {
	Name         *string                  `json:"name,omitempty"`
	Description  *string                  `json:"description,omitempty"`
	Config       map[string]any           `json:"config,omitempty"`
	EventFilters []intmodel.EventFilter   `json:"event_filters,omitempty"`
}

type UpdateStatusRequest struct {
	Status intmodel.IntegrationStatus `json:"status"`
}

func (r *UpdateStatusRequest) Validate() error {
	switch r.Status {
	case intmodel.IntegrationStatusActive, intmodel.IntegrationStatusInactive:
		return nil
	default:
		return fmt.Errorf("status must be active or inactive")
	}
}

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

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

type ListQuery struct {
	Page    int
	PerPage int
	Search  string
	Type    string
	Status  string
	Sort    string
	Order   string
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

func ParseListQuery(r *http.Request) (*ListQuery, error) {
	values := r.URL.Query()
	query := &ListQuery{
		Page:    1,
		PerPage: 20,
		Sort:    "created_at",
		Order:   "desc",
	}

	if v := strings.TrimSpace(values.Get("search")); v != "" {
		query.Search = v
	}
	if v := strings.TrimSpace(values.Get("type")); v != "" {
		query.Type = v
	}
	// Support multi-value status: ?status=active&status=inactive
	if statuses := values["status"]; len(statuses) > 0 {
		query.Status = strings.Join(statuses, ",")
	}
	if v := strings.TrimSpace(values.Get("sort")); v != "" {
		switch v {
		case "created_at", "updated_at", "last_used_at", "delivery_count", "name", "type", "status":
			query.Sort = v
		default:
			return nil, fmt.Errorf("invalid sort field")
		}
	}
	if v := strings.TrimSpace(values.Get("order")); v != "" {
		switch strings.ToLower(v) {
		case "asc", "desc":
			query.Order = strings.ToLower(v)
		default:
			return nil, fmt.Errorf("invalid order")
		}
	}
	if v := strings.TrimSpace(values.Get("page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("page must be a positive integer")
		}
		query.Page = n
	}
	if v := strings.TrimSpace(values.Get("per_page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("per_page must be a positive integer")
		}
		if n > 100 {
			n = 100
		}
		query.PerPage = n
	}

	return query, nil
}

type DeliveryQuery struct {
	Page      int
	PerPage   int
	Status    string
	EventType string
	DateFrom  *time.Time
	DateTo    *time.Time
}

func (q DeliveryQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

func ParseDeliveryQuery(r *http.Request) (*DeliveryQuery, error) {
	values := r.URL.Query()
	query := &DeliveryQuery{
		Page:    1,
		PerPage: 20,
	}

	if v := strings.TrimSpace(values.Get("status")); v != "" {
		query.Status = v
	}
	if v := strings.TrimSpace(values.Get("event_type")); v != "" {
		query.EventType = v
	}
	if v := strings.TrimSpace(values.Get("date_from")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, fmt.Errorf("invalid date_from")
		}
		query.DateFrom = &t
	}
	if v := strings.TrimSpace(values.Get("date_to")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, fmt.Errorf("invalid date_to")
		}
		query.DateTo = &t
	}
	if v := strings.TrimSpace(values.Get("page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("page must be a positive integer")
		}
		query.Page = n
	}
	if v := strings.TrimSpace(values.Get("per_page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("per_page must be a positive integer")
		}
		if n > 100 {
			n = 100
		}
		query.PerPage = n
	}
	return query, nil
}

type TicketLinkQuery struct {
	IntegrationID string
	EntityType     string
	EntityID       string
	ExternalSystem string
}
