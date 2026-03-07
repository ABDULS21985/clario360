package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/connector"
	dataconfig "github.com/clario360/platform/internal/data/config"
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
	config          *dataconfig.Config
	sourceRepo      *repository.SourceRepository
	syncRepo        *repository.SyncRepository
	tester          *ConnectionTester
	discovery       *SchemaDiscoveryService
	ingestion       *IngestionService
	encryptor       *ConfigEncryptor
	producer        *events.Producer
	metrics         *datametrics.Metrics
	logger          zerolog.Logger
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
			"id":          id,
			"rows_read":   history.RowsRead,
			"rows_written": history.RowsWritten,
			"duration_ms": history.DurationMs,
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
	default:
		return nil, fmt.Errorf("unsupported source type %q", sourceType)
	}
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
