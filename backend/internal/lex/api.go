package lex

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
	allowedContractTypes  = map[string]struct{}{"nda": {}, "service_agreement": {}, "employment": {}, "vendor": {}, "license": {}, "other": {}}
	allowedContractStatus = map[string]struct{}{"draft": {}, "review": {}, "negotiation": {}, "active": {}, "expired": {}, "terminated": {}}
	allowedDocumentStatus = map[string]struct{}{"draft": {}, "review": {}, "approved": {}, "archived": {}}
	allowedRuleSeverities = map[string]struct{}{"critical": {}, "high": {}, "medium": {}, "low": {}}
	allowedAlertStatuses  = map[string]struct{}{"new": {}, "acknowledged": {}, "resolved": {}, "dismissed": {}}
)

type Contract struct {
	ID            uuid.UUID       `json:"id"`
	Title         string          `json:"title"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	Parties       json.RawMessage `json:"parties"`
	EffectiveDate *time.Time      `json:"effective_date,omitempty"`
	ExpiryDate    *time.Time      `json:"expiry_date,omitempty"`
	Value         *float64        `json:"value,omitempty"`
	Currency      string          `json:"currency"`
	FileURL       *string         `json:"file_url,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type LegalDocument struct {
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	Type      string     `json:"type"`
	Content   string     `json:"content"`
	FileURL   *string    `json:"file_url,omitempty"`
	Status    string     `json:"status"`
	Version   int        `json:"version"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Tags      []string   `json:"tags"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ComplianceRule struct {
	ID                  uuid.UUID       `json:"id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	Jurisdiction        *string         `json:"jurisdiction,omitempty"`
	RegulationReference *string         `json:"regulation_reference,omitempty"`
	RuleLogic           json.RawMessage `json:"rule_logic"`
	Severity            string          `json:"severity"`
	Enabled             bool            `json:"enabled"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type ComplianceAlert struct {
	ID          uuid.UUID  `json:"id"`
	RuleID      uuid.UUID  `json:"rule_id"`
	EntityType  string     `json:"entity_type"`
	EntityID    uuid.UUID  `json:"entity_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Severity    string     `json:"severity"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type ComplianceDashboard struct {
	TotalRules        int               `json:"total_rules"`
	EnabledRules      int               `json:"enabled_rules"`
	OpenAlerts        int               `json:"open_alerts"`
	BySeverity        map[string]int    `json:"by_severity"`
	ByStatus          map[string]int    `json:"by_status"`
	RecentAlerts      []ComplianceAlert `json:"recent_alerts"`
	ContractsExpiring int               `json:"contracts_expiring_30d"`
}

type ComplianceCheckRequest struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
}

type ComplianceCheckResult struct {
	RuleID   uuid.UUID  `json:"rule_id"`
	RuleName string     `json:"rule_name"`
	Severity string     `json:"severity"`
	Status   string     `json:"status"`
	Message  string     `json:"message"`
	AlertID  *uuid.UUID `json:"alert_id,omitempty"`
}

type contractRequest struct {
	Title         string           `json:"title"`
	Type          string           `json:"type"`
	Status        string           `json:"status"`
	Parties       []map[string]any `json:"parties"`
	EffectiveDate *time.Time       `json:"effective_date"`
	ExpiryDate    *time.Time       `json:"expiry_date"`
	Value         *float64         `json:"value"`
	Currency      string           `json:"currency"`
	FileURL       *string          `json:"file_url"`
	Metadata      map[string]any   `json:"metadata"`
}

type documentRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags"`
	FileURL *string  `json:"file_url"`
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

	r.Get("/contracts", handler.ListContracts)
	r.Post("/contracts", handler.CreateContract)
	r.Get("/contracts/{id}", handler.GetContract)
	r.Put("/contracts/{id}", handler.UpdateContract)

	r.Get("/documents", handler.ListDocuments)

	r.Get("/regulations", handler.ListRegulations)
	r.Get("/compliance", handler.GetComplianceDashboard)
	r.Post("/compliance/check", handler.RunComplianceCheck)

	r.Get("/cases", handler.ListCases)
	r.Post("/cases", handler.CreateCase)
	r.Get("/cases/{id}", handler.GetCase)
	r.Put("/cases/{id}", handler.UpdateCase)
}

func (h *Handler) ListContracts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListContracts(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"), r.URL.Query().Get("status"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateContract(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req contractRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	contract, err := h.service.CreateContract(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationLex) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, contract)
}

func (h *Handler) GetContract(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	contract, err := h.service.GetContract(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "contract not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, contract)
}

func (h *Handler) UpdateContract(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req contractRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	contract, err := h.service.UpdateContract(r.Context(), tenantID, userID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, errValidationLex):
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, pgx.ErrNoRows):
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "contract not found", nil)
		default:
			h.writeInternalError(w, r, err)
		}
		return
	}
	suiteapi.WriteData(w, http.StatusOK, contract)
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListDocuments(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"), "")
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) ListRegulations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListRegulations(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) GetComplianceDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	dashboard, err := h.service.GetComplianceDashboard(r.Context(), tenantID)
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, dashboard)
}

