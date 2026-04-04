package connector

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/colinmarc/hdfs/v2"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

const hdfsConnectorType = "hdfs"

type HDFSConnector struct {
	config   model.HDFSConnectionConfig
	client   *hdfs.Client
	sourceID uuid.UUID
	tenantID uuid.UUID
	logger   zerolog.Logger
	limits   ConnectorLimits
}

func NewHDFSConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.HDFSConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(hdfsConnectorType, "connect", ErrorCodeConfigurationError, "decode hdfs config", err)
	}
	if len(cfg.BasePaths) == 0 {
		cfg.BasePaths = []string{"/user/hive/warehouse"}
	}
	if cfg.MaxFileSizeMB == 0 {
		cfg.MaxFileSizeMB = 100
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(hdfsConnectorType, "connect", ErrorCodeConfigurationError, "validate hdfs config", err)
	}
	connector := &HDFSConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", hdfsConnectorType).Logger(),
		limits: options.Limits,
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(hdfsConnectorType).Inc()
	return connector, nil
}

func (c *HDFSConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *HDFSConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "test", start, err) }()
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	base := c.config.BasePaths[0]
	info, err := client.Stat(base)
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "test", ErrorCodeConnectionFailed, "stat hdfs base path", err)
	}
	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     "HDFS",
		Message:     fmt.Sprintf("Connected to HDFS. Base path %s accessible.", base),
		Permissions: []string{info.Name()},
	}, nil
}

func (c *HDFSConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "discover", start, err) }()
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	grouped, err := c.walkBasePaths(ctx, client)
	if err != nil {
		return nil, err
	}
	tables := make([]model.DiscoveredTable, 0, len(grouped))
	totalColumns := 0
	containsPII := false
	highest := model.DataClassificationPublic
	for dir, files := range grouped {
		table, tableErr := c.inferTableFromFiles(ctx, client, dir, files, opts)
		if tableErr != nil {
			return nil, tableErr
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highest = discovery.MaxClassification(highest, table.InferredClass)
	}
	observeSchemaMetrics(hdfsConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *HDFSConnector) FetchData(ctx context.Context, table string, params FetchParams) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "fetch", start, err) }()
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	file, err := client.Open(table)
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "fetch", ErrorCodeQueryFailed, "open hdfs file", err)
	}
	defer file.Close()
	header := make([]byte, 4096)
	n, _ := io.ReadFull(file, header)
	format := detectHDFSFormat(table, header[:n])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, newConnectorError(hdfsConnectorType, "fetch", ErrorCodeDriverError, "rewind hdfs file", err)
	}
	switch format {
	case "csv", "tsv":
		return fetchDelimitedFile(file, params, format == "tsv")
	case "json", "jsonl":
		return fetchJSONFile(file, params)
	default:
		return nil, newConnectorError(hdfsConnectorType, "fetch", ErrorCodeUnsupportedOperation, "HDFS fetch is only supported for csv/tsv/json/jsonl files", ErrCapabilityUnsupported)
	}
}

func (c *HDFSConnector) ReadQuery(ctx context.Context, query string, args []any) (*DataBatch, error) {
	_ = ctx
	_ = query
	_ = args
	return nil, newConnectorError(hdfsConnectorType, "read_query", ErrorCodeUnsupportedOperation, "HDFS does not support SQL querying", ErrCapabilityUnsupported)
}

func (c *HDFSConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (*WriteResult, error) {
	_ = ctx
	_ = table
	_ = rows
	_ = params
	return nil, newConnectorError(hdfsConnectorType, "write", ErrorCodeUnsupportedOperation, "HDFS connector is read-only", ErrCapabilityUnsupported)
}

func (c *HDFSConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "estimate", start, err) }()
	schema, err := c.DiscoverSchema(ctx, DiscoveryOptions{MaxTables: c.limits.MaxTables})
	if err != nil {
		return nil, err
	}
	estimate := &SizeEstimate{TableCount: schema.TableCount}
	for _, table := range schema.Tables {
		estimate.TotalRows += table.EstimatedRows
		estimate.TotalBytes += table.SizeBytes
	}
	return estimate, nil
}

