//go:build integration

package connector

import (
	"testing"
	"time"

	"github.com/clario360/platform/internal/data/connector/testhelpers"
)

func mockWarehouseCatalog() testhelpers.MockHiveCatalog {
	ddlTime := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	return testhelpers.MockHiveCatalog{
		DefaultDatabase: "default",
		Databases: map[string]*testhelpers.MockHiveDatabase{
			"default": {
				Name: "default",
				Tables: map[string]*testhelpers.MockHiveTable{
					"customers": {
						Name: "customers",
						Columns: []testhelpers.MockHiveColumn{
							{Name: "id", Type: "int"},
							{Name: "user_email", Type: "string"},
							{Name: "region", Type: "string"},
							{Name: "event_time", Type: "timestamp"},
						},
						Rows: []map[string]string{
							{"id": "1", "user_email": "alice@example.com", "region": "EMEA", "event_time": "2026-03-08T10:00:00Z"},
							{"id": "2", "user_email": "bob@example.com", "region": "NA", "event_time": "2026-03-08T10:05:00Z"},
						},
						Location:         "hdfs://namenode:8020/user/hive/warehouse/default.db/customers",
						InputFormat:      "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat",
						NumRows:          2,
						RawDataSize:      4096,
						PartitionColumns: []string{"region"},
						LastDDLTime:      ddlTime,
					},
				},
			},
			"sys": {
				Name: "sys",
				Tables: map[string]*testhelpers.MockHiveTable{
					"impala_audit": {
						Name: "impala_audit",
						Columns: []testhelpers.MockHiveColumn{
							{Name: "user_name", Type: "string"},
							{Name: "source_ip", Type: "string"},
							{Name: "action", Type: "string"},
							{Name: "database_name", Type: "string"},
							{Name: "table_name", Type: "string"},
							{Name: "statement", Type: "string"},
							{Name: "event_time", Type: "string"},
							{Name: "rows_read", Type: "bigint"},
							{Name: "rows_written", Type: "bigint"},
							{Name: "bytes_read", Type: "bigint"},
							{Name: "bytes_written", Type: "bigint"},
							{Name: "duration_ms", Type: "bigint"},
							{Name: "success", Type: "string"},
							{Name: "error_message", Type: "string"},
						},
						Rows: []map[string]string{
							{
								"user_name":     "bi-analyst",
								"source_ip":     "10.10.10.20",
								"action":        "query",
								"database_name": "default",
								"table_name":    "customers",
								"statement":     "SELECT id, user_email FROM default.customers",
								"event_time":    "2026-03-08T10:15:00Z",
								"rows_read":     "2",
								"rows_written":  "0",
								"bytes_read":    "4096",
								"bytes_written": "0",
								"duration_ms":   "25",
								"success":       "true",
								"error_message": "",
							},
						},
						Location:    "hdfs://namenode:8020/user/hive/warehouse/sys.db/impala_audit",
						InputFormat: "org.apache.hadoop.mapred.TextInputFormat",
						NumRows:     1,
						RawDataSize: 2048,
						LastDDLTime: ddlTime,
					},
				},
			},
		},
	}
}

func startMockWarehouseThriftServer(t testing.TB) *testhelpers.MockThriftServer {
	t.Helper()
	return testhelpers.NewMockThriftServer(t, mockWarehouseCatalog())
}
