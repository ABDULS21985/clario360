package connector

import (
	"testing"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

func TestClickHouseTypeMapping(t *testing.T) {
	tests := []struct {
		name         string
		native       string
		wantMapped   string
		wantSubtype  string
		wantNullable bool
	}{
		{name: "UInt8", native: "UInt8", wantMapped: "integer"},
		{name: "Float64", native: "Float64", wantMapped: "float"},
		{name: "String", native: "String", wantMapped: "string"},
		{name: "DateTime64", native: "DateTime64(3)", wantMapped: "datetime"},
		{name: "UUID", native: "UUID", wantMapped: "string", wantSubtype: "uuid"},
		{name: "Nullable", native: "Nullable(String)", wantMapped: "string", wantNullable: true},
		{name: "LowCardinality", native: "LowCardinality(String)", wantMapped: "string"},
		{name: "NestedNullable", native: "Nullable(LowCardinality(String))", wantMapped: "string", wantNullable: true},
		{name: "Array", native: "Array(Int32)", wantMapped: "array"},
		{name: "Map", native: "Map(String, Int32)", wantMapped: "json"},
		{name: "Enum8", native: "Enum8('a'=1,'b'=2)", wantMapped: "string", wantSubtype: "enum"},
		{name: "Decimal", native: "Decimal(10,2)", wantMapped: "decimal"},
		{name: "IPv4", native: "IPv4", wantMapped: "string", wantSubtype: "ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMapped, gotSubtype, gotNullable := clickHouseTypeMapping(tt.native)
			if gotMapped != tt.wantMapped || gotSubtype != tt.wantSubtype || gotNullable != tt.wantNullable {
				t.Fatalf("clickHouseTypeMapping(%q) = (%q, %q, %v), want (%q, %q, %v)", tt.native, gotMapped, gotSubtype, gotNullable, tt.wantMapped, tt.wantSubtype, tt.wantNullable)
			}
		})
	}
}

func TestClickHousePIIDetection(t *testing.T) {
	columns := discovery.DetectPII([]model.DiscoveredColumn{
		{Name: "user_email"},
		{Name: "phone_number"},
		{Name: "event_count"},
	})

	if !columns[0].InferredPII || columns[0].InferredPIIType != "email" {
		t.Fatalf("user_email inference = %+v, want email pii", columns[0])
	}
	if !columns[1].InferredPII || columns[1].InferredPIIType != "phone" {
		t.Fatalf("phone_number inference = %+v, want phone pii", columns[1])
	}
	if columns[2].InferredPII {
		t.Fatalf("event_count inference = %+v, want non-pii", columns[2])
	}
}
