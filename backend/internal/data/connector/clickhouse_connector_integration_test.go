//go:build integration

package connector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/clario360/platform/internal/data/connector/testhelpers"
)

func TestClickHouse_FullLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := testhelpers.StartClickHouseContainer(ctx, t, "analytics")
	db, err := container.OpenDB(ctx, "")
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	mustExecClickHouse(t, ctx, db, `CREATE TABLE analytics.customers (
		id UInt64,
		user_email String,
		phone_number String,
		event_count UInt32,
		created_at DateTime
	) ENGINE = MergeTree ORDER BY id`)
	mustExecClickHouse(t, ctx, db, `CREATE TABLE analytics.audit_events (
		id UInt64,
		event_name String,
		created_at DateTime
	) ENGINE = MergeTree ORDER BY id`)
	mustExecClickHouse(t, ctx, db, `INSERT INTO analytics.customers (id, user_email, phone_number, event_count, created_at) VALUES
		(1, 'alice@example.com', '555-0101', 3, now()),
		(2, 'bob@example.com', '555-0102', 8, now())`)
	mustExecClickHouse(t, ctx, db, `INSERT INTO analytics.audit_events (id, event_name, created_at) VALUES
		(1, 'login', now()),
		(2, 'logout', now())`)

	conn, err := NewClickHouseConnector(mustRawJSON(t, container.NativeConfig("")), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewClickHouseConnector() error = %v", err)
	}
	clickhouseConn := conn.(*ClickHouseConnector)
	sourceID, tenantID := integrationSourceContext()
	clickhouseConn.SetSourceContext(sourceID, tenantID)
	defer clickhouseConn.Close()

	testResult, err := clickhouseConn.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
	if !testResult.Success || !strings.Contains(testResult.Version, "ClickHouse") {
		t.Fatalf("TestConnection() = %+v", testResult)
	}

	schema, err := clickhouseConn.DiscoverSchema(ctx, DiscoveryOptions{
		MaxTables:    20,
		MaxColumns:   20,
		SampleValues: true,
		MaxSamples:   2,
	})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	customers := requireDiscoveredTable(t, schema, "customers")
	if !customers.ContainsPII || customers.PIIColumnCount < 2 {
		t.Fatalf("customers discovery missing pii metadata: %+v", customers)
	}
	emailColumn := requireDiscoveredColumn(t, customers, "user_email")
	if !emailColumn.InferredPII || emailColumn.InferredPIIType != "email" {
		t.Fatalf("user_email pii inference = %+v", emailColumn)
	}

	batch, err := clickhouseConn.FetchData(ctx, "customers", FetchParams{
		Columns:   []string{"id", "user_email"},
		OrderBy:   "id",
		BatchSize: 2,
	})
	if err != nil {
		t.Fatalf("FetchData() error = %v", err)
	}
	if batch.RowCount != 2 || len(batch.Rows) != 2 {
		t.Fatalf("FetchData() = %+v, want 2 rows", batch)
	}
	if fmt.Sprint(batch.Rows[0]["user_email"]) != "alice@example.com" {
		t.Fatalf("first fetched row = %+v, want alice@example.com", batch.Rows[0])
	}

	estimate, err := clickhouseConn.EstimateSize(ctx)
	if err != nil {
		t.Fatalf("EstimateSize() error = %v", err)
	}
	if estimate.TableCount < 2 || estimate.TotalRows < 4 {
		t.Fatalf("EstimateSize() = %+v, want at least 2 tables / 4 rows", estimate)
	}

	if _, err = clickhouseConn.ReadQuery(ctx, "SELECT id, user_email FROM analytics.customers ORDER BY id LIMIT 1", nil); err != nil {
		t.Fatalf("ReadQuery() error = %v", err)
	}
	mustExecClickHouse(t, ctx, db, "SYSTEM FLUSH LOGS")

	waitForCondition(t, 20*time.Second, func(waitCtx context.Context) (bool, error) {
		events, accessErr := clickhouseConn.QueryAccessLogs(waitCtx, time.Now().Add(-10*time.Minute))
		if accessErr != nil {
			return false, accessErr
		}
		for _, event := range events {
			if event.SourceID == sourceID && event.TenantID == tenantID &&
				(event.Table == "customers" || strings.Contains(strings.ToLower(event.QueryPreview), "from analytics.customers")) {
				return true, nil
			}
		}
		return false, nil
	})

	locations, err := clickhouseConn.ListDataLocations(ctx)
	if err != nil {
		t.Fatalf("ListDataLocations() error = %v", err)
	}
	if len(locations) < 2 {
		t.Fatalf("ListDataLocations() = %+v, want at least 2 tables", locations)
	}
	if !strings.HasPrefix(locations[0].Location, "clickhouse://") {
		t.Fatalf("ListDataLocations()[0] = %+v, want clickhouse:// location", locations[0])
	}

	if err := clickhouseConn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if clickhouseConn.db != nil {
		t.Fatal("Close() did not clear db handle")
	}
}

