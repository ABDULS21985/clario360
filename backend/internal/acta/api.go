package acta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/suiteapi"
)

var (
	allowedMeetingStatuses = map[string]struct{}{"scheduled": {}, "in_progress": {}, "completed": {}, "cancelled": {}}
	allowedMinuteStatuses  = map[string]struct{}{"draft": {}, "review": {}, "approved": {}, "published": {}}
	allowedWorkflowStatus  = map[string]struct{}{"active": {}, "inactive": {}, "archived": {}}
)

type Committee struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	Description      string          `json:"description"`
	Members          json.RawMessage `json:"members"`
	MeetingFrequency *string         `json:"meeting_frequency,omitempty"`
	Status           string          `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Meeting struct {
	ID              uuid.UUID       `json:"id"`
	CommitteeID     uuid.UUID       `json:"committee_id"`
	CommitteeName   string          `json:"committee_name"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	Location        *string         `json:"location,omitempty"`
	VirtualLink     *string         `json:"virtual_link,omitempty"`
	Status          string          `json:"status"`
	DurationMinutes *int            `json:"duration_minutes,omitempty"`
	Attendees       json.RawMessage `json:"attendees"`
	ActionItemCount int             `json:"action_item_count"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MeetingMinute struct {
	ID            uuid.UUID       `json:"id"`
	MeetingID     uuid.UUID       `json:"meeting_id"`
	MeetingTitle  string          `json:"meeting_title"`
	Content       string          `json:"content"`
	AISummary     *string         `json:"ai_summary,omitempty"`
	AIActionItems json.RawMessage `json:"ai_action_items"`
	Status        string          `json:"status"`
	ApprovedBy    *uuid.UUID      `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time      `json:"approved_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type ActionItem struct {
	ID           uuid.UUID  `json:"id"`
	MeetingID    uuid.UUID  `json:"meeting_id"`
	MeetingTitle string     `json:"meeting_title"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	Status       string     `json:"status"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type WorkflowTemplate struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Definition json.RawMessage `json:"definition"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type meetingRequest struct {
	CommitteeID     uuid.UUID `json:"committee_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	Location        *string   `json:"location"`
	VirtualLink     *string   `json:"virtual_link"`
	Status          string    `json:"status"`
	DurationMinutes *int      `json:"duration_minutes"`
}

type minuteRequest struct {
	MeetingID     uuid.UUID        `json:"meeting_id"`
	Content       string           `json:"content"`
	AISummary     *string          `json:"ai_summary"`
	AIActionItems []map[string]any `json:"ai_action_items"`
	Status        string           `json:"status"`
}

type workflowRequest struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Definition map[string]any `json:"definition"`
	Status     string         `json:"status"`
}

type Repository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

type Handler struct {
	service *Service
	logger  zerolog.Logger
}

func NewRepository(db *pgxpool.Pool, logger zerolog.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func MountRoutes(r chi.Router, db *pgxpool.Pool, logger zerolog.Logger) {
	handler := NewHandler(NewService(NewRepository(db, logger), logger), logger)

	r.Get("/committees", handler.ListCommittees)
	r.Get("/meetings", handler.ListMeetings)
	r.Post("/meetings", handler.CreateMeeting)
	r.Get("/meetings/{id}", handler.GetMeeting)
	r.Put("/meetings/{id}", handler.UpdateMeeting)
	r.Get("/action-items", handler.ListActionItems)

	r.Get("/documents", handler.ListDocuments)
	r.Post("/documents", handler.CreateDocument)
	r.Get("/documents/{id}", handler.GetDocument)
	r.Put("/documents/{id}", handler.UpdateDocument)
	r.Delete("/documents/{id}", handler.DeleteDocument)
	r.Post("/documents/{id}/sign", handler.SignDocument)

	r.Get("/templates", handler.ListTemplates)
	r.Post("/templates", handler.CreateTemplate)
}

func (h *Handler) ListCommittees(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListCommittees(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) ListMeetings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListMeetings(r.Context(), tenantID, page, perPage, r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateMeeting(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req meetingRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.CreateMeeting(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationActa) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *Handler) GetMeeting(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GetMeeting(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "meeting not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) UpdateMeeting(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req meetingRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.UpdateMeeting(r.Context(), tenantID, userID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, errValidationActa):
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, pgx.ErrNoRows):
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "meeting not found", nil)
		default:
			h.writeInternalError(w, r, err)
		}
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) ListActionItems(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListActionItems(r.Context(), tenantID, page, perPage, r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListDocuments(r.Context(), tenantID, page, perPage)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req minuteRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.CreateDocument(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationActa) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GetDocument(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "document not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req minuteRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.UpdateDocument(r.Context(), tenantID, userID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, errValidationActa):
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, pgx.ErrNoRows):
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "document not found", nil)
		default:
			h.writeInternalError(w, r, err)
		}
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if err := h.service.DeleteDocument(r.Context(), tenantID, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "document not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SignDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.SignDocument(r.Context(), tenantID, userID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "document not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListTemplates(r.Context(), tenantID, page, perPage)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req workflowRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.CreateTemplate(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationActa) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

var errValidationActa = errors.New("validation failed")

func (s *Service) ListCommittees(ctx context.Context, tenantID uuid.UUID, page, perPage int, search string) ([]Committee, int, error) {
	return s.repo.ListCommittees(ctx, tenantID, page, perPage, search)
}

func (s *Service) ListMeetings(ctx context.Context, tenantID uuid.UUID, page, perPage int, status string) ([]Meeting, int, error) {
	return s.repo.ListMeetings(ctx, tenantID, page, perPage, status)
}

func (s *Service) CreateMeeting(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req meetingRequest) (*Meeting, error) {
	req = normalizeMeeting(req)
	if err := validateMeeting(req); err != nil {
		return nil, err
	}
	return s.repo.CreateMeeting(ctx, tenantID, userID, req)
}

func (s *Service) GetMeeting(ctx context.Context, tenantID, id uuid.UUID) (*Meeting, error) {
	return s.repo.GetMeeting(ctx, tenantID, id)
}

func (s *Service) UpdateMeeting(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req meetingRequest) (*Meeting, error) {
	req = normalizeMeeting(req)
	if err := validateMeeting(req); err != nil {
		return nil, err
	}
	return s.repo.UpdateMeeting(ctx, tenantID, userID, id, req)
}

func (s *Service) ListActionItems(ctx context.Context, tenantID uuid.UUID, page, perPage int, status string) ([]ActionItem, int, error) {
	return s.repo.ListActionItems(ctx, tenantID, page, perPage, status)
}

func (s *Service) ListDocuments(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]MeetingMinute, int, error) {
	return s.repo.ListDocuments(ctx, tenantID, page, perPage)
}

func (s *Service) CreateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req minuteRequest) (*MeetingMinute, error) {
	req = normalizeMinute(req)
	if err := validateMinute(req); err != nil {
		return nil, err
	}
	return s.repo.CreateDocument(ctx, tenantID, userID, req)
}

func (s *Service) GetDocument(ctx context.Context, tenantID, id uuid.UUID) (*MeetingMinute, error) {
	return s.repo.GetDocument(ctx, tenantID, id)
}

func (s *Service) UpdateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req minuteRequest) (*MeetingMinute, error) {
	req = normalizeMinute(req)
	if err := validateMinute(req); err != nil {
		return nil, err
	}
	return s.repo.UpdateDocument(ctx, tenantID, userID, id, req)
}

func (s *Service) DeleteDocument(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteDocument(ctx, tenantID, id)
}

func (s *Service) SignDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID) (*MeetingMinute, error) {
	return s.repo.SignDocument(ctx, tenantID, userID, id)
}

func (s *Service) ListTemplates(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]WorkflowTemplate, int, error) {
	return s.repo.ListTemplates(ctx, tenantID, page, perPage)
}

func (s *Service) CreateTemplate(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req workflowRequest) (*WorkflowTemplate, error) {
	req = normalizeWorkflow(req)
	if err := validateWorkflow(req); err != nil {
		return nil, err
	}
	return s.repo.CreateTemplate(ctx, tenantID, userID, req)
}

func (r *Repository) ListCommittees(ctx context.Context, tenantID uuid.UUID, page, perPage int, search string) ([]Committee, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.type, a.description, a.members, a.meeting_frequency, a.status::text, a.created_at, a.updated_at
		FROM committees a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.OrderBy("created_at", "desc", []string{"created_at", "name"})
	qb.Paginate(page, perPage)
	query, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Committee, 0, perPage)
	for rows.Next() {
		var item Committee
		var members []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Description, &members, &item.MeetingFrequency, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Members = normalizeJSON(members, "[]")
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) ListMeetings(ctx context.Context, tenantID uuid.UUID, page, perPage int, status string) ([]Meeting, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.committee_id, c.name, a.title, a.description, a.scheduled_at, a.location, a.virtual_link,
		       a.status::text, a.duration_minutes, c.members,
		       COALESCE((SELECT COUNT(*) FROM action_items ai WHERE ai.tenant_id = a.tenant_id AND ai.meeting_id = a.id), 0),
		       a.created_at, a.updated_at
		FROM meetings a
		JOIN committees c ON c.id = a.committee_id AND c.tenant_id = a.tenant_id`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(status) != "", "a.status::text = ?", strings.ToLower(strings.TrimSpace(status)))
	qb.OrderBy("created_at", "desc", []string{"created_at", "title", "status"})
	qb.Paginate(page, perPage)
	query, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Meeting, 0, perPage)
	for rows.Next() {
		var item Meeting
		var attendees []byte
		if err := rows.Scan(&item.ID, &item.CommitteeID, &item.CommitteeName, &item.Title, &item.Description, &item.ScheduledAt, &item.Location, &item.VirtualLink, &item.Status, &item.DurationMinutes, &attendees, &item.ActionItemCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Attendees = normalizeJSON(attendees, "[]")
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) CreateMeeting(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req meetingRequest) (*Meeting, error) {
	if _, err := r.getCommittee(ctx, tenantID, req.CommitteeID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: committee does not exist", errValidationActa)
		}
		return nil, err
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO meetings (tenant_id, committee_id, title, description, scheduled_at, location, virtual_link, status, duration_minutes, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id`,
		tenantID, req.CommitteeID, req.Title, req.Description, req.ScheduledAt, req.Location, req.VirtualLink, req.Status, req.DurationMinutes, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetMeeting(ctx, tenantID, id)
}

func (r *Repository) GetMeeting(ctx context.Context, tenantID, id uuid.UUID) (*Meeting, error) {
	var item Meeting
	var attendees []byte
	if err := r.db.QueryRow(ctx, `
		SELECT a.id, a.committee_id, c.name, a.title, a.description, a.scheduled_at, a.location, a.virtual_link,
		       a.status::text, a.duration_minutes, c.members,
		       COALESCE((SELECT COUNT(*) FROM action_items ai WHERE ai.tenant_id = a.tenant_id AND ai.meeting_id = a.id), 0),
		       a.created_at, a.updated_at
		FROM meetings a
		JOIN committees c ON c.id = a.committee_id AND c.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.CommitteeID, &item.CommitteeName, &item.Title, &item.Description, &item.ScheduledAt, &item.Location, &item.VirtualLink, &item.Status, &item.DurationMinutes, &attendees, &item.ActionItemCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Attendees = normalizeJSON(attendees, "[]")
	return &item, nil
}

func (r *Repository) UpdateMeeting(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req meetingRequest) (*Meeting, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE meetings
		SET committee_id = $3, title = $4, description = $5, scheduled_at = $6, location = $7, virtual_link = $8, status = $9, duration_minutes = $10, updated_by = $11
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, req.CommitteeID, req.Title, req.Description, req.ScheduledAt, req.Location, req.VirtualLink, req.Status, req.DurationMinutes, userID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetMeeting(ctx, tenantID, id)
}

func (r *Repository) ListActionItems(ctx context.Context, tenantID uuid.UUID, page, perPage int, status string) ([]ActionItem, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.meeting_id, m.title, a.title, a.description, a.assigned_to, a.due_date, a.status::text, a.completed_at, a.created_at, a.updated_at
		FROM action_items a
		JOIN meetings m ON m.id = a.meeting_id AND m.tenant_id = a.tenant_id`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(status) != "", "a.status::text = ?", strings.ToLower(strings.TrimSpace(status)))
	qb.OrderBy("created_at", "desc", []string{"created_at", "status", "title"})
	qb.Paginate(page, perPage)
	query, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]ActionItem, 0, perPage)
	for rows.Next() {
		var item ActionItem
		if err := rows.Scan(&item.ID, &item.MeetingID, &item.MeetingTitle, &item.Title, &item.Description, &item.AssignedTo, &item.DueDate, &item.Status, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) ListDocuments(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]MeetingMinute, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.meeting_id, m.title, a.content, a.ai_summary, a.ai_action_items, a.status::text, a.approved_by, a.approved_at, a.created_at, a.updated_at
		FROM meeting_minutes a
		JOIN meetings m ON m.id = a.meeting_id AND m.tenant_id = a.tenant_id`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.OrderBy("created_at", "desc", []string{"created_at"})
	qb.Paginate(page, perPage)
	query, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]MeetingMinute, 0, perPage)
	for rows.Next() {
		var item MeetingMinute
		var aiActionItems []byte
		if err := rows.Scan(&item.ID, &item.MeetingID, &item.MeetingTitle, &item.Content, &item.AISummary, &aiActionItems, &item.Status, &item.ApprovedBy, &item.ApprovedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.AIActionItems = normalizeJSON(aiActionItems, "[]")
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) CreateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req minuteRequest) (*MeetingMinute, error) {
	if _, err := r.GetMeeting(ctx, tenantID, req.MeetingID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: meeting does not exist", errValidationActa)
		}
		return nil, err
	}
	payload, _ := json.Marshal(req.AIActionItems)
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO meeting_minutes (tenant_id, meeting_id, content, ai_summary, ai_action_items, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id`,
		tenantID, req.MeetingID, req.Content, req.AISummary, payload, req.Status, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetDocument(ctx, tenantID, id)
}

func (r *Repository) GetDocument(ctx context.Context, tenantID, id uuid.UUID) (*MeetingMinute, error) {
	var item MeetingMinute
	var aiActionItems []byte
	if err := r.db.QueryRow(ctx, `
		SELECT a.id, a.meeting_id, m.title, a.content, a.ai_summary, a.ai_action_items, a.status::text, a.approved_by, a.approved_at, a.created_at, a.updated_at
		FROM meeting_minutes a
		JOIN meetings m ON m.id = a.meeting_id AND m.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.MeetingID, &item.MeetingTitle, &item.Content, &item.AISummary, &aiActionItems, &item.Status, &item.ApprovedBy, &item.ApprovedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.AIActionItems = normalizeJSON(aiActionItems, "[]")
	return &item, nil
}

func (r *Repository) UpdateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req minuteRequest) (*MeetingMinute, error) {
	payload, _ := json.Marshal(req.AIActionItems)
	tag, err := r.db.Exec(ctx, `
		UPDATE meeting_minutes
		SET meeting_id = $3, content = $4, ai_summary = $5, ai_action_items = $6, status = $7, updated_by = $8
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, req.MeetingID, req.Content, req.AISummary, payload, req.Status, userID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetDocument(ctx, tenantID, id)
}

func (r *Repository) DeleteDocument(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM meeting_minutes WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) SignDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID) (*MeetingMinute, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE meeting_minutes
		SET status = 'approved', approved_by = $3, approved_at = NOW(), updated_by = $3
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, userID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetDocument(ctx, tenantID, id)
}

func (r *Repository) ListTemplates(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]WorkflowTemplate, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.type, a.definition, a.status::text, a.created_at, a.updated_at
		FROM governance_workflows a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.OrderBy("created_at", "desc", []string{"created_at", "name"})
	qb.Paginate(page, perPage)
	query, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]WorkflowTemplate, 0, perPage)
	for rows.Next() {
		var item WorkflowTemplate
		var definition []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &definition, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Definition = normalizeJSON(definition, "{}")
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) CreateTemplate(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req workflowRequest) (*WorkflowTemplate, error) {
	definition, _ := json.Marshal(req.Definition)
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO governance_workflows (tenant_id, name, type, definition, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id`,
		tenantID, req.Name, req.Type, definition, req.Status, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.getTemplate(ctx, tenantID, id)
}

func (r *Repository) getCommittee(ctx context.Context, tenantID, id uuid.UUID) (*Committee, error) {
	var item Committee
	var members []byte
	if err := r.db.QueryRow(ctx, `
		SELECT id, name, type, description, members, meeting_frequency, status::text, created_at, updated_at
		FROM committees
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.Name, &item.Type, &item.Description, &members, &item.MeetingFrequency, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Members = normalizeJSON(members, "[]")
	return &item, nil
}

func (r *Repository) getTemplate(ctx context.Context, tenantID, id uuid.UUID) (*WorkflowTemplate, error) {
	var item WorkflowTemplate
	var definition []byte
	if err := r.db.QueryRow(ctx, `
		SELECT id, name, type, definition, status::text, created_at, updated_at
		FROM governance_workflows
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.Name, &item.Type, &definition, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Definition = normalizeJSON(definition, "{}")
	return &item, nil
}

func (h *Handler) tenantOnly(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

func (h *Handler) tenantUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, *uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, nil, false
	}
	userID, err := suiteapi.UserID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, nil, false
	}
	return tenantID, userID, true
}

