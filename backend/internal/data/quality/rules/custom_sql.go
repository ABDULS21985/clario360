package rules

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/sqlutil"
)

type CustomSQLChecker struct {
	registry  *connector.ConnectorRegistry
	sourceRepo *repository.SourceRepository
	decryptor ConfigDecryptor
}

type CustomSQLConfig struct {
	SQL string `json:"sql"`
}

func NewCustomSQLChecker(registry *connector.ConnectorRegistry, sourceRepo *repository.SourceRepository, decryptor ConfigDecryptor) Checker {
	return &CustomSQLChecker{registry: registry, sourceRepo: sourceRepo, decryptor: decryptor}
}

func (c *CustomSQLChecker) Type() string { return "custom_sql" }

func (c *CustomSQLChecker) Check(ctx context.Context, dataset Dataset) (*CheckResult, error) {
	var cfg CustomSQLConfig
	if err := json.Unmarshal(dataset.Rule.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode custom_sql config: %w", err)
	}
	if err := sqlutil.ValidateReadOnlySQL(cfg.SQL); err != nil {
		return nil, err
	}
	decrypted, err := c.decryptor.Decrypt(dataset.Source.EncryptedConfig, dataset.Source.Source.EncryptionKeyID)
	if err != nil {
		return nil, err
	}
	conn, err := c.registry.Create(dataset.Source.Source.Type, decrypted)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	batch, err := conn.ReadQuery(ctx, cfg.SQL, nil)
	if err != nil {
		return nil, err
	}
	var violations float64
	if len(batch.Rows) > 0 {
		for _, value := range batch.Rows[0] {
			if parsed, ok := asFloat(value); ok {
				violations = parsed
				break
			}
		}
	}
	status := "passed"
	if violations > 0 {
		status = "failed"
	}
	return &CheckResult{
		Status:         status,
		RecordsChecked: int64(len(dataset.Rows)),
		RecordsPassed:  int64(len(dataset.Rows)) - int64(violations),
		RecordsFailed:  int64(violations),
		PassRate:       passRate(int64(len(dataset.Rows)), int64(violations)),
		FailureSummary: fmt.Sprintf("%.0f records matched the custom condition", violations),
		MetricValue:    violations,
	}, nil
}