func (h *Handler) RunComplianceCheck(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req ComplianceCheckRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	result, err := h.service.RunComplianceCheck(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, errValidationLex) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.ListDocuments(r.Context(), tenantID, page, perPage, r.URL.Query().Get("search"), "case")
	if err != nil {
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	var req documentRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	doc, err := h.service.CreateDocument(r.Context(), tenantID, userID, req, "case")
	if err != nil {
		if errors.Is(err, errValidationLex) {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, doc)
}

func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantOnly(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	doc, err := h.service.GetDocument(r.Context(), tenantID, id, "case")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "case not found", nil)
			return
		}
		h.writeInternalError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, doc)
}

func (h *Handler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req documentRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", map[string]string{"body": err.Error()})
		return
	}
	doc, err := h.service.UpdateDocument(r.Context(), tenantID, userID, id, req, "case")
	if err != nil {
		switch {
		case errors.Is(err, errValidationLex):
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, pgx.ErrNoRows):
			suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "case not found", nil)
		default:
			h.writeInternalError(w, r, err)
		}
		return
	}
	suiteapi.WriteData(w, http.StatusOK, doc)
}

var errValidationLex = errors.New("validation failed")

func (s *Service) ListContracts(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, status string) ([]Contract, int, error) {
	return s.repo.ListContracts(ctx, tenantID, page, perPage, search, status)
}

func (s *Service) CreateContract(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req contractRequest) (*Contract, error) {
	req = normalizeContract(req)
	if err := validateContract(req); err != nil {
		return nil, err
	}
	return s.repo.CreateContract(ctx, tenantID, userID, req)
}

func (s *Service) GetContract(ctx context.Context, tenantID, id uuid.UUID) (*Contract, error) {
	return s.repo.GetContract(ctx, tenantID, id)
}

func (s *Service) UpdateContract(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req contractRequest) (*Contract, error) {
	req = normalizeContract(req)
	if err := validateContract(req); err != nil {
		return nil, err
	}
	return s.repo.UpdateContract(ctx, tenantID, userID, id, req)
}

func (s *Service) ListDocuments(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, docType string) ([]LegalDocument, int, error) {
	return s.repo.ListDocuments(ctx, tenantID, page, perPage, search, docType)
}

func (s *Service) CreateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req documentRequest, docType string) (*LegalDocument, error) {
	req = normalizeDocument(req)
	if err := validateDocument(req); err != nil {
		return nil, err
	}
	return s.repo.CreateDocument(ctx, tenantID, userID, req, docType)
}

func (s *Service) GetDocument(ctx context.Context, tenantID, id uuid.UUID, docType string) (*LegalDocument, error) {
	return s.repo.GetDocument(ctx, tenantID, id, docType)
}

func (s *Service) UpdateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req documentRequest, docType string) (*LegalDocument, error) {
	req = normalizeDocument(req)
	if err := validateDocument(req); err != nil {
		return nil, err
	}
	return s.repo.UpdateDocument(ctx, tenantID, userID, id, req, docType)
}

