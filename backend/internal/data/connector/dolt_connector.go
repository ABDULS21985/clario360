package connector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

const doltConnectorType = "dolt"

type DoltConnector struct {
	config   model.DoltConnectionConfig
	db       *sql.DB
	sourceID uuid.UUID
	tenantID uuid.UUID
	logger   zerolog.Logger
	limits   ConnectorLimits
}

type DataChangeEvent struct {
	CommitHash    string    `json:"commit_hash"`
	Committer     string    `json:"committer"`
	CommitDate    time.Time `json:"commit_date"`
	Message       string    `json:"message"`
	Table         string    `json:"table"`
	RowsAdded     int       `json:"rows_added"`
	RowsDeleted   int       `json:"rows_deleted"`
	RowsModified  int       `json:"rows_modified"`
	CellsAdded    int       `json:"cells_added"`
	CellsDeleted  int       `json:"cells_deleted"`
	CellsModified int       `json:"cells_modified"`
}

type DoltBranch struct {
	Name                 string    `json:"name"`
	Hash                 string    `json:"hash"`
	LatestCommitter      string    `json:"latest_committer"`
	LatestCommitterEmail string    `json:"latest_committer_email"`
	LatestCommitDate     time.Time `json:"latest_commit_date"`
	LatestCommitMessage  string    `json:"latest_commit_message"`
}

func NewDoltConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.DoltConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(doltConnectorType, "connect", ErrorCodeConfigurationError, "decode dolt config", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 3306
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(doltConnectorType, "connect", ErrorCodeConfigurationError, "validate dolt config", err)
	}
	connector := &DoltConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", doltConnectorType).Logger(),
		limits: options.Limits,
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(doltConnectorType).Inc()
	return connector, nil
}

func (c *DoltConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *DoltConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "test", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}

	var version string
	if err = db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return nil, newConnectorError(doltConnectorType, "test", ErrorCodeQueryFailed, "query dolt version", err)
	}

	branches, err := c.GetBranches(ctx)
	if err != nil {
		return nil, err
	}
	permissions := make([]string, 0, len(branches))
	for _, branch := range branches {
		permissions = append(permissions, branch.Name)
	}
	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     version,
		Message:     fmt.Sprintf("Connected to Dolt on branch %s.", c.config.Branch),
		Permissions: permissions,
	}, nil
}

