package analytics

import (
	"testing"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/data/model"
)

func TestValidateQueryRestrictedDenied(t *testing.T) {
	dataModel := &model.DataModel{
		Name:               "employees",
		Status:             model.DataModelStatusActive,
		DataClassification: model.DataClassificationRestricted,
		SchemaDefinition:   []model.ModelField{{Name: "email", PIIType: "email"}},
	}
	query := &model.AnalyticsQuery{Columns: []string{"email"}}
	if _, err := AnalyzeQuery(query, dataModel, []string{auth.PermDataRead}, false); err == nil {
		t.Fatalf("AnalyzeQuery() error = nil, want permission error")
	}
}

func TestValidateQueryUnknownColumn(t *testing.T) {
	dataModel := &model.DataModel{
		Name:               "customers",
		Status:             model.DataModelStatusActive,
		DataClassification: model.DataClassificationInternal,
		SchemaDefinition:   []model.ModelField{{Name: "customer_id"}},
	}
	query := &model.AnalyticsQuery{Columns: []string{"missing"}}
	if _, err := AnalyzeQuery(query, dataModel, []string{auth.PermDataRead}, false); err == nil {
		t.Fatalf("AnalyzeQuery() error = nil, want unknown-column error")
	}
}

func TestValidateQueryPIITracked(t *testing.T) {
	dataModel := &model.DataModel{
		Name:               "customers",
		Status:             model.DataModelStatusActive,
		DataClassification: model.DataClassificationConfidential,
		SchemaDefinition: []model.ModelField{
			{Name: "email", PIIType: "email"},
			{Name: "customer_id"},
		},
	}
	query := &model.AnalyticsQuery{Columns: []string{"customer_id", "email"}}
	result, err := AnalyzeQuery(query, dataModel, []string{auth.PermDataConfidential}, false)
	if err != nil {
		t.Fatalf("AnalyzeQuery() unexpected error = %v", err)
	}
	if len(result.PIIColumnsAccessed) != 1 || result.PIIColumnsAccessed[0] != "email" {
		t.Fatalf("PIIColumnsAccessed = %#v, want [email]", result.PIIColumnsAccessed)
	}
}