func (h *Handler) writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().Err(err).Msg("acta service request failed")
	suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "request failed", nil)
}

func validateMeeting(req meetingRequest) error {
	if req.CommitteeID == uuid.Nil {
		return fmt.Errorf("%w: committee_id is required", errValidationActa)
	}
	if req.Title == "" {
		return fmt.Errorf("%w: title is required", errValidationActa)
	}
	if _, ok := allowedMeetingStatuses[req.Status]; !ok {
		return fmt.Errorf("%w: invalid meeting status", errValidationActa)
	}
	return nil
}

func validateMinute(req minuteRequest) error {
	if req.MeetingID == uuid.Nil {
		return fmt.Errorf("%w: meeting_id is required", errValidationActa)
	}
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("%w: content is required", errValidationActa)
	}
	if _, ok := allowedMinuteStatuses[req.Status]; !ok {
		return fmt.Errorf("%w: invalid document status", errValidationActa)
	}
	return nil
}

func validateWorkflow(req workflowRequest) error {
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", errValidationActa)
	}
	if strings.TrimSpace(req.Type) == "" {
		return fmt.Errorf("%w: type is required", errValidationActa)
	}
	if _, ok := allowedWorkflowStatus[req.Status]; !ok {
		return fmt.Errorf("%w: invalid template status", errValidationActa)
	}
	return nil
}

func normalizeMeeting(req meetingRequest) meetingRequest {
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "scheduled"
	}
	return req
}

func normalizeMinute(req minuteRequest) minuteRequest {
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "draft"
	}
	return req
}

func normalizeWorkflow(req workflowRequest) workflowRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "active"
	}
	return req
}

func normalizeJSON(value []byte, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(value)
}