func (c *HDFSConnector) QueryAccessLogs(ctx context.Context, since time.Time) (_ []DataAccessEvent, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "access_logs", start, err) }()
	if strings.TrimSpace(c.config.AuditLogPath) == "" {
		return []DataAccessEvent{}, nil
	}
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	file, err := client.Open(c.config.AuditLogPath)
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "access_logs", ErrorCodeQueryFailed, "open hdfs audit log", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	events := make([]DataAccessEvent, 0)
	for scanner.Scan() {
		line := scanner.Text()
		event, ok := parseHDFSAuditLine(line, c.sourceID, c.tenantID)
		if !ok || event.Timestamp.Before(since) {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, newConnectorError(hdfsConnectorType, "access_logs", ErrorCodeDriverError, "scan hdfs audit log", err)
	}
	getConnectorMetrics().AccessEventsTotal.WithLabelValues(hdfsConnectorType).Add(float64(len(events)))
	return events, nil
}

func (c *HDFSConnector) ListDataLocations(ctx context.Context) (_ []DataLocation, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "locations", start, err) }()
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	grouped, err := c.walkBasePaths(ctx, client)
	if err != nil {
		return nil, err
	}
	locations := make([]DataLocation, 0, len(grouped))
	for dir, files := range grouped {
		var size int64
		lastModified := time.Time{}
		format := "unknown"
		for _, file := range files {
			size += file.Size()
			if file.ModTime().After(lastModified) {
				lastModified = file.ModTime().UTC()
			}
			if format == "unknown" {
				format = detectHDFSFormat(file.Name(), nil)
			}
		}
		locations = append(locations, DataLocation{
			SourceID:     c.sourceID,
			SourceType:   hdfsConnectorType,
			Table:        path.Base(dir),
			Database:     path.Dir(dir),
			Location:     dir,
			Format:       format,
			SizeBytes:    size,
			LastModified: lastModified,
			Partitioned:  strings.Count(strings.TrimPrefix(dir, "/"), "/") > 1,
		})
	}
	return locations, nil
}

func (c *HDFSConnector) ScanFile(ctx context.Context, filePath string, sampleBytes int64) (_ *FileScanResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "scan_file", start, err) }()
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	info, err := client.Stat(filePath)
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "scan_file", ErrorCodeQueryFailed, "stat hdfs file", err)
	}
	file, err := client.Open(filePath)
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "scan_file", ErrorCodeQueryFailed, "open hdfs file", err)
	}
	defer file.Close()
	if sampleBytes <= 0 {
		sampleBytes = 1 << 20
	}
	buffer := make([]byte, sampleBytes)
	n, _ := io.ReadFull(file, buffer)
	buffer = buffer[:n]
	format := detectHDFSFormat(filePath, buffer)
	columns := make([]model.DiscoveredColumn, 0)
	findings := make([]PIIFinding, 0)
	switch format {
	case "csv", "tsv":
		columns, findings = scanDelimitedBuffer(buffer, format == "tsv")
	case "json", "jsonl":
		columns, findings = scanJSONBuffer(buffer)
	default:
		// Binary warehouse formats are detected, but deep schema extraction is deferred.
	}
	classification := classifyPIIFindings(findings)
	getConnectorMetrics().DSPMFilesScanned.WithLabelValues(hdfsConnectorType, format).Inc()
	return &FileScanResult{
		Path:           filePath,
		Format:         format,
		SizeBytes:      info.Size(),
		Columns:        columns,
		PIIFindings:    findings,
		Classification: classification,
		SampledBytes:   int64(len(buffer)),
	}, nil
}

