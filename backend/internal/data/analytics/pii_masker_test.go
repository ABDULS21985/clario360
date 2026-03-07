package analytics

import (
	"testing"

	"github.com/clario360/platform/internal/data/model"
)

func TestMaskPIIColumns(t *testing.T) {
	dataModel := &model.DataModel{
		SchemaDefinition: []model.ModelField{
			{Name: "email", PIIType: "email"},
			{Name: "phone", PIIType: "phone"},
			{Name: "full_name", PIIType: "name"},
			{Name: "ssn", PIIType: "national_id"},
			{Name: "salary", PIIType: "financial"},
		},
	}
	rows := []map[string]any{{
		"email":     "john@example.com",
		"phone":     "+1-555-123-4567",
		"full_name": "John Doe",
		"ssn":       "123-45-6789",
		"salary":    85000,
	}}
	masked, result := ApplyPIIMasking(rows, dataModel, false)
	if masked[0]["salary"] != "***" {
		t.Fatalf("salary = %v, want ***", masked[0]["salary"])
	}
	if masked[0]["ssn"] != "***-**-6789" {
		t.Fatalf("ssn = %v, want masked last four", masked[0]["ssn"])
	}
	if result.TotalMasked != 5 {
		t.Fatalf("TotalMasked = %d, want 5", result.TotalMasked)
	}
}

func TestMaskPIIAuthorizedUser(t *testing.T) {
	dataModel := &model.DataModel{
		SchemaDefinition: []model.ModelField{{Name: "email", PIIType: "email"}},
	}
	rows := []map[string]any{{"email": "john@example.com"}}
	masked, result := ApplyPIIMasking(rows, dataModel, true)
	if masked[0]["email"] != "john@example.com" {
		t.Fatalf("authorized email = %v, want unmasked", masked[0]["email"])
	}
	if result.TotalMasked != 0 {
		t.Fatalf("TotalMasked = %d, want 0", result.TotalMasked)
	}
}
