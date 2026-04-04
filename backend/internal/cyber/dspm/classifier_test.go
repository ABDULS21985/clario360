package dspm

import (
	"encoding/json"
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
)

func makeAsset(metaJSON string, tags ...string) *model.Asset {
	a := &model.Asset{Tags: tags}
	if metaJSON != "" {
		a.Metadata = json.RawMessage(metaJSON)
	}
	return a
}

// TestClassify_Restricted verifies that SSN/bank columns produce restricted classification.
func TestClassify_Restricted(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset(`{"columns":["user_id","ssn","credit_card_number","first_name"]}`)
	r := c.Classify(asset)
	if r.Classification != "restricted" {
		t.Errorf("expected restricted, got %s", r.Classification)
	}
	if !r.ContainsPII {
		t.Error("expected ContainsPII=true")
	}
	if r.SensitivityScore != 90 {
		t.Errorf("expected sensitivity 90, got %.1f", r.SensitivityScore)
	}
}

// TestClassify_Confidential_Salary verifies salary columns produce confidential.
func TestClassify_Confidential_Salary(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset(`{"columns":["employee_id","salary","wage"]}`)
	r := c.Classify(asset)
	if r.Classification != "confidential" {
		t.Errorf("expected confidential, got %s", r.Classification)
	}
	if r.SensitivityScore != 70 {
		t.Errorf("expected sensitivity 70, got %.1f", r.SensitivityScore)
	}
}

// TestClassify_Confidential_Name verifies name columns alone produce confidential.
func TestClassify_Confidential_Name(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset(`{"columns":["first_name","last_name","email"]}`)
	r := c.Classify(asset)
	if r.Classification != "confidential" {
		t.Errorf("expected confidential, got %s", r.Classification)
	}
}

// TestClassify_Public verifies public-tagged asset gets public classification.
func TestClassify_Public(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset(`{"columns":["product_id","name","price"]}`, "public")
	r := c.Classify(asset)
	if r.Classification != "public" {
		t.Errorf("expected public, got %s", r.Classification)
	}
	if r.SensitivityScore != 10 {
		t.Errorf("expected sensitivity 10, got %.1f", r.SensitivityScore)
	}
	if r.ContainsPII {
		t.Error("expected ContainsPII=false for public asset with no PII columns")
	}
}

// TestClassify_Internal_NoColumns verifies assets with no schema return internal default.
func TestClassify_Internal_NoColumns(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset("")
	r := c.Classify(asset)
	if r.Classification != "internal" {
		t.Errorf("expected internal, got %s", r.Classification)
	}
	if r.SensitivityScore != 40 {
		t.Errorf("expected sensitivity 40, got %.1f", r.SensitivityScore)
	}
}

// TestClassify_TagColumns verifies col: tag prefix is used for column detection.
func TestClassify_TagColumns(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset("", "col:bank_account", "col:iban")
	r := c.Classify(asset)
	if r.Classification != "restricted" {
		t.Errorf("expected restricted via tag columns, got %s", r.Classification)
	}
}

// TestClassify_SchemaInfo verifies schema_info.tables path is parsed.
func TestClassify_SchemaInfo(t *testing.T) {
	c := NewDSPMClassifier()
	meta := map[string]interface{}{
		"schema_info": map[string]interface{}{
			"tables": []interface{}{
				map[string]interface{}{
					"columns": []interface{}{"diagnosis", "treatment"},
				},
			},
		},
	}
	raw, _ := json.Marshal(meta)
	asset := makeAsset(string(raw))
	r := c.Classify(asset)
	if r.Classification != "restricted" {
		t.Errorf("expected restricted for medical columns, got %s", r.Classification)
	}
}

// TestClassify_PIITypesDeduplication verifies same PII type isn't counted twice.
func TestClassify_PIITypesDeduplication(t *testing.T) {
	c := NewDSPMClassifier()
	asset := makeAsset(`{"columns":["ssn","social_security","national_id"]}`)
	r := c.Classify(asset)
	// All three match the "ssn" pattern — should only count once.
	count := 0
	for _, pii := range r.PIITypes {
		if pii == "ssn" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected ssn deduplicated to 1 entry, got %d", count)
	}
}
