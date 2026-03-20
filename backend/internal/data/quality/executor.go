package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/quality/rules"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

type configDecryptor interface {
	Decrypt(ciphertext []byte, keyID string) ([]byte, error)
}

type QualityExecutor struct {
	checkers    map[string]rules.Checker
	connRegistry *connector.ConnectorRegistry
	sourceRepo  *repository.SourceRepository
	modelRepo   *repository.ModelRepository
	ruleRepo    *repository.QualityRuleRepository
	resultRepo  *repository.QualityResultRepository
	decryptor   configDecryptor
	producer    *events.Producer
	logger      zerolog.Logger
}

func NewExecutor(
	connRegistry *connector.ConnectorRegistry,
	sourceRepo *repository.SourceRepository,
	modelRepo *repository.ModelRepository,
	ruleRepo *repository.QualityRuleRepository,
	resultRepo *repository.QualityResultRepository,
	decryptor configDecryptor,
	producer *events.Producer,
	logger zerolog.Logger,
) *QualityExecutor {
	executor := &QualityExecutor{
		checkers:    map[string]rules.Checker{},
		connRegistry: connRegistry,
		sourceRepo:  sourceRepo,
		modelRepo:   modelRepo,
		ruleRepo:    ruleRepo,
		resultRepo:  resultRepo,
		decryptor:   decryptor,
		producer:    producer,
		logger:      logger,
	}
	executor.register(
		rules.NewNotNullChecker(),
		rules.NewUniqueChecker(),
		rules.NewRangeChecker(),
		rules.NewRegexChecker(),
		rules.NewEnumChecker(),
		rules.NewFreshnessChecker(),
		rules.NewRowCountChecker(),
		rules.NewStatisticalChecker(),
		rules.NewCustomSQLChecker(connRegistry, sourceRepo, decryptor),
		rules.NewReferentialChecker(connRegistry, sourceRepo, decryptor),
	)
	return executor
}

func (e *QualityExecutor) register(checkers ...rules.Checker) {
	for _, checker := range checkers {
		e.checkers[checker.Type()] = checker
	}
}

func (e *QualityExecutor) RunCheck(ctx context.Context, tenantID, ruleID uuid.UUID, pipelineRunID *uuid.UUID) (*model.QualityResult, error) {
	rule, err := e.ruleRepo.Get(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}
	modelItem, err := e.modelRepo.Get(ctx, tenantID, rule.ModelID)
	if err != nil {
		return nil, err
	}
	if modelItem.SourceID == nil || modelItem.SourceTable == nil {
		return nil, fmt.Errorf("quality rule model is not linked to a source table")
	}
	sourceRecord, err := e.sourceRepo.Get(ctx, tenantID, *modelItem.SourceID)
	if err != nil {
		return nil, err
	}
	previousResult, _ := e.resultRepo.LatestByRule(ctx, tenantID, rule.ID)

	rows, err := e.fetchModelRows(ctx, sourceRecord, *modelItem.SourceTable)
	if err != nil {
		return nil, err
	}
	checker, ok := e.checkers[string(rule.RuleType)]
	if !ok {
		return nil, fmt.Errorf("unsupported quality rule type %q", rule.RuleType)
	}
	start := time.Now()
	checkResult, err := checker.Check(ctx, rules.Dataset{
		Rule:           rule,
		Model:          modelItem,
		Source:         sourceRecord,
		Rows:           rows,
		PreviousResult: previousResult,
	})
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		message := err.Error()
		result := &model.QualityResult{
			ID:             uuid.New(),
			TenantID:       tenantID,
			RuleID:         rule.ID,
			ModelID:        rule.ModelID,
			PipelineRunID:  pipelineRunID,
			Status:         model.QualityResultError,
			FailureSamples: json.RawMessage(`[]`),
			CheckedAt:      time.Now().UTC(),
			DurationMs:     &durationMs,
			ErrorMessage:   &message,
			CreatedAt:      time.Now().UTC(),
		}
		if createErr := e.resultRepo.Create(ctx, result); createErr != nil {
			return nil, createErr
		}
		_ = e.ruleRepo.UpdateExecutionState(ctx, tenantID, rule.ID, result.CheckedAt, result.Status, rule.ConsecutiveFailures+1)
		e.publish(ctx, "data.quality.events", "data.quality.check_failed", tenantID, map[string]any{
			"rule_id":  rule.ID,
			"model_id": rule.ModelID,
			"error":    message,
		})
		return result, nil
	}

	result := &model.QualityResult{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RuleID:         rule.ID,
		ModelID:        rule.ModelID,
		PipelineRunID:  pipelineRunID,
		Status:         model.QualityResultStatus(checkResult.Status),
		RecordsChecked: checkResult.RecordsChecked,
		RecordsPassed:  checkResult.RecordsPassed,
		RecordsFailed:  checkResult.RecordsFailed,
		PassRate:       &checkResult.PassRate,
		FailureSamples: json.RawMessage(marshalSamples(checkResult.FailureSamples)),
		FailureSummary: stringPtr(checkResult.FailureSummary),
		CheckedAt:      time.Now().UTC(),
		DurationMs:     &durationMs,
		CreatedAt:      time.Now().UTC(),
	}
	if err := e.resultRepo.Create(ctx, result); err != nil {
		return nil, err
	}
	consecutiveFailures := 0
	if result.Status == model.QualityResultFailed || result.Status == model.QualityResultError {
		consecutiveFailures = rule.ConsecutiveFailures + 1
	}
	if err := e.ruleRepo.UpdateExecutionState(ctx, tenantID, rule.ID, result.CheckedAt, result.Status, consecutiveFailures); err != nil {
		return nil, err
	}
	eventType := "data.quality.check_passed"
	switch result.Status {
	case model.QualityResultFailed:
		eventType = "data.quality.check_failed"
	case model.QualityResultWarning:
		eventType = "data.quality.check_warning"
	}
	e.publish(ctx, "data.quality.events", eventType, tenantID, map[string]any{
		"rule_id":        rule.ID,
		"model_id":       rule.ModelID,
		"records_failed": result.RecordsFailed,
		"pass_rate":      result.PassRate,
		"severity":       rule.Severity,
	})
	return result, nil
}

func (e *QualityExecutor) fetchModelRows(ctx context.Context, source *repository.SourceRecord, table string) ([]map[string]interface{}, error) {
	decrypted, err := e.decryptor.Decrypt(source.EncryptedConfig, source.Source.EncryptionKeyID)
	if err != nil {
		return nil, fmt.Errorf("decrypt source config: %w", err)
	}
	conn, err := e.connRegistry.Create(source.Source.Type, json.RawMessage(decrypted))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows := make([]map[string]interface{}, 0)
	offset := int64(0)
	for {
		batch, err := conn.FetchData(ctx, table, connector.FetchParams{BatchSize: 1000, Offset: offset})
		if err != nil {
			return nil, err
		}
		for _, row := range batch.Rows {
			value := make(map[string]interface{}, len(row))
			for key, item := range row {
				value[key] = item
			}
			rows = append(rows, value)
		}
		if !batch.HasMore || batch.RowCount == 0 {
			break
		}
		offset += int64(batch.RowCount)
	}
	return rows, nil
}

func (e *QualityExecutor) publish(ctx context.Context, topic, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if e.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return
	}
	_ = e.producer.Publish(ctx, topic, event)
}

func marshalSamples(samples []map[string]interface{}) []byte {
	payload, _ := json.Marshal(samples)
	return payload
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}
