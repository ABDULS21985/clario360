package rules

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/repository"
)

type ReferentialChecker struct {
	registry   *connector.ConnectorRegistry
	sourceRepo *repository.SourceRepository
	decryptor  ConfigDecryptor
}

type ReferentialConfig struct {
	ReferenceSourceID string `json:"reference_source_id"`
	ReferenceTable    string `json:"reference_table"`
	ReferenceColumn   string `json:"reference_column"`
}

func NewReferentialChecker(registry *connector.ConnectorRegistry, sourceRepo *repository.SourceRepository, decryptor ConfigDecryptor) Checker {
	return &ReferentialChecker{registry: registry, sourceRepo: sourceRepo, decryptor: decryptor}
}

func (c *ReferentialChecker) Type() string { return "referential" }

func (c *ReferentialChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	if dataset.Rule.ColumnName == nil {
		return nil, fmt.Errorf("referential rule requires column_name")
	}
	var cfg ReferentialConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode referential config: %w", err)
	}
	sourceID, err := uuid.Parse(cfg.ReferenceSourceID)
	if err != nil {
		return nil, err
	}
	refSource, err := c.sourceRepo.Get(ctx, dataset.Rule.TenantID, sourceID)
	if err != nil {
		return nil, err
	}
	decrypted, err := c.decryptor.Decrypt(refSource.EncryptedConfig, refSource.Source.EncryptionKeyID)
	if err != nil {
		return nil, err
	}
	conn, err := c.registry.Create(refSource.Source.Type, decrypted)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	referenceValues := make(map[string]struct{})
	offset := int64(0)
	for {
		batch, err := conn.FetchData(ctx, cfg.ReferenceTable, connector.FetchParams{Columns: []string{cfg.ReferenceColumn}, BatchSize: 1000, Offset: offset})
		if err != nil {
			return nil, err
		}
		for _, row := range batch.Rows {
			referenceValues[fmt.Sprint(row[cfg.ReferenceColumn])] = struct{}{}
		}
		if !batch.HasMore || batch.RowCount == 0 {
			break
		}
		offset += int64(batch.RowCount)
	}

	failed := make([]map[string]interface{}, 0)
	for _, row := range dataset.Rows {
		if _, ok := referenceValues[fmt.Sprint(row[*dataset.Rule.ColumnName])]; !ok {
			failed = append(failed, row)
		}
	}
	checked := int64(len(dataset.Rows))
	failedCount := int64(len(failed))
	return &CheckResult{
		Status:         statusFromCounts(failedCount),
		RecordsChecked: checked,
		RecordsPassed:  checked - failedCount,
		RecordsFailed:  failedCount,
		PassRate:       passRate(checked, failedCount),
		FailureSamples: limitedSamples(failed, 10),
		FailureSummary: fmt.Sprintf("%d referential integrity violations detected", failedCount),
	}, nil
}