func (c *HDFSConnector) ListRecentFiles(ctx context.Context, since time.Time, basePath string) (_ []FileInfo, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hdfsConnectorType, "recent_files", start, err) }()
	client, err := c.openClient()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(basePath) == "" {
		basePath = c.config.BasePaths[0]
	}
	files := make([]FileInfo, 0)
	err = client.Walk(basePath, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || info.ModTime().Before(since) || isMarkerFile(info.Name()) {
			return nil
		}
		files = append(files, FileInfo{
			Path:      filePath,
			SizeBytes: info.Size(),
			ModTime:   info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "recent_files", ErrorCodeQueryFailed, "walk recent hdfs files", err)
	}
	return files, nil
}

func (c *HDFSConnector) Close() error {
	c.client = nil
	getConnectorMetrics().ActiveConnections.WithLabelValues(hdfsConnectorType).Dec()
	return nil
}

func (c *HDFSConnector) openClient() (*hdfs.Client, error) {
	if c.client != nil {
		return c.client, nil
	}
	options := hdfs.ClientOptions{
		Addresses: c.config.NameNodes,
		User:      c.config.User,
	}
	client, err := hdfs.NewClient(options)
	if err != nil {
		return nil, newConnectorError(hdfsConnectorType, "connect", ErrorCodeConnectionFailed, "connect to hdfs", err)
	}
	c.client = client
	return client, nil
}

func (c *HDFSConnector) walkBasePaths(ctx context.Context, client *hdfs.Client) (map[string][]os.FileInfo, error) {
	grouped := make(map[string][]os.FileInfo)
	directories := 0
	files := 0
	for _, base := range c.config.BasePaths {
		err := client.Walk(base, func(filePath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if info.IsDir() {
				directories++
				if directories > 1000 {
					return io.EOF
				}
				return nil
			}
			if isMarkerFile(info.Name()) {
				return nil
			}
			files++
			if files > 10000 {
				return io.EOF
			}
			grouped[path.Dir(filePath)] = append(grouped[path.Dir(filePath)], info)
			return nil
		})
		if err != nil && err != io.EOF {
			return nil, newConnectorError(hdfsConnectorType, "discover", ErrorCodeQueryFailed, "walk hdfs paths", err)
		}
	}
	return grouped, nil
}

func (c *HDFSConnector) inferTableFromFiles(ctx context.Context, client *hdfs.Client, dir string, files []os.FileInfo, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	var sizeBytes int64
	var lastModified time.Time
	format := "unknown"
	columns := make([]model.DiscoveredColumn, 0)
	for _, info := range files {
		sizeBytes += info.Size()
		if info.ModTime().After(lastModified) {
			lastModified = info.ModTime().UTC()
		}
		filePath := path.Join(dir, info.Name())
		file, err := client.Open(filePath)
		if err != nil {
			return nil, newConnectorError(hdfsConnectorType, "discover", ErrorCodeQueryFailed, "open hdfs file for schema inference", err)
		}
		head := make([]byte, 4096)
		n, _ := io.ReadFull(file, head)
		file.Close()
		detected := detectHDFSFormat(filePath, head[:n])
		if format == "unknown" {
			format = detected
		}
		if len(columns) == 0 && (detected == "csv" || detected == "tsv" || detected == "json" || detected == "jsonl") {
			if detected == "csv" || detected == "tsv" {
				columns = inferHDFSDelimitedColumns(head[:n], detected == "tsv", opts.MaxSamples)
			} else {
				columns = inferJSONColumnsFromBuffer(head[:n], opts.MaxSamples)
			}
		}
	}
	columns = discovery.DetectPII(columns)
	piiCount := 0
	nullableCount := 0
	for _, column := range columns {
		if column.InferredPII {
			piiCount++
		}
		if column.Nullable {
			nullableCount++
		}
	}
	return &model.DiscoveredTable{
		Name:           path.Base(dir),
		Type:           format,
		Comment:        dir,
		Columns:        columns,
		EstimatedRows:  0,
		SizeBytes:      sizeBytes,
		InferredClass:  discovery.TableClassification(columns),
		ContainsPII:    piiCount > 0,
		PIIColumnCount: piiCount,
		NullableCount:  nullableCount,
	}, nil
}

