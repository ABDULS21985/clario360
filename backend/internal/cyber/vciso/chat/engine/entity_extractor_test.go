package engine

import (
	"testing"
	"time"
)

func TestEntityExtractor_Extract(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 15, 30, 0, 0, time.UTC)
	extractor := NewEntityExtractor(func() time.Time { return now })

	tests := []struct {
		name    string
		message string
		intent  string
		check   func(t *testing.T, entities map[string]string)
	}{
		{
			name:    "extract uuid",
			message: "Investigate alert 550e8400-e29b-41d4-a716-446655440000",
			intent:  "investigation_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["alert_id"] != "550e8400-e29b-41d4-a716-446655440000" {
					t.Fatalf("alert_id = %q", entities["alert_id"])
				}
			},
		},
		{
			name:    "extract short uuid",
			message: "Tell me about alert 550e8400",
			intent:  "alert_detail",
			check: func(t *testing.T, entities map[string]string) {
				if entities["alert_id"] != "550e8400" {
					t.Fatalf("alert_id = %q", entities["alert_id"])
				}
			},
		},
		{
			name:    "extract hash id",
			message: "Investigate alert #1234",
			intent:  "investigation_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["alert_id"] != "#1234" {
					t.Fatalf("alert_id = %q", entities["alert_id"])
				}
			},
		},
		{
			name:    "extract hostname",
			message: "Show details for web-prod-01",
			intent:  "asset_lookup",
			check: func(t *testing.T, entities map[string]string) {
				if entities["asset_name"] != "web-prod-01" {
					t.Fatalf("asset_name = %q", entities["asset_name"])
				}
			},
		},
		{
			name:    "extract fqdn",
			message: "What do we know about db.corp.local?",
			intent:  "asset_lookup",
			check: func(t *testing.T, entities map[string]string) {
				if entities["asset_name"] != "db.corp.local" {
					t.Fatalf("asset_name = %q", entities["asset_name"])
				}
			},
		},
		{
			name:    "extract ipv4",
			message: "Show details for 10.0.1.50",
			intent:  "asset_lookup",
			check: func(t *testing.T, entities map[string]string) {
				if entities["asset_ip"] != "10.0.1.50" {
					t.Fatalf("asset_ip = %q", entities["asset_ip"])
				}
			},
		},
		{
			name:    "ignore invalid ipv4",
			message: "Show details for 999.999.999.999",
			intent:  "asset_lookup",
			check: func(t *testing.T, entities map[string]string) {
				if entities["asset_ip"] != "" {
					t.Fatalf("asset_ip = %q, want empty", entities["asset_ip"])
				}
			},
		},
		{
			name:    "time today",
			message: "Show critical alerts today",
			intent:  "alert_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["start_time"] != "2026-03-08T00:00:00Z" {
					t.Fatalf("start_time = %q", entities["start_time"])
				}
			},
		},
		{
			name:    "time last 14 days",
			message: "Show alerts from the last 14 days",
			intent:  "alert_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["start_time"] != "2026-02-22T15:30:00Z" {
					t.Fatalf("start_time = %q", entities["start_time"])
				}
			},
		},
		{
			name:    "severity multiple",
			message: "Show critical and high alerts",
			intent:  "alert_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["severity"] != "critical,high" {
					t.Fatalf("severity = %q", entities["severity"])
				}
			},
		},
		{
			name:    "count",
			message: "Show top 10 vulnerabilities",
			intent:  "vulnerability_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["count"] != "10" {
					t.Fatalf("count = %q", entities["count"])
				}
			},
		},
		{
			name:    "framework",
			message: "ISO 27001 compliance status",
			intent:  "compliance_query",
			check: func(t *testing.T, entities map[string]string) {
				if entities["framework"] != "iso27001" {
					t.Fatalf("framework = %q", entities["framework"])
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.check(t, extractor.Extract(tc.message, tc.intent))
		})
	}
}
