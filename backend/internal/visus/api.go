package visus

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
	allowedWidgetTypes = map[string]struct{}{"kpi_card": {}, "line_chart": {}, "bar_chart": {}, "pie_chart": {}, "table": {}, "heatmap": {}, "gauge": {}, "alert_feed": {}, "text": {}}
	allowedReportTypes = map[string]struct{}{"scheduled": {}, "on_demand": {}, "automated": {}}
)

type Dashboard struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Layout      json.RawMessage `json:"layout"`
	IsDefault   bool            `json:"is_default"`
	OwnerUserID *uuid.UUID      `json:"owner_user_id,omitempty"`
	SharedWith  json.RawMessage `json:"shared_with"`
	WidgetCount int             `json:"widget_count"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Widget struct {
	ID                     uuid.UUID       `json:"id"`
	DashboardID            uuid.UUID       `json:"dashboard_id"`
	Type                   string          `json:"type"`
	Title                  string          `json:"title"`
	Config                 json.RawMessage `json:"config"`
	Position               json.RawMessage `json:"position"`
	RefreshIntervalSeconds *int            `json:"refresh_interval_seconds,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

type Report struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Config          json.RawMessage `json:"config"`
	Schedule        *string         `json:"schedule,omitempty"`
	LastGeneratedAt *time.Time      `json:"last_generated_at,omitempty"`
	FileURL         *string         `json:"file_url,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ReportGeneration struct {
	ReportID    uuid.UUID       `json:"report_id"`
	SnapshotID  uuid.UUID       `json:"snapshot_id"`
	GeneratedAt time.Time       `json:"generated_at"`
	FileURL     string          `json:"file_url"`
	Metadata    json.RawMessage `json:"metadata"`
}

type dashboardRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Layout      map[string]any   `json:"layout"`
	IsDefault   bool             `json:"is_default"`
	SharedWith  []map[string]any `json:"shared_with"`
}

type reportRequest struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Config   map[string]any `json:"config"`
	Schedule *string        `json:"schedule"`
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
	r.Get("/dashboards", handler.ListDashboards)
	r.Post("/dashboards", handler.CreateDashboard)
	r.Get("/dashboards/{id}", handler.GetDashboard)
	r.Put("/dashboards/{id}", handler.UpdateDashboard)
	r.Get("/reports", handler.ListReports)
	r.Post("/reports", handler.CreateReport)
	r.Post("/reports/{id}/generate", handler.GenerateReport)
	r.Get("/widgets", handler.ListWidgets)
}

func (h *Handler) ListDashboards(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListDashboards(r.Context(), tenantID, page, perPage)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req dashboardRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.CreateDashboard(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationVisus) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GetDashboard(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "dashboard not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) UpdateDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dashboardRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.UpdateDashboard(r.Context(), tenantID, userID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, errValidationVisus):
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, pgx.ErrNoRows):
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "dashboard not found", nil)
		default:
			h.writeInternalError(w, r, err)
		}
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListReports(r.Context(), tenantID, page, perPage)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req reportRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	item, err := h.service.CreateReport(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationVisus) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *Handler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GenerateReport(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "report not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusAccepted, item)
}

func (h *Handler) ListWidgets(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	dashboardID := r.URL.Query().Get("dashboard_id")
	items, total, err := h.service.ListWidgets(r.Context(), tenantID, page, perPage, dashboardID)
	if err != nil {
		if errors.Is(err, errValidationVisus) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

var errValidationVisus = errors.New("validation failed")

func (s *Service) ListDashboards(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]Dashboard, int, error) {
	return s.repo.ListDashboards(ctx, tenantID, page, perPage)
}

func (s *Service) CreateDashboard(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req dashboardRequest) (*Dashboard, error) {
	req = normalizeDashboard(req)
	if err := validateDashboard(req); err != nil {
		return nil, err
	}
	return s.repo.CreateDashboard(ctx, tenantID, userID, req)
}

func (s *Service) GetDashboard(ctx context.Context, tenantID, id uuid.UUID) (*Dashboard, error) {
	return s.repo.GetDashboard(ctx, tenantID, id)
}

func (s *Service) UpdateDashboard(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req dashboardRequest) (*Dashboard, error) {
	req = normalizeDashboard(req)
	if err := validateDashboard(req); err != nil {
		return nil, err
	}
	return s.repo.UpdateDashboard(ctx, tenantID, userID, id, req)
}

func (s *Service) ListReports(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]Report, int, error) {
	return s.repo.ListReports(ctx, tenantID, page, perPage)
}

func (s *Service) CreateReport(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req reportRequest) (*Report, error) {
	req = normalizeReport(req)
	if err := validateReport(req); err != nil {
		return nil, err
	}
	return s.repo.CreateReport(ctx, tenantID, userID, req)
}

func (s *Service) GenerateReport(ctx context.Context, tenantID, id uuid.UUID) (*ReportGeneration, error) {
	return s.repo.GenerateReport(ctx, tenantID, id)
}

func (s *Service) ListWidgets(ctx context.Context, tenantID uuid.UUID, page, perPage int, dashboardID string) ([]Widget, int, error) {
	var dashboardUUID *uuid.UUID
	if strings.TrimSpace(dashboardID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(dashboardID))
		if err != nil {
			return nil, 0, fmt.Errorf("%w: invalid dashboard_id", errValidationVisus)
		}
		dashboardUUID = &id
	}
	return s.repo.ListWidgets(ctx, tenantID, page, perPage, dashboardUUID)
}

func (r *Repository) ListDashboards(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]Dashboard, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.description, a.layout, a.is_default, a.owner_user_id, a.shared_with,
		       COALESCE((SELECT COUNT(*) FROM dashboard_widgets w WHERE w.tenant_id = a.tenant_id AND w.dashboard_id = a.id), 0),
		       a.created_at, a.updated_at
		FROM dashboards a`)
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
	items := make([]Dashboard, 0, perPage)
	for rows.Next() {
		var item Dashboard
		var layout, sharedWith []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &layout, &item.IsDefault, &item.OwnerUserID, &sharedWith, &item.WidgetCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Layout = normalizeJSON(layout, "{}")
		item.SharedWith = normalizeJSON(sharedWith, "[]")
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

func (r *Repository) CreateDashboard(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req dashboardRequest) (*Dashboard, error) {
	layout, _ := json.Marshal(req.Layout)
	sharedWith, _ := json.Marshal(req.SharedWith)
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if req.IsDefault {
		if _, err = tx.Exec(ctx, `UPDATE dashboards SET is_default = false WHERE tenant_id = $1`, tenantID); err != nil {
			return nil, err
		}
	}
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `
		INSERT INTO dashboards (tenant_id, name, description, layout, is_default, owner_user_id, shared_with, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $6, $6)
		RETURNING id`,
		tenantID, req.Name, req.Description, layout, req.IsDefault, userID, sharedWith,
	).Scan(&id); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetDashboard(ctx, tenantID, id)
}

