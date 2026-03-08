package connector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beltran/gohive"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

const sparkConnectorType = "spark"

type SparkConnector struct {
	config     model.SparkConnectionConfig
	thriftConn *gohive.Connection
	httpClient *http.Client
	sourceID   uuid.UUID
	tenantID   uuid.UUID
	logger     zerolog.Logger
	limits     ConnectorLimits
}

type SparkApplication struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	User      string        `json:"user"`
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration"`
	State     string        `json:"state"`
}

type SparkApplicationDetail struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	User         string            `json:"user"`
	State        string            `json:"state"`
	StartTime    time.Time         `json:"start_time"`
	Duration     time.Duration     `json:"duration"`
	Jobs         []SparkJob        `json:"jobs"`
	Executors    []SparkExecutor   `json:"executors"`
	StageMetrics SparkStageMetrics `json:"stage_metrics"`
}

type SparkJob struct {
	JobID        int    `json:"job_id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	NumTasks     int    `json:"num_tasks"`
	NumCompleted int    `json:"num_completed"`
}

type SparkExecutor struct {
	ID            string `json:"id"`
	HostPort      string `json:"host_port"`
	TotalTasks    int    `json:"total_tasks"`
	TotalDuration int64  `json:"total_duration"`
}

type SparkStageMetrics struct {
	InputBytes   int64 `json:"input_bytes"`
	InputRecords int64 `json:"input_records"`
}

func NewSparkConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.SparkConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(sparkConnectorType, "connect", ErrorCodeConfigurationError, "decode spark config", err)
	}
	if cfg.QueryTimeoutSeconds == 0 {
		cfg.QueryTimeoutSeconds = 120
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = minInt(options.Limits.MaxPoolSize, 5)
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = max(1, cfg.MaxOpenConns/2)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(sparkConnectorType, "connect", ErrorCodeConfigurationError, "validate spark config", err)
	}
	connector := &SparkConnector{
		config:     cfg,
		httpClient: &http.Client{Timeout: time.Duration(cfg.QueryTimeoutSeconds) * time.Second},
		logger:     options.Logger.With().Str("connector", sparkConnectorType).Logger(),
		limits:     options.Limits,
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(sparkConnectorType).Inc()
	return connector, nil
}

func (c *SparkConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *SparkConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "test", start, err) }()

	warnings := make([]string, 0)
	permissions := make([]string, 0)
	success := false
	if c.config.Thrift != nil {
		if conn, thriftErr := c.openThrift(ctx); thriftErr == nil {
			success = true
			_, rows, queryErr := gohiveQueryRows(ctx, conn, "SHOW DATABASES", 20)
			if queryErr == nil {
				for _, row := range rows {
					permissions = append(permissions, firstRowValue(row))
				}
			}
		} else {
			warnings = append(warnings, "Spark Thrift unavailable: "+thriftErr.Error())
		}
	}
	if _, restErr := c.restGET(ctx, c.config.REST.MasterURL+"/api/v1/applications", nil); restErr == nil {
		success = true
	} else {
		warnings = append(warnings, "Spark REST unavailable: "+restErr.Error())
	}
	if !success {
		return nil, newConnectorError(sparkConnectorType, "test", ErrorCodeConnectionFailed, "unable to connect to Spark Thrift or REST API", nil)
	}
	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     "Spark",
		Message:     "Spark connection established.",
		Permissions: permissions,
		Warnings:    warnings,
	}, nil
}

func (c *SparkConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "discover", start, err) }()
	conn, err := c.requireThrift(ctx, "discover")
	if err != nil {
		return nil, err
	}
	databaseName := "default"
	if c.config.Thrift != nil && c.config.Thrift.Database != "" {
		databaseName = c.config.Thrift.Database
	}
	_, rows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("SHOW TABLES IN %s", backtickQuote(databaseName)), opts.MaxTables)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "show spark tables", err)
	}
	tables := make([]model.DiscoveredTable, 0)
	totalColumns := 0
	containsPII := false
	highest := model.DataClassificationPublic
	for _, row := range rows {
		tableName := firstRowValue(row)
		if tableName == "" {
			continue
		}
		table, tableErr := c.discoverTable(ctx, conn, databaseName, tableName, opts)
		if tableErr != nil {
			return nil, tableErr
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highest = discovery.MaxClassification(highest, table.InferredClass)
	}
	observeSchemaMetrics(sparkConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *SparkConnector) FetchData(ctx context.Context, table string, params FetchParams) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "fetch", start, err) }()
	conn, err := c.requireThrift(ctx, "fetch")
	if err != nil {
		return nil, err
	}
	columns := "*"
	if len(params.Columns) > 0 {
		quoted := make([]string, 0, len(params.Columns))
		for _, column := range params.Columns {
			quoted = append(quoted, backtickQuote(column))
		}
		columns = strings.Join(quoted, ", ")
	}
	query := fmt.Sprintf("SELECT %s FROM %s", columns, quoteDotBacktickIdentifier(table))
	if len(params.Filters) > 0 {
		conditions := make([]string, 0, len(params.Filters))
		for column, value := range params.Filters {
			conditions = append(conditions, fmt.Sprintf("%s = %s", backtickQuote(column), sqlLiteral(value)))
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if params.OrderBy != "" {
		query += " ORDER BY " + backtickQuote(params.OrderBy)
	}
	if params.BatchSize <= 0 {
		params.BatchSize = 1000
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", params.BatchSize, params.Offset)
	description, rows, err := gohiveQueryRows(ctx, conn, query, params.BatchSize)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "fetch", ErrorCodeQueryFailed, "fetch spark rows", err)
	}
	observeFetchMetrics(sparkConnectorType, len(rows), int64(len(mustJSON(rows))))
	return &DataBatch{
		Columns:  descriptionNames(description),
		Rows:     rows,
		RowCount: len(rows),
		HasMore:  len(rows) == params.BatchSize,
	}, nil
}

func (c *SparkConnector) ReadQuery(ctx context.Context, query string, args []any) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "read_query", start, err) }()
	_ = args
	if !isReadOnlyQuery(query) {
		return nil, newConnectorError(sparkConnectorType, "read_query", ErrorCodeUnsupportedOperation, "only read-only queries are allowed", ErrCapabilityUnsupported)
	}
	conn, err := c.requireThrift(ctx, "read_query")
	if err != nil {
		return nil, err
	}
	description, rows, err := gohiveQueryRows(ctx, conn, query, 0)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "read_query", ErrorCodeQueryFailed, "execute spark query", err)
	}
	return &DataBatch{Columns: descriptionNames(description), Rows: rows, RowCount: len(rows)}, nil
}

func (c *SparkConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (_ *WriteResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "write", start, err) }()
	if params.Strategy == "merge" {
		return nil, newConnectorError(sparkConnectorType, "write", ErrorCodeUnsupportedOperation, "Spark Thrift connector does not support merge writes", ErrCapabilityUnsupported)
	}
	if len(rows) == 0 {
		return &WriteResult{}, nil
	}
	conn, err := c.requireThrift(ctx, "write")
	if err != nil {
		return nil, err
	}
	columns := writeColumns(rows)
	literals := make([]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			values = append(values, sqlLiteral(row[column]))
		}
		literals = append(literals, "("+strings.Join(values, ", ")+")")
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, backtickQuote(column))
	}
	if params.Replace {
		if err = gohiveExec(ctx, conn, "TRUNCATE TABLE "+quoteDotBacktickIdentifier(table)); err != nil {
			return nil, newConnectorError(sparkConnectorType, "write", ErrorCodeQueryFailed, "truncate spark table", err)
		}
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", quoteDotBacktickIdentifier(table), strings.Join(quotedColumns, ", "), strings.Join(literals, ", "))
	if err = gohiveExec(ctx, conn, query); err != nil {
		return nil, newConnectorError(sparkConnectorType, "write", ErrorCodeQueryFailed, "insert spark rows", err)
	}
	return &WriteResult{RowsWritten: int64(len(rows)), BytesWritten: int64(len(mustJSON(rows)))}, nil
}

func (c *SparkConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "estimate", start, err) }()
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

func (c *SparkConnector) QueryAccessLogs(ctx context.Context, since time.Time) (_ []DataAccessEvent, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "access_logs", start, err) }()

	baseURL := c.config.REST.HistoryURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = c.config.REST.MasterURL
	}
	var rawApps []map[string]any
	if _, err = c.restGET(ctx, baseURL+"/api/v1/applications", &rawApps); err != nil {
		return nil, newConnectorError(sparkConnectorType, "access_logs", ErrorCodeQueryFailed, "query spark applications", err)
	}
	events := make([]DataAccessEvent, 0)
	for _, app := range rawApps {
		appID := fmt.Sprint(app["id"])
		appName := fmt.Sprint(app["name"])
		user := fmt.Sprint(app["sparkUser"])
		var jobs []map[string]any
		_, _ = c.restGET(ctx, baseURL+"/api/v1/applications/"+appID+"/jobs", &jobs)
		var stages []map[string]any
		_, _ = c.restGET(ctx, baseURL+"/api/v1/applications/"+appID+"/stages", &stages)
		var bytesRead, rowsRead int64
		for _, stage := range stages {
			bytesRead += int64Number(stage["inputBytes"])
			rowsRead += int64Number(stage["inputRecords"])
		}
		started := parseSparkTime(anyString(app["attempts"], "startTime"))
		if started.Before(since) {
			continue
		}
		completedAt := parseSparkTime(anyString(app["attempts"], "endTime"))
		successValue := strings.ToLower(anyString(app["attempts"], "completed"))
		if successValue == "" {
			successValue = strings.ToLower(fmt.Sprint(app["completed"]))
		}
		event := DataAccessEvent{
			Timestamp:    completedAt,
			User:         user,
			Action:       "spark_job",
			Database:     appName,
			QueryHash:    sha256Hex(appID),
			QueryPreview: truncateString(appName, 500),
			RowsRead:     rowsRead,
			BytesRead:    bytesRead,
			DurationMs:   completedAt.Sub(started).Milliseconds(),
			Success:      successValue == "true" || successValue == "success",
			SourceType:   sparkConnectorType,
			SourceID:     c.sourceID,
			TenantID:     c.tenantID,
		}
		if len(jobs) == 0 {
			event.ErrorMsg = "no job details returned"
		}
		events = append(events, event)
	}
	getConnectorMetrics().AccessEventsTotal.WithLabelValues(sparkConnectorType).Add(float64(len(events)))
	return events, nil
}

func (c *SparkConnector) ListDataLocations(ctx context.Context) ([]DataLocation, error) {
	conn, err := c.requireThrift(ctx, "locations")
	if err != nil {
		return nil, err
	}
	databaseName := "default"
	if c.config.Thrift != nil && c.config.Thrift.Database != "" {
		databaseName = c.config.Thrift.Database
	}
	_, rows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("SHOW TABLES IN %s", backtickQuote(databaseName)), c.limits.MaxTables)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "locations", ErrorCodeQueryFailed, "show spark tables for locations", err)
	}
	locations := make([]DataLocation, 0, len(rows))
	for _, row := range rows {
		tableName := firstRowValue(row)
		_, formattedRows, queryErr := gohiveQueryRows(ctx, conn, fmt.Sprintf("DESCRIBE FORMATTED %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), 1000)
		if queryErr != nil {
			return nil, newConnectorError(sparkConnectorType, "locations", ErrorCodeQueryFailed, "describe formatted spark table", queryErr)
		}
		meta := parseHiveDescribeFormatted(formattedRows)
		locations = append(locations, DataLocation{
			SourceID:     c.sourceID,
			SourceType:   sparkConnectorType,
			Table:        tableName,
			Database:     databaseName,
			Location:     meta.Location,
			Format:       inputFormatToFormat(meta.InputFormat),
			SizeBytes:    meta.RawDataSize,
			LastModified: time.Now().UTC(),
			Partitioned:  len(meta.PartitionColumns) > 0,
			Partitions:   len(meta.PartitionColumns),
		})
	}
	return locations, nil
}

func (c *SparkConnector) GetActiveApplications(ctx context.Context) (_ []SparkApplication, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "active_apps", start, err) }()
	var payload []map[string]any
	if _, err = c.restGET(ctx, c.config.REST.MasterURL+"/api/v1/applications", &payload); err != nil {
		return nil, newConnectorError(sparkConnectorType, "active_apps", ErrorCodeQueryFailed, "query spark active applications", err)
	}
	apps := make([]SparkApplication, 0, len(payload))
	for _, item := range payload {
		attempts, _ := item["attempts"].([]any)
		state := fmt.Sprint(item["state"])
		if len(attempts) > 0 {
			if attempt, ok := attempts[0].(map[string]any); ok {
				state = fmt.Sprint(attempt["completed"])
			}
		}
		apps = append(apps, SparkApplication{
			ID:        fmt.Sprint(item["id"]),
			Name:      fmt.Sprint(item["name"]),
			User:      fmt.Sprint(item["sparkUser"]),
			StartTime: parseSparkTime(anyString(item["attempts"], "startTime")),
			State:     state,
		})
	}
	return apps, nil
}

func (c *SparkConnector) GetApplicationDetail(ctx context.Context, appID string) (_ *SparkApplicationDetail, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(sparkConnectorType, "app_detail", start, err) }()
	var app map[string]any
	if _, err = c.restGET(ctx, c.config.REST.MasterURL+"/api/v1/applications/"+appID, &app); err != nil {
		return nil, newConnectorError(sparkConnectorType, "app_detail", ErrorCodeQueryFailed, "query spark application detail", err)
	}
	var jobsPayload []map[string]any
	_, _ = c.restGET(ctx, c.config.REST.MasterURL+"/api/v1/applications/"+appID+"/jobs", &jobsPayload)
	var executorsPayload []map[string]any
	_, _ = c.restGET(ctx, c.config.REST.MasterURL+"/api/v1/applications/"+appID+"/executors", &executorsPayload)
	var stagesPayload []map[string]any
	_, _ = c.restGET(ctx, c.config.REST.MasterURL+"/api/v1/applications/"+appID+"/stages", &stagesPayload)
	detail := &SparkApplicationDetail{
		ID:        fmt.Sprint(app["id"]),
		Name:      fmt.Sprint(app["name"]),
		User:      fmt.Sprint(app["sparkUser"]),
		StartTime: parseSparkTime(anyString(app["attempts"], "startTime")),
	}
	for _, job := range jobsPayload {
		detail.Jobs = append(detail.Jobs, SparkJob{
			JobID:        int(int64Number(job["jobId"])),
			Name:         fmt.Sprint(job["name"]),
			Status:       fmt.Sprint(job["status"]),
			NumTasks:     int(int64Number(job["numTasks"])),
			NumCompleted: int(int64Number(job["numCompletedTasks"])),
		})
	}
	for _, executor := range executorsPayload {
		detail.Executors = append(detail.Executors, SparkExecutor{
			ID:            fmt.Sprint(executor["id"]),
			HostPort:      fmt.Sprint(executor["hostPort"]),
			TotalTasks:    int(int64Number(executor["totalTasks"])),
			TotalDuration: int64Number(executor["totalDuration"]),
		})
	}
	for _, stage := range stagesPayload {
		detail.StageMetrics.InputBytes += int64Number(stage["inputBytes"])
		detail.StageMetrics.InputRecords += int64Number(stage["inputRecords"])
	}
	return detail, nil
}

func (c *SparkConnector) Close() error {
	if c.thriftConn != nil {
		c.thriftConn.Close()
		c.thriftConn = nil
	}
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(sparkConnectorType).Dec()
	return nil
}

func (c *SparkConnector) openThrift(ctx context.Context) (*gohive.Connection, error) {
	if c.thriftConn != nil {
		return c.thriftConn, nil
	}
	if c.config.Thrift == nil {
		return nil, newConnectorError(sparkConnectorType, "connect", ErrorCodeConfigurationError, "spark thrift configuration is required for SQL operations", nil)
	}
	cfg := gohive.NewConnectConfiguration()
	cfg.Username = c.config.Thrift.Username
	cfg.Password = c.config.Thrift.Password
	cfg.Service = "spark"
	cfg.Database = defaultString(c.config.Thrift.Database, "default")
	cfg.FetchSize = 1000
	cfg.ConnectTimeout = c.limits.ConnectTimeout
	cfg.SocketTimeout = time.Duration(c.config.QueryTimeoutSeconds) * time.Second
	cfg.HttpTimeout = time.Duration(c.config.QueryTimeoutSeconds) * time.Second
	if strings.EqualFold(c.config.Thrift.AuthType, "kerberos") {
		cfg.TLSConfig = &tls.Config{ServerName: c.config.Thrift.Host, MinVersion: tls.VersionTLS12}
	}
	auth := "NOSASL"
	switch c.config.Thrift.AuthType {
	case "plain":
		auth = "NONE"
	case "kerberos":
		auth = "KERBEROS"
	}
	conn, err := gohive.Connect(c.config.Thrift.Host, c.config.Thrift.Port, auth, cfg)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "connect", ErrorCodeConnectionFailed, "connect to Spark Thrift Server", err)
	}
	c.thriftConn = conn
	return conn, nil
}

func (c *SparkConnector) requireThrift(ctx context.Context, operation string) (*gohive.Connection, error) {
	conn, err := c.openThrift(ctx)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, operation, ErrorCodeConfigurationError, "spark thrift configuration is required for SQL operations", err)
	}
	return conn, nil
}

func (c *SparkConnector) discoverTable(ctx context.Context, conn *gohive.Connection, databaseName, tableName string, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	_, describeRows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("DESCRIBE %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), max(100, opts.MaxColumns))
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "describe spark table", err)
	}
	_, formattedRows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("DESCRIBE FORMATTED %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), 1000)
	if err != nil {
		return nil, newConnectorError(sparkConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "describe formatted spark table", err)
	}
	meta := parseHiveDescribeFormatted(formattedRows)
	columns := make([]model.DiscoveredColumn, 0)
	nullableCount := 0
	for _, row := range describeRows {
		name := strings.TrimSpace(fmt.Sprint(row["col_name"]))
		nativeType := strings.TrimSpace(fmt.Sprint(row["data_type"]))
		if name == "" || strings.HasPrefix(name, "#") || nativeType == "" {
			continue
		}
		mappedType, subtype := hiveLikeTypeMapping(nativeType)
		column := model.DiscoveredColumn{
			Name:       name,
			DataType:   nativeType,
			NativeType: nativeType,
			MappedType: mappedType,
			Subtype:    subtype,
			Nullable:   true,
			Comment:    strings.TrimSpace(fmt.Sprint(row["comment"])),
		}
		if opts.SampleValues {
			samples, sampleErr := c.sampleColumnValues(ctx, conn, databaseName, tableName, name, opts.MaxSamples)
			if sampleErr == nil {
				column.SampleValues = samples
				column.SampleStats = discovery.AnalyzeSamples(samples)
			}
		}
		columns = append(columns, column)
		nullableCount++
	}
	columns = discovery.DetectPII(columns)
	piiCount := 0
	for _, column := range columns {
		if column.InferredPII {
			piiCount++
		}
	}
	return &model.DiscoveredTable{
		SchemaName:     databaseName,
		Name:           tableName,
		Type:           inputFormatToFormat(meta.InputFormat),
		Comment:        meta.Location,
		Columns:        columns,
		EstimatedRows:  meta.NumRows,
		SizeBytes:      meta.RawDataSize,
		InferredClass:  discovery.TableClassification(columns),
		ContainsPII:    piiCount > 0,
		PIIColumnCount: piiCount,
		NullableCount:  nullableCount,
	}, nil
}

func (c *SparkConnector) sampleColumnValues(ctx context.Context, conn *gohive.Connection, databaseName, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	_, rows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s IS NOT NULL LIMIT %d",
		backtickQuote(columnName),
		quoteDotBacktickIdentifier(databaseName+"."+tableName),
		backtickQuote(columnName),
		maxSamples,
	), maxSamples)
	if err != nil {
		return nil, err
	}
	samples := make([]string, 0, len(rows))
	for _, row := range rows {
		samples = append(samples, firstRowValue(row))
	}
	return samples, nil
}

func (c *SparkConnector) restGET(ctx context.Context, url string, out any) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if out == nil {
			resp.Body.Close()
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, fmt.Errorf("spark rest status %s: %s", resp.Status, string(body))
	}
	if out == nil {
		return resp, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}
	return resp, nil
}

func parseSparkTime(value string) time.Time {
	if value == "" {
		return time.Now().UTC()
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05.000GMT", "2006-01-02T15:04:05.000-0700"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	if ms := parseInt64Loose(value); ms > 0 {
		return time.UnixMilli(ms).UTC()
	}
	return time.Now().UTC()
}

func int64Number(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	case string:
		return parseInt64Loose(typed)
	default:
		return 0
	}
}

func anyString(value any, field string) string {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	return fmt.Sprint(item[field])
}
