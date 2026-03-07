package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

const dataModelEventsTopic = "data.model.events"

type ModelService struct {
	modelRepo  *repository.ModelRepository
	sourceRepo *repository.SourceRepository
	producer   *events.Producer
	metrics    *datametrics.Metrics
	logger     zerolog.Logger
}

func NewModelService(modelRepo *repository.ModelRepository, sourceRepo *repository.SourceRepository, producer *events.Producer, metrics *datametrics.Metrics, logger zerolog.Logger) *ModelService {
	return &ModelService{
		modelRepo:  modelRepo,
		sourceRepo: sourceRepo,
		producer:   producer,
		metrics:    metrics,
		logger:     logger,
	}
}

func (s *ModelService) Create(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateModelRequest) (*model.DataModel, error) {
	fields, err := decodeModelFields(req.SchemaDefinition)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	rules, err := decodeValidationRules(req.QualityRules)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	nextVersion, previousVersionID, err := s.modelRepo.NextVersion(ctx, tenantID, req.Name)
	if err != nil {
		return nil, err
	}
	if previousVersionID != nil {
		return nil, fmt.Errorf("%w: a data model named %q already exists", ErrConflict, req.Name)
	}
	now := time.Now().UTC()
	classification := model.DataClassification(req.DataClassification)
	if !classification.IsValid() {
		classification = model.DataClassificationInternal
	}
	item := &model.DataModel{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Name:               req.Name,
		DisplayName:        req.DisplayName,
		Description:        req.Description,
		Status:             defaultModelStatus(req.Status),
		SchemaDefinition:   fields,
		SourceID:           req.SourceID,
		SourceTable:        req.SourceTable,
		QualityRules:       rules,
		DataClassification: classification,
		ContainsPII:        req.ContainsPII,
		PIIColumns:         req.PIIColumns,
		FieldCount:         len(fields),
		Version:            nextVersion,
		PreviousVersionID:  previousVersionID,
		Tags:               req.Tags,
		Metadata:           coalesceJSON(req.Metadata, json.RawMessage(`{}`)),
		CreatedBy:          userID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.modelRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataModelsTotal.WithLabelValues(tenantID.String(), string(item.Status)).Inc()
	}
	_ = s.publishModelEvent(ctx, "data.model.created", tenantID, map[string]any{
		"id":        item.ID,
		"name":      item.Name,
		"source_id": item.SourceID,
	})
	return item, nil
}

func (s *ModelService) List(ctx context.Context, tenantID uuid.UUID, params dto.ListModelsParams) ([]*model.DataModel, int, error) {
	return s.modelRepo.List(ctx, tenantID, params)
}

func (s *ModelService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DataModel, error) {
	return s.modelRepo.Get(ctx, tenantID, id)
}

func (s *ModelService) Update(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateModelRequest) (*model.DataModel, error) {
	current, err := s.modelRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	nextVersion, previousVersionID, err := s.modelRepo.NextVersion(ctx, tenantID, current.Name)
	if err != nil {
		return nil, err
	}
	fields := current.SchemaDefinition
	if len(req.SchemaDefinition) > 0 {
		fields, err = decodeModelFields(req.SchemaDefinition)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}
	rules := current.QualityRules
	if len(req.QualityRules) > 0 {
		rules, err = decodeValidationRules(req.QualityRules)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	updated := &model.DataModel{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Name:               current.Name,
		DisplayName:        coalesceStringPtr(req.DisplayName, current.DisplayName),
		Description:        coalesceStringPtr(req.Description, current.Description),
		Status:             coalesceModelStatus(req.Status, current.Status),
		SchemaDefinition:   fields,
		SourceID:           current.SourceID,
		SourceTable:        current.SourceTable,
		QualityRules:       rules,
		DataClassification: coalesceClassification(req.DataClassification, current.DataClassification),
		ContainsPII:        coalesceBool(req.ContainsPII, current.ContainsPII),
		PIIColumns:         coalesceStringSlice(req.PIIColumns, current.PIIColumns),
		FieldCount:         len(fields),
		Version:            nextVersion,
		PreviousVersionID:  previousVersionID,
		Tags:               coalesceStringSlice(req.Tags, current.Tags),
		Metadata:           coalesceJSON(req.Metadata, current.Metadata),
		CreatedBy:          userID,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := s.modelRepo.Create(ctx, updated); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataSourceOperationsTotal.WithLabelValues("model_update").Inc()
		s.metrics.DataModelsTotal.WithLabelValues(tenantID.String(), string(updated.Status)).Inc()
	}
	_ = s.publishModelEvent(ctx, "data.model.updated", tenantID, map[string]any{
		"id":      updated.ID,
		"name":    updated.Name,
		"version": updated.Version,
	})
	return updated, nil
}

func (s *ModelService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	current, err := s.modelRepo.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := s.modelRepo.SoftDelete(ctx, tenantID, id, time.Now().UTC()); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.DataSourceOperationsTotal.WithLabelValues("model_delete").Inc()
		s.metrics.DataModelsTotal.WithLabelValues(tenantID.String(), string(current.Status)).Dec()
	}
	_ = s.publishModelEvent(ctx, "data.model.deleted", tenantID, map[string]any{
		"id":   current.ID,
		"name": current.Name,
	})
	return nil
}