func (c *DoltConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "discover", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT TABLE_NAME, TABLE_TYPE, COALESCE(TABLE_ROWS, 0), COALESCE(DATA_LENGTH, 0)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
		LIMIT ?`
	rows, err := db.QueryContext(ctx, query, c.config.Database, opts.MaxTables)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "list dolt tables", err)
	}
	type tableMeta struct {
		name          string
		tableType     string
		estimatedRows int64
		sizeBytes     int64
	}
	tableMetas := make([]tableMeta, 0)
	for rows.Next() {
		var meta tableMeta
		if err = rows.Scan(&meta.name, &meta.tableType, &meta.estimatedRows, &meta.sizeBytes); err != nil {
			_ = rows.Close()
			return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "scan dolt table", err)
		}
		tableMetas = append(tableMetas, meta)
	}
	if closeErr := rows.Close(); closeErr != nil {
		return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeDriverError, "close dolt table rows", closeErr)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "iterate dolt tables", err)
	}

	tables := make([]model.DiscoveredTable, 0, len(tableMetas))
	totalColumns := 0
	containsPII := false
	highest := model.DataClassificationPublic
	for _, meta := range tableMetas {
		table, tableErr := c.discoverTable(ctx, db, meta.name, meta.tableType, meta.estimatedRows, meta.sizeBytes, opts)
		if tableErr != nil {
			return nil, tableErr
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highest = discovery.MaxClassification(highest, table.InferredClass)
	}
	observeSchemaMetrics(doltConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *DoltConnector) FetchData(ctx context.Context, table string, params FetchParams) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "fetch", start, err) }()

	db, err := c.openDB(ctx)
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
	args := make([]any, 0, len(params.Filters)+2)
	conditions := make([]string, 0, len(params.Filters))
	for column, value := range params.Filters {
		conditions = append(conditions, fmt.Sprintf("%s = ?", backtickQuote(column)))
		args = append(args, value)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if params.OrderBy != "" {
		query += " ORDER BY " + backtickQuote(params.OrderBy)
	}
	if params.BatchSize <= 0 {
		params.BatchSize = 100
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.BatchSize, params.Offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "fetch", ErrorCodeQueryFailed, "fetch dolt rows", err)
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "fetch", ErrorCodeDriverError, "read dolt columns", err)
	}
	resultRows := make([]map[string]any, 0, params.BatchSize)
	for rows.Next() {
		values := make([]any, len(columnNames))
		targets := make([]any, len(columnNames))
		for i := range values {
			targets[i] = &values[i]
		}
		if err = rows.Scan(targets...); err != nil {
			return nil, newConnectorError(doltConnectorType, "fetch", ErrorCodeDriverError, "scan dolt row", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = normalizeSQLValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(doltConnectorType, "fetch", ErrorCodeDriverError, "iterate dolt rows", err)
	}
	observeFetchMetrics(doltConnectorType, len(resultRows), int64(len(mustJSON(resultRows))))
	return &DataBatch{
		Columns:  columnNames,
		Rows:     resultRows,
		RowCount: len(resultRows),
		HasMore:  len(resultRows) == params.BatchSize,
	}, nil
}

func (c *DoltConnector) ReadQuery(ctx context.Context, query string, args []any) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "read_query", start, err) }()

	if !isReadOnlyQuery(query) {
		return nil, newConnectorError(doltConnectorType, "read_query", ErrorCodeUnsupportedOperation, "only read-only queries are allowed", ErrCapabilityUnsupported)
	}
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "read_query", ErrorCodeQueryFailed, "execute dolt query", err)
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "read_query", ErrorCodeDriverError, "read dolt query columns", err)
	}
	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columnNames))
		targets := make([]any, len(columnNames))
		for i := range values {
			targets[i] = &values[i]
		}
		if err = rows.Scan(targets...); err != nil {
			return nil, newConnectorError(doltConnectorType, "read_query", ErrorCodeDriverError, "scan dolt query row", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = normalizeSQLValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(doltConnectorType, "read_query", ErrorCodeDriverError, "iterate dolt query rows", err)
	}
	return &DataBatch{Columns: columnNames, Rows: resultRows, RowCount: len(resultRows)}, nil
}

func (c *DoltConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (_ *WriteResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "write", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "write", ErrorCodeDriverError, "begin dolt write transaction", err)
	}
	defer tx.Rollback()

	if params.Replace {
		if _, err = tx.ExecContext(ctx, "TRUNCATE TABLE "+quoteDotBacktickIdentifier(table)); err != nil {
			return nil, newConnectorError(doltConnectorType, "write", ErrorCodeQueryFailed, "truncate dolt table", err)
		}
	}

	columns := writeColumns(rows)
	if len(columns) == 0 {
		return &WriteResult{}, nil
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, backtickQuote(column))
	}
	valueGroups := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*len(columns))
	for range rows {
		placeholders := make([]string, 0, len(columns))
		for range columns {
			placeholders = append(placeholders, "?")
		}
		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ", ")+")")
	}
	for _, row := range rows {
		args = append(args, rowValues(row, columns)...)
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		quoteDotBacktickIdentifier(table),
		strings.Join(quotedColumns, ", "),
		strings.Join(valueGroups, ", "),
	)
	if params.Strategy == "merge" && len(params.MergeKeys) > 0 {
		assignments := make([]string, 0, len(columns))
		mergeSet := make(map[string]struct{}, len(params.MergeKeys))
		for _, key := range params.MergeKeys {
			mergeSet[key] = struct{}{}
		}
		for _, column := range columns {
			if _, skip := mergeSet[column]; skip {
				continue
			}
			quoted := backtickQuote(column)
			assignments = append(assignments, fmt.Sprintf("%s = VALUES(%s)", quoted, quoted))
		}
		if len(assignments) > 0 {
			query += " ON DUPLICATE KEY UPDATE " + strings.Join(assignments, ", ")
		}
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "write", ErrorCodeQueryFailed, "insert dolt rows", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, newConnectorError(doltConnectorType, "write", ErrorCodeDriverError, "commit dolt transaction", err)
	}
	affected, _ := result.RowsAffected()
	return &WriteResult{
		RowsWritten:  affected,
		BytesWritten: int64(len(mustJSON(rows))),
	}, nil
}

func (c *DoltConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "estimate", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	var estimate SizeEstimate
	if err = db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(TABLE_ROWS), 0), COALESCE(SUM(DATA_LENGTH), 0)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?`, c.config.Database).Scan(&estimate.TableCount, &estimate.TotalRows, &estimate.TotalBytes); err != nil {
		return nil, newConnectorError(doltConnectorType, "estimate", ErrorCodeQueryFailed, "estimate dolt size", err)
	}
	return &estimate, nil
}

