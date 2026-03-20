//go:build integration

package connector

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/clario360/platform/internal/data/connector/testhelpers"
)

func TestDolt_FullLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := testhelpers.StartDoltContainer(ctx, t, "app")
	db, err := container.OpenUserDB(ctx)
	if err != nil {
		t.Fatalf("OpenUserDB() error = %v", err)
	}
	defer db.Close()

	mustExecSQL(t, ctx, db, `CREATE TABLE customers (id INT PRIMARY KEY, name VARCHAR(100), user_email VARCHAR(255))`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'initial customer load')`)

	conn, err := NewDoltConnector(mustRawJSON(t, container.ConnectionConfig("main")), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewDoltConnector() error = %v", err)
	}
	doltConn := conn.(*DoltConnector)
	sourceID, tenantID := integrationSourceContext()
	doltConn.SetSourceContext(sourceID, tenantID)
	defer doltConn.Close()

	testResult, err := doltConn.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if !testResult.Success || testResult.Version == "" {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema, err := doltConn.DiscoverSchema(ctx, DiscoveryOptions{
		MaxTables:    20,
		MaxColumns:   20,
		SampleValues: true,
		MaxSamples:   2,
	})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	customers := requireDiscoveredTable(t, schema, "customers")
	if customers.PIIColumnCount == 0 {
		t.Fatalf("DiscoverSchema() did not flag user_email as pii: %+v", customers)
	}

	batch, err := doltConn.FetchData(ctx, "customers", FetchParams{
		OrderBy:   "id",
		BatchSize: 10,
	})
	if err != nil {
		t.Fatalf("FetchData() error = %v", err)
	}
	if batch.RowCount != 2 || fmt.Sprint(batch.Rows[0]["name"]) != "Alice" {
		t.Fatalf("FetchData() = %+v, want Alice and Bob rows", batch)
	}

	estimate, err := doltConn.EstimateSize(ctx)
	if err != nil {
		t.Fatalf("EstimateSize() error = %v", err)
	}
	if estimate.TableCount == 0 || estimate.TotalRows < 2 {
		t.Fatalf("EstimateSize() = %+v", estimate)
	}

	if err := doltConn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if doltConn.db != nil {
		t.Fatal("Close() did not clear db handle")
	}
}

func TestDolt_VersionHistory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := testhelpers.StartDoltContainer(ctx, t, "app")
	db, err := container.OpenUserDB(ctx)
	if err != nil {
		t.Fatalf("OpenUserDB() error = %v", err)
	}
	defer db.Close()

	since := time.Now().UTC()
	mustExecSQL(t, ctx, db, `CREATE TABLE customers (id INT PRIMARY KEY, name VARCHAR(100))`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (1, 'Alice')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'initial commit')`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (2, 'Bob')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'second commit')`)

	conn, err := NewDoltConnector(mustRawJSON(t, container.ConnectionConfig("main")), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewDoltConnector() error = %v", err)
	}
	doltConn := conn.(*DoltConnector)
	sourceID, tenantID := integrationSourceContext()
	doltConn.SetSourceContext(sourceID, tenantID)
	defer doltConn.Close()

	changes, err := doltConn.GetRecentChanges(ctx, since)
	if err != nil {
		t.Fatalf("GetRecentChanges() error = %v", err)
	}
	if len(changes) < 2 {
		t.Fatalf("GetRecentChanges() = %+v, want at least 2 change events", changes)
	}

	events, err := doltConn.QueryAccessLogs(ctx, since)
	if err != nil {
		t.Fatalf("QueryAccessLogs() error = %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("QueryAccessLogs() = %+v, want at least 2 commit events", events)
	}
	if events[0].Action != "commit" || events[0].SourceID != sourceID || events[0].TenantID != tenantID {
		t.Fatalf("first access event = %+v", events[0])
	}
}

func TestDolt_BranchAccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := testhelpers.StartDoltContainer(ctx, t, "app")
	db, err := container.OpenUserDB(ctx)
	if err != nil {
		t.Fatalf("OpenUserDB() error = %v", err)
	}
	defer db.Close()

	mustExecSQL(t, ctx, db, `CREATE TABLE customers (id INT PRIMARY KEY, name VARCHAR(100))`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (1, 'Alice')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'main baseline')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_BRANCH('staging')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_CHECKOUT('staging')`)
	mustExecSQL(t, ctx, db, `INSERT INTO customers VALUES (2, 'Bob')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_ADD('.')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_COMMIT('-am', 'staging change')`)
	mustCallDoltProcedure(t, ctx, db, `CALL DOLT_CHECKOUT('main')`)

	conn, err := NewDoltConnector(mustRawJSON(t, container.ConnectionConfig("main")), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewDoltConnector() error = %v", err)
	}
	doltConn := conn.(*DoltConnector)
	defer doltConn.Close()

	mainBatch, err := doltConn.FetchData(ctx, "customers", FetchParams{OrderBy: "id", BatchSize: 10})
	if err != nil {
		t.Fatalf("FetchData(main) error = %v", err)
	}
	if mainBatch.RowCount != 1 {
		t.Fatalf("main branch rows = %d, want 1", mainBatch.RowCount)
	}

	if err := doltConn.SwitchBranch(ctx, "staging"); err != nil {
		t.Fatalf("SwitchBranch(staging) error = %v", err)
	}
	stagingBatch, err := doltConn.FetchData(ctx, "customers", FetchParams{OrderBy: "id", BatchSize: 10})
	if err != nil {
		t.Fatalf("FetchData(staging) error = %v", err)
	}
	if stagingBatch.RowCount != 2 {
		t.Fatalf("staging branch rows = %d, want 2", stagingBatch.RowCount)
	}
}

func mustExecSQL(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("ExecContext(%q) error = %v", query, err)
	}
}

func mustCallDoltProcedure(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		t.Fatalf("QueryContext(%q) error = %v", query, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		t.Fatalf("rows.Columns(%q) error = %v", query, err)
	}
	values := make([]any, len(columns))
	targets := make([]any, len(columns))
	for i := range values {
		targets[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			t.Fatalf("rows.Scan(%q) error = %v", query, err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(%q) error = %v", query, err)
	}
}
