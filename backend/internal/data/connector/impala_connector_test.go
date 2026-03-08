package connector

import "testing"

func TestImpalaTypeMapping(t *testing.T) {
	tests := []struct {
		native string
		want   string
	}{
		{native: "INT", want: "integer"},
		{native: "DOUBLE", want: "float"},
		{native: "STRING", want: "string"},
		{native: "TIMESTAMP", want: "datetime"},
		{native: "BOOLEAN", want: "boolean"},
		{native: "DECIMAL(18,4)", want: "decimal"},
		{native: "ARRAY<STRING>", want: "array"},
		{native: "MAP<STRING,INT>", want: "json"},
		{native: "STRUCT<a:INT,b:STRING>", want: "json"},
	}

	for _, tt := range tests {
		got, _ := hiveLikeTypeMapping(tt.native)
		if got != tt.want {
			t.Fatalf("hiveLikeTypeMapping(%q) = %q, want %q", tt.native, got, tt.want)
		}
	}
}

func TestImpalaDescribeFormattedParsing(t *testing.T) {
	rows := []map[string]any{
		{"col_name": "# Detailed Table Information", "data_type": "", "comment": ""},
		{"col_name": "Location:", "data_type": "hdfs://namenode:8020/user/hive/warehouse/db.db/orders", "comment": ""},
		{"col_name": "numRows:", "data_type": "1250", "comment": ""},
		{"col_name": "rawDataSize:", "data_type": "8192", "comment": ""},
		{"col_name": "# Partition Information", "data_type": "", "comment": ""},
		{"col_name": "dt", "data_type": "string", "comment": ""},
		{"col_name": "# Storage Information", "data_type": "", "comment": ""},
		{"col_name": "InputFormat:", "data_type": "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat", "comment": ""},
	}

	meta := parseHiveDescribeFormatted(rows)
	if meta.Location != "hdfs://namenode:8020/user/hive/warehouse/db.db/orders" {
		t.Fatalf("Location = %q", meta.Location)
	}
	if meta.NumRows != 1250 {
		t.Fatalf("NumRows = %d", meta.NumRows)
	}
	if meta.RawDataSize != 8192 {
		t.Fatalf("RawDataSize = %d", meta.RawDataSize)
	}
	if len(meta.PartitionColumns) != 1 || meta.PartitionColumns[0] != "dt" {
		t.Fatalf("PartitionColumns = %v", meta.PartitionColumns)
	}
}