func (r *Repository) GetDashboard(ctx context.Context, tenantID, id uuid.UUID) (*Dashboard, error) {
	var item Dashboard
	var layout, sharedWith []byte
	if err := r.db.QueryRow(ctx, `
		SELECT a.id, a.name, a.description, a.layout, a.is_default, a.owner_user_id, a.shared_with,
		       COALESCE((SELECT COUNT(*) FROM dashboard_widgets w WHERE w.tenant_id = a.tenant_id AND w.dashboard_id = a.id), 0),
		       a.created_at, a.updated_at
		FROM dashboards a
		WHERE a.tenant_id = $1 AND a.id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.Name, &item.Description, &layout, &item.IsDefault, &item.OwnerUserID, &sharedWith, &item.WidgetCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Layout = normalizeJSON(layout, "{}")
	item.SharedWith = normalizeJSON(sharedWith, "[]")
	return &item, nil
}

func (r *Repository) UpdateDashboard(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req dashboardRequest) (*Dashboard, error) {
	layout, _ := json.Marshal(req.Layout)
	sharedWith, _ := json.Marshal(req.SharedWith)
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if req.IsDefault {
		if _, err = tx.Exec(ctx, `UPDATE dashboards SET is_default = false WHERE tenant_id = $1 AND id <> $2`, tenantID, id); err != nil {
			return nil, err
		}
	}
	tag, err := tx.Exec(ctx, `
		UPDATE dashboards
		SET name = $3, description = $4, layout = $5, is_default = $6, shared_with = $7, updated_by = $8
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, req.Name, req.Description, layout, req.IsDefault, sharedWith, userID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetDashboard(ctx, tenantID, id)
}

func (r *Repository) ListReports(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]Report, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.type::text, a.config, a.schedule, a.last_generated_at, a.file_url, a.created_at, a.updated_at
		FROM reports a`)
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
	items := make([]Report, 0, perPage)
	for rows.Next() {
		var item Report
		var config []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &config, &item.Schedule, &item.LastGeneratedAt, &item.FileURL, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Config = normalizeJSON(config, "{}")
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

func (r *Repository) CreateReport(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req reportRequest) (*Report, error) {
	config, _ := json.Marshal(req.Config)
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO reports (tenant_id, name, type, config, schedule, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id`,
		tenantID, req.Name, req.Type, config, req.Schedule, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.getReport(ctx, tenantID, id)
}

