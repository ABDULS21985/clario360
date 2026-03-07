package data

import (
	"context"
	"database/sql"
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
	allowedSourceTypes   = map[string]struct{}{"database": {}, "api": {}, "file": {}, "stream": {}, "cloud_storage": {}}
	allowedSourceStatus  = map[string]struct{}{"active": {}, "inactive": {}, "error": {}, "syncing": {}}
	allowedPipelineTypes = map[string]struct{}{"etl": {}, "elt": {}, "streaming": {}, "batch": {}}
	allowedPipelineState = map[string]struct{}{"active": {}, "paused": {}, "failed": {}, "completed": {}}
)

type Source struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	ConnectionConfig json.RawMessage `json:"connection_config"`
	SchemaMetadata   json.RawMessage `json:"schema_metadata"`
	LastSyncedAt     *time.Time      `json:"last_synced_at,omitempty"`
	SyncFrequency    string          `json:"sync_frequency,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Pipeline struct {
	ID                 uuid.UUID       `json:"id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Type               string          `json:"type"`
	Status             string          `json:"status"`
	SourceID           *uuid.UUID      `json:"source_id,omitempty"`
	SourceName         *string         `json:"source_name,omitempty"`
	TargetID           *uuid.UUID      `json:"target_id,omitempty"`
	TargetName         *string         `json:"target_name,omitempty"`
	Schedule           *string         `json:"schedule,omitempty"`
	Config             json.RawMessage `json:"config"`
	LastRunAt          *time.Time      `json:"last_run_at,omitempty"`
	NextRunAt          *time.Time      `json:"next_run_at,omitempty"`
	LastRunStatus      *string         `json:"last_run_status,omitempty"`
	LastRunFailed      *int64          `json:"last_run_records_failed,omitempty"`
	LastRunProcessed   *int64          `json:"last_run_records_processed,omitempty"`
	LastRunCompletedAt *time.Time      `json:"last_run_completed_at,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type Dataset struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Version          int             `json:"version"`
	Status           string          `json:"status"`
	SourceID         *uuid.UUID      `json:"source_id,omitempty"`
	SourceName       *string         `json:"source_name,omitempty"`
	SchemaDefinition json.RawMessage `json:"schema_definition"`
	Lineage          json.RawMessage `json:"lineage"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type PipelineRun struct {
	ID               uuid.UUID       `json:"id"`
	PipelineID       uuid.UUID       `json:"pipeline_id"`
	Status           string          `json:"status"`
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	RecordsProcessed int64           `json:"records_processed"`
	RecordsFailed    int64           `json:"records_failed"`
	ErrorLog         *string         `json:"error_log,omitempty"`
	Metrics          json.RawMessage `json:"metrics"`
}

type QualityFailure struct {
	RuleName       string          `json:"rule_name"`
	ModelName      string          `json:"model_name"`
	Severity       string          `json:"severity"`
	RecordsFailed  int64           `json:"records_failed"`
	FailureSamples json.RawMessage `json:"failure_samples"`
	CheckedAt      time.Time       `json:"checked_at"`
}

type QualityDashboard struct {
	Score            float64          `json:"score"`
	Trend            float64          `json:"trend"`
	TotalRules       int              `json:"total_rules"`
	EnabledRules     int              `json:"enabled_rules"`
	ResultsLast7Days int              `json:"results_last_7_days"`
	FailedLast7Days  int              `json:"failed_last_7_days"`
	CriticalFailures int              `json:"critical_failures"`
	PassRate         float64          `json:"pass_rate"`
	RecentFailures   []QualityFailure `json:"recent_failures"`
}

type createSourceRequest struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Status           string         `json:"status"`
	ConnectionConfig map[string]any `json:"connection_config"`
	SchemaMetadata   map[string]any `json:"schema_metadata"`
	SyncFrequency    string         `json:"sync_frequency"`
}

type createPipelineRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	SourceID    *uuid.UUID     `json:"source_id"`
	TargetID    *uuid.UUID     `json:"target_id"`
	Schedule    *string        `json:"schedule"`
	Config      map[string]any `json:"config"`
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
	r.Get("/sources", handler.ListSources)
	r.Post("/sources", handler.CreateSource)
	r.Get("/sources/{id}", handler.GetSource)

	r.Get("/pipelines", handler.ListPipelines)
	r.Post("/pipelines", handler.CreatePipeline)
	r.Get("/pipelines/count", handler.CountPipelines)
	r.Post("/pipelines/{id}/run", handler.RunPipeline)

	r.Get("/datasets", handler.ListDatasets)
	r.Get("/quality", handler.GetQualityDashboard)
	r.Get("/quality/score", handler.GetQualityScore)
}

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListSources(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"), r.URL.Query().Get("type"), r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateSource(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req createSourceRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	source, err := h.service.CreateSource(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidation) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, source)
}

func (h *Handler) GetSource(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	source, err := h.service.GetSource(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "data source not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, source)
}

func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListPipelines(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"), r.URL.Query().Get("type"), r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CountPipelines(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	count, err := h.service.CountPipelines(r.Context(), tenantID, r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]int{"count": count})
}

func (h *Handler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req createPipelineRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	pipeline, err := h.service.CreatePipeline(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidation) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, pipeline)
}

func (h *Handler) RunPipeline(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	run, err := h.service.RunPipeline(r.Context(), tenantID, userID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "pipeline not found", nil)
			return
		}
		if errors.Is(err, errValidation) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusAccepted, run)
}

func (h *Handler) ListDatasets(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListDatasets(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"), r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) GetQualityDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	dashboard, err := h.service.GetQualityDashboard(r.Context(), tenantID)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, dashboard)
}

func (h *Handler) GetQualityScore(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	dashboard, err := h.service.GetQualityDashboard(r.Context(), tenantID)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]float64{
		"score": dashboard.Score,
		"trend": dashboard.Trend,
	})
}

var errValidation = errors.New("validation failed")

func (s *Service) ListSources(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, sourceType, status string) ([]Source, int, error) {
	return s.repo.ListSources(ctx, tenantID, page, perPage, search, sourceType, status)
}

func (s *Service) CreateSource(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req createSourceRequest) (*Source, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", errValidation)
	}
	if _, ok := allowedSourceTypes[req.Type]; !ok {
		return nil, fmt.Errorf("%w: invalid source type", errValidation)
	}
	if req.Status == "" {
		req.Status = "inactive"
	}
	if _, ok := allowedSourceStatus[req.Status]; !ok {
		return nil, fmt.Errorf("%w: invalid source status", errValidation)
	}
	return s.repo.CreateSource(ctx, tenantID, userID, req)
}

func (s *Service) GetSource(ctx context.Context, tenantID, id uuid.UUID) (*Source, error) {
	return s.repo.GetSource(ctx, tenantID, id)
}

func (s *Service) ListPipelines(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, pipelineType, status string) ([]Pipeline, int, error) {
	return s.repo.ListPipelines(ctx, tenantID, page, perPage, search, pipelineType, status)
}

func (s *Service) CountPipelines(ctx context.Context, tenantID uuid.UUID, status string) (int, error) {
	return s.repo.CountPipelines(ctx, tenantID, status)
}

func (s *Service) CreatePipeline(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req createPipelineRequest) (*Pipeline, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", errValidation)
	}
	if _, ok := allowedPipelineTypes[req.Type]; !ok {
		return nil, fmt.Errorf("%w: invalid pipeline type", errValidation)
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if _, ok := allowedPipelineState[req.Status]; !ok {
		return nil, fmt.Errorf("%w: invalid pipeline status", errValidation)
	}
	if req.SourceID != nil {
		if _, err := s.repo.GetSource(ctx, tenantID, *req.SourceID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: source_id does not exist in tenant", errValidation)
			}
			return nil, err
		}
	}
	if req.TargetID != nil {
		if _, err := s.repo.GetSource(ctx, tenantID, *req.TargetID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: target_id does not exist in tenant", errValidation)
			}
			return nil, err
		}
	}
	return s.repo.CreatePipeline(ctx, tenantID, userID, req)
}