func (c *DoltConnector) QueryAccessLogs(ctx context.Context, since time.Time) (_ []DataAccessEvent, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "access_logs", start, err) }()

	changes, err := c.GetRecentChanges(ctx, since)
	if err != nil {
		return nil, err
	}
	events := make([]DataAccessEvent, 0, len(changes))
	for _, change := range changes {
		events = append(events, DataAccessEvent{
			Timestamp:    change.CommitDate,
			User:         change.Committer,
			Action:       "commit",
			Database:     c.config.Database,
			Table:        change.Table,
			QueryHash:    sha256Hex(change.CommitHash),
			QueryPreview: truncateString(change.Message, 500),
			RowsWritten:  int64(change.RowsAdded + change.RowsDeleted + change.RowsModified),
			DurationMs:   0,
			Success:      true,
			SourceType:   doltConnectorType,
			SourceID:     c.sourceID,
			TenantID:     c.tenantID,
		})
	}
	getConnectorMetrics().AccessEventsTotal.WithLabelValues(doltConnectorType).Add(float64(len(events)))
	return events, nil
}

func (c *DoltConnector) ListDataLocations(ctx context.Context) (_ []DataLocation, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(doltConnectorType, "locations", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT TABLE_NAME, COALESCE(DATA_LENGTH, 0), COALESCE(UPDATE_TIME, NOW())
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?`, c.config.Database)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "locations", ErrorCodeQueryFailed, "query dolt table locations", err)
	}
	defer rows.Close()

	locations := make([]DataLocation, 0)
	for rows.Next() {
		var table string
		var sizeBytes int64
		var updatedAt time.Time
		if err = rows.Scan(&table, &sizeBytes, &updatedAt); err != nil {
			return nil, newConnectorError(doltConnectorType, "locations", ErrorCodeDriverError, "scan dolt location", err)
		}
		locations = append(locations, DataLocation{
			SourceID:     c.sourceID,
			SourceType:   doltConnectorType,
			Table:        table,
			Database:     c.config.Database,
			Location:     fmt.Sprintf("dolt://%s:%d/%s/%s@%s", c.config.Host, c.config.Port, c.config.Database, table, c.config.Branch),
			Format:       "managed",
			SizeBytes:    sizeBytes,
			LastModified: updatedAt.UTC(),
		})
	}
	return locations, rows.Err()
}

func (c *DoltConnector) GetRecentChanges(ctx context.Context, since time.Time) (_ []DataChangeEvent, err error) {
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT commit_hash, CONCAT(committer, ' <', email, '>'), date, message
		FROM dolt_log
		WHERE date > ?
		ORDER BY date DESC
		LIMIT 10000`, since)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeQueryFailed, "query dolt log", err)
	}
	type commitMeta struct {
		hash      string
		committer string
		date      time.Time
		message   string
	}
	commits := make([]commitMeta, 0)
	for rows.Next() {
		var commit commitMeta
		if err = rows.Scan(&commit.hash, &commit.committer, &commit.date, &commit.message); err != nil {
			_ = rows.Close()
			return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeDriverError, "scan dolt log row", err)
		}
		commits = append(commits, commit)
	}
	if closeErr := rows.Close(); closeErr != nil {
		return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeDriverError, "close dolt log rows", closeErr)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeDriverError, "iterate dolt log rows", err)
	}

	changes := make([]DataChangeEvent, 0)
	for _, commit := range commits {
		previousRef := commit.hash + "~1"
		diffRows, diffErr := db.QueryContext(ctx, `
			SELECT table_name, rows_added, rows_deleted, rows_modified,
			       cells_added, cells_deleted, cells_modified
			FROM dolt_diff_stat(?, ?)`, previousRef, commit.hash)
		if diffErr != nil {
			return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeQueryFailed, "query dolt diff stat", diffErr)
		}
		for diffRows.Next() {
			var event DataChangeEvent
			event.CommitHash = commit.hash
			event.Committer = commit.committer
			event.CommitDate = commit.date.UTC()
			event.Message = commit.message
			if err = diffRows.Scan(&event.Table, &event.RowsAdded, &event.RowsDeleted, &event.RowsModified, &event.CellsAdded, &event.CellsDeleted, &event.CellsModified); err != nil {
				diffRows.Close()
				return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeDriverError, "scan dolt diff stat", err)
			}
			changes = append(changes, event)
		}
		if closeErr := diffRows.Close(); closeErr != nil && err == nil {
			return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeDriverError, "close dolt diff rows", closeErr)
		}
		if err = diffRows.Err(); err != nil {
			return nil, newConnectorError(doltConnectorType, "changes", ErrorCodeDriverError, "iterate dolt diff rows", err)
		}
	}
	return changes, nil
}

