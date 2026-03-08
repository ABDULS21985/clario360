package engine

import "testing"

func TestIntentClassifier_Classify(t *testing.T) {
	t.Parallel()

	classifier := NewIntentClassifier()
	tests := []struct {
		name       string
		message    string
		intent     string
		method     string
		confidence float64
	}{
		{"risk regex 1", "What is our risk score?", "risk_score_query", "regex", 0.90},
		{"risk regex 2", "How safe are we?", "risk_score_query", "regex", 0.90},
		{"alert regex 1", "How many critical alerts do we have?", "alert_query", "regex", 0.90},
		{"alert regex 2", "Show open incidents", "alert_query", "regex", 0.90},
		{"alert detail 1", "Tell me about alert 550e8400-e29b-41d4-a716-446655440000", "alert_detail", "regex", 0.90},
		{"alert detail 2", "What happened with alert #1234?", "alert_detail", "regex", 0.90},
		{"asset 1", "Show details for web-prod-01", "asset_lookup", "regex", 0.90},
		{"asset 2", "What do we know about 10.0.1.50?", "asset_lookup", "regex", 0.90},
		{"vuln 1", "Top 10 critical vulnerabilities", "vulnerability_query", "regex", 0.90},
		{"vuln 2", "Show unpatched CVEs", "vulnerability_query", "regex", 0.90},
		{"mitre 1", "MITRE ATT&CK coverage", "mitre_query", "regex", 0.90},
		{"mitre 2", "What detection gaps do we have?", "mitre_query", "regex", 0.90},
		{"ueba 1", "Who are the riskiest users?", "ueba_query", "regex", 0.90},
		{"ueba 2", "Any suspicious activity?", "ueba_query", "regex", 0.90},
		{"pipeline 1", "Are any pipelines failing?", "pipeline_query", "regex", 0.90},
		{"pipeline 2", "Data health status", "pipeline_query", "regex", 0.90},
		{"compliance 1", "ISO 27001 compliance status", "compliance_query", "regex", 0.90},
		{"compliance 2", "Are we audit ready?", "compliance_query", "regex", 0.90},
		{"recommend 1", "What should I focus on today?", "recommendation_query", "regex", 0.90},
		{"recommend 2", "Top priorities", "recommendation_query", "regex", 0.90},
		{"dashboard 1", "Build me a security dashboard", "dashboard_build", "regex", 0.90},
		{"dashboard 2", "Create a view for alerts and risk", "dashboard_build", "regex", 0.90},
		{"investigation 1", "Investigate alert 550e8400-e29b-41d4-a716-446655440000", "investigation_query", "regex", 0.90},
		{"investigation 2", "Deep dive into alert #5678", "investigation_query", "regex", 0.90},
		{"trend 1", "How has risk changed this week?", "trend_query", "regex", 0.90},
		{"trend 2", "Alert volume trend", "trend_query", "regex", 0.90},
		{"remediation 1", "Start remediation for alert 550e8400-e29b-41d4-a716-446655440000", "remediation_query", "regex", 0.90},
		{"remediation 2", "Fix alert #1234", "remediation_query", "regex", 0.90},
		{"report 1", "Generate executive report", "report_query", "regex", 0.90},
		{"report 2", "Weekly security briefing", "report_query", "regex", 0.90},
		{"risk keyword", "risk overview", "risk_score_query", "keyword", 0.50},
		{"alert keyword", "alerts today", "alert_query", "keyword", 0.50},
		{"security keyword", "security status", "risk_score_query", "keyword", 0.50},
		{"unknown hello", "Hello", "unknown", "fallback", 0.0},
		{"unknown weather", "What's the weather?", "unknown", "fallback", 0.0},
		{"unknown empty", "", "unknown", "fallback", 0.0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := classifier.Classify(tc.message)
			if result.Intent != tc.intent {
				t.Fatalf("intent = %q, want %q", result.Intent, tc.intent)
			}
			if result.MatchMethod != tc.method {
				t.Fatalf("method = %q, want %q", result.MatchMethod, tc.method)
			}
			if result.Confidence < tc.confidence {
				t.Fatalf("confidence = %.2f, want >= %.2f", result.Confidence, tc.confidence)
			}
		})
	}
}

func TestIntentClassifier_PriorityOrdering(t *testing.T) {
	t.Parallel()

	classifier := NewIntentClassifier()

	result := classifier.Classify("Investigate alert #1234")
	if result.Intent != "investigation_query" {
		t.Fatalf("intent = %q, want investigation_query", result.Intent)
	}

	result = classifier.Classify("Investigate alert details")
	if result.Intent != "alert_detail" {
		t.Fatalf("intent = %q, want alert_detail", result.Intent)
	}
}