func (s *Service) ListRegulations(ctx context.Context, tenantID uuid.UUID, page, perPage int, search string) ([]ComplianceRule, int, error) {
	return s.repo.ListRegulations(ctx, tenantID, page, perPage, search)
}

func (s *Service) GetComplianceDashboard(ctx context.Context, tenantID uuid.UUID) (*ComplianceDashboard, error) {
	return s.repo.GetComplianceDashboard(ctx, tenantID)
}

func (s *Service) RunComplianceCheck(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req ComplianceCheckRequest) ([]ComplianceCheckResult, error) {
	req.EntityType = strings.TrimSpace(strings.ToLower(req.EntityType))
	switch req.EntityType {
	case "contract", "document", "case":
	default:
		return nil, fmt.Errorf("%w: unsupported entity type", errValidationLex)
	}

	entity, err := s.repo.LoadComplianceEntity(ctx, tenantID, req.EntityType, req.EntityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: entity not found", errValidationLex)
		}
		return nil, err
	}
	rules, _, err := s.repo.ListRegulations(ctx, tenantID, 1, 500, "")
	if err != nil {
		return nil, err
	}

	results := make([]ComplianceCheckResult, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		passed, message := evaluateRule(rule.RuleLogic, entity)
		status := "compliant"
		var alertID *uuid.UUID
		if !passed {
			status = "non_compliant"
			id, err := s.repo.UpsertComplianceAlert(ctx, tenantID, userID, rule, req.EntityType, req.EntityID, message)
			if err != nil {
				return nil, err
			}
			alertID = &id
		}
		results = append(results, ComplianceCheckResult{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Severity: rule.Severity,
			Status:   status,
			Message:  message,
			AlertID:  alertID,
		})
	}
	return results, nil
}

