package service

import (
	"testing"

	"github.com/clario360/platform/internal/data/model"
)

func TestDerive_GeneratesFields(t *testing.T) {
	table := model.DiscoveredTable{
		Name: "customer_master",
		Columns: []model.DiscoveredColumn{
			{Name: "id", MappedType: "integer", NativeType: "int4", Nullable: false, IsPrimaryKey: true, InferredClass: model.DataClassificationInternal},
			{Name: "email", MappedType: "string", NativeType: "varchar(255)", Nullable: false, InferredPIIType: "email", InferredClass: model.DataClassificationConfidential, SampleValues: []string{"a@example.com", "b@example.com"}},
			{Name: "status", MappedType: "string", NativeType: "varchar(20)", Nullable: false, InferredClass: model.DataClassificationInternal, SampleValues: []string{"active", "inactive", "pending"}},
			{Name: "created_at", MappedType: "datetime", NativeType: "timestamptz", Nullable: false, InferredClass: model.DataClassificationInternal},
		},
	}

	fields := deriveModelFields(table)
	if len(fields) != 4 {
		t.Fatalf("deriveModelFields() len = %d, want 4", len(fields))
	}
	if fields[1].PIIType != "email" {
		t.Fatalf("fields[1].PIIType = %q, want email", fields[1].PIIType)
	}

	rules := deriveValidationRules(fields)
	assertHasRule(t, rules, "not_null", "id")
	assertHasRule(t, rules, "unique", "id")
	assertHasRule(t, rules, "max_length", "email")
	assertHasRule(t, rules, "enum", "status")
	assertHasRule(t, rules, "format", "email")
	assertHasRule(t, rules, "not_future", "created_at")
}

func TestDerive_Classification(t *testing.T) {
	fields := deriveModelFields(model.DiscoveredTable{
		Columns: []model.DiscoveredColumn{
			{Name: "employee_id", MappedType: "integer", NativeType: "int4", InferredClass: model.DataClassificationInternal},
			{Name: "ssn", MappedType: "string", NativeType: "varchar(20)", InferredPIIType: "national_id", InferredClass: model.DataClassificationRestricted},
		},
	})

	classification := model.DataClassificationPublic
	piiColumns := make([]string, 0)
	for _, field := range fields {
		classification = maxFieldClassification(classification, field.Classification)
		if field.PIIType != "" {
			piiColumns = append(piiColumns, field.Name)
		}
	}

	if classification != model.DataClassificationRestricted {
		t.Fatalf("classification = %q, want %q", classification, model.DataClassificationRestricted)
	}
	if len(piiColumns) != 1 || piiColumns[0] != "ssn" {
		t.Fatalf("piiColumns = %#v, want [\"ssn\"]", piiColumns)
	}
}

func assertHasRule(t *testing.T, rules []model.ValidationRule, ruleType, field string) {
	t.Helper()
	for _, rule := range rules {
		if rule.Type == ruleType && rule.Field == field {
			return
		}
	}
	t.Fatalf("missing rule type=%q field=%q in %#v", ruleType, field, rules)
}
