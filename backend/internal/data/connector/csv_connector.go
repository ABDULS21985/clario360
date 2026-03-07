package connector

import (
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

type CSVConnector struct {
	config model.CSVConnectionConfig
	client *minio.Client
	logger zerolog.Logger
	limits ConnectorLimits
}

func NewCSVConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.CSVConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("decode csv config: %w", err)
	}
	if cfg.Delimiter == "" {
		cfg.Delimiter = ","
	}
	if cfg.Encoding == "" {
		cfg.Encoding = "utf-8"
	}
	if cfg.QuoteChar == "" {
		cfg.QuoteChar = `"`
	}
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &CSVConnector{
		config: cfg,
		client: client,
		logger: options.Logger.With().Str("connector", "csv").Logger(),
		limits: options.Limits,
	}, nil
}

func (c *CSVConnector) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	info, err := c.client.StatObject(ctx, c.config.Bucket, c.config.FilePath, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("stat csv object %s/%s: %w", c.config.Bucket, c.config.FilePath, err)
	}
	return &ConnectionTestResult{
		Success:   true,
		LatencyMs: 0,
		Message:   fmt.Sprintf("CSV file found (%d bytes).", info.Size),
	}, nil
}

func (c *CSVConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error) {
	records, headers, err := c.readPreview(ctx)
	if err != nil {
		return nil, err
	}
	columns := inferDelimitedColumns(headers, records)
	columns = discovery.DetectPII(columns)

	table := model.DiscoveredTable{
		Name:            filepath.Base(c.config.FilePath),
		Type:            "csv",
		Columns:         columns,
		InferredClass:   discovery.TableClassification(columns),
		ContainsPII:     hasPII(columns),
		PIIColumnCount:  countPII(columns),
		SampledRowCount: len(records),
	}
	return &model.DiscoveredSchema{
		Tables:       []model.DiscoveredTable{table},
		TableCount:   1,
		ColumnCount:  len(columns),
		ContainsPII:  table.ContainsPII,
		HighestClass: table.InferredClass,
	}, nil
}

func (c *CSVConnector) FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error) {
	object, err := c.client.GetObject(ctx, c.config.Bucket, c.config.FilePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open csv object: %w", err)
	}
	defer object.Close()

	reader, err := c.newReader(object)
	if err != nil {
		return nil, err
	}

	headerRow, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	headers := headerRow
	if !c.config.HasHeader {
		headers = generatedHeaders(len(headerRow))
		reader.FieldsPerRecord = len(headers)
	}

	if !c.config.HasHeader {
		if params.Offset > 0 {
			params.Offset--
		}
	}

	batchSize := defaultBatchSize(params.BatchSize)
	rows := make([]map[string]any, 0, batchSize)
	var index int64
	if !c.config.HasHeader {
		row := recordToMap(headers, headerRow)
		if params.Offset == 0 {
			rows = append(rows, row)
		}
		index = 1
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv record: %w", err)
		}
		if index < params.Offset {
			index++
			continue
		}
		if len(rows) >= batchSize {
			return &DataBatch{
				Columns:  headers,
				Rows:     rows,
				RowCount: len(rows),
				HasMore:  true,
			}, nil
		}
		rows = append(rows, recordToMap(headers, record))
		index++
	}

	return &DataBatch{
		Columns:  headers,
		Rows:     rows,
		RowCount: len(rows),
		HasMore:  false,
	}, nil
}

func (c *CSVConnector) EstimateSize(ctx context.Context) (*SizeEstimate, error) {
	object, err := c.client.GetObject(ctx, c.config.Bucket, c.config.FilePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open csv object for estimate: %w", err)
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat csv object for estimate: %w", err)
	}

	reader, err := c.newReader(object)
	if err != nil {
		return nil, err
	}
	var rowCount int64
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("count csv rows: %w", err)
		}
		rowCount++
	}
	if c.config.HasHeader && rowCount > 0 {
		rowCount--
	}

	return &SizeEstimate{
		TableCount: 1,
		TotalRows:  rowCount,
		TotalBytes: info.Size,
	}, nil
}

func (c *CSVConnector) Close() error { return nil }

func (c *CSVConnector) readPreview(ctx context.Context) ([][]string, []string, error) {
	opts := minio.GetObjectOptions{}
	opts.SetRange(0, 1<<20-1)
	object, err := c.client.GetObject(ctx, c.config.Bucket, c.config.FilePath, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("open csv preview object: %w", err)
	}
	defer object.Close()

	reader, err := c.newReader(object)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]string, 0, c.limits.MaxSampleRows)
	headers := []string{}

	firstRow, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("read csv preview header: %w", err)
	}
	if c.config.HasHeader {
		headers = firstRow
	} else {
		headers = generatedHeaders(len(firstRow))
		rows = append(rows, firstRow)
	}

	for len(rows) < c.limits.MaxSampleRows {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read csv preview row: %w", err)
		}
		rows = append(rows, record)
	}

	return rows, headers, nil
}

func (c *CSVConnector) newReader(reader io.Reader) (*csv.Reader, error) {
	if c.config.QuoteChar != `"` {
		return nil, fmt.Errorf("unsupported CSV quote_char %q: only double quote is supported", c.config.QuoteChar)
	}
	csvReader := csv.NewReader(reader)
	if c.config.Delimiter == "\t" || strings.EqualFold(c.config.Delimiter, "tab") {
		csvReader.Comma = '\t'
	} else if c.config.Delimiter != "" {
		runes := []rune(c.config.Delimiter)
		if len(runes) != 1 {
			return nil, fmt.Errorf("delimiter must be a single character")
		}
		csvReader.Comma = runes[0]
	}
	csvReader.FieldsPerRecord = -1
	csvReader.ReuseRecord = false
	return csvReader, nil
}

func inferDelimitedColumns(headers []string, records [][]string) []model.DiscoveredColumn {
	columns := make([]model.DiscoveredColumn, 0, len(headers))
	for index, header := range headers {
		samples := make([]string, 0, len(records))
		for _, record := range records {
			if index < len(record) {
				samples = append(samples, record[index])
			}
		}
		inferredType := discovery.InferSampleType(samples)
		column := model.DiscoveredColumn{
			Name:         header,
			DataType:     inferredType,
			NativeType:   "csv",
			MappedType:   inferredType,
			Nullable:     hasEmpty(samples),
			SampleValues: truncateSamples(samples, 5),
			SampleStats:  discovery.AnalyzeSamples(samples),
		}
		columns = append(columns, column)
	}
	return columns
}

func generatedHeaders(count int) []string {
	headers := make([]string, 0, count)
	for i := 0; i < count; i++ {
		headers = append(headers, fmt.Sprintf("col%d", i+1))
	}
	return headers
}

func recordToMap(headers, record []string) map[string]any {
	row := make(map[string]any, len(headers))
	for i, header := range headers {
		if i < len(record) {
			row[header] = record[i]
		} else {
			row[header] = ""
		}
	}
	return row
}

func truncateSamples(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func hasEmpty(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}