func (r *Repository) GenerateReport(ctx context.Context, tenantID, id uuid.UUID) (*ReportGeneration, error) {
	report, err := r.getReport(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	var kpiCount, executiveAlerts, snapshotsLast30d int
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM kpi_definitions WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM executive_alerts WHERE tenant_id = $1 AND status IN ('new','viewed')),
			(SELECT COUNT(*) FROM kpi_snapshots WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '30 days')`,
		tenantID,
	).Scan(&kpiCount, &executiveAlerts, &snapshotsLast30d); err != nil {
		return nil, err
	}
	snapshotID := uuid.New()
	generatedAt := time.Now().UTC()
	fileURL := fmt.Sprintf("/api/v1/visus/reports/%s/snapshots/%s", id, snapshotID)
	metadata, _ := json.Marshal(map[string]any{
		"report_name":          report.Name,
		"kpi_definition_count": kpiCount,
		"open_exec_alerts":     executiveAlerts,
		"kpi_snapshots_30d":    snapshotsLast30d,
		"config":               json.RawMessage(report.Config),
	})
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if _, err = tx.Exec(ctx, `
		INSERT INTO report_snapshots (id, tenant_id, report_id, generated_at, file_url, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		snapshotID, tenantID, id, generatedAt, fileURL, metadata,
	); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `
		UPDATE reports
		SET last_generated_at = $3, file_url = $4
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, generatedAt, fileURL,
	); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &ReportGeneration{
		ReportID:    id,
		SnapshotID:  snapshotID,
		GeneratedAt: generatedAt,
		FileURL:     fileURL,
		Metadata:    metadata,
	}, nil
}

func (r *Repository) ListWidgets(ctx context.Context, tenantID uuid.UUID, page, perPage int, dashboardID *uuid.UUID) ([]Widget, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.dashboard_id, a.type::text, a.title, a.config, a.position, a.refresh_interval_seconds, a.created_at, a.updated_at
		FROM dashboard_widgets a`)
	qb.Where("a.tenant_id = ?", tenantID)
	if dashboardID != nil {
		qb.Where("a.dashboard_id = ?", *dashboardID)
	}
	qb.OrderBy("created_at", "desc", []string{"created_at", "title"})
	qb.Paginate(page, perPage)
	query, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Widget, 0, perPage)
	for rows.Next() {
		var item Widget
		var config, position []byte
		if err := rows.Scan(&item.ID, &item.DashboardID, &item.Type, &item.Title, &config, &position, &item.RefreshIntervalSeconds, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Config = normalizeJSON(config, "{}")
		item.Position = normalizeJSON(position, "{}")
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

func (r *Repository) getReport(ctx context.Context, tenantID, id uuid.UUID) (*Report, error) {
	var item Report
	var config []byte
	if err := r.db.QueryRow(ctx, `
		SELECT id, name, type::text, config, schedule, last_generated_at, file_url, created_at, updated_at
		FROM reports
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.Name, &item.Type, &config, &item.Schedule, &item.LastGeneratedAt, &item.FileURL, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Config = normalizeJSON(config, "{}")
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
	h.logger.Error().Err(err).Msg("visus service request failed")
	suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "request failed", nil)
}

func validateDashboard(req dashboardRequest) error {
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", errValidationVisus)
	}
	return nil
}

func validateReport(req reportRequest) error {
	if req.Name == "" {
		return fmt.Errorf("%w: name is required", errValidationVisus)
	}
	if _, ok := allowedReportTypes[req.Type]; !ok {
		return fmt.Errorf("%w: invalid report type", errValidationVisus)
	}
	return nil
}

func normalizeDashboard(req dashboardRequest) dashboardRequest {
	req.Name = strings.TrimSpace(req.Name)
	return req
}

func normalizeReport(req reportRequest) reportRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	return req
}

func normalizeJSON(value []byte, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(value)
}