func (s *ModelService) DeriveFromSource(ctx context.Context, tenantID, userID uuid.UUID, req dto.DeriveModelRequest) (*model.DataModel, error) {
	record, err := s.sourceRepo.Get(ctx, tenantID, req.SourceID)
	if err != nil {
		return nil, err
	}
	if record.Source.SchemaMetadata == nil {
		return nil, pgx.ErrNoRows
	}

	var table *model.DiscoveredTable
	for index := range record.Source.SchemaMetadata.Tables {
		if strings.EqualFold(record.Source.SchemaMetadata.Tables[index].Name, req.TableName) {
			table = &record.Source.SchemaMetadata.Tables[index]
			break
		}
	}
	if table == nil {
		return nil, fmt.Errorf("table %q not found in source schema", req.TableName)
	}

	return s.createDerivedModel(ctx, tenantID, userID, record.Source, *table, req.Name, true)
}

func (s *ModelService) CreateDerivedModelFromTable(ctx context.Context, tenantID, userID uuid.UUID, source *model.DataSource, table model.DiscoveredTable, name string, assignQualityRules bool) (*model.DataModel, error) {
	if source == nil {
		return nil, fmt.Errorf("%w: source is required", ErrValidation)
	}
	return s.createDerivedModel(ctx, tenantID, userID, source, table, name, assignQualityRules)
}

