package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	dataconfig "github.com/clario360/platform/internal/data/config"
	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/dto"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

const (
	dataSourceEventsTopic = "data.source.events"
)

type SourceService struct {
	config           *dataconfig.Config
	sourceRepo       *repository.SourceRepository
	syncRepo         *repository.SyncRepository
	tester           *ConnectionTester
	discovery        *SchemaDiscoveryService
	ingestion        *IngestionService
	encryptor        *ConfigEncryptor
	producer         *events.Producer
	metrics          *datametrics.Metrics
	logger           zerolog.Logger
	predictionLogger *aigovmiddleware.PredictionLogger
}

func NewSourceService(
	config *dataconfig.Config,
	sourceRepo *repository.SourceRepository,
	syncRepo *repository.SyncRepository,
	tester *ConnectionTester,
	discovery *SchemaDiscoveryService,
	ingestion *IngestionService,
	encryptor *ConfigEncryptor,
	producer *events.Producer,
	metrics *datametrics.Metrics,
	logger zerolog.Logger,
) *SourceService {
	return &SourceService{
		config:     config,
		sourceRepo: sourceRepo,
		syncRepo:   syncRepo,
		tester:     tester,
		discovery:  discovery,
		ingestion:  ingestion,
		encryptor:  encryptor,
		producer:   producer,
		metrics:    metrics,
		logger:     logger,
	}
}

func (s *SourceService) Create(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateSourceRequest) (*model.DataSource, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	sourceType := model.DataSourceType(strings.TrimSpace(req.Type))
	if !sourceType.IsValid() {
		return nil, fmt.Errorf("%w: invalid data source type", ErrValidation)
	}

	normalizedConfig, err := normalizeConnectionConfig(sourceType, req.ConnectionConfig, s.config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if exists, err := s.sourceRepo.ExistsByName(ctx, tenantID, req.Name, nil); err != nil {
		return nil, err
	} else if exists {
		return nil, fmt.Errorf("%w: a data source named %q already exists", ErrConflict, req.Name)
	}

	testCtx, cancel := context.WithTimeout(ctx, s.config.ConnectorConnectTimeout)
	defer cancel()
	testResult, err := s.tester.Test(testCtx, sourceType, normalizedConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionTestFailed, err)
	}
	if !testResult.Success {
		return nil, fmt.Errorf("%w: %s", ErrConnectionTestFailed, testResult.Message)
	}

	plaintext := append([]byte(nil), normalizedConfig...)
	encryptedConfig, keyID, err := s.encryptor.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataEncryptionOperationsTotal.WithLabelValues("encrypt").Inc()
	}

	now := time.Now().UTC()
	source := &model.DataSource{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		Type:            sourceType,
		EncryptionKeyID: keyID,
		Status:          model.DataSourceStatusActive,
		SyncFrequency:   req.SyncFrequency,
		NextSyncAt:      computeNextSync(req.SyncFrequency, now),
		Tags:            safeTags(req.Tags),
		Metadata:        coalesceJSON(req.Metadata, json.RawMessage(`{}`)),
		CreatedBy:       userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.sourceRepo.Create(ctx, &repository.SourceRecord{
		Source:          source,
		EncryptedConfig: encryptedConfig,
	}); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataSourceOperationsTotal.WithLabelValues("create").Inc()
		s.metrics.DataSourcesTotal.WithLabelValues(tenantID.String(), string(source.Type), string(source.Status)).Inc()
	}
	_ = s.publishSourceEvent(ctx, "data.source.created", tenantID, map[string]any{
		"id":        source.ID,
		"name":      source.Name,
		"type":      source.Type,
		"tenant_id": source.TenantID,
	})
	_ = s.publishSourceEvent(ctx, "data.source.connection_tested", tenantID, map[string]any{
		"id":         source.ID,
		"success":    testResult.Success,
		"latency_ms": testResult.LatencyMs,
	})

	sanitized := *source
	sanitized.ConnectionConfig = connector.SanitizeConnectionConfig(source.Type, normalizedConfig)
	return &sanitized, nil
}

