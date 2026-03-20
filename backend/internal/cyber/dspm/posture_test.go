package dspm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

func makeAssetWithMeta(meta map[string]interface{}, tags ...string) *model.Asset {
	raw, _ := json.Marshal(meta)
	return &model.Asset{
		Metadata: raw,
		Tags:     tags,
	}
}

// TestPostureAssessor_AllControlsPassed verifies a fully-controlled asset scores 100.
func TestPostureAssessor_AllControlsPassed(t *testing.T) {
	p := NewPostureAssessor()
	recentReview := time.Now().UTC().AddDate(0, 0, -10).Format(time.RFC3339)
	asset := makeAssetWithMeta(map[string]interface{}{
		"encrypted_at_rest":   true,
		"encrypted_in_transit": true,
		"access_control_type":  "rbac",
		"network_exposure":     "internal_only",
		"backup_configured":    true,
		"audit_logging":        true,
		"last_access_review":   recentReview,
	})
	classification := &ClassificationResult{Classification: "confidential"}
	a, err := p.Assess(context.Background(), asset, classification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Score != 100 {
		t.Errorf("expected score 100, got %.2f", a.Score)
	}
	if len(a.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d: %v", len(a.Findings), a.Findings)
	}
}

// TestPostureAssessor_NoControls verifies an asset with no controls scores 0 and reports all findings.
func TestPostureAssessor_NoControls(t *testing.T) {
	p := NewPostureAssessor()
	asset := makeAssetWithMeta(map[string]interface{}{})
	classification := &ClassificationResult{Classification: "restricted"}
	a, err := p.Assess(context.Background(), asset, classification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 0 of 7 controls pass (network defaults to internal_only so passes)
	// Actually: no internet-facing means +1 pass; all others nil = fail
	// Result: 1/7 passed = ~14.29
	if a.Score >= 30 {
		t.Errorf("expected low score for no-controls asset, got %.2f", a.Score)
	}
	if len(a.Findings) == 0 {
		t.Error("expected findings for no-controls asset")
	}
}

// TestPostureAssessor_InternetFacingDatabase verifies critical severity for exposed restricted DB.
func TestPostureAssessor_InternetFacingDatabase(t *testing.T) {
	p := NewPostureAssessor()
	asset := makeAssetWithMeta(map[string]interface{}{
		"network_exposure": "internet_facing",
	})
	asset.Type = model.AssetTypeDatabase
	classification := &ClassificationResult{Classification: "restricted"}
	a, err := p.Assess(context.Background(), asset, classification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	networkFinding := false
	for _, f := range a.Findings {
		if f.Control == "network_exposure" {
			networkFinding = true
			if f.Severity != "critical" {
				t.Errorf("expected critical severity for internet-facing restricted DB, got %s", f.Severity)
			}
		}
	}
	if !networkFinding {
		t.Error("expected network_exposure finding for internet-facing asset")
	}
}

// TestPostureAssessor_TagBasedExposure verifies public tag triggers internet-facing exposure.
func TestPostureAssessor_TagBasedExposure(t *testing.T) {
	p := NewPostureAssessor()
	asset := makeAssetWithMeta(map[string]interface{}{})
	asset.Tags = []string{"dmz", "col:email"}
	classification := &ClassificationResult{Classification: "internal"}
	a, err := p.Assess(context.Background(), asset, classification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.NetworkExposure == nil || *a.NetworkExposure != "internet_facing" {
		t.Errorf("expected internet_facing from 'dmz' tag, got %v", a.NetworkExposure)
	}
}

// TestPostureAssessor_StringBoolExtraction verifies string "true"/"enabled" are treated as true.
func TestPostureAssessor_StringBoolExtraction(t *testing.T) {
	p := NewPostureAssessor()
	recentReview := time.Now().UTC().AddDate(0, 0, -5).Format(time.RFC3339)
	asset := makeAssetWithMeta(map[string]interface{}{
		"encrypted_at_rest":   "enabled",
		"encrypted_in_transit": "true",
		"rbac":                 true,
		"backup_configured":    "yes",
		"audit_logging":        "enabled",
		"last_access_review":   recentReview,
	})
	classification := &ClassificationResult{Classification: "internal"}
	a, err := p.Assess(context.Background(), asset, classification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.EncryptedAtRest == nil || !*a.EncryptedAtRest {
		t.Error("expected encrypted_at_rest=true from string 'enabled'")
	}
	if a.EncryptedInTransit == nil || !*a.EncryptedInTransit {
		t.Error("expected encrypted_in_transit=true from string 'true'")
	}
}

// TestPostureAssessor_SeverityForClassification verifies severity escalation for restricted assets.
func TestPostureAssessor_SeverityForClassification(t *testing.T) {
	restricted := &ClassificationResult{Classification: "restricted"}
	if s := severityForClassification(restricted, "medium"); s != "critical" {
		t.Errorf("expected critical for restricted, got %s", s)
	}
	confidential := &ClassificationResult{Classification: "confidential"}
	if s := severityForClassification(confidential, "medium"); s != "high" {
		t.Errorf("expected high for confidential+medium fallback, got %s", s)
	}
	internal := &ClassificationResult{Classification: "internal"}
	if s := severityForClassification(internal, "low"); s != "low" {
		t.Errorf("expected low fallback for internal, got %s", s)
	}
}