func (s *ModelService) createDerivedModel(ctx context.Context, tenantID, userID uuid.UUID, source *model.DataSource, table model.DiscoveredTable, name string, assignQualityRules bool) (*model.DataModel, error) {
	if name == "" {
		name = sanitizeModelName(table.Name)
	}
	nextVersion, previousVersionID, err := s.modelRepo.NextVersion(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	if previousVersionID != nil {
		return nil, fmt.Errorf("%w: a data model named %q already exists", ErrConflict, name)
	}

	fields := deriveModelFields(table)
	rules := []model.ValidationRule{}
	if assignQualityRules {
		rules = deriveValidationRules(fields)
	}
	piiColumns := make([]string, 0)
	classification := model.DataClassificationPublic
	containsPII := false
	for _, field := range fields {
		classification = maxFieldClassification(classification, field.Classification)
		if field.PIIType != "" {
			containsPII = true
			piiColumns = append(piiColumns, field.Name)
		}
	}

	now := time.Now().UTC()
	item := &model.DataModel{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Name:               name,
		DisplayName:        humanizeName(name),
		Description:        fmt.Sprintf("Auto-derived from %s.%s", source.Name, table.Name),
		Status:             model.DataModelStatusDraft,
		SchemaDefinition:   fields,
		SourceID:           &source.ID,
		SourceTable:        &table.Name,
		QualityRules:       rules,
		DataClassification: classification,
		ContainsPII:        containsPII,
		PIIColumns:         piiColumns,
		FieldCount:         len(fields),
		Version:            nextVersion,
		PreviousVersionID:  previousVersionID,
		Tags:               []string{},
		Metadata:           json.RawMessage(`{}`),
		CreatedBy:          userID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.modelRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.DataModelDerivationsTotal.Inc()
		s.metrics.DataModelsTotal.WithLabelValues(tenantID.String(), string(item.Status)).Inc()
	}
	_ = s.publishModelEvent(ctx, "data.model.derived", tenantID, map[string]any{
		"id":           item.ID,
		"name":         item.Name,
		"source_id":    item.SourceID,
		"table_name":   table.Name,
		"pii_detected": item.ContainsPII,
	})
	return item, nil
}

func (s *ModelService) ValidateAgainstSource(ctx context.Context, tenantID, id uuid.UUID) (*model.ModelValidationResult, error) {
	item, err := s.modelRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if item.SourceID == nil || item.SourceTable == nil {
		return &model.ModelValidationResult{Success: false, Errors: []model.ModelValidationError{{
			Code:    "missing_source",
			Message: "model is not linked to a source table",
		}}}, nil
	}
	record, err := s.sourceRepo.Get(ctx, tenantID, *item.SourceID)
	if err != nil {
		return nil, err
	}
	if record.Source.SchemaMetadata == nil {
		return nil, pgx.ErrNoRows
	}
	var table *model.DiscoveredTable
	for index := range record.Source.SchemaMetadata.Tables {
		if strings.EqualFold(record.Source.SchemaMetadata.Tables[index].Name, *item.SourceTable) {
			table = &record.Source.SchemaMetadata.Tables[index]
			break
		}
	}
	if table == nil {
		return nil, fmt.Errorf("source table %q not found", *item.SourceTable)
	}

	columnByName := make(map[string]model.DiscoveredColumn, len(table.Columns))
	for _, column := range table.Columns {
		columnByName[strings.ToLower(column.Name)] = column
	}

	errorsList := make([]model.ModelValidationError, 0)
	for _, field := range item.SchemaDefinition {
		column, ok := columnByName[strings.ToLower(field.Name)]
		if !ok {
			errorsList = append(errorsList, model.ModelValidationError{
				Field:   field.Name,
				Code:    "missing_column",
				Message: "field does not exist in source schema",
			})
			continue
		}
		if field.DataType != column.MappedType {
			errorsList = append(errorsList, model.ModelValidationError{
				Field:   field.Name,
				Code:    "type_mismatch",
				Message: fmt.Sprintf("field type %s does not match source type %s", field.DataType, column.MappedType),
			})
		}
	}

	result := &model.ModelValidationResult{
		Success: len(errorsList) == 0,
		Errors:  errorsList,
	}
	_ = s.publishModelEvent(ctx, "data.model.validated", tenantID, map[string]any{
		"id":      item.ID,
		"success": result.Success,
		"errors":  len(result.Errors),
	})
	return result, nil
}

func (s *ModelService) ListVersions(ctx context.Context, tenantID, id uuid.UUID) ([]*model.DataModel, error) {
	return s.modelRepo.ListVersions(ctx, tenantID, id)
}

func (s *ModelService) GetLineage(ctx context.Context, tenantID, id uuid.UUID) (*model.ModelLineage, error) {
	item, err := s.modelRepo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	lineage := &model.ModelLineage{Model: *item}
	if item.SourceID != nil {
		source, err := s.sourceRepo.Get(ctx, tenantID, *item.SourceID)
		if err == nil {
			sanitized := *source.Source
			sanitized.ConnectionConfig = nil
			lineage.Source = &sanitized
			if sanitized.SchemaMetadata != nil && item.SourceTable != nil {
				for index := range sanitized.SchemaMetadata.Tables {
					if strings.EqualFold(sanitized.SchemaMetadata.Tables[index].Name, *item.SourceTable) {
						lineage.SourceTable = &sanitized.SchemaMetadata.Tables[index]
						break
					}
				}
			}
		}
	}
	return lineage, nil
}

func (s *ModelService) publishModelEvent(ctx context.Context, eventType string, tenantID uuid.UUID, payload any) error {
	if s.producer == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return s.producer.Publish(ctx, dataModelEventsTopic, event)
}

func decodeModelFields(raw json.RawMessage) ([]model.ModelField, error) {
	var fields []model.ModelField
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func decodeValidationRules(raw json.RawMessage) ([]model.ValidationRule, error) {
	if len(raw) == 0 {
		return []model.ValidationRule{}, nil
	}
	var rules []model.ValidationRule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func defaultModelStatus(raw string) model.DataModelStatus {
	if status := model.DataModelStatus(raw); status.IsValid() {
		return status
	}
	return model.DataModelStatusDraft
}

func coalesceModelStatus(raw *string, fallback model.DataModelStatus) model.DataModelStatus {
	if raw == nil {
		return fallback
	}
	value := model.DataModelStatus(*raw)
	if value.IsValid() {
		return value
	}
	return fallback
}

func coalesceClassification(raw *string, fallback model.DataClassification) model.DataClassification {
	if raw == nil {
		return fallback
	}
	value := model.DataClassification(*raw)
	if value.IsValid() {
		return value
	}
	return fallback
}

func coalesceStringPtr(raw *string, fallback string) string {
	if raw == nil {
		return fallback
	}
	return *raw
}

func coalesceBool(raw *bool, fallback bool) bool {
	if raw == nil {
		return fallback
	}
	return *raw
}

func coalesceStringSlice(raw []string, fallback []string) []string {
	if raw == nil {
		return fallback
	}
	return raw
}

func sanitizeModelName(value string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	sanitized := re.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_")
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		return "derived_model"
	}
	return sanitized
}

func humanizeName(value string) string {
	parts := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func truncateSamples(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return append([]string(nil), values...)
	}
	return append([]string(nil), values[:limit]...)
}

func deriveModelFields(table model.DiscoveredTable) []model.ModelField {
	fields := make([]model.ModelField, 0, len(table.Columns))
	for _, column := range table.Columns {
		field := model.ModelField{
			Name:           column.Name,
			DisplayName:    humanizeName(column.Name),
			DataType:       column.MappedType,
			NativeType:     column.NativeType,
			Nullable:       column.Nullable,
			IsPrimaryKey:   column.IsPrimaryKey,
			IsForeignKey:   column.IsForeignKey,
			ForeignKeyRef:  column.ForeignKeyRef,
			Description:    chooseDescription(column.Comment, fmt.Sprintf("Derived from source column %s", column.Name)),
			DefaultValue:   column.DefaultValue,
			PIIType:        column.InferredPIIType,
			Classification: column.InferredClass,
			SampleValues:   truncateSamples(column.SampleValues, 5),
		}
		fields = append(fields, field)
	}
	return fields
}

func deriveValidationRules(fields []model.ModelField) []model.ValidationRule {
	rules := make([]model.ValidationRule, 0)
	for _, field := range fields {
		if !field.Nullable {
			rules = append(rules, model.ValidationRule{Type: "not_null", Field: field.Name})
		}
		if field.IsPrimaryKey {
			rules = append(rules, model.ValidationRule{Type: "unique", Field: field.Name})
		}
		if field.DataType == "string" && len(field.SampleValues) > 0 {
			maxLength := 0
			enumValues := make(map[string]struct{})
			for _, sample := range field.SampleValues {
				if len(sample) > maxLength {
					maxLength = len(sample)
				}
				enumValues[sample] = struct{}{}
			}
			if maxLength > 0 {
				rules = append(rules, model.ValidationRule{
					Type:   "max_length",
					Field:  field.Name,
					Params: map[string]any{"max": maxLength},
				})
			}
			if len(enumValues) > 0 && len(enumValues) < 20 {
				values := make([]string, 0, len(enumValues))
				for value := range enumValues {
					values = append(values, value)
				}
				sort.Strings(values)
				rules = append(rules, model.ValidationRule{
					Type:   "enum",
					Field:  field.Name,
					Params: map[string]any{"values": values},
				})
			}
		}
		if field.PIIType == "email" {
			rules = append(rules, model.ValidationRule{
				Type:   "format",
				Field:  field.Name,
				Params: map[string]any{"pattern": "email"},
			})
		}
		if field.DataType == "datetime" {
			rules = append(rules, model.ValidationRule{Type: "not_future", Field: field.Name})
		}
	}
	return rules
}

func maxFieldClassification(current, candidate model.DataClassification) model.DataClassification {
	switch candidate {
	case model.DataClassificationRestricted:
		return candidate
	case model.DataClassificationConfidential:
		if current != model.DataClassificationRestricted {
			return candidate
		}
	case model.DataClassificationInternal:
		if current == model.DataClassificationPublic {
			return candidate
		}
	}
	return current
}

func chooseDescription(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