func (s *SourceService) List(ctx context.Context, tenantID uuid.UUID, params dto.ListSourcesParams) ([]*model.DataSource, int, error) {
	items, total, err := s.sourceRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, 0, err
	}
	values := make([]*model.DataSource, 0, len(items))
	for _, item := range items {
		sanitized, err := s.sanitizeSourceRecord(ctx, item)
		if err != nil {
			return nil, 0, err
		}
		values = append(values, sanitized)
	}
	return values, total, nil
}

func (s *SourceService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DataSource, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return s.sanitizeSourceRecord(ctx, record)
}

func (s *SourceService) Update(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateSourceRequest) (*model.DataSource, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	source := record.Source

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" && !strings.EqualFold(strings.TrimSpace(*req.Name), source.Name) {
		if exists, err := s.sourceRepo.ExistsByName(ctx, tenantID, strings.TrimSpace(*req.Name), &id); err != nil {
			return nil, err
		} else if exists {
			return nil, fmt.Errorf("%w: a data source named %q already exists", ErrConflict, strings.TrimSpace(*req.Name))
		}
		source.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		source.Description = *req.Description
	}
	if req.SyncFrequency != nil {
		source.SyncFrequency = req.SyncFrequency
		now := time.Now().UTC()
		source.NextSyncAt = computeNextSync(req.SyncFrequency, now)
	}
	if req.Tags != nil {
		source.Tags = safeTags(req.Tags)
	}
	if len(req.Metadata) > 0 {
		source.Metadata = req.Metadata
	}
	source.UpdatedAt = time.Now().UTC()

	encryptedConfig := record.EncryptedConfig
	if len(req.ConnectionConfig) > 0 {
		normalizedConfig, err := normalizeConnectionConfig(source.Type, req.ConnectionConfig, s.config)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		testCtx, cancel := context.WithTimeout(ctx, s.config.ConnectorConnectTimeout)
		defer cancel()
		if _, err := s.tester.Test(testCtx, source.Type, normalizedConfig); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrConnectionTestFailed, err)
		}
		plaintext := append([]byte(nil), normalizedConfig...)
		encryptedConfig, source.EncryptionKeyID, err = s.encryptor.Encrypt(plaintext)
		if err != nil {
			return nil, err
		}
		if s.metrics != nil {
			s.metrics.DataEncryptionOperationsTotal.WithLabelValues("encrypt").Inc()
		}
	}

	if err := s.sourceRepo.Update(ctx, &repository.SourceRecord{
		Source:          source,
		EncryptedConfig: encryptedConfig,
	}); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataSourceOperationsTotal.WithLabelValues("update").Inc()
	}

	_ = s.publishSourceEvent(ctx, "data.source.updated", tenantID, map[string]any{
		"id":             source.ID,
		"name":           source.Name,
		"changed_fields": changedFields(req),
	})
	updatedRecord := &repository.SourceRecord{Source: source, EncryptedConfig: encryptedConfig}
	return s.sanitizeSourceRecord(ctx, updatedRecord)
}

func (s *SourceService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := s.sourceRepo.SoftDelete(ctx, tenantID, id, time.Now().UTC()); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.DataSourceOperationsTotal.WithLabelValues("delete").Inc()
		s.metrics.DataSourcesTotal.WithLabelValues(tenantID.String(), string(record.Source.Type), string(record.Source.Status)).Dec()
	}
	_ = s.publishSourceEvent(ctx, "data.source.deleted", tenantID, map[string]any{
		"id":   record.Source.ID,
		"name": record.Source.Name,
	})
	return nil
}

