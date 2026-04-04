package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/data/analytics"
	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

type AnalyticsService struct {
	repo          *repository.AnalyticsRepository
	modelRepo     *repository.ModelRepository
	sourceRepo    *repository.SourceRepository
	registry      *connector.ConnectorRegistry
	encryptor     *ConfigEncryptor
	auditRecorder *analytics.AuditRecorder
	lineage       *LineageService
	producer      *events.Producer
	logger        zerolog.Logger
}

func NewAnalyticsService(repo *repository.AnalyticsRepository, modelRepo *repository.ModelRepository, sourceRepo *repository.SourceRepository, registry *connector.ConnectorRegistry, encryptor *ConfigEncryptor, auditRecorder *analytics.AuditRecorder, lineage *LineageService, producer *events.Producer, logger zerolog.Logger) *AnalyticsService {
	return &AnalyticsService{
		repo:          repo,
		modelRepo:     modelRepo,
		sourceRepo:    sourceRepo,
		registry:      registry,
		encryptor:     encryptor,
		auditRecorder: auditRecorder,
		lineage:       lineage,
		producer:      producer,
		logger:        logger,
	}
}

func (s *AnalyticsService) Execute(ctx context.Context, tenantID, userID uuid.UUID, permissions []string, req dto.ExecuteAnalyticsQueryRequest, savedQueryID *uuid.UUID, ipAddress, userAgent string) (*model.QueryResult, error) {
	modelItem, source, sourceConfig, err := s.loadModelAndSource(ctx, tenantID, req.ModelID)
	if err != nil {
		return nil, err
	}
	source.ConnectionConfig = sourceConfig

	validation, err := analytics.AnalyzeQuery(&req.Query, modelItem, permissions, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", classifyAnalyticsValidationError(err), err)
	}
	built, err := analytics.BuildSQL(&req.Query, modelItem, source)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	conn, err := s.registry.Create(source.Type, sourceConfig)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	started := time.Now()

	var result *model.QueryResult
	var executionErr error
	defer func() {
		executionMs := time.Since(started).Milliseconds()
		if s.auditRecorder == nil {
			return
		}
		rowsReturned := 0
		truncated := false
		if result != nil {
			rowsReturned = result.RowCount
			truncated = result.Truncated
		}
		errorMessage := ""
		if executionErr != nil {
			errorMessage = executionErr.Error()
		}
		s.auditRecorder.RecordQueryExecution(ctx, tenantID, userID, modelItem.ID, source.ID, req.Query, string(modelItem.DataClassification), validation.ColumnsAccessed, validation.PIIColumnsAccessed, !validation.UserHasPIIPermission && len(validation.PIIColumnsAccessed) > 0, rowsReturned, truncated, executionMs, executionErr != nil, errorMessage, savedQueryID, ipAddress, userAgent)
	}()

	dataBatch, err := conn.ReadQuery(queryCtx, built.SQL, built.Args)
	if err != nil {
		executionErr = translateAnalyticsError(err, queryCtx)
		return nil, executionErr
	}
	countBatch, err := conn.ReadQuery(queryCtx, built.CountSQL, built.CountArgs)
	if err != nil {
		executionErr = translateAnalyticsError(err, queryCtx)
		return nil, executionErr
	}
	totalCount := extractCount(countBatch)
	maskedRows, masking := analytics.ApplyPIIMasking(dataBatch.Rows, modelItem, validation.UserHasPIIPermission)
	result = analytics.FormatResult(dataBatch.Columns, maskedRows, modelItem, totalCount, masking, time.Since(started).Milliseconds())

	if savedQueryID != nil {
		_ = s.repo.TouchSavedQueryRun(ctx, tenantID, *savedQueryID, time.Now().UTC())
	}
	if s.lineage != nil {
		_ = s.lineage.RecordQueryExecution(ctx, tenantID, userID, modelItem.ID, req.Query)
	}
	_ = s.publishQueryEvent(ctx, tenantID, userID, modelItem, validation, result.RowCount)
	return result, nil
}

func (s *AnalyticsService) Explore(ctx context.Context, tenantID, userID, modelID uuid.UUID, permissions []string, query model.AnalyticsQuery, ipAddress, userAgent string) (*model.QueryResult, error) {
	query.Limit = minInt(query.Limit, 100)
	if query.Limit <= 0 {
		query.Limit = 100
	}
	return s.Execute(ctx, tenantID, userID, permissions, dto.ExecuteAnalyticsQueryRequest{
		ModelID: modelID,
		Query:   query,
	}, nil, ipAddress, userAgent)
}