func TestClickHouse_PIIDetection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container := testhelpers.StartClickHouseContainer(ctx, t, "analytics")
	db, err := container.OpenDB(ctx, "")
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	mustExecClickHouse(t, ctx, db, `CREATE TABLE analytics.pii_probe (
		id UInt64,
		user_email String,
		phone_number String,
		ssn String,
		event_id UInt64,
		event_time DateTime
	) ENGINE = MergeTree ORDER BY id`)

	conn, err := NewClickHouseConnector(mustRawJSON(t, container.NativeConfig("")), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewClickHouseConnector() error = %v", err)
	}
	clickhouseConn := conn.(*ClickHouseConnector)
	defer clickhouseConn.Close()

	schema, err := clickhouseConn.DiscoverSchema(ctx, DiscoveryOptions{MaxTables: 10, MaxColumns: 20})
	if err != nil {
		t.Fatalf("DiscoverSchema() error = %v", err)
	}
	table := requireDiscoveredTable(t, schema, "pii_probe")
	for _, columnName := range []string{"user_email", "phone_number", "ssn"} {
		column := requireDiscoveredColumn(t, table, columnName)
		if !column.InferredPII {
			t.Fatalf("column %s was not flagged as pii: %+v", columnName, column)
		}
	}
}

func TestClickHouse_LargeTable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	container := testhelpers.StartClickHouseContainer(ctx, t, "analytics")
	db, err := container.OpenDB(ctx, "")
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	mustExecClickHouse(t, ctx, db, `CREATE TABLE analytics.large_events (
		id UInt64,
		user_email String
	) ENGINE = MergeTree ORDER BY id`)
	mustExecClickHouse(t, ctx, db, `
		INSERT INTO analytics.large_events (id, user_email)
		SELECT number + 1, concat('user', toString(number), '@example.com')
		FROM numbers(100000)`)

	conn, err := NewClickHouseConnector(mustRawJSON(t, container.NativeConfig("")), integrationFactoryOptions())
	if err != nil {
		t.Fatalf("NewClickHouseConnector() error = %v", err)
	}
	clickhouseConn := conn.(*ClickHouseConnector)
	defer clickhouseConn.Close()

	total := 0
	offset := int64(0)
	for batchNum := 0; batchNum < 12; batchNum++ {
		batch, fetchErr := clickhouseConn.FetchData(ctx, "large_events", FetchParams{
			OrderBy:   "id",
			BatchSize: 10000,
			Offset:    offset,
		})
		if fetchErr != nil {
			t.Fatalf("FetchData(offset=%d) error = %v", offset, fetchErr)
		}
		total += batch.RowCount
		if !batch.HasMore {
			break
		}
		offset += int64(batch.RowCount)
	}
	if total != 100000 {
		t.Fatalf("retrieved rows = %d, want 100000", total)
	}
}

func mustExecClickHouse(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("ExecContext(%q) error = %v", query, err)
	}
}