func (s *SourceService) ChangeStatus(ctx context.Context, tenantID, id uuid.UUID, status model.DataSourceStatus) (*model.DataSource, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	oldStatus := record.Source.Status
	if err := s.sourceRepo.UpdateStatus(ctx, tenantID, id, status, nil); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataSourceOperationsTotal.WithLabelValues("change_status").Inc()
		s.metrics.DataSourcesTotal.WithLabelValues(tenantID.String(), string(record.Source.Type), string(oldStatus)).Dec()
		s.metrics.DataSourcesTotal.WithLabelValues(tenantID.String(), string(record.Source.Type), string(status)).Inc()
	}
	_ = s.publishSourceEvent(ctx, "data.source.status_changed", tenantID, map[string]any{
		"id":         id,
		"old_status": oldStatus,
		"new_status": status,
	})
	record.Source.Status = status
	return s.sanitizeSourceRecord(ctx, record)
}

func (s *SourceService) TestConnection(ctx context.Context, tenantID, id uuid.UUID) (*connector.ConnectionTestResult, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	configJSON, err := s.loadDecryptedConfig(ctx, record)
	if err != nil {
		return nil, err
	}
	result, err := s.tester.Test(ctx, record.Source.Type, configJSON)
	if err != nil {
		return nil, err
	}
	_ = s.publishSourceEvent(ctx, "data.source.connection_tested", tenantID, map[string]any{
		"id":         id,
		"success":    result.Success,
		"latency_ms": result.LatencyMs,
	})
	return result, nil
}

