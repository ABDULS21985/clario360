package testhelpers

import (
	"context"
	"fmt"
	"io"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/beltran/gohive/hiveserver"
	tc "github.com/testcontainers/testcontainers-go"
)

type MockHiveCatalog struct {
	DefaultDatabase string
	Databases       map[string]*MockHiveDatabase
}

type MockHiveDatabase struct {
	Name   string
	Tables map[string]*MockHiveTable
}

type MockHiveTable struct {
	Name             string
	Columns          []MockHiveColumn
	Rows             []map[string]string
	Location         string
	InputFormat      string
	NumRows          int64
	RawDataSize      int64
	PartitionColumns []string
	LastDDLTime      time.Time
}

type MockHiveColumn struct {
	Name    string
	Type    string
	Comment string
}

type MockThriftServer struct {
	addr      string
	server    *thrift.TSimpleServer
	service   *mockTCLIService
	closeOnce sync.Once
}

type mockSession struct {
	currentDatabase string
}

type mockOperation struct {
	columns    []mockResultColumn
	rows       []map[string]string
	hasResults bool
	fetched    bool
}

type mockResultColumn struct {
	Name    string
	Type    hiveserver.TTypeId
	Comment string
}

type mockTCLIService struct {
	mu          sync.RWMutex
	catalog     *MockHiveCatalog
	sessions    map[string]*mockSession
	operations  map[string]*mockOperation
	recorded    []string
}

var (
	usePattern             = regexp.MustCompile(`(?i)^use\s+([a-zA-Z0-9_]+)$`)
	showTablesPattern      = regexp.MustCompile("(?i)^show\\s+tables\\s+in\\s+(.+)$")
	describePattern        = regexp.MustCompile("(?i)^describe\\s+(.+)$")
	describeFormattedPattern = regexp.MustCompile("(?i)^describe\\s+formatted\\s+(.+)$")
	selectPattern          = regexp.MustCompile(`(?i)^select\s+(.+?)\s+from\s+([a-zA-Z0-9_` + "`" + `\.]+)(?:\s+where\s+(.+?))?(?:\s+order\s+by\s+([a-zA-Z0-9_` + "`" + `]+)(?:\s+(asc|desc))?)?(?:\s+limit\s+(\d+))?(?:\s+offset\s+(\d+))?$`)
)

func NewMockThriftServer(t testing.TB, catalog MockHiveCatalog) *MockThriftServer {
	t.Helper()
	service := &mockTCLIService{
		catalog:    &catalog,
		sessions:   make(map[string]*mockSession),
		operations: make(map[string]*mockOperation),
	}
	if service.catalog.DefaultDatabase == "" {
		service.catalog.DefaultDatabase = "default"
	}
	addr, err := reserveTCPAddress()
	if err != nil {
		t.Fatalf("reserve thrift server address: %v", err)
	}
	socket, err := thrift.NewTServerSocket(addr)
	if err != nil {
		t.Fatalf("create thrift server socket: %v", err)
	}
	server := thrift.NewTSimpleServer4(
		hiveserver.NewTCLIServiceProcessor(service),
		socket,
		thrift.NewTTransportFactory(),
		thrift.NewTBinaryProtocolFactoryDefault(),
	)
	mock := &MockThriftServer{
		addr:    socket.Addr().String(),
		server:  server,
		service: service,
	}
	go func() {
		_ = server.Serve()
	}()
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waitForTCP(waitCtx, mock.addr); err != nil {
		t.Fatalf("wait for mock thrift server: %v", err)
	}
	t.Cleanup(func() {
		_ = mock.Close()
	})
	return mock
}

func (m *MockThriftServer) Address() string {
	return m.addr
}

func (m *MockThriftServer) HostPort() (string, int) {
	host, portText, _ := net.SplitHostPort(m.addr)
	port, _ := strconv.Atoi(portText)
	return host, port
}

func (m *MockThriftServer) RecordedQueries() []string {
	m.service.mu.RLock()
	defer m.service.mu.RUnlock()
	values := make([]string, len(m.service.recorded))
	copy(values, m.service.recorded)
	return values
}

func (m *MockThriftServer) Close() error {
	var err error
	m.closeOnce.Do(func() {
		err = m.server.Stop()
	})
	return err
}

