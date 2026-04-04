package connector

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

type S3Connector struct {
	config model.S3ConnectionConfig
	client *minio.Client
	logger zerolog.Logger
	limits ConnectorLimits
}

func NewS3Connector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.S3ConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("decode s3 config: %w", err)
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	if cfg.MaxObjects <= 0 {
		cfg.MaxObjects = options.Limits.MaxTables
	}
	return &S3Connector{
		config: cfg,
		client: client,
		logger: options.Logger.With().Str("connector", "s3").Logger(),
		limits: options.Limits,
	}, nil
}

func (c *S3Connector) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	found := false
	for object := range c.client.ListObjects(ctx, c.config.Bucket, minio.ListObjectsOptions{Prefix: c.config.Prefix, Recursive: true, MaxKeys: 1}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list s3 objects: %w", object.Err)
		}
		found = true
		break
	}
	message := "S3 bucket is reachable."
	if found {
		message = "S3 bucket is reachable and contains at least one object."
	}
	return &ConnectionTestResult{Success: true, Message: message}, nil
}

func (c *S3Connector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error) {
	tables := make([]model.DiscoveredTable, 0)
	totalColumns := 0
	highest := model.DataClassificationPublic
	containsPII := false

	for object := range c.client.ListObjects(ctx, c.config.Bucket, minio.ListObjectsOptions{Prefix: c.config.Prefix, Recursive: true, MaxKeys: opts.MaxTables}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list s3 objects for discovery: %w", object.Err)
		}
		if object.Key == "" {
			continue
		}
		table, err := c.discoverObject(ctx, object.Key, object.Size)
		if err != nil {
			c.logger.Warn().Err(err).Str("object", object.Key).Msg("skipping unsupported S3 object during discovery")
			continue
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		highest = discovery.MaxClassification(highest, table.InferredClass)
		containsPII = containsPII || table.ContainsPII
		if len(tables) >= opts.MaxTables {
			break
		}
	}

	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *S3Connector) FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error) {
	ext := strings.ToLower(filepath.Ext(table))
	switch ext {
	case ".csv", ".tsv":
		return c.fetchDelimitedObject(ctx, table, params)
	case ".json", ".jsonl", ".ndjson":
		return c.fetchJSONLinesObject(ctx, table, params)
	default:
		return nil, fmt.Errorf("unsupported object format for fetch: %s", ext)
	}
}

func (c *S3Connector) EstimateSize(ctx context.Context) (*SizeEstimate, error) {
	var count int
	var totalBytes int64
	for object := range c.client.ListObjects(ctx, c.config.Bucket, minio.ListObjectsOptions{Prefix: c.config.Prefix, Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list s3 objects for estimate: %w", object.Err)
		}
		count++
		totalBytes += object.Size
	}
	return &SizeEstimate{
		TableCount: count,
		TotalRows:  0,
		TotalBytes: totalBytes,
	}, nil
}

func (c *S3Connector) ReadQuery(ctx context.Context, query string, args []any) (*DataBatch, error) {
	return nil, fmt.Errorf("%w: S3 connector does not support SQL query execution", ErrCapabilityUnsupported)
}

func (c *S3Connector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (*WriteResult, error) {
	return nil, fmt.Errorf("%w: S3 connector is read-only", ErrCapabilityUnsupported)
}

func (c *S3Connector) Close() error { return nil }

func (c *S3Connector) discoverObject(ctx context.Context, objectKey string, objectSize int64) (*model.DiscoveredTable, error) {
	ext := strings.ToLower(filepath.Ext(objectKey))
	switch ext {
	case ".csv", ".tsv":
		return c.discoverDelimitedObject(ctx, objectKey, ext, objectSize)
	case ".json", ".jsonl", ".ndjson":
		return c.discoverJSONLinesObject(ctx, objectKey, objectSize)
	default:
		return nil, fmt.Errorf("unsupported object format: %s", ext)
	}
}

func (c *S3Connector) discoverDelimitedObject(ctx context.Context, objectKey, ext string, objectSize int64) (*model.DiscoveredTable, error) {
	opts := minio.GetObjectOptions{}
	opts.SetRange(0, 1<<20-1)
	object, err := c.client.GetObject(ctx, c.config.Bucket, objectKey, opts)
	if err != nil {
		return nil, fmt.Errorf("open s3 object %s: %w", objectKey, err)
	}
	defer object.Close()

	reader := csv.NewReader(object)
	if ext == ".tsv" {
		reader.Comma = '\t'
	}
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read s3 delimited header %s: %w", objectKey, err)
	}
	records := make([][]string, 0, c.limits.MaxSampleRows)
	for len(records) < c.limits.MaxSampleRows {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read s3 delimited sample %s: %w", objectKey, err)
		}
		records = append(records, record)
	}
	columns := inferDelimitedColumns(header, records)
	columns = discovery.DetectPII(columns)

	return &model.DiscoveredTable{
		Name:            objectKey,
		Type:            "s3_object",
		Columns:         columns,
		EstimatedRows:   0,
		SizeBytes:       objectSize,
		InferredClass:   discovery.TableClassification(columns),
		ContainsPII:     hasPII(columns),
		PIIColumnCount:  countPII(columns),
		SampledRowCount: len(records),
	}, nil
}