func (s *SourceService) TestConfig(ctx context.Context, tenantID uuid.UUID, req dto.TestSourceConfigRequest) (*connector.ConnectionTestResult, error) {
	sourceType := model.DataSourceType(strings.TrimSpace(req.Type))
	if !sourceType.IsValid() {
		return nil, fmt.Errorf("%w: invalid data source type", ErrValidation)
	}
	normalizedConfig, err := normalizeConnectionConfig(sourceType, req.ConnectionConfig, s.config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	result, err := s.tester.Test(ctx, sourceType, normalizedConfig)
	if err != nil {
		return nil, err
	}
	_ = s.publishSourceEvent(ctx, "data.source.connection_tested", tenantID, map[string]any{
		"success":    result.Success,
		"latency_ms": result.LatencyMs,
		"type":       sourceType,
	})
	return result, nil
}

func (s *SourceService) DiscoverSchema(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	configJSON, err := s.loadDecryptedConfig(ctx, record)
	if err != nil {
		return nil, err
	}
	schema, estimate, err := s.discovery.Discover(ctx, record.Source.Type, configJSON)
	if err != nil {
		lastError := err.Error()
		_ = s.sourceRepo.UpdateStatus(ctx, tenantID, id, model.DataSourceStatusError, &lastError)
		return nil, err
	}
	if err := s.sourceRepo.UpdateSchema(ctx, tenantID, id, schema, time.Now().UTC(), &repository.SizeEstimatePatch{
		TableCount: estimate.TableCount,
		TotalRows:  estimate.TotalRows,
		TotalBytes: estimate.TotalBytes,
	}); err != nil {
		return nil, err
	}
	_ = s.publishSourceEvent(ctx, "data.source.schema_discovered", tenantID, map[string]any{
		"id":           id,
		"table_count":  schema.TableCount,
		"column_count": schema.ColumnCount,
		"pii_detected": schema.ContainsPII,
	})
	s.recordPIIPrediction(ctx, tenantID, id, record.Source, schema)
	return schema, nil
}

func (s *SourceService) GetSchema(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if record.Source.SchemaMetadata == nil {
		return nil, pgx.ErrNoRows
	}
	return record.Source.SchemaMetadata, nil
}

func (s *SourceService) TriggerSync(ctx context.Context, tenantID, id uuid.UUID, syncType model.SyncType, userID *uuid.UUID) (*model.SyncHistory, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	configJSON, err := s.loadDecryptedConfig(ctx, record)
	if err != nil {
		return nil, err
	}
	_ = s.publishSourceEvent(ctx, "data.source.sync_started", tenantID, map[string]any{
		"id":        id,
		"sync_type": syncType,
	})
	history, err := s.ingestion.Run(ctx, record, configJSON, syncType, model.SyncTriggerAPI, userID)
	if err != nil {
		_ = s.publishSourceEvent(ctx, "data.source.sync_failed", tenantID, map[string]any{
			"id":    id,
			"error": err.Error(),
		})
		return nil, err
	}
	if history.Status == model.SyncStatusSuccess || history.Status == model.SyncStatusPartial {
		_ = s.publishSourceEvent(ctx, "data.source.sync_completed", tenantID, map[string]any{
			"id":           id,
			"rows_read":    history.RowsRead,
			"rows_written": history.RowsWritten,
			"duration_ms":  history.DurationMs,
		})
	}
	return history, nil
}

func (s *SourceService) ListSyncHistory(ctx context.Context, tenantID, id uuid.UUID, limit int) ([]*model.SyncHistory, error) {
	return s.syncRepo.ListBySource(ctx, tenantID, id, limit)
}

func (s *SourceService) GetStats(ctx context.Context, tenantID, id uuid.UUID) (*dto.SourceStatsResponse, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return &dto.SourceStatsResponse{
		TableCount:         derefInt(record.Source.TableCount),
		TotalRowCount:      derefInt64(record.Source.TotalRowCount),
		TotalSizeBytes:     derefInt64(record.Source.TotalSizeBytes),
		SchemaDiscoveredAt: record.Source.SchemaDiscoveredAt,
		LastSyncedAt:       record.Source.LastSyncedAt,
		LastSyncStatus:     record.Source.LastSyncStatus,
	}, nil
}

func (s *SourceService) AggregateStats(ctx context.Context, tenantID uuid.UUID) (*dto.AggregateSourceStatsResponse, error) {
	return s.sourceRepo.AggregateStats(ctx, tenantID)
}

func (s *SourceService) SetPredictionLogger(predictionLogger *aigovmiddleware.PredictionLogger) {
	s.predictionLogger = predictionLogger
}

func (s *SourceService) recordPIIPrediction(ctx context.Context, tenantID, sourceID uuid.UUID, source *model.DataSource, schema *model.DiscoveredSchema) {
	if s.predictionLogger == nil || source == nil || schema == nil {
		return
	}

	piiColumns := make([]string, 0, 12)
	piiTypes := make(map[string]int)
	matchedConditions := make([]string, 0, 12)
	for _, table := range schema.Tables {
		for _, column := range table.Columns {
			if !column.InferredPII || strings.TrimSpace(column.InferredPIIType) == "" {
				continue
			}
			piiTypes[column.InferredPIIType]++
			if len(piiColumns) < 12 {
				piiColumns = append(piiColumns, fmt.Sprintf("%s.%s", table.Name, column.Name))
			}
		}
	}
	for piiType, count := range piiTypes {
		matchedConditions = append(matchedConditions, fmt.Sprintf("%s:%d", piiType, count))
	}
	if len(matchedConditions) == 0 {
		matchedConditions = append(matchedConditions, "no_pii_columns_detected")
	}

	matchedRules := []string{"column_name_heuristics"}
	if len(piiColumns) > 0 {
		matchedRules = append(matchedRules, "sample_value_patterns")
	}

	confidence := 0.9
	if !schema.ContainsPII {
		confidence = 0.82
	}

	input := map[string]any{
		"source_id":    sourceID.String(),
		"source_name":  source.Name,
		"source_type":  source.Type,
		"table_count":  schema.TableCount,
		"column_count": schema.ColumnCount,
		"contains_pii": schema.ContainsPII,
	}
	_, _ = s.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "data-pii-classifier",
		UseCase:      "pii_classification",
		EntityType:   "data_source",
		EntityID:     &sourceID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"contains_pii": schema.ContainsPII,
					"pii_columns":  piiColumns,
					"pii_types":    piiTypes,
					"table_count":  schema.TableCount,
					"column_count": schema.ColumnCount,
				},
				Confidence: confidence,
				Metadata: map[string]any{
					"matched_rules":      matchedRules,
					"matched_conditions": matchedConditions,
					"rule_weights": map[string]any{
						"column_name_heuristics": 0.45,
						"sample_value_patterns":  0.55,
					},
					"pii_columns":  piiColumns,
					"pii_types":    piiTypes,
					"table_count":  schema.TableCount,
					"column_count": schema.ColumnCount,
				},
			}, nil
		},
	})
}