func waitForTCP(ctx context.Context, addr string) error {
	var lastErr error
	for {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return fmt.Errorf("dial %s: %w", addr, lastErr)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func reserveTCPAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()
	if closeErr := listener.Close(); closeErr != nil {
		return "", closeErr
	}
	return addr, nil
}

func (s *mockTCLIService) OpenSession(_ context.Context, _ *hiveserver.TOpenSessionReq) (*hiveserver.TOpenSessionResp, error) {
	sessionID := newHandleID()
	s.mu.Lock()
	s.sessions[sessionID] = &mockSession{currentDatabase: s.catalog.DefaultDatabase}
	s.mu.Unlock()
	protocol := hiveserver.TProtocolVersion_HIVE_CLI_SERVICE_PROTOCOL_V6
	return &hiveserver.TOpenSessionResp{
		Status:                successStatus(),
		ServerProtocolVersion: protocol,
		SessionHandle:         sessionHandle(sessionID),
		Configuration:         map[string]string{},
	}, nil
}

func (s *mockTCLIService) CloseSession(_ context.Context, req *hiveserver.TCloseSessionReq) (*hiveserver.TCloseSessionResp, error) {
	s.mu.Lock()
	delete(s.sessions, handleKey(req.GetSessionHandle().GetSessionId()))
	s.mu.Unlock()
	return &hiveserver.TCloseSessionResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetInfo(_ context.Context, _ *hiveserver.TGetInfoReq) (*hiveserver.TGetInfoResp, error) {
	return &hiveserver.TGetInfoResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) ExecuteStatement(_ context.Context, req *hiveserver.TExecuteStatementReq) (*hiveserver.TExecuteStatementResp, error) {
	query := strings.TrimSpace(req.GetStatement())
	s.mu.Lock()
	s.recorded = append(s.recorded, query)
	s.mu.Unlock()
	result, sessionErr := s.executeQuery(handleKey(req.GetSessionHandle().GetSessionId()), query)
	if sessionErr != nil {
		return &hiveserver.TExecuteStatementResp{Status: errorStatus(sessionErr.Error())}, nil
	}
	operationID := newHandleID()
	s.mu.Lock()
	s.operations[operationID] = result
	s.mu.Unlock()
	return &hiveserver.TExecuteStatementResp{
		Status: successStatus(),
		OperationHandle: &hiveserver.TOperationHandle{
			OperationId:   handleIdentifier(operationID),
			OperationType: hiveserver.TOperationType_EXECUTE_STATEMENT,
			HasResultSet:  result.hasResults,
		},
	}, nil
}

func (s *mockTCLIService) GetTypeInfo(_ context.Context, _ *hiveserver.TGetTypeInfoReq) (*hiveserver.TGetTypeInfoResp, error) {
	return &hiveserver.TGetTypeInfoResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetCatalogs(_ context.Context, _ *hiveserver.TGetCatalogsReq) (*hiveserver.TGetCatalogsResp, error) {
	return &hiveserver.TGetCatalogsResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetSchemas(_ context.Context, _ *hiveserver.TGetSchemasReq) (*hiveserver.TGetSchemasResp, error) {
	return &hiveserver.TGetSchemasResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetTables(_ context.Context, _ *hiveserver.TGetTablesReq) (*hiveserver.TGetTablesResp, error) {
	return &hiveserver.TGetTablesResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetTableTypes(_ context.Context, _ *hiveserver.TGetTableTypesReq) (*hiveserver.TGetTableTypesResp, error) {
	return &hiveserver.TGetTableTypesResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetColumns(_ context.Context, _ *hiveserver.TGetColumnsReq) (*hiveserver.TGetColumnsResp, error) {
	return &hiveserver.TGetColumnsResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetFunctions(_ context.Context, _ *hiveserver.TGetFunctionsReq) (*hiveserver.TGetFunctionsResp, error) {
	return &hiveserver.TGetFunctionsResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetPrimaryKeys(_ context.Context, _ *hiveserver.TGetPrimaryKeysReq) (*hiveserver.TGetPrimaryKeysResp, error) {
	return &hiveserver.TGetPrimaryKeysResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetCrossReference(_ context.Context, _ *hiveserver.TGetCrossReferenceReq) (*hiveserver.TGetCrossReferenceResp, error) {
	return &hiveserver.TGetCrossReferenceResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetOperationStatus(_ context.Context, req *hiveserver.TGetOperationStatusReq) (*hiveserver.TGetOperationStatusResp, error) {
	s.mu.RLock()
	operation := s.operations[handleKey(req.GetOperationHandle().GetOperationId())]
	s.mu.RUnlock()
	if operation == nil {
		return &hiveserver.TGetOperationStatusResp{Status: errorStatus("operation not found")}, nil
	}
	state := hiveserver.TOperationState_FINISHED_STATE
	hasResults := operation.hasResults
	now := time.Now().UnixMilli()
	return &hiveserver.TGetOperationStatusResp{
		Status:             successStatus(),
		OperationState:     &state,
		HasResultSet:       &hasResults,
		OperationStarted:   &now,
		OperationCompleted: &now,
	}, nil
}

func (s *mockTCLIService) CancelOperation(_ context.Context, _ *hiveserver.TCancelOperationReq) (*hiveserver.TCancelOperationResp, error) {
	return &hiveserver.TCancelOperationResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) CloseOperation(_ context.Context, req *hiveserver.TCloseOperationReq) (*hiveserver.TCloseOperationResp, error) {
	s.mu.Lock()
	delete(s.operations, handleKey(req.GetOperationHandle().GetOperationId()))
	s.mu.Unlock()
	return &hiveserver.TCloseOperationResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetResultSetMetadata(_ context.Context, req *hiveserver.TGetResultSetMetadataReq) (*hiveserver.TGetResultSetMetadataResp, error) {
	s.mu.RLock()
	operation := s.operations[handleKey(req.GetOperationHandle().GetOperationId())]
	s.mu.RUnlock()
	if operation == nil {
		return &hiveserver.TGetResultSetMetadataResp{Status: errorStatus("operation not found")}, nil
	}
	return &hiveserver.TGetResultSetMetadataResp{
		Status: successStatus(),
		Schema: buildTableSchema(operation.columns),
	}, nil
}

func (s *mockTCLIService) FetchResults(_ context.Context, req *hiveserver.TFetchResultsReq) (*hiveserver.TFetchResultsResp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	operation := s.operations[handleKey(req.GetOperationHandle().GetOperationId())]
	if operation == nil {
		return &hiveserver.TFetchResultsResp{Status: errorStatus("operation not found")}, nil
	}
	rows := []map[string]string{}
	if operation.hasResults && !operation.fetched {
		rows = operation.rows
		operation.fetched = true
	}
	return &hiveserver.TFetchResultsResp{
		Status:  successStatus(),
		HasMoreRows: boolPtr(false),
		Results: buildRowSet(operation.columns, rows),
	}, nil
}

func (s *mockTCLIService) GetDelegationToken(_ context.Context, _ *hiveserver.TGetDelegationTokenReq) (*hiveserver.TGetDelegationTokenResp, error) {
	return &hiveserver.TGetDelegationTokenResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) CancelDelegationToken(_ context.Context, _ *hiveserver.TCancelDelegationTokenReq) (*hiveserver.TCancelDelegationTokenResp, error) {
	return &hiveserver.TCancelDelegationTokenResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) RenewDelegationToken(_ context.Context, _ *hiveserver.TRenewDelegationTokenReq) (*hiveserver.TRenewDelegationTokenResp, error) {
	return &hiveserver.TRenewDelegationTokenResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) GetQueryId(_ context.Context, _ *hiveserver.TGetQueryIdReq) (*hiveserver.TGetQueryIdResp, error) {
	return &hiveserver.TGetQueryIdResp{QueryId: newHandleID()}, nil
}

func (s *mockTCLIService) SetClientInfo(_ context.Context, _ *hiveserver.TSetClientInfoReq) (*hiveserver.TSetClientInfoResp, error) {
	return &hiveserver.TSetClientInfoResp{Status: successStatus()}, nil
}

func (s *mockTCLIService) executeQuery(sessionID, query string) (*mockOperation, error) {
	s.mu.RLock()
	session := s.sessions[sessionID]
	s.mu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	trimmed := strings.TrimSpace(query)
	normalized := normalizeQuery(trimmed)

	if strings.EqualFold(normalized, "show databases") {
		names := make([]string, 0, len(s.catalog.Databases))
		for name := range s.catalog.Databases {
			names = append(names, name)
		}
		slices.Sort(names)
		rows := make([]map[string]string, 0, len(names))
		for _, name := range names {
			rows = append(rows, map[string]string{"database_name": name})
		}
		return &mockOperation{
			hasResults: true,
			columns:    []mockResultColumn{{Name: "database_name", Type: hiveserver.TTypeId_STRING_TYPE}},
			rows:       rows,
		}, nil
	}
	if matches := usePattern.FindStringSubmatch(normalized); len(matches) == 2 {
		dbName := strings.ToLower(matches[1])
		if _, ok := s.catalog.Databases[dbName]; !ok {
			return nil, fmt.Errorf("unknown database %s", dbName)
		}
		s.mu.Lock()
		session.currentDatabase = dbName
		s.mu.Unlock()
		return &mockOperation{hasResults: false}, nil
	}
	if matches := showTablesPattern.FindStringSubmatch(normalized); len(matches) == 2 {
		dbName := normalizeIdentifier(matches[1])
		db, ok := s.catalog.Databases[dbName]
		if !ok {
			return nil, fmt.Errorf("unknown database %s", dbName)
		}
		names := make([]string, 0, len(db.Tables))
		for name := range db.Tables {
			names = append(names, name)
		}
		slices.Sort(names)
		rows := make([]map[string]string, 0, len(names))
		for _, name := range names {
			rows = append(rows, map[string]string{"tab_name": name})
		}
		return &mockOperation{
			hasResults: true,
			columns:    []mockResultColumn{{Name: "tab_name", Type: hiveserver.TTypeId_STRING_TYPE}},
			rows:       rows,
		}, nil
	}
	if matches := describeFormattedPattern.FindStringSubmatch(normalized); len(matches) == 2 {
		table, err := s.lookupTable(session.currentDatabase, matches[1])
		if err != nil {
			return nil, err
		}
		rows := describeFormattedRows(table)
		return &mockOperation{
			hasResults: true,
			columns: []mockResultColumn{
				{Name: "col_name", Type: hiveserver.TTypeId_STRING_TYPE},
				{Name: "data_type", Type: hiveserver.TTypeId_STRING_TYPE},
				{Name: "comment", Type: hiveserver.TTypeId_STRING_TYPE},
			},
			rows: rows,
		}, nil
	}
	if matches := describePattern.FindStringSubmatch(normalized); len(matches) == 2 {
		table, err := s.lookupTable(session.currentDatabase, matches[1])
		if err != nil {
			return nil, err
		}
		rows := describeRows(table)
		return &mockOperation{
			hasResults: true,
			columns: []mockResultColumn{
				{Name: "col_name", Type: hiveserver.TTypeId_STRING_TYPE},
				{Name: "data_type", Type: hiveserver.TTypeId_STRING_TYPE},
				{Name: "comment", Type: hiveserver.TTypeId_STRING_TYPE},
			},
			rows: rows,
		}, nil
	}
	if matches := selectPattern.FindStringSubmatch(normalized); len(matches) == 8 {
		return s.executeSelect(session.currentDatabase, matches[1], matches[2], matches[3], matches[4], matches[5], matches[6], matches[7])
	}
	return nil, fmt.Errorf("unsupported query %q", query)
}

func (s *mockTCLIService) executeSelect(currentDB, columnsPart, tableIdent, wherePart, orderByPart, orderDirection, limitPart, offsetPart string) (*mockOperation, error) {
	table, err := s.lookupTable(currentDB, tableIdent)
	if err != nil {
		return nil, err
	}

	selectedColumns := make([]MockHiveColumn, 0)
	if strings.TrimSpace(columnsPart) == "*" {
		selectedColumns = append(selectedColumns, table.Columns...)
	} else {
		for _, column := range strings.Split(columnsPart, ",") {
			name := normalizeIdentifier(column)
			found := false
			for _, candidate := range table.Columns {
				if strings.EqualFold(candidate.Name, name) {
					selectedColumns = append(selectedColumns, candidate)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown column %s", name)
			}
		}
	}

	rows := make([]map[string]string, 0, len(table.Rows))
	for _, row := range table.Rows {
		if !matchesWhereClause(row, wherePart) {
			continue
		}
		selected := make(map[string]string, len(selectedColumns))
		for _, column := range selectedColumns {
			selected[column.Name] = lookupRowValue(row, column.Name)
		}
		rows = append(rows, selected)
	}

	if orderByPart != "" {
		orderColumn := normalizeIdentifier(orderByPart)
		slices.SortFunc(rows, func(a, b map[string]string) int {
			return strings.Compare(lookupRowValue(a, orderColumn), lookupRowValue(b, orderColumn))
		})
		if strings.EqualFold(strings.TrimSpace(orderDirection), "desc") {
			slices.Reverse(rows)
		}
	}
	limit := len(rows)
	if value, err := strconv.Atoi(limitPart); err == nil && value >= 0 && value < limit {
		limit = value
	}
	offset := 0
	if value, err := strconv.Atoi(offsetPart); err == nil && value >= 0 {
		offset = value
	}
	if offset > len(rows) {
		offset = len(rows)
	}
	rows = rows[offset:]
	if limit < len(rows) {
		rows = rows[:limit]
	}

	resultColumns := make([]mockResultColumn, 0, len(selectedColumns))
	for _, column := range selectedColumns {
		resultColumns = append(resultColumns, mockResultColumn{
			Name:    column.Name,
			Type:    toTypeID(column.Type),
			Comment: column.Comment,
		})
	}
	return &mockOperation{
		hasResults: true,
		columns:    resultColumns,
		rows:       rows,
	}, nil
}

func (s *mockTCLIService) lookupTable(currentDB, identifier string) (*MockHiveTable, error) {
	qualified := normalizeIdentifier(identifier)
	dbName := currentDB
	tableName := qualified
	if strings.Contains(qualified, ".") {
		parts := strings.SplitN(qualified, ".", 2)
		dbName, tableName = parts[0], parts[1]
	}
	db, ok := s.catalog.Databases[strings.ToLower(dbName)]
	if !ok {
		return nil, fmt.Errorf("unknown database %s", dbName)
	}
	table, ok := db.Tables[strings.ToLower(tableName)]
	if !ok {
		return nil, fmt.Errorf("unknown table %s.%s", dbName, tableName)
	}
	return table, nil
}

func describeRows(table *MockHiveTable) []map[string]string {
	rows := make([]map[string]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		rows = append(rows, map[string]string{
			"col_name":  column.Name,
			"data_type": column.Type,
			"comment":   column.Comment,
		})
	}
	return rows
}

func describeFormattedRows(table *MockHiveTable) []map[string]string {
	lastDDL := table.LastDDLTime
	if lastDDL.IsZero() {
		lastDDL = time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	}
	rows := describeRows(table)
	rows = append(rows,
		map[string]string{"col_name": "", "data_type": "", "comment": ""},
		map[string]string{"col_name": "# Detailed Table Information", "data_type": "", "comment": ""},
		map[string]string{"col_name": "Location:", "data_type": table.Location, "comment": ""},
		map[string]string{"col_name": "numRows:", "data_type": strconv.FormatInt(nonZeroOr(table.NumRows, int64(len(table.Rows))), 10), "comment": ""},
		map[string]string{"col_name": "rawDataSize:", "data_type": strconv.FormatInt(nonZeroOr(table.RawDataSize, 4096), 10), "comment": ""},
		map[string]string{"col_name": "transient_lastDdlTime:", "data_type": strconv.FormatInt(lastDDL.Unix(), 10), "comment": ""},
		map[string]string{"col_name": "# Storage Information", "data_type": "", "comment": ""},
		map[string]string{"col_name": "InputFormat:", "data_type": table.InputFormat, "comment": ""},
	)
	if len(table.PartitionColumns) > 0 {
		rows = append(rows, map[string]string{"col_name": "# Partition Information", "data_type": "", "comment": ""})
		for _, partition := range table.PartitionColumns {
			rows = append(rows, map[string]string{"col_name": partition, "data_type": "string", "comment": ""})
		}
	}
	return rows
}

func buildTableSchema(columns []mockResultColumn) *hiveserver.TTableSchema {
	schema := &hiveserver.TTableSchema{Columns: make([]*hiveserver.TColumnDesc, 0, len(columns))}
	for index, column := range columns {
		comment := column.Comment
		schema.Columns = append(schema.Columns, &hiveserver.TColumnDesc{
			ColumnName: column.Name,
			TypeDesc: &hiveserver.TTypeDesc{
				Types: []*hiveserver.TTypeEntry{{
					PrimitiveEntry: &hiveserver.TPrimitiveTypeEntry{Type: hiveserver.TTypeId_STRING_TYPE},
				}},
			},
			Position: int32(index + 1),
			Comment:  &comment,
		})
	}
	return schema
}

func buildRowSet(columns []mockResultColumn, rows []map[string]string) *hiveserver.TRowSet {
	columnValues := make([]*hiveserver.TColumn, 0, len(columns))
	for _, column := range columns {
		values := make([]string, 0, len(rows))
		for _, row := range rows {
			values = append(values, lookupRowValue(row, column.Name))
		}
		columnValues = append(columnValues, &hiveserver.TColumn{
			StringVal: &hiveserver.TStringColumn{Values: values, Nulls: []byte{}},
		})
	}
	columnCount := int32(len(columns))
	return &hiveserver.TRowSet{
		StartRowOffset: 0,
		Rows:           []*hiveserver.TRow{},
		Columns:        columnValues,
		ColumnCount:    &columnCount,
	}
}

func normalizeQuery(query string) string {
	normalized := strings.ReplaceAll(query, "`", "")
	normalized = strings.ReplaceAll(normalized, "\n", " ")
	normalized = strings.ReplaceAll(normalized, "\t", " ")
	return strings.Join(strings.Fields(strings.TrimSpace(normalized)), " ")
}

func normalizeIdentifier(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "`", ""))
	value = strings.Trim(value, "'")
	return strings.ToLower(value)
}

func matchesWhereClause(row map[string]string, where string) bool {
	where = strings.TrimSpace(where)
	if where == "" {
		return true
	}
	parts := strings.Split(strings.ToLower(where), " and ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasSuffix(part, " is not null"):
			column := normalizeIdentifier(strings.TrimSuffix(part, " is not null"))
			if strings.TrimSpace(lookupRowValue(row, column)) == "" {
				return false
			}
		case strings.Contains(part, ">="):
			pieces := strings.SplitN(part, ">=", 2)
			if lookupRowValue(row, normalizeIdentifier(pieces[0])) < strings.Trim(strings.TrimSpace(pieces[1]), "'") {
				return false
			}
		case strings.Contains(part, ">"):
			pieces := strings.SplitN(part, ">", 2)
			if lookupRowValue(row, normalizeIdentifier(pieces[0])) <= strings.Trim(strings.TrimSpace(pieces[1]), "'") {
				return false
			}
		case strings.Contains(part, "="):
			pieces := strings.SplitN(part, "=", 2)
			if lookupRowValue(row, normalizeIdentifier(pieces[0])) != strings.Trim(strings.TrimSpace(pieces[1]), "'") {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func lookupRowValue(row map[string]string, column string) string {
	for key, value := range row {
		if strings.EqualFold(key, column) {
			return value
		}
	}
	return ""
}

func successStatus() *hiveserver.TStatus {
	return &hiveserver.TStatus{StatusCode: hiveserver.TStatusCode_SUCCESS_STATUS}
}

func errorStatus(message string) *hiveserver.TStatus {
	return &hiveserver.TStatus{
		StatusCode:   hiveserver.TStatusCode_ERROR_STATUS,
		ErrorMessage: &message,
	}
}

func sessionHandle(id string) *hiveserver.TSessionHandle {
	return &hiveserver.TSessionHandle{SessionId: handleIdentifier(id)}
}

func handleIdentifier(id string) *hiveserver.THandleIdentifier {
	return &hiveserver.THandleIdentifier{
		GUID:   []byte(id),
		Secret: []byte("secret-" + id),
	}
}

func handleKey(identifier *hiveserver.THandleIdentifier) string {
	if identifier == nil {
		return ""
	}
	return string(identifier.GUID)
}

func newHandleID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func boolPtr(value bool) *bool {
	return &value
}

func nonZeroOr(value, fallback int64) int64 {
	if value != 0 {
		return value
	}
	return fallback
}

func toTypeID(nativeType string) hiveserver.TTypeId {
	switch strings.ToLower(strings.TrimSpace(nativeType)) {
	case "boolean":
		return hiveserver.TTypeId_BOOLEAN_TYPE
	case "tinyint":
		return hiveserver.TTypeId_TINYINT_TYPE
	case "smallint":
		return hiveserver.TTypeId_SMALLINT_TYPE
	case "int":
		return hiveserver.TTypeId_INT_TYPE
	case "bigint":
		return hiveserver.TTypeId_BIGINT_TYPE
	case "float":
		return hiveserver.TTypeId_FLOAT_TYPE
	case "double":
		return hiveserver.TTypeId_DOUBLE_TYPE
	case "timestamp":
		return hiveserver.TTypeId_TIMESTAMP_TYPE
	case "date":
		return hiveserver.TTypeId_DATE_TYPE
	default:
		return hiveserver.TTypeId_STRING_TYPE
	}
}

func SkipIfDockerUnavailable(t testing.TB) {
	t.Helper()
	tester, ok := t.(*testing.T)
	if !ok {
		return
	}
	tc.SkipIfProviderIsNotHealthy(tester)
}

func ReadExecOutput(reader io.Reader) string {
	if reader == nil {
		return ""
	}
	body, _ := io.ReadAll(reader)
	return string(body)
}