func (c *DoltConnector) GetBranches(ctx context.Context) (_ []DoltBranch, err error) {
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT name, hash, latest_committer, latest_committer_email, latest_commit_date, latest_commit_message
		FROM dolt_branches
		ORDER BY latest_commit_date DESC`)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "branches", ErrorCodeQueryFailed, "query dolt branches", err)
	}
	defer rows.Close()
	branches := make([]DoltBranch, 0)
	for rows.Next() {
		var branch DoltBranch
		if err = rows.Scan(&branch.Name, &branch.Hash, &branch.LatestCommitter, &branch.LatestCommitterEmail, &branch.LatestCommitDate, &branch.LatestCommitMessage); err != nil {
			return nil, newConnectorError(doltConnectorType, "branches", ErrorCodeDriverError, "scan dolt branch", err)
		}
		branches = append(branches, branch)
	}
	return branches, rows.Err()
}

func (c *DoltConnector) SwitchBranch(ctx context.Context, branch string) error {
	db, err := c.openDB(ctx)
	if err != nil {
		return err
	}
	if _, err = db.ExecContext(ctx, "CALL DOLT_CHECKOUT(?)", branch); err != nil {
		return newConnectorError(doltConnectorType, "checkout", ErrorCodeQueryFailed, "switch dolt branch", err)
	}
	c.config.Branch = branch
	return nil
}

func (c *DoltConnector) Close() error {
	if c.db != nil {
		err := c.db.Close()
		c.db = nil
		getConnectorMetrics().ActiveConnections.WithLabelValues(doltConnectorType).Dec()
		return err
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(doltConnectorType).Dec()
	return nil
}

func (c *DoltConnector) openDB(ctx context.Context) (*sql.DB, error) {
	if c.db != nil {
		return c.db, nil
	}
	cfg := mysqlDriver.NewConfig()
	cfg.User = c.config.Username
	cfg.Passwd = c.config.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	cfg.DBName = c.config.Database
	cfg.ParseTime = true
	if c.config.UseTLS {
		cfg.TLSConfig = "preferred"
	}
	cfg.Timeout = c.limits.ConnectTimeout
	cfg.ReadTimeout = c.limits.StatementTimeout
	cfg.WriteTimeout = c.limits.StatementTimeout

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "connect", ErrorCodeConnectionFailed, "open dolt connection", err)
	}
	maxOpen := c.config.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = c.limits.MaxPoolSize
	}
	maxIdle := c.config.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = max(1, maxOpen/2)
	}
	if c.config.Branch != "" {
		// Branch selection is session-scoped; keep a sticky pool.
		maxOpen = 1
		maxIdle = 1
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	lifetime := 30 * time.Minute
	if c.config.ConnMaxLifetimeMins > 0 {
		lifetime = time.Duration(c.config.ConnMaxLifetimeMins) * time.Minute
	}
	db.SetConnMaxLifetime(lifetime)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, newConnectorError(doltConnectorType, "connect", ErrorCodeConnectionFailed, "ping dolt source", err)
	}
	c.db = db
	if c.config.Branch != "" {
		if err := c.SwitchBranch(ctx, c.config.Branch); err != nil {
			db.Close()
			c.db = nil
			return nil, err
		}
	}
	return db, nil
}

func (c *DoltConnector) discoverTable(ctx context.Context, db *sql.DB, tableName, tableType string, estimatedRows, sizeBytes int64, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE,
		       COLUMN_DEFAULT, COLUMN_COMMENT, COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, c.config.Database, tableName)
	if err != nil {
		return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "query dolt columns", err)
	}

	type columnMeta struct {
		columnName   string
		dataType     string
		columnType   string
		maxLength    *int
		nullable     string
		defaultValue *string
		comment      string
		columnKey    string
	}
	columnMetas := make([]columnMeta, 0)
	for rows.Next() {
		var meta columnMeta
		if err = rows.Scan(&meta.columnName, &meta.dataType, &meta.columnType, &meta.maxLength, &meta.nullable, &meta.defaultValue, &meta.comment, &meta.columnKey); err != nil {
			return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeDriverError, "scan dolt column", err)
		}
		columnMetas = append(columnMetas, meta)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeDriverError, "iterate dolt columns", err)
	}
	if err = rows.Close(); err != nil {
		return nil, newConnectorError(doltConnectorType, "discover", ErrorCodeDriverError, "close dolt columns", err)
	}

	columns := make([]model.DiscoveredColumn, 0, len(columnMetas))
	primaryKeys := make([]string, 0)
	for _, meta := range columnMetas {
		mapped := discovery.MapNativeType(meta.columnType)
		column := model.DiscoveredColumn{
			Name:         meta.columnName,
			DataType:     meta.dataType,
			NativeType:   meta.columnType,
			MappedType:   mapped.Type,
			Subtype:      mapped.Subtype,
			MaxLength:    meta.maxLength,
			Nullable:     strings.EqualFold(meta.nullable, "YES"),
			DefaultValue: meta.defaultValue,
			Comment:      meta.comment,
			IsPrimaryKey: meta.columnKey == "PRI",
		}
		if meta.columnKey == "PRI" {
			primaryKeys = append(primaryKeys, meta.columnName)
		}
		if opts.SampleValues {
			samples, sampleErr := c.sampleColumnValues(ctx, db, tableName, meta.columnName, opts.MaxSamples)
			if sampleErr == nil {
				column.SampleValues = samples
				column.SampleStats = discovery.AnalyzeSamples(samples)
			}
		}
		columns = append(columns, column)
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
		Name:           tableName,
		Type:           strings.ToLower(tableType),
		Columns:        columns,
		PrimaryKeys:    primaryKeys,
		EstimatedRows:  estimatedRows,
		SizeBytes:      sizeBytes,
		InferredClass:  discovery.TableClassification(columns),
		ContainsPII:    piiCount > 0,
		PIIColumnCount: piiCount,
		NullableCount:  nullableCount,
	}, nil
}

func (c *DoltConnector) sampleColumnValues(ctx context.Context, db *sql.DB, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL LIMIT %d",
		backtickQuote(columnName),
		backtickQuote(tableName),
		backtickQuote(columnName),
		maxSamples,
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	samples := make([]string, 0, maxSamples)
	for rows.Next() {
		var value any
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		samples = append(samples, fmt.Sprint(value))
	}
	return samples, rows.Err()
}
