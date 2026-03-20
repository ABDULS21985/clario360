package connector

import "testing"

func TestHiveTypeMapping(t *testing.T) {
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

func TestHiveInputFormatToFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat", want: "parquet"},
		{input: "org.apache.hadoop.hive.ql.io.orc.OrcInputFormat", want: "orc"},
		{input: "org.apache.hadoop.mapred.TextInputFormat", want: "text"},
		{input: "org.apache.hadoop.hive.serde2.avro.AvroContainerInputFormat", want: "avro"},
	}

	for _, tt := range tests {
		if got := inputFormatToFormat(tt.input); got != tt.want {
			t.Fatalf("inputFormatToFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHiveDataLocationExtraction(t *testing.T) {
	rows := []map[string]any{
		{"col_name": "Location:", "data_type": "hdfs://namenode:8020/user/hive/warehouse/sales.db/customers", "comment": ""},
		{"col_name": "InputFormat:", "data_type": "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat", "comment": ""},
	}
	meta := parseHiveDescribeFormatted(rows)
	if meta.Location != "hdfs://namenode:8020/user/hive/warehouse/sales.db/customers" {
		t.Fatalf("Location = %q", meta.Location)
	}
	if got := inputFormatToFormat(meta.InputFormat); got != "parquet" {
		t.Fatalf("inputFormatToFormat(meta.InputFormat) = %q, want parquet", got)
	}
}