func detectHDFSFormat(filePath string, head []byte) string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".parquet"):
		return "parquet"
	case strings.HasSuffix(lower, ".orc"):
		return "orc"
	case strings.HasSuffix(lower, ".avro"):
		return "avro"
	case strings.HasSuffix(lower, ".tsv"):
		return "tsv"
	case strings.HasSuffix(lower, ".csv"):
		return "csv"
	case strings.HasSuffix(lower, ".jsonl"), strings.HasSuffix(lower, ".ndjson"):
		return "jsonl"
	case strings.HasSuffix(lower, ".json"):
		return "json"
	}
	if len(head) >= 4 && string(head[:4]) == "PAR1" {
		return "parquet"
	}
	if len(head) >= 3 && string(head[:3]) == "ORC" {
		return "orc"
	}
	if len(head) >= 4 && string(head[:4]) == "Obj\x01" {
		return "avro"
	}
	if bytesContains(head, []byte{','}) {
		return "csv"
	}
	if bytesContains(head, []byte{'\t'}) {
		return "tsv"
	}
	if bytesContains(head, []byte{'{'}) {
		return "json"
	}
	return "unknown"
}

func inferHDFSDelimitedColumns(head []byte, tsv bool, maxSamples int) []model.DiscoveredColumn {
	reader := csv.NewReader(strings.NewReader(string(head)))
	if tsv {
		reader.Comma = '\t'
	}
	records, _ := reader.ReadAll()
	if len(records) == 0 {
		return nil
	}
	header := records[0]
	columns := make([]model.DiscoveredColumn, 0, len(header))
	for idx, name := range header {
		samples := make([]string, 0, maxSamples)
		for _, row := range records[1:] {
			if idx < len(row) {
				samples = append(samples, row[idx])
			}
			if maxSamples > 0 && len(samples) >= maxSamples {
				break
			}
		}
		mapped := discovery.InferSampleType(samples)
		column := model.DiscoveredColumn{
			Name:         strings.TrimSpace(name),
			DataType:     mapped,
			NativeType:   mapped,
			MappedType:   mapped,
			Nullable:     true,
			SampleValues: samples,
			SampleStats:  discovery.AnalyzeSamples(samples),
		}
		columns = append(columns, column)
	}
	return columns
}

func inferJSONColumnsFromBuffer(head []byte, maxSamples int) []model.DiscoveredColumn {
	lines := strings.Split(string(head), "\n")
	samplesByKey := make(map[string][]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var object map[string]any
		if err := json.Unmarshal([]byte(line), &object); err != nil {
			continue
		}
		for key, value := range object {
			samplesByKey[key] = append(samplesByKey[key], fmt.Sprint(value))
			if maxSamples > 0 && len(samplesByKey[key]) > maxSamples {
				samplesByKey[key] = samplesByKey[key][:maxSamples]
			}
		}
	}
	columns := make([]model.DiscoveredColumn, 0, len(samplesByKey))
	for key, samples := range samplesByKey {
		mapped := discovery.InferSampleType(samples)
		columns = append(columns, model.DiscoveredColumn{
			Name:         key,
			DataType:     mapped,
			NativeType:   mapped,
			MappedType:   mapped,
			Nullable:     true,
			SampleValues: samples,
			SampleStats:  discovery.AnalyzeSamples(samples),
		})
	}
	return columns
}

func fetchDelimitedFile(reader io.Reader, params FetchParams, tsv bool) (*DataBatch, error) {
	csvReader := csv.NewReader(reader)
	if tsv {
		csvReader.Comma = '\t'
	}
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &DataBatch{}, nil
	}
	header := records[0]
	start := int(params.Offset) + 1
	if start > len(records) {
		return &DataBatch{Columns: header, Rows: []map[string]any{}}, nil
	}
	limit := params.BatchSize
	if limit <= 0 {
		limit = 1000
	}
	end := minInt(len(records), start+limit)
	rows := make([]map[string]any, 0, end-start)
	for _, record := range records[start:end] {
		row := make(map[string]any, len(header))
		for i, column := range header {
			if i < len(record) {
				row[column] = record[i]
			} else {
				row[column] = nil
			}
		}
		rows = append(rows, row)
	}
	return &DataBatch{
		Columns:  header,
		Rows:     rows,
		RowCount: len(rows),
		HasMore:  end < len(records),
	}, nil
}