func (r *Repository) ListContracts(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, status string) ([]Contract, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.title, a.type::text, a.status::text, a.parties, a.effective_date, a.expiry_date,
		       a.value::float8, a.currency, a.file_url, a.metadata, a.created_at, a.updated_at
		FROM contracts a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.title ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.WhereIf(strings.TrimSpace(status) != "", "a.status::text = ?", strings.ToLower(strings.TrimSpace(status)))
	qb.OrderBy("created_at", "desc", []string{"created_at", "title", "status"})
	qb.Paginate(page, perPage)
	sqlQuery, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Contract, 0, perPage)
	for rows.Next() {
		var item Contract
		var parties, metadata []byte
		var value sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Status, &parties, &item.EffectiveDate, &item.ExpiryDate, &value, &item.Currency, &item.FileURL, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if value.Valid {
			item.Value = &value.Float64
		}
		item.Parties = normalizeJSON(parties, "[]")
		item.Metadata = normalizeJSON(metadata, "{}")
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

func (r *Repository) CreateContract(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req contractRequest) (*Contract, error) {
	parties, _ := json.Marshal(req.Parties)
	metadata, _ := json.Marshal(req.Metadata)
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO contracts (tenant_id, title, type, status, parties, effective_date, expiry_date, value, currency, file_url, metadata, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		RETURNING id`,
		tenantID, req.Title, req.Type, req.Status, parties, req.EffectiveDate, req.ExpiryDate, req.Value, req.Currency, req.FileURL, metadata, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetContract(ctx, tenantID, id)
}

func (r *Repository) GetContract(ctx context.Context, tenantID, id uuid.UUID) (*Contract, error) {
	var item Contract
	var parties, metadata []byte
	var value sql.NullFloat64
	if err := r.db.QueryRow(ctx, `
		SELECT id, title, type::text, status::text, parties, effective_date, expiry_date,
		       value::float8, currency, file_url, metadata, created_at, updated_at
		FROM contracts
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&item.ID, &item.Title, &item.Type, &item.Status, &parties, &item.EffectiveDate, &item.ExpiryDate, &value, &item.Currency, &item.FileURL, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if value.Valid {
		item.Value = &value.Float64
	}
	item.Parties = normalizeJSON(parties, "[]")
	item.Metadata = normalizeJSON(metadata, "{}")
	return &item, nil
}

func (r *Repository) UpdateContract(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req contractRequest) (*Contract, error) {
	parties, _ := json.Marshal(req.Parties)
	metadata, _ := json.Marshal(req.Metadata)
	tag, err := r.db.Exec(ctx, `
		UPDATE contracts
		SET title = $3, type = $4, status = $5, parties = $6, effective_date = $7,
		    expiry_date = $8, value = $9, currency = $10, file_url = $11, metadata = $12, updated_by = $13
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, req.Title, req.Type, req.Status, parties, req.EffectiveDate, req.ExpiryDate, req.Value, req.Currency, req.FileURL, metadata, userID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetContract(ctx, tenantID, id)
}

func (r *Repository) ListDocuments(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, docType string) ([]LegalDocument, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.title, a.type, a.content, a.file_url, a.status::text, a.version, a.parent_id, a.tags, a.created_at, a.updated_at
		FROM legal_documents a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.title ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.WhereIf(strings.TrimSpace(docType) != "", "a.type = ?", docType)
	qb.OrderBy("created_at", "desc", []string{"created_at", "title"})
	qb.Paginate(page, perPage)
	sqlQuery, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]LegalDocument, 0, perPage)
	for rows.Next() {
		var item LegalDocument
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Content, &item.FileURL, &item.Status, &item.Version, &item.ParentID, &item.Tags, &item.CreatedAt, &item.UpdatedAt); err != nil {
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

func (r *Repository) CreateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req documentRequest, docType string) (*LegalDocument, error) {
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO legal_documents (tenant_id, title, type, content, file_url, status, tags, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id`,
		tenantID, req.Title, docType, req.Content, req.FileURL, req.Status, req.Tags, userID,
	).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetDocument(ctx, tenantID, id, docType)
}

func (r *Repository) GetDocument(ctx context.Context, tenantID, id uuid.UUID, docType string) (*LegalDocument, error) {
	var item LegalDocument
	row := r.db.QueryRow(ctx, `
		SELECT id, title, type, content, file_url, status::text, version, parent_id, tags, created_at, updated_at
		FROM legal_documents
		WHERE tenant_id = $1 AND id = $2 AND ($3 = '' OR type = $3)`,
		tenantID, id, docType,
	)
	if err := row.Scan(&item.ID, &item.Title, &item.Type, &item.Content, &item.FileURL, &item.Status, &item.Version, &item.ParentID, &item.Tags, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpdateDocument(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID, req documentRequest, docType string) (*LegalDocument, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE legal_documents
		SET title = $4, content = $5, file_url = $6, status = $7, tags = $8, updated_by = $9
		WHERE tenant_id = $1 AND id = $2 AND type = $3`,
		tenantID, id, docType, req.Title, req.Content, req.FileURL, req.Status, req.Tags, userID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetDocument(ctx, tenantID, id, docType)
}

func (r *Repository) ListRegulations(ctx context.Context, tenantID uuid.UUID, page, perPage int, search string) ([]ComplianceRule, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.name, a.description, a.jurisdiction, a.regulation_reference, a.rule_logic, a.severity::text, a.enabled, a.created_at, a.updated_at
		FROM compliance_rules a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(strings.TrimSpace(search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(search)+"%")
	qb.OrderBy("created_at", "desc", []string{"created_at", "name"})
	qb.Paginate(page, perPage)
	sqlQuery, args := qb.Build()
	countQuery, countArgs := qb.BuildCount()
	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]ComplianceRule, 0, perPage)
	for rows.Next() {
		var item ComplianceRule
		var ruleLogic []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Jurisdiction, &item.RegulationReference, &ruleLogic, &item.Severity, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.RuleLogic = normalizeJSON(ruleLogic, "{}")
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

func (r *Repository) GetComplianceDashboard(ctx context.Context, tenantID uuid.UUID) (*ComplianceDashboard, error) {
	dashboard := &ComplianceDashboard{
		BySeverity: make(map[string]int),
		ByStatus:   make(map[string]int),
	}
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM compliance_rules WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM compliance_rules WHERE tenant_id = $1 AND enabled = true),
			(SELECT COUNT(*) FROM compliance_alerts WHERE tenant_id = $1 AND status IN ('new','acknowledged')),
			(SELECT COUNT(*) FROM contracts WHERE tenant_id = $1 AND status = 'active' AND expiry_date <= CURRENT_DATE + INTERVAL '30 days')`,
		tenantID,
	).Scan(&dashboard.TotalRules, &dashboard.EnabledRules, &dashboard.OpenAlerts, &dashboard.ContractsExpiring); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT severity::text, status::text, COUNT(*)
		FROM compliance_alerts
		WHERE tenant_id = $1
		GROUP BY severity, status`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var severity, status string
		var count int
		if err := rows.Scan(&severity, &status, &count); err != nil {
			return nil, err
		}
		dashboard.BySeverity[severity] += count
		dashboard.ByStatus[status] += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	alertRows, err := r.db.Query(ctx, `
		SELECT id, rule_id, entity_type, entity_id, title, description, severity::text, status::text, created_at, resolved_at
		FROM compliance_alerts
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 10`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer alertRows.Close()
	dashboard.RecentAlerts = make([]ComplianceAlert, 0, 10)
	for alertRows.Next() {
		var item ComplianceAlert
		if err := alertRows.Scan(&item.ID, &item.RuleID, &item.EntityType, &item.EntityID, &item.Title, &item.Description, &item.Severity, &item.Status, &item.CreatedAt, &item.ResolvedAt); err != nil {
			return nil, err
		}
		dashboard.RecentAlerts = append(dashboard.RecentAlerts, item)
	}
	if err := alertRows.Err(); err != nil {
		return nil, err
	}
	return dashboard, nil
}

func (r *Repository) LoadComplianceEntity(ctx context.Context, tenantID uuid.UUID, entityType string, id uuid.UUID) (map[string]any, error) {
	switch entityType {
	case "contract":
		contract, err := r.GetContract(ctx, tenantID, id)
		if err != nil {
			return nil, err
		}
		entity := map[string]any{
			"title":        contract.Title,
			"type":         contract.Type,
			"status":       contract.Status,
			"currency":     contract.Currency,
			"file_present": contract.FileURL != nil && *contract.FileURL != "",
			"party_count":  len(mustJSONArray(contract.Parties)),
			"metadata":     mustJSONObject(contract.Metadata),
		}
		if contract.Value != nil {
			entity["value"] = *contract.Value
		}
		if contract.ExpiryDate != nil {
			entity["days_to_expiry"] = int(contract.ExpiryDate.Sub(time.Now().UTC()).Hours() / 24)
		}
		return entity, nil
	case "document", "case":
		docType := entityType
		if entityType == "document" {
			docType = ""
		}
		document, err := r.GetDocument(ctx, tenantID, id, docType)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"title":        document.Title,
			"type":         document.Type,
			"status":       document.Status,
			"version":      document.Version,
			"tag_count":    len(document.Tags),
			"file_present": document.FileURL != nil && *document.FileURL != "",
		}, nil
	default:
		return nil, pgx.ErrNoRows
	}
}

func (r *Repository) UpsertComplianceAlert(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, rule ComplianceRule, entityType string, entityID uuid.UUID, message string) (uuid.UUID, error) {
	var existing uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM compliance_alerts
		WHERE tenant_id = $1
		  AND rule_id = $2
		  AND entity_type = $3
		  AND entity_id = $4
		  AND status IN ('new', 'acknowledged')
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, rule.ID, entityType, entityID,
	).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO compliance_alerts (tenant_id, rule_id, entity_type, entity_id, title, description, severity, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'new', $8, $8)
		RETURNING id`,
		tenantID, rule.ID, entityType, entityID, rule.Name, message, rule.Severity, userID,
	).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
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
	h.logger.Error().Err(err).Msg("lex service request failed")
	suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "request failed", nil)
}

func validateContract(req contractRequest) error {
	if req.Title == "" {
		return fmt.Errorf("%w: title is required", errValidationLex)
	}
	if _, ok := allowedContractTypes[req.Type]; !ok {
		return fmt.Errorf("%w: invalid contract type", errValidationLex)
	}
	if _, ok := allowedContractStatus[req.Status]; !ok {
		return fmt.Errorf("%w: invalid contract status", errValidationLex)
	}
	return nil
}

func validateDocument(req documentRequest) error {
	if req.Title == "" {
		return fmt.Errorf("%w: title is required", errValidationLex)
	}
	if _, ok := allowedDocumentStatus[req.Status]; !ok {
		return fmt.Errorf("%w: invalid document status", errValidationLex)
	}
	return nil
}

func normalizeContract(req contractRequest) contractRequest {
	req.Title = strings.TrimSpace(req.Title)
	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Currency == "" {
		req.Currency = "SAR"
	}
	return req
}

func normalizeDocument(req documentRequest) documentRequest {
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "draft"
	}
	return req
}

