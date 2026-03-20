package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
)

type DocumentService struct {
	db        *pgxpool.Pool
	contracts *repository.ContractRepository
	documents *repository.DocumentRepository
	publisher Publisher
	metrics   *metrics.Metrics
	topic     string
	logger    zerolog.Logger
}

func NewDocumentService(db *pgxpool.Pool, contracts *repository.ContractRepository, documents *repository.DocumentRepository, publisher Publisher, appMetrics *metrics.Metrics, topic string, logger zerolog.Logger) *DocumentService {
	return &DocumentService{
		db:        db,
		contracts: contracts,
		documents: documents,
		publisher: publisherOrNoop(publisher),
		metrics:   appMetrics,
		topic:     topic,
		logger:    logger.With().Str("service", "lex-documents").Logger(),
	}
}

func (s *DocumentService) Create(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateLegalDocumentRequest) (*model.LegalDocument, error) {
	req.Normalize()
	if strings.TrimSpace(req.Title) == "" {
		return nil, validationError("title is required", map[string]string{"title": "required"})
	}
	if req.ContractID != nil {
		if _, err := s.contracts.Get(ctx, tenantID, *req.ContractID); err != nil {
			if err == pgx.ErrNoRows {
				return nil, validationError("linked contract not found", map[string]string{"contract_id": "not found"})
			}
			return nil, internalError("load linked contract", err)
		}
	}
	document := &model.LegalDocument{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Title:           req.Title,
		Type:            req.Type,
		Description:     req.Description,
		Category:        normalizeOptionalString(req.Category),
		Confidentiality: req.Confidentiality,
		ContractID:      req.ContractID,
		CurrentVersion:  1,
		Status:          model.DocumentStatusActive,
		Tags:            req.Tags,
		Metadata:        req.Metadata,
		CreatedBy:       userID,
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start document transaction", err)
	}
	defer tx.Rollback(ctx)
	if err := s.documents.Create(ctx, tx, document); err != nil {
		return nil, internalError("create legal document", err)
	}
	if req.Document != nil {
		version := &model.DocumentVersion{
			ID:            uuid.New(),
			TenantID:      tenantID,
			DocumentID:    document.ID,
			Version:       1,
			FileID:        req.Document.FileID,
			FileName:      req.Document.FileName,
			FileSizeBytes: req.Document.FileSizeBytes,
			ContentHash:   req.Document.ContentHash,
			ChangeSummary: normalizeOptionalString(&req.Document.ChangeSummary),
			UploadedBy:    userID,
		}
		if err := s.documents.InsertVersion(ctx, tx, version); err != nil {
			return nil, internalError("create document version", err)
		}
		if err := s.documents.UpdateFile(ctx, tx, tenantID, document.ID, req.Document.FileID, req.Document.FileName, req.Document.FileSizeBytes, 1); err != nil {
			return nil, internalError("update document file", err)
		}
		document.FileID = &req.Document.FileID
		document.FileName = &req.Document.FileName
		document.FileSizeBytes = &req.Document.FileSizeBytes
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit document transaction", err)
	}
	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.document.uploaded", tenantID, &userID, map[string]any{
		"id":      document.ID,
		"title":   document.Title,
		"type":    document.Type,
		"file_id": document.FileID,
	}, s.logger)
	return document, nil
}

func (s *DocumentService) List(ctx context.Context, tenantID uuid.UUID, docType, status, search string, page, perPage int) ([]model.LegalDocument, int, error) {
	return s.documents.List(ctx, tenantID, docType, status, search, page, perPage)
}

func (s *DocumentService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.LegalDocument, error) {
	document, err := s.documents.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("document not found")
		}
		return nil, internalError("get document", err)
	}
	return document, nil
}

func (s *DocumentService) Update(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateLegalDocumentRequest) (*model.LegalDocument, error) {
	document, err := s.documents.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("document not found")
		}
		return nil, internalError("load document", err)
	}
	if req.Title != nil {
		document.Title = strings.TrimSpace(*req.Title)
	}
	if req.Type != nil {
		document.Type = *req.Type
	}
	if req.Description != nil {
		document.Description = strings.TrimSpace(*req.Description)
	}
	if req.Category != nil {
		document.Category = normalizeOptionalString(req.Category)
	}
	if req.Confidentiality != nil {
		document.Confidentiality = *req.Confidentiality
	}
	if req.ContractID != nil {
		document.ContractID = req.ContractID
	}
	if req.Status != nil {
		document.Status = *req.Status
	}
	if req.Tags != nil {
		document.Tags = dto.NormalizeTags(req.Tags)
	}
	if req.Metadata != nil {
		document.Metadata = req.Metadata
	}
	if err := s.documents.Update(ctx, s.db, document); err != nil {
		return nil, internalError("update document", err)
	}
	return document, nil
}

func (s *DocumentService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.documents.SoftDelete(ctx, tenantID, id); err != nil {
		if err == pgx.ErrNoRows {
			return notFoundError("document not found")
		}
		return internalError("delete document", err)
	}
	return nil
}

func (s *DocumentService) UploadVersion(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UploadDocumentVersionRequest) ([]model.DocumentVersion, error) {
	document, err := s.documents.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("document not found")
		}
		return nil, internalError("load document", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start document version transaction", err)
	}
	defer tx.Rollback(ctx)
	version := &model.DocumentVersion{
		ID:            uuid.New(),
		TenantID:      tenantID,
		DocumentID:    document.ID,
		Version:       document.CurrentVersion + 1,
		FileID:        req.FileID,
		FileName:      req.FileName,
		FileSizeBytes: req.FileSizeBytes,
		ContentHash:   req.ContentHash,
		ChangeSummary: normalizeOptionalString(&req.ChangeSummary),
		UploadedBy:    userID,
	}
	if err := s.documents.InsertVersion(ctx, tx, version); err != nil {
		return nil, internalError("create document version", err)
	}
	if err := s.documents.UpdateFile(ctx, tx, tenantID, document.ID, req.FileID, req.FileName, req.FileSizeBytes, version.Version); err != nil {
		return nil, internalError("update document current version", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit document version", err)
	}
	return s.documents.ListVersions(ctx, tenantID, document.ID)
}

func (s *DocumentService) ListVersions(ctx context.Context, tenantID, id uuid.UUID) ([]model.DocumentVersion, error) {
	return s.documents.ListVersions(ctx, tenantID, id)
}