func (s *SourceService) loadDecryptedConfig(ctx context.Context, record *repository.SourceRecord) (json.RawMessage, error) {
	if len(record.EncryptedConfig) == 0 {
		return nil, fmt.Errorf("%w: connection config is empty", ErrValidation)
	}
	plaintext, err := s.encryptor.Decrypt(record.EncryptedConfig)
	if err == nil {
		if s.metrics != nil {
			s.metrics.DataEncryptionOperationsTotal.WithLabelValues("decrypt").Inc()
		}
		defer zeroBytes(plaintext)
		return json.RawMessage(append([]byte(nil), plaintext...)), nil
	}
	if json.Valid(record.EncryptedConfig) {
		legacy := append([]byte(nil), record.EncryptedConfig...)
		encrypted, keyID, encryptErr := s.encryptor.Encrypt(append([]byte(nil), legacy...))
		if encryptErr == nil {
			if s.metrics != nil {
				s.metrics.DataEncryptionOperationsTotal.WithLabelValues("encrypt").Inc()
			}
			record.Source.EncryptionKeyID = keyID
			updateErr := s.sourceRepo.Update(ctx, &repository.SourceRecord{
				Source:          record.Source,
				EncryptedConfig: encrypted,
			})
			if updateErr != nil {
				s.logger.Warn().Err(updateErr).Str("source_id", record.Source.ID.String()).Msg("failed to migrate legacy plaintext connection config")
			} else {
				record.EncryptedConfig = encrypted
			}
		}
		return legacy, nil
	}
	s.logger.Error().Err(err).Str("source_id", record.Source.ID.String()).Msg("failed to decrypt data source connection config")
	return nil, fmt.Errorf("decrypt data source connection config")
}

func (s *SourceService) sanitizeSourceRecord(ctx context.Context, record *repository.SourceRecord) (*model.DataSource, error) {
	configJSON, err := s.loadDecryptedConfig(ctx, record)
	if err != nil {
		return nil, err
	}
	sanitized := *record.Source
	sanitized.ConnectionConfig = connector.SanitizeConnectionConfig(record.Source.Type, configJSON)
	return &sanitized, nil
}

func (s *SourceService) publishSourceEvent(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if s.producer == nil {
		return nil
	}
	evt, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, dataSourceEventsTopic, evt)
}