func evaluateRule(ruleLogic json.RawMessage, entity map[string]any) (bool, string) {
	if len(ruleLogic) == 0 || string(ruleLogic) == "{}" {
		return true, "rule has no executable logic"
	}
	var payload map[string]any
	if err := json.Unmarshal(ruleLogic, &payload); err != nil {
		return false, "rule logic is invalid"
	}
	ok, msg := evalCondition(payload, entity)
	if ok {
		return true, "entity satisfies rule conditions"
	}
	return false, msg
}

func evalCondition(condition map[string]any, entity map[string]any) (bool, string) {
	if all, ok := condition["all"].([]any); ok {
		for _, raw := range all {
			next, ok := raw.(map[string]any)
			if !ok {
				return false, "invalid nested condition"
			}
			if passed, msg := evalCondition(next, entity); !passed {
				return false, msg
			}
		}
		return true, "all conditions passed"
	}
	if anyOf, ok := condition["any"].([]any); ok {
		var last string
		for _, raw := range anyOf {
			next, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if passed, _ := evalCondition(next, entity); passed {
				return true, "one condition passed"
			}
			_, last = evalCondition(next, entity)
		}
		if last == "" {
			last = "no conditions matched"
		}
		return false, last
	}

	field, _ := condition["field"].(string)
	operator, _ := condition["operator"].(string)
	expected := condition["value"]
	actual, exists := entity[field]
	if !exists {
		return false, fmt.Sprintf("missing field %s", field)
	}

	switch strings.ToLower(operator) {
	case "equals":
		if fmt.Sprint(actual) == fmt.Sprint(expected) {
			return true, "field matches expected value"
		}
		return false, fmt.Sprintf("%s expected %v but found %v", field, expected, actual)
	case "not_equals":
		if fmt.Sprint(actual) != fmt.Sprint(expected) {
			return true, "field differs from blocked value"
		}
		return false, fmt.Sprintf("%s must not equal %v", field, expected)
	case "lte":
		if actualFloat, ok := asFloat(actual); ok {
			if expectedFloat, ok := asFloat(expected); ok && actualFloat <= expectedFloat {
				return true, "value within threshold"
			}
		}
		return false, fmt.Sprintf("%s exceeds threshold", field)
	case "gte":
		if actualFloat, ok := asFloat(actual); ok {
			if expectedFloat, ok := asFloat(expected); ok && actualFloat >= expectedFloat {
				return true, "value meets threshold"
			}
		}
		return false, fmt.Sprintf("%s below required threshold", field)
	case "contains":
		if strings.Contains(strings.ToLower(fmt.Sprint(actual)), strings.ToLower(fmt.Sprint(expected))) {
			return true, "field contains expected token"
		}
		return false, fmt.Sprintf("%s does not contain %v", field, expected)
	default:
		return false, "unsupported operator"
	}
}

func asFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		v, err := typed.Float64()
		return v, err == nil
	default:
		return 0, false
	}
}

func normalizeJSON(value []byte, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(value)
}

func mustJSONArray(raw json.RawMessage) []any {
	var items []any
	_ = json.Unmarshal(raw, &items)
	return items
}

func mustJSONObject(raw json.RawMessage) map[string]any {
	var item map[string]any
	_ = json.Unmarshal(raw, &item)
	if item == nil {
		item = map[string]any{}
	}
	return item
}
