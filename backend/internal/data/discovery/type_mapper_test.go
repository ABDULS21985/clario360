package discovery

import "testing"

func TestMapNativeType(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   string
		wantSubtype string
	}{
		{"varchar", "varchar(255)", "string", ""},
		{"int4", "int4", "integer", ""},
		{"numeric", "numeric(10,2)", "float", ""},
		{"bool", "boolean", "boolean", ""},
		{"timestamptz", "timestamptz", "datetime", ""},
		{"jsonb", "jsonb", "json", ""},
		{"uuid", "uuid", "string", "uuid"},
		{"unknown", "custom_type", "string", ""},
		{"tinyint1", "tinyint(1)", "boolean", ""},
		{"mysql-datetime", "datetime", "datetime", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapNativeType(tt.input)
			if got.Type != tt.wantType || got.Subtype != tt.wantSubtype {
				t.Fatalf("MapNativeType(%q) = %#v, want type=%q subtype=%q", tt.input, got, tt.wantType, tt.wantSubtype)
			}
		})
	}
}
