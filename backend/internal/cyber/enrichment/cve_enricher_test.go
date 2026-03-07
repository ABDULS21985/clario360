package enrichment

import "testing"

func TestBuildCPEString(t *testing.T) {
	tests := []struct {
		name      string
		os        string
		version   string
		wantEmpty bool
		wantPart  string
	}{
		{name: "ubuntu", os: "linux", version: "Ubuntu 22.04", wantPart: "canonical:ubuntu_linux:22.04"},
		{name: "windows", os: "windows", version: "Server 2022", wantPart: "microsoft:windows_server:2022"},
		{name: "unknown", os: "obscureos", version: "1.0", wantEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCPEString(tt.os, tt.version)
			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("expected empty cpe, got %s", got)
				}
				return
			}
			if got == "" || !contains(got, tt.wantPart) {
				t.Fatalf("expected %q in %q", tt.wantPart, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}