func (c *S3Connector) discoverJSONLinesObject(ctx context.Context, objectKey string, objectSize int64) (*model.DiscoveredTable, error) {
	opts := minio.GetObjectOptions{}
	opts.SetRange(0, 1<<20-1)
	object, err := c.client.GetObject(ctx, c.config.Bucket, objectKey, opts)
	if err != nil {
		return nil, fmt.Errorf("open json object %s: %w", objectKey, err)
	}
	defer object.Close()

	payload, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("read json object %s: %w", objectKey, err)
	}
	rows, err := parseJSONRows(payload)
	if err != nil {
		return nil, err
	}
	if len(rows) > c.limits.MaxSampleRows {
		rows = rows[:c.limits.MaxSampleRows]
	}
	columns := inferJSONColumns(rows)
	columns = discovery.DetectPII(columns)
	return &model.DiscoveredTable{
		Name:            objectKey,
		Type:            "s3_object",
		Columns:         columns,
		EstimatedRows:   0,
		SizeBytes:       objectSize,
		InferredClass:   discovery.TableClassification(columns),
		ContainsPII:     hasPII(columns),
		PIIColumnCount:  countPII(columns),
		SampledRowCount: len(rows),
	}, nil
}

func (c *S3Connector) fetchDelimitedObject(ctx context.Context, objectKey string, params FetchParams) (*DataBatch, error) {
	object, err := c.client.GetObject(ctx, c.config.Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open s3 delimited object: %w", err)
	}
	defer object.Close()

	reader := csv.NewReader(object)
	if strings.HasSuffix(strings.ToLower(objectKey), ".tsv") {
		reader.Comma = '\t'
	}
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read s3 delimited header: %w", err)
	}

	batchSize := defaultBatchSize(params.BatchSize)
	rows := make([]map[string]any, 0, batchSize)
	var index int64
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read s3 delimited record: %w", err)
		}
		if index < params.Offset {
			index++
			continue
		}
		if len(rows) >= batchSize {
			return &DataBatch{Columns: header, Rows: rows, RowCount: len(rows), HasMore: true}, nil
		}
		rows = append(rows, recordToMap(header, record))
		index++
	}
	return &DataBatch{Columns: header, Rows: rows, RowCount: len(rows)}, nil
}

func (c *S3Connector) fetchJSONLinesObject(ctx context.Context, objectKey string, params FetchParams) (*DataBatch, error) {
	object, err := c.client.GetObject(ctx, c.config.Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open s3 json object: %w", err)
	}
	defer object.Close()

	payload, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("read s3 json object: %w", err)
	}
	rows, err := parseJSONRows(payload)
	if err != nil {
		return nil, err
	}
	batchSize := defaultBatchSize(params.BatchSize)
	start := int(params.Offset)
	if start > len(rows) {
		start = len(rows)
	}
	end := start + batchSize
	hasMore := false
	if end < len(rows) {
		hasMore = true
	} else {
		end = len(rows)
	}
	selected := rows[start:end]
	columns := make([]string, 0)
	if len(selected) > 0 {
		for key := range selected[0] {
			columns = append(columns, key)
		}
	}
	return &DataBatch{
		Columns:  columns,
		Rows:     selected,
		RowCount: len(selected),
		HasMore:  hasMore,
	}, nil
}

func parseJSONRows(payload []byte) ([]map[string]any, error) {
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err == nil {
		switch typed := parsed.(type) {
		case []any:
			rows := make([]map[string]any, 0, len(typed))
			for _, entry := range typed {
				if row, ok := entry.(map[string]any); ok {
					rows = append(rows, row)
				}
			}
			return rows, nil
		case map[string]any:
			return []map[string]any{typed}, nil
		}
	}

	lines := bytes.Split(payload, []byte("\n"))
	rows := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, fmt.Errorf("parse json lines object: %w", err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}