func fetchJSONFile(reader io.Reader, params FetchParams) (*DataBatch, error) {
	scanner := bufio.NewScanner(reader)
	rows := make([]map[string]any, 0)
	line := int64(0)
	limit := params.BatchSize
	if limit <= 0 {
		limit = 1000
	}
	columns := make([]string, 0)
	seen := make(map[string]struct{})
	for scanner.Scan() {
		if line < params.Offset {
			line++
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		for key := range record {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			columns = append(columns, key)
		}
		rows = append(rows, record)
		if len(rows) >= limit {
			break
		}
		line++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &DataBatch{
		Columns:  columns,
		Rows:     rows,
		RowCount: len(rows),
		HasMore:  len(rows) == limit,
	}, nil
}

func scanDelimitedBuffer(buffer []byte, tsv bool) ([]model.DiscoveredColumn, []PIIFinding) {
	columns := discovery.DetectPII(inferHDFSDelimitedColumns(buffer, tsv, 10))
	return columns, piiFindingsFromColumns(columns)
}

func scanJSONBuffer(buffer []byte) ([]model.DiscoveredColumn, []PIIFinding) {
	columns := discovery.DetectPII(inferJSONColumnsFromBuffer(buffer, 10))
	return columns, piiFindingsFromColumns(columns)
}

func classifyPIIFindings(findings []PIIFinding) string {
	classification := "public"
	for _, finding := range findings {
		switch finding.PIIType {
		case "credit_card", "ssn", "national_id":
			return "restricted"
		case "email", "phone", "address", "name":
			classification = "confidential"
		}
	}
	if len(findings) == 0 {
		return "internal"
	}
	return classification
}

func parseHDFSAuditLine(line string, sourceID, tenantID uuid.UUID) (DataAccessEvent, bool) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return DataAccessEvent{}, false
	}
	timestamp, err := time.Parse("2006-01-02 15:04:05,000", parts[0]+" "+parts[1])
	if err != nil {
		return DataAccessEvent{}, false
	}
	event := DataAccessEvent{
		Timestamp:  timestamp.UTC(),
		Action:     "hdfs_access",
		Database:   "hdfs",
		SourceType: hdfsConnectorType,
		SourceID:   sourceID,
		TenantID:   tenantID,
	}
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "ugi="):
			event.User = strings.TrimPrefix(part, "ugi=")
		case strings.HasPrefix(part, "ip=/"):
			event.SourceIP = strings.TrimPrefix(part, "ip=/")
		case strings.HasPrefix(part, "cmd="):
			event.Action = strings.TrimPrefix(part, "cmd=")
		case strings.HasPrefix(part, "src="):
			event.Table = strings.TrimPrefix(part, "src=")
			event.QueryPreview = event.Table
			event.QueryHash = sha256Hex(event.Table)
		}
	}
	return event, true
}

func isMarkerFile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "_success" || strings.HasSuffix(lower, ".crc")
}

func bytesContains(buffer []byte, chars []byte) bool {
	for _, value := range buffer {
		for _, candidate := range chars {
			if value == candidate {
				return true
			}
		}
	}
	return false
}

func piiFindingsFromColumns(columns []model.DiscoveredColumn) []PIIFinding {
	findings := make([]PIIFinding, 0)
	for _, column := range columns {
		if column.InferredPII {
			findings = append(findings, PIIFinding{
				Column:      column.Name,
				PIIType:     column.InferredPIIType,
				Confidence:  0.9,
				SampleCount: len(column.SampleValues),
			})
			continue
		}
		if column.SampleStats.LooksLikeIP {
			findings = append(findings, PIIFinding{
				Column:      column.Name,
				PIIType:     "ip_address",
				Confidence:  0.75,
				SampleCount: len(column.SampleValues),
			})
		}
	}
	return findings
}