func normalizeConnectionConfig(sourceType model.DataSourceType, raw json.RawMessage, cfg *dataconfig.Config) (json.RawMessage, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("connection_config must be valid JSON")
	}

	switch sourceType {
	case model.DataSourceTypePostgreSQL:
		var value model.PostgresConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode PostgreSQL config: %w", err)
		}
		if value.Port == 0 {
			value.Port = 5432
		}
		if value.Schema == "" {
			value.Schema = "public"
		}
		if value.StatementTimeoutMs == 0 {
			value.StatementTimeoutMs = int(cfg.ConnectorStatementTimeout.Milliseconds())
		}
		return json.Marshal(value)
	case model.DataSourceTypeMySQL:
		var value model.MySQLConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode MySQL config: %w", err)
		}
		if value.Port == 0 {
			value.Port = 3306
		}
		return json.Marshal(value)
	case model.DataSourceTypeAPI:
		var value model.APIConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode API config: %w", err)
		}
		if value.RateLimit == 0 {
			value.RateLimit = cfg.ConnectorAPIRateLimit
		}
		if value.QueryParams == nil {
			value.QueryParams = map[string]string{}
		}
		if value.Headers == nil {
			value.Headers = map[string]string{}
		}
		if value.AuthConfig == nil {
			value.AuthConfig = map[string]any{}
		}
		return json.Marshal(value)
	case model.DataSourceTypeCSV:
		var value model.CSVConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode CSV config: %w", err)
		}
		if value.MinioEndpoint == "" {
			value.MinioEndpoint = cfg.MinIOEndpoint
		}
		if value.AccessKey == "" {
			value.AccessKey = cfg.MinIOAccessKey
		}
		if value.SecretKey == "" {
			value.SecretKey = cfg.MinIOSecretKey
		}
		if value.Bucket == "" {
			value.Bucket = cfg.MinIOBucket
		}
		if value.Delimiter == "" {
			value.Delimiter = ","
		}
		if value.Encoding == "" {
			value.Encoding = "utf-8"
		}
		if value.QuoteChar == "" {
			value.QuoteChar = `"`
		}
		return json.Marshal(value)
	case model.DataSourceTypeS3:
		var value model.S3ConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode S3 config: %w", err)
		}
		if value.Endpoint == "" {
			value.Endpoint = cfg.MinIOEndpoint
		}
		if value.Bucket == "" {
			value.Bucket = cfg.MinIOBucket
		}
		if value.AccessKey == "" {
			value.AccessKey = cfg.MinIOAccessKey
		}
		if value.SecretKey == "" {
			value.SecretKey = cfg.MinIOSecretKey
		}
		if len(value.AllowedFormats) == 0 {
			value.AllowedFormats = []string{"csv", "tsv", "json", "jsonl", "ndjson"}
		}
		return json.Marshal(value)
	case model.DataSourceTypeClickHouse:
		var value model.ClickHouseConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode ClickHouse config: %w", err)
		}
		if value.Port == 0 {
			value.Port = 9000
		}
		if value.Protocol == "" {
			value.Protocol = "native"
		}
		if value.MaxOpenConns == 0 {
			value.MaxOpenConns = cfg.ConnectorMaxPoolSize
		}
		if value.MaxIdleConns == 0 {
			value.MaxIdleConns = max(1, cfg.ConnectorMaxPoolSize/2)
		}
		if value.DialTimeoutSeconds == 0 {
			value.DialTimeoutSeconds = int(cfg.ConnectorConnectTimeout.Seconds())
		}
		if value.ReadTimeoutSeconds == 0 {
			value.ReadTimeoutSeconds = int(cfg.ConnectorStatementTimeout.Seconds())
		}
		if !bytes.Contains(raw, []byte(`"compression"`)) {
			value.Compression = true
		}
		return json.Marshal(value)
	case model.DataSourceTypeImpala:
		var value model.ImpalaConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode Impala config: %w", err)
		}
		if value.Port == 0 {
			value.Port = 21050
		}
		if value.AuthType == "" {
			value.AuthType = "noauth"
		}
		if value.QueryTimeoutSeconds == 0 {
			value.QueryTimeoutSeconds = int(cfg.ConnectorStatementTimeout.Seconds())
		}
		if value.MaxOpenConns == 0 {
			value.MaxOpenConns = minInt(cfg.ConnectorMaxPoolSize, 5)
		}
		if value.MaxIdleConns == 0 {
			value.MaxIdleConns = max(1, value.MaxOpenConns/2)
		}
		return json.Marshal(value)
	case model.DataSourceTypeHive:
		var value model.HiveConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode Hive config: %w", err)
		}
		if value.Port == 0 {
			value.Port = 10000
		}
		if value.AuthType == "" {
			value.AuthType = "noauth"
		}
		if value.TransportMode == "" {
			value.TransportMode = "binary"
		}
		if value.HTTPPath == "" {
			value.HTTPPath = "cliservice"
		}
		if value.QueryTimeoutSeconds == 0 {
			value.QueryTimeoutSeconds = int((2 * cfg.ConnectorStatementTimeout).Seconds())
		}
		if value.FetchSize == 0 {
			value.FetchSize = cfg.ConnectorMaxSampleRows
		}
		return json.Marshal(value)
	case model.DataSourceTypeHDFS:
		var value model.HDFSConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode HDFS config: %w", err)
		}
		if len(value.BasePaths) == 0 {
			value.BasePaths = []string{"/user/hive/warehouse"}
		}
		if value.MaxFileSizeMB == 0 {
			value.MaxFileSizeMB = 100
		}
		return json.Marshal(value)
	case model.DataSourceTypeSpark:
		var value model.SparkConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode Spark config: %w", err)
		}
		if value.Thrift != nil {
			if value.Thrift.Port == 0 {
				value.Thrift.Port = 10001
			}
			if value.Thrift.AuthType == "" {
				value.Thrift.AuthType = "noauth"
			}
		}
		if value.QueryTimeoutSeconds == 0 {
			value.QueryTimeoutSeconds = int((2 * cfg.ConnectorStatementTimeout).Seconds())
		}
		if value.MaxOpenConns == 0 {
			value.MaxOpenConns = minInt(cfg.ConnectorMaxPoolSize, 5)
		}
		if value.MaxIdleConns == 0 {
			value.MaxIdleConns = max(1, value.MaxOpenConns/2)
		}
		return json.Marshal(value)
	case model.DataSourceTypeDagster:
		var value model.DagsterConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode Dagster config: %w", err)
		}
		if value.TimeoutSeconds == 0 {
			value.TimeoutSeconds = int(cfg.ConnectorStatementTimeout.Seconds())
		}
		return json.Marshal(value)
	case model.DataSourceTypeDolt:
		var value model.DoltConnectionConfig
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode Dolt config: %w", err)
		}
		if value.Port == 0 {
			value.Port = 3306
		}
		if value.Branch == "" {
			value.Branch = "main"
		}
		if value.MaxOpenConns == 0 {
			value.MaxOpenConns = cfg.ConnectorMaxPoolSize
		}
		if value.MaxIdleConns == 0 {
			value.MaxIdleConns = max(1, cfg.ConnectorMaxPoolSize/2)
		}
		return json.Marshal(value)
	default:
		return nil, fmt.Errorf("unsupported source type %q", sourceType)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func safeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func coalesceJSON(value json.RawMessage, fallback json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return fallback
	}
	return value
}