func (s *Service) RunPipeline(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, pipelineID uuid.UUID) (*PipelineRun, error) {
	pipeline, err := s.repo.GetPipeline(ctx, tenantID, pipelineID)
	if err != nil {
		return nil, err
	}
	if pipeline.Status == "paused" {
		return nil, fmt.Errorf("%w: paused pipelines cannot be run", errValidation)
	}

	status := "completed"
	var errorLog *string
	sourceStatus := "unknown"
	targetStatus := "unknown"
	if pipeline.SourceID != nil {
		source, err := s.repo.GetSource(ctx, tenantID, *pipeline.SourceID)
		if err != nil {
			return nil, err
		}
		sourceStatus = source.Status
		if source.Status == "error" || source.Status == "inactive" {
			status = "failed"
			msg := fmt.Sprintf("source %s is %s", source.Name, source.Status)
			errorLog = &msg
		}
	}
	if pipeline.TargetID != nil {
		target, err := s.repo.GetSource(ctx, tenantID, *pipeline.TargetID)
		if err != nil {
			return nil, err
		}
		targetStatus = target.Status
		if target.Status == "error" || target.Status == "inactive" {
			status = "failed"
			msg := fmt.Sprintf("target %s is %s", target.Name, target.Status)
			errorLog = &msg
		}
	}

	processed, failed, lineageEdges, err := s.repo.EstimatePipelineImpact(ctx, tenantID, pipelineID, pipeline.SourceID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	completedAt := now
	metricsPayload, _ := json.Marshal(map[string]any{
		"source_status":     sourceStatus,
		"target_status":     targetStatus,
		"lineage_edges":     lineageEdges,
		"quality_failures":  failed,
		"execution_control": "metadata_evaluation",
	})

	run := &PipelineRun{
		ID:               uuid.New(),
		PipelineID:       pipelineID,
		Status:           status,
		StartedAt:        now,
		CompletedAt:      &completedAt,
		RecordsProcessed: processed,
		RecordsFailed:    failed,
		ErrorLog:         errorLog,
		Metrics:          metricsPayload,
	}
	if err := s.repo.CreatePipelineRun(ctx, tenantID, userID, run, pipeline.Schedule); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) ListDatasets(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, status string) ([]Dataset, int, error) {
	return s.repo.ListDatasets(ctx, tenantID, page, perPage, search, status)
}

func (s *Service) GetQualityDashboard(ctx context.Context, tenantID uuid.UUID) (*QualityDashboard, error) {
	return s.repo.GetQualityDashboard(ctx, tenantID)
}

func (r *Repository) ListSources(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, sourceType, status string) ([]Source, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.type::text, a.status::text, a.connection_config, a.schema_metadata,
		       a.last_synced_at, a.sync_frequency, a.created_at, a.updated_at
		FROM data_sources a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.WhereIf(strings.TrimSpace(sourceType) != "", "a.type::text = ?", strings.ToLower(strings.TrimSpace(sourceType)))
	qb.WhereIf(strings.TrimSpace(status) != "", "a.status::text = ?", strings.ToLower(strings.TrimSpace(status)))
	qb.OrderBy("created_at", "desc", []string{"name", "type", "status", "created_at"})
	qb.Paginate(page, perPage)
	sqlQuery, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Source, 0, perPage)
	for rows.Next() {
		var item Source
		var connectionConfig []byte
		var schemaMetadata []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Status, &connectionConfig, &schemaMetadata, &item.LastSyncedAt, &item.SyncFrequency, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.ConnectionConfig = normalizeJSON(connectionConfig, "{}")
		item.SchemaMetadata = normalizeJSON(schemaMetadata, "{}")
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

func (r *Repository) CreateSource(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req createSourceRequest) (*Source, error) {
	connectionConfig, _ := json.Marshal(req.ConnectionConfig)
	schemaMetadata, _ := json.Marshal(req.SchemaMetadata)
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO data_sources (tenant_id, name, type, connection_config, status, schema_metadata, sync_frequency, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $8)
		RETURNING id`,
		tenantID, req.Name, req.Type, connectionConfig, req.Status, schemaMetadata, req.SyncFrequency, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetSource(ctx, tenantID, id)
}

func (r *Repository) GetSource(ctx context.Context, tenantID, id uuid.UUID) (*Source, error) {
	var item Source
	var connectionConfig []byte
	var schemaMetadata []byte
	if err := r.db.QueryRow(ctx, `
		SELECT id, name, type::text, status::text, connection_config, schema_metadata,
		       last_synced_at, sync_frequency, created_at, updated_at
		FROM data_sources
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.Name, &item.Type, &item.Status, &connectionConfig, &schemaMetadata, &item.LastSyncedAt, &item.SyncFrequency, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.ConnectionConfig = normalizeJSON(connectionConfig, "{}")
	item.SchemaMetadata = normalizeJSON(schemaMetadata, "{}")
	return &item, nil
}

func (r *Repository) ListPipelines(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, pipelineType, status string) ([]Pipeline, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.description, a.type::text, a.status::text,
		       a.source_id, s.name, a.target_id, t.name, a.schedule, a.config,
		       a.last_run_at, a.next_run_at,
		       (SELECT pr.status::text FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       (SELECT pr.records_failed FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       (SELECT pr.records_processed FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       (SELECT pr.completed_at FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       a.created_at, a.updated_at
		FROM pipelines a
		LEFT JOIN data_sources s ON s.id = a.source_id AND s.tenant_id = a.tenant_id
		LEFT JOIN data_sources t ON t.id = a.target_id AND t.tenant_id = a.tenant_id`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.WhereIf(strings.TrimSpace(pipelineType) != "", "a.type::text = ?", strings.ToLower(strings.TrimSpace(pipelineType)))
	qb.WhereIf(strings.TrimSpace(status) != "", "a.status::text = ?", strings.ToLower(strings.TrimSpace(status)))
	qb.OrderBy("created_at", "desc", []string{"name", "type", "status", "created_at"})
	qb.Paginate(page, perPage)
	sqlQuery, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Pipeline, 0, perPage)
	for rows.Next() {
		var item Pipeline
		var config []byte
		var schedule sql.NullString
		var sourceName, targetName sql.NullString
		var lastRunStatus sql.NullString
		var lastRunFailed, lastRunProcessed sql.NullInt64
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.Type, &item.Status,
			&item.SourceID, &sourceName, &item.TargetID, &targetName, &schedule, &config,
			&item.LastRunAt, &item.NextRunAt, &lastRunStatus, &lastRunFailed, &lastRunProcessed, &item.LastRunCompletedAt,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.Config = normalizeJSON(config, "{}")
		if schedule.Valid {
			item.Schedule = &schedule.String
		}
		if sourceName.Valid {
			item.SourceName = &sourceName.String
		}
		if targetName.Valid {
			item.TargetName = &targetName.String
		}
		if lastRunStatus.Valid {
			item.LastRunStatus = &lastRunStatus.String
		}
		if lastRunFailed.Valid {
			item.LastRunFailed = &lastRunFailed.Int64
		}
		if lastRunProcessed.Valid {
			item.LastRunProcessed = &lastRunProcessed.Int64
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

func (r *Repository) CountPipelines(ctx context.Context, tenantID uuid.UUID, status string) (int, error) {
	query := `SELECT COUNT(*) FROM pipelines WHERE tenant_id = $1`
	args := []any{tenantID}
	if status = strings.TrimSpace(strings.ToLower(status)); status != "" {
		query += ` AND status::text = $2`
		args = append(args, status)
	}
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) GetPipeline(ctx context.Context, tenantID, id uuid.UUID) (*Pipeline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.name, a.description, a.type::text, a.status::text,
		       a.source_id, s.name, a.target_id, t.name, a.schedule, a.config,
		       a.last_run_at, a.next_run_at,
		       (SELECT pr.status::text FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       (SELECT pr.records_failed FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       (SELECT pr.records_processed FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       (SELECT pr.completed_at FROM pipeline_runs pr WHERE pr.tenant_id = a.tenant_id AND pr.pipeline_id = a.id ORDER BY pr.created_at DESC LIMIT 1),
		       a.created_at, a.updated_at
		FROM pipelines a
		LEFT JOIN data_sources s ON s.id = a.source_id AND s.tenant_id = a.tenant_id
		LEFT JOIN data_sources t ON t.id = a.target_id AND t.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.id = $2`,
		tenantID, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	var item Pipeline
	var config []byte
	var schedule sql.NullString
	var sourceName, targetName sql.NullString
	var lastRunStatus sql.NullString
	var lastRunFailed, lastRunProcessed sql.NullInt64
	if err := rows.Scan(
		&item.ID, &item.Name, &item.Description, &item.Type, &item.Status,
		&item.SourceID, &sourceName, &item.TargetID, &targetName, &schedule, &config,
		&item.LastRunAt, &item.NextRunAt, &lastRunStatus, &lastRunFailed, &lastRunProcessed, &item.LastRunCompletedAt,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Config = normalizeJSON(config, "{}")
	if schedule.Valid {
		item.Schedule = &schedule.String
	}
	if sourceName.Valid {
		item.SourceName = &sourceName.String
	}
	if targetName.Valid {
		item.TargetName = &targetName.String
	}
	if lastRunStatus.Valid {
		item.LastRunStatus = &lastRunStatus.String
	}
	if lastRunFailed.Valid {
		item.LastRunFailed = &lastRunFailed.Int64
	}
	if lastRunProcessed.Valid {
		item.LastRunProcessed = &lastRunProcessed.Int64
	}
	return &item, rows.Err()
}

func (r *Repository) CreatePipeline(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req createPipelineRequest) (*Pipeline, error) {
	config, _ := json.Marshal(req.Config)
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO pipelines (tenant_id, name, description, type, source_id, target_id, schedule, config, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id`,
		tenantID, req.Name, req.Description, req.Type, req.SourceID, req.TargetID, req.Schedule, config, req.Status, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetPipeline(ctx, tenantID, id)
}

func (r *Repository) EstimatePipelineImpact(ctx context.Context, tenantID, pipelineID uuid.UUID, sourceID *uuid.UUID) (int64, int64, int64, error) {
	var lineageEdges int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM data_lineage WHERE tenant_id = $1 AND pipeline_id = $2`, tenantID, pipelineID).Scan(&lineageEdges); err != nil {
		return 0, 0, 0, err
	}

	var processed int64
	if sourceID != nil {
		if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM data_models WHERE tenant_id = $1 AND source_id = $2`, tenantID, *sourceID).Scan(&processed); err != nil {
			return 0, 0, 0, err
		}
	}

	var failed int64
	if sourceID != nil {
		if err := r.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(dqr.records_failed), 0)
			FROM data_models dm
			JOIN data_quality_results dqr ON dqr.model_id = dm.id AND dqr.tenant_id = dm.tenant_id
			WHERE dm.tenant_id = $1
			  AND dm.source_id = $2
			  AND dqr.status = 'failed'
			  AND dqr.checked_at > NOW() - INTERVAL '30 days'`,
			tenantID, *sourceID,
		).Scan(&failed); err != nil {
			return 0, 0, 0, err
		}
	}

	if processed == 0 {
		processed = lineageEdges
	}
	return processed, failed, lineageEdges, nil
}

func (r *Repository) CreatePipelineRun(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, run *PipelineRun, schedule *string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, `
		INSERT INTO pipeline_runs (id, tenant_id, pipeline_id, status, started_at, completed_at, records_processed, records_failed, error_log, metrics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		run.ID, tenantID, run.PipelineID, run.Status, run.StartedAt, run.CompletedAt, run.RecordsProcessed, run.RecordsFailed, run.ErrorLog, run.Metrics,
	); err != nil {
		return err
	}

	var nextRunAt *time.Time
	if schedule != nil && strings.TrimSpace(*schedule) != "" {
		t := run.StartedAt.Add(24 * time.Hour)
		nextRunAt = &t
	}
	if _, err = tx.Exec(ctx, `
		UPDATE pipelines
		SET last_run_at = $3, next_run_at = $4, status = CASE WHEN $5 = 'failed' THEN 'failed' ELSE status END
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, run.PipelineID, run.StartedAt, nextRunAt, run.Status,
	); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repository) ListDatasets(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, status string) ([]Dataset, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.description, a.version, a.status::text, a.source_id, s.name, a.schema_definition, a.lineage, a.created_at, a.updated_at
		FROM data_models a
		LEFT JOIN data_sources s ON s.id = a.source_id AND s.tenant_id = a.tenant_id`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.WhereIf(strings.TrimSpace(status) != "", "a.status::text = ?", strings.ToLower(strings.TrimSpace(status)))
	qb.OrderBy("created_at", "desc", []string{"name", "status", "created_at"})
	qb.Paginate(page, perPage)
	sqlQuery, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Dataset, 0, perPage)
	for rows.Next() {
		var item Dataset
		var schemaDefinition, lineage []byte
		var sourceName sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Version, &item.Status, &item.SourceID, &sourceName, &schemaDefinition, &lineage, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if sourceName.Valid {
			item.SourceName = &sourceName.String
		}
		item.SchemaDefinition = normalizeJSON(schemaDefinition, "{}")
		item.Lineage = normalizeJSON(lineage, "{}")
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

func (r *Repository) GetQualityDashboard(ctx context.Context, tenantID uuid.UUID) (*QualityDashboard, error) {
	dashboard := &QualityDashboard{}
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM data_quality_rules WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM data_quality_rules WHERE tenant_id = $1 AND enabled = true),
			(SELECT COUNT(*) FROM data_quality_results WHERE tenant_id = $1 AND checked_at > NOW() - INTERVAL '7 days'),
			(SELECT COUNT(*) FROM data_quality_results WHERE tenant_id = $1 AND checked_at > NOW() - INTERVAL '7 days' AND status = 'failed'),
			(SELECT COUNT(*)
			 FROM data_quality_results dqr
			 JOIN data_quality_rules dqrule ON dqrule.id = dqr.rule_id AND dqrule.tenant_id = dqr.tenant_id
			 WHERE dqr.tenant_id = $1
			   AND dqr.checked_at > NOW() - INTERVAL '7 days'
			   AND dqr.status = 'failed'
			   AND dqrule.severity = 'critical')`,
		tenantID,
	).Scan(&dashboard.TotalRules, &dashboard.EnabledRules, &dashboard.ResultsLast7Days, &dashboard.FailedLast7Days, &dashboard.CriticalFailures); err != nil {
		return nil, err
	}

	dashboard.PassRate = 100
	if dashboard.ResultsLast7Days > 0 {
		dashboard.PassRate = float64(dashboard.ResultsLast7Days-dashboard.FailedLast7Days) / float64(dashboard.ResultsLast7Days) * 100
	}
	scorePenalty := 0.0
	if dashboard.ResultsLast7Days > 0 {
		scorePenalty += float64(dashboard.FailedLast7Days) / float64(dashboard.ResultsLast7Days) * 60
	}
	if dashboard.EnabledRules > 0 {
		scorePenalty += float64(dashboard.CriticalFailures) / float64(dashboard.EnabledRules) * 40
	}
	dashboard.Score = clamp(100-scorePenalty, 0, 100)

	var previousResults, previousFailed int
	if err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE checked_at >= NOW() - INTERVAL '14 days' AND checked_at < NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE checked_at >= NOW() - INTERVAL '14 days' AND checked_at < NOW() - INTERVAL '7 days' AND status = 'failed')
		FROM data_quality_results
		WHERE tenant_id = $1`, tenantID,
	).Scan(&previousResults, &previousFailed); err != nil {
		return nil, err
	}
	prevScore := 100.0
	if previousResults > 0 {
		prevPenalty := float64(previousFailed) / float64(previousResults) * 60
		prevScore = clamp(100-prevPenalty, 0, 100)
	}
	dashboard.Trend = dashboard.Score - prevScore

	rows, err := r.db.Query(ctx, `
		SELECT r.name, m.name, r.severity::text, qr.records_failed, qr.failure_samples, qr.checked_at
		FROM data_quality_results qr
		JOIN data_quality_rules r ON r.id = qr.rule_id AND r.tenant_id = qr.tenant_id
		JOIN data_models m ON m.id = qr.model_id AND m.tenant_id = qr.tenant_id
		WHERE qr.tenant_id = $1
		  AND qr.status = 'failed'
		ORDER BY qr.checked_at DESC
		LIMIT 10`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dashboard.RecentFailures = make([]QualityFailure, 0, 10)
	for rows.Next() {
		var item QualityFailure
		var samples []byte
		if err := rows.Scan(&item.RuleName, &item.ModelName, &item.Severity, &item.RecordsFailed, &samples, &item.CheckedAt); err != nil {
			return nil, err
		}
		item.FailureSamples = normalizeJSON(samples, "[]")
		dashboard.RecentFailures = append(dashboard.RecentFailures, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dashboard, nil
}

func (h *Handler) tenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
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
	h.logger.Error().Err(err).Msg("data service request failed")
	suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "request failed", nil)
}

func normalizeJSON(value []byte, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(value)
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