func (s *AnalyticsService) Explain(ctx context.Context, tenantID uuid.UUID, permissions []string, req dto.ExplainAnalyticsQueryRequest) (*model.QueryExplain, error) {
	modelItem, source, sourceConfig, err := s.loadModelAndSource(ctx, tenantID, req.ModelID)
	if err != nil {
		return nil, err
	}
	source.ConnectionConfig = sourceConfig
	if _, err := analytics.AnalyzeQuery(&req.Query, modelItem, permissions, false); err != nil {
		return nil, fmt.Errorf("%w: %v", classifyAnalyticsValidationError(err), err)
	}
	built, err := analytics.BuildSQL(&req.Query, modelItem, source)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return &model.QueryExplain{
		SQL:        built.SQL,
		CountSQL:   built.CountSQL,
		Parameters: built.Args,
	}, nil
}

func (s *AnalyticsService) CreateSavedQuery(ctx context.Context, tenantID, userID uuid.UUID, permissions []string, req dto.SaveQueryRequest) (*model.SavedQuery, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	modelItem, _, _, err := s.loadModelAndSource(ctx, tenantID, req.ModelID)
	if err != nil {
		return nil, err
	}
	if _, err := analytics.AnalyzeQuery(&req.QueryDefinition, modelItem, permissions, false); err != nil {
		return nil, fmt.Errorf("%w: %v", classifyAnalyticsValidationError(err), err)
	}
	visibility := model.SavedQueryVisibility(strings.TrimSpace(req.Visibility))
	if visibility == "" {
		visibility = model.SavedQueryVisibilityPrivate
	}
	switch visibility {
	case model.SavedQueryVisibilityPrivate, model.SavedQueryVisibilityTeam, model.SavedQueryVisibilityOrganization:
	default:
		return nil, fmt.Errorf("%w: invalid visibility", ErrValidation)
	}
	item := &model.SavedQuery{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            strings.TrimSpace(req.Name),
		Description:     req.Description,
		ModelID:         req.ModelID,
		QueryDefinition: req.QueryDefinition,
		Visibility:      visibility,
		Tags:            req.Tags,
		CreatedBy:       userID,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if err := s.repo.CreateSavedQuery(ctx, item); err != nil {
		return nil, err
	}
	_ = s.publish(ctx, "data.analytics.saved_query_created", tenantID, map[string]any{
		"id":       item.ID,
		"name":     item.Name,
		"model_id": item.ModelID,
	})
	return item, nil
}

func (s *AnalyticsService) ListSavedQueries(ctx context.Context, tenantID, userID uuid.UUID, params dto.ListSavedQueriesParams) ([]*model.SavedQuery, int, error) {
	return s.repo.ListSavedQueries(ctx, tenantID, userID, params)
}

func (s *AnalyticsService) GetSavedQuery(ctx context.Context, tenantID, userID uuid.UUID, id uuid.UUID) (*model.SavedQuery, error) {
	item, err := s.repo.GetSavedQuery(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := ensureSavedQueryAccess(item, userID); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *AnalyticsService) UpdateSavedQuery(ctx context.Context, tenantID, userID uuid.UUID, permissions []string, id uuid.UUID, req dto.UpdateSavedQueryRequest) (*model.SavedQuery, error) {
	item, err := s.repo.GetSavedQuery(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if item.CreatedBy != userID {
		return nil, fmt.Errorf("%w: only the creator can update a saved query", ErrForbiddenOperation)
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.QueryDefinition != nil {
		modelItem, _, _, err := s.loadModelAndSource(ctx, tenantID, item.ModelID)
		if err != nil {
			return nil, err
		}
		if _, err := analytics.AnalyzeQuery(req.QueryDefinition, modelItem, permissions, false); err != nil {
			return nil, fmt.Errorf("%w: %v", classifyAnalyticsValidationError(err), err)
		}
		item.QueryDefinition = *req.QueryDefinition
	}
	if req.Visibility != nil {
		visibility := model.SavedQueryVisibility(strings.TrimSpace(*req.Visibility))
		switch visibility {
		case model.SavedQueryVisibilityPrivate, model.SavedQueryVisibilityTeam, model.SavedQueryVisibilityOrganization:
			item.Visibility = visibility
		default:
			return nil, fmt.Errorf("%w: invalid visibility", ErrValidation)
		}
	}
	if req.Tags != nil {
		item.Tags = req.Tags
	}
	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateSavedQuery(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *AnalyticsService) DeleteSavedQuery(ctx context.Context, tenantID, userID, id uuid.UUID) error {
	return s.repo.SoftDeleteSavedQuery(ctx, tenantID, id, userID, time.Now().UTC())
}

func (s *AnalyticsService) RunSavedQuery(ctx context.Context, tenantID, userID uuid.UUID, permissions []string, id uuid.UUID, ipAddress, userAgent string) (*model.QueryResult, error) {
	item, err := s.repo.GetSavedQuery(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := ensureSavedQueryAccess(item, userID); err != nil {
		return nil, err
	}
	return s.Execute(ctx, tenantID, userID, permissions, dto.ExecuteAnalyticsQueryRequest{
		ModelID: item.ModelID,
		Query:   item.QueryDefinition,
	}, &item.ID, ipAddress, userAgent)
}

func (s *AnalyticsService) ListAudit(ctx context.Context, tenantID uuid.UUID, params dto.ListAnalyticsAuditParams) ([]*model.AnalyticsAuditLog, int, error) {
	return s.repo.ListAuditLogs(ctx, tenantID, params)
}

func (s *AnalyticsService) loadModelAndSource(ctx context.Context, tenantID, modelID uuid.UUID) (*model.DataModel, *model.DataSource, json.RawMessage, error) {
	modelItem, err := s.modelRepo.Get(ctx, tenantID, modelID)
	if err != nil {
		return nil, nil, nil, err
	}
	if modelItem.SourceID == nil {
		return nil, nil, nil, fmt.Errorf("%w: model %q is not linked to a source", ErrValidation, modelItem.Name)
	}
	sourceRecord, err := s.sourceRepo.Get(ctx, tenantID, *modelItem.SourceID)
	if err != nil {
		return nil, nil, nil, err
	}
	if sourceRecord.Source.Type != model.DataSourceTypePostgreSQL && sourceRecord.Source.Type != model.DataSourceTypeMySQL {
		return nil, nil, nil, fmt.Errorf("%w: analytics queries are supported for postgresql and mysql sources only", ErrUnsupportedType)
	}
	config, err := s.decryptSourceConfig(sourceRecord)
	if err != nil {
		return nil, nil, nil, err
	}
	sourceCopy := *sourceRecord.Source
	return modelItem, &sourceCopy, config, nil
}

func (s *AnalyticsService) decryptSourceConfig(record *repository.SourceRecord) (json.RawMessage, error) {
	plaintext, err := s.encryptor.Decrypt(record.EncryptedConfig)
	if err != nil {
		s.logger.Error().Err(err).Str("source_id", record.Source.ID.String()).Msg("failed to decrypt analytics source config")
		return nil, fmt.Errorf("decrypt data source connection config")
	}
	defer zeroBytes(plaintext)
	return json.RawMessage(append([]byte(nil), plaintext...)), nil
}

func (s *AnalyticsService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if s.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, "data.analytics.events", event)
}

func (s *AnalyticsService) publishQueryEvent(ctx context.Context, tenantID, userID uuid.UUID, modelItem *model.DataModel, validation *analytics.ValidationContext, rowsReturned int) error {
	return s.publish(ctx, "data.analytics.query_executed", tenantID, map[string]any{
		"user_id":        userID,
		"model_id":       modelItem.ID,
		"rows_returned":  rowsReturned,
		"pii_accessed":   len(validation.PIIColumnsAccessed) > 0,
		"classification": modelItem.DataClassification,
	})
}

func ensureSavedQueryAccess(item *model.SavedQuery, userID uuid.UUID) error {
	if item.Visibility == model.SavedQueryVisibilityPrivate && item.CreatedBy != userID {
		return fmt.Errorf("%w: private saved query", ErrForbiddenOperation)
	}
	return nil
}

func extractCount(batch *connector.DataBatch) int64 {
	if batch == nil || len(batch.Rows) == 0 {
		return 0
	}
	for _, row := range batch.Rows {
		for _, value := range row {
			switch cast := value.(type) {
			case int64:
				return cast
			case int32:
				return int64(cast)
			case int:
				return int64(cast)
			case float64:
				return int64(cast)
			case string:
				if parsed, err := parseInt64(cast); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func parseInt64(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty")
	}
	var parsed int64
	_, err := fmt.Sscan(value, &parsed)
	return parsed, err
}

func translateAnalyticsError(err error, ctx context.Context) error {
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%w: analytics query exceeded the 30s execution timeout", ErrTimeout)
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fmt.Errorf("%w: analytics query exceeded the 30s execution timeout", ErrTimeout)
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return fmt.Errorf("%w: analytics query exceeded the 30s execution timeout", ErrTimeout)
	}
	return err
}

func classifyAnalyticsValidationError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "permission") {
		return ErrForbiddenOperation
	}
	return ErrValidation
}

func minInt(left, right int) int {
	if left <= 0 {
		return right
	}
	if left < right {
		return left
	}
	return right
}

func PermissionsFromContext(ctx context.Context) []string {
	claims := auth.ClaimsFromContext(ctx)
	user := auth.UserFromContext(ctx)
	values := make([]string, 0)
	if claims != nil {
		values = append(values, claims.Permissions...)
	}
	if user != nil {
		for _, role := range user.Roles {
			perms := auth.RolePermissions[role]
			values = append(values, perms...)
		}
	}
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