func computeNextSync(syncFrequency *string, now time.Time) *time.Time {
	if syncFrequency == nil || strings.TrimSpace(*syncFrequency) == "" {
		return nil
	}
	switch strings.TrimSpace(*syncFrequency) {
	case "@hourly":
		next := now.Add(time.Hour)
		return &next
	case "@daily":
		next := now.Add(24 * time.Hour)
		return &next
	case "@weekly":
		next := now.Add(7 * 24 * time.Hour)
		return &next
	default:
		return nil
	}
}

func changedFields(req dto.UpdateSourceRequest) []string {
	fields := make([]string, 0, 5)
	if req.Name != nil {
		fields = append(fields, "name")
	}
	if req.Description != nil {
		fields = append(fields, "description")
	}
	if len(req.ConnectionConfig) > 0 {
		fields = append(fields, "connection_config")
	}
	if req.SyncFrequency != nil {
		fields = append(fields, "sync_frequency")
	}
	if req.Tags != nil {
		fields = append(fields, "tags")
	}
	if len(req.Metadata) > 0 {
		fields = append(fields, "metadata")
	}
	return fields
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func IsConflict(err error) bool {
	return err != nil && (errors.Is(err, ErrConflict) || strings.Contains(strings.ToLower(err.Error()), "already exists"))
}
