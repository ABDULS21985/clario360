package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/sqlutil"
)

var safeIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_\.]*$`)

type configDecryptor interface {
	Decrypt(ciphertext []byte, keyID string) ([]byte, error)
}

type Extractor struct {
	registry  *connector.ConnectorRegistry
	decryptor configDecryptor
}

func NewExtractor(registry *connector.ConnectorRegistry, decryptor configDecryptor) *Extractor {
	return &Extractor{registry: registry, decryptor: decryptor}
}

type ExtractResult struct {
	Rows            []map[string]interface{}
	RecordsExtracted int64
	BytesRead       int64
	IncrementalFrom *string
	IncrementalTo   *string
}

func (e *Extractor) Extract(ctx context.Context, source *repository.SourceRecord, cfg model.PipelineConfig) (*ExtractResult, error) {
	decrypted, err := e.decryptor.Decrypt(source.EncryptedConfig, source.Source.EncryptionKeyID)
	if err != nil {
		return nil, fmt.Errorf("decrypt source config: %w", err)
	}
	conn, err := e.registry.Create(source.Source.Type, json.RawMessage(decrypted))
	if err != nil {
		return nil, fmt.Errorf("create source connector: %w", err)
	}
	defer conn.Close()

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}

	if strings.TrimSpace(cfg.SourceQuery) != "" {
		if err := sqlutil.ValidateReadOnlySQL(cfg.SourceQuery); err != nil {
			return nil, err
		}
		batch, err := conn.ReadQuery(ctx, cfg.SourceQuery, nil)
		if err != nil {
			return nil, err
		}
		rows, incrementalFrom, incrementalTo := incrementalRange(batch.Rows, cfg.IncrementalField)
		return &ExtractResult{
			Rows:             rows,
			RecordsExtracted: int64(len(rows)),
			BytesRead:        int64(len(mustJSON(rows))),
			IncrementalFrom:  incrementalFrom,
			IncrementalTo:    incrementalTo,
		}, nil
	}

	if cfg.SourceTable == "" {
		return nil, fmt.Errorf("pipeline config.source_table is required when source_query is not set")
	}

	if cfg.IncrementalField != "" && cfg.IncrementalValue != nil {
		query, err := buildIncrementalQuery(source.Source.Type, cfg.SourceTable, cfg.IncrementalField)
		if err != nil {
			return nil, err
		}
		batch, err := conn.ReadQuery(ctx, query, []any{*cfg.IncrementalValue})
		if err != nil {
			return nil, err
		}
		rows, incrementalFrom, incrementalTo := incrementalRange(batch.Rows, cfg.IncrementalField)
		return &ExtractResult{
			Rows:             rows,
			RecordsExtracted: int64(len(rows)),
			BytesRead:        int64(len(mustJSON(rows))),
			IncrementalFrom:  incrementalFrom,
			IncrementalTo:    incrementalTo,
		}, nil
	}

	rows := make([]map[string]interface{}, 0)
	offset := int64(0)
	for {
		batch, err := conn.FetchData(ctx, cfg.SourceTable, connector.FetchParams{
			BatchSize: batchSize,
			Offset:    offset,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch source batch: %w", err)
		}
		for _, row := range batch.Rows {
			rows = append(rows, toInterfaceMap(row))
		}
		if !batch.HasMore || batch.RowCount == 0 {
			break
		}
		offset += int64(batch.RowCount)
	}

	incrementalFrom, incrementalTo := incrementalRangeInterface(rows, cfg.IncrementalField)
	return &ExtractResult{
		Rows:             rows,
		RecordsExtracted: int64(len(rows)),
		BytesRead:        int64(len(mustJSON(rows))),
		IncrementalFrom:  incrementalFrom,
		IncrementalTo:    incrementalTo,
	}, nil
}

func buildIncrementalQuery(sourceType model.DataSourceType, tableName, field string) (string, error) {
	if !safeIdentifierPattern.MatchString(tableName) || !safeIdentifierPattern.MatchString(field) {
		return "", fmt.Errorf("invalid incremental source table or field")
	}
	switch sourceType {
	case model.DataSourceTypePostgreSQL:
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			return fmt.Sprintf(`SELECT * FROM "%s"."%s" WHERE "%s" > $1 ORDER BY "%s"`, parts[0], parts[1], field, field), nil
		}
		return fmt.Sprintf(`SELECT * FROM "%s" WHERE "%s" > $1 ORDER BY "%s"`, tableName, field, field), nil
	case model.DataSourceTypeMySQL:
		return fmt.Sprintf("SELECT * FROM %s WHERE %s > ? ORDER BY %s", mysqlQuoted(tableName), mysqlQuoted(field), mysqlQuoted(field)), nil
	default:
		return "", fmt.Errorf("incremental extraction is not supported for source type %s", sourceType)
	}
}

func incrementalRange(rows []map[string]any, field string) ([]map[string]interface{}, *string, *string) {
	values := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		values = append(values, toInterfaceMap(row))
	}
	from, to := incrementalRangeInterface(values, field)
	return values, from, to
}

func incrementalRangeInterface(rows []map[string]interface{}, field string) (*string, *string) {
	if field == "" || len(rows) == 0 {
		return nil, nil
	}
	var minValue *string
	var maxValue *string
	for _, row := range rows {
		raw, ok := row[field]
		if !ok || raw == nil {
			continue
		}
		text := fmt.Sprint(raw)
		if minValue == nil || text < *minValue {
			copyValue := text
			minValue = &copyValue
		}
		if maxValue == nil || text > *maxValue {
			copyValue := text
			maxValue = &copyValue
		}
	}
	return minValue, maxValue
}

func toInterfaceMap(row map[string]any) map[string]interface{} {
	value := make(map[string]interface{}, len(row))
	for key, item := range row {
		value[key] = item
	}
	return value
}

func mustJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}

func mysqlQuoted(identifier string) string {
	parts := strings.Split(identifier, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, "`"+strings.ReplaceAll(part, "`", "``")+"`")
	}
	return strings.Join(quoted, ".")
}
