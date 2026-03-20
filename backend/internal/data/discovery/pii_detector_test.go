package discovery

import (
	"testing"

	"github.com/clario360/platform/internal/data/model"
)

func TestPIIDetection(t *testing.T) {
	tests := []struct {
		name           string
		column         model.DiscoveredColumn
		wantPII        bool
		wantType       string
		wantClass      model.DataClassification
	}{
		{"email", model.DiscoveredColumn{Name: "email_address"}, true, "email", model.DataClassificationConfidential},
		{"phone", model.DiscoveredColumn{Name: "phone_number"}, true, "phone", model.DataClassificationConfidential},
		{"ssn", model.DiscoveredColumn{Name: "ssn"}, true, "national_id", model.DataClassificationRestricted},
		{"bvn", model.DiscoveredColumn{Name: "bvn"}, true, "national_id", model.DataClassificationRestricted},
		{"salary", model.DiscoveredColumn{Name: "base_salary"}, true, "financial", model.DataClassificationRestricted},
		{"password", model.DiscoveredColumn{Name: "password_hash"}, true, "credential", model.DataClassificationRestricted},
		{"medical", model.DiscoveredColumn{Name: "diagnosis_code"}, true, "medical", model.DataClassificationRestricted},
		{"credit-card", model.DiscoveredColumn{Name: "card_number"}, true, "credit_card", model.DataClassificationRestricted},
		{"name", model.DiscoveredColumn{Name: "first_name"}, true, "name", model.DataClassificationConfidential},
		{"address", model.DiscoveredColumn{Name: "street_address"}, true, "address", model.DataClassificationConfidential},
		{"dob", model.DiscoveredColumn{Name: "date_of_birth"}, true, "dob", model.DataClassificationConfidential},
		{"gender", model.DiscoveredColumn{Name: "gender"}, true, "demographic", model.DataClassificationRestricted},
		{"product-id", model.DiscoveredColumn{Name: "product_id"}, false, "", model.DataClassificationPublic},
		{"created-at", model.DiscoveredColumn{Name: "created_at"}, false, "", model.DataClassificationPublic},
		{"total-amount", model.DiscoveredColumn{Name: "total_amount"}, false, "", model.DataClassificationPublic},
		{"sample-email", model.DiscoveredColumn{Name: "contact_info", SampleValues: []string{"user@example.com"}}, true, "email", model.DataClassificationConfidential},
		{"sample-card", model.DiscoveredColumn{Name: "reference", SampleValues: []string{"4111111111111111"}}, true, "credit_card", model.DataClassificationRestricted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPII([]model.DiscoveredColumn{tt.column})
			if got := result[0].InferredPII; got != tt.wantPII {
				t.Fatalf("InferredPII = %v, want %v", got, tt.wantPII)
			}
			if got := result[0].InferredPIIType; got != tt.wantType {
				t.Fatalf("InferredPIIType = %q, want %q", got, tt.wantType)
			}
			if got := result[0].InferredClass; got != tt.wantClass {
				t.Fatalf("InferredClass = %q, want %q", got, tt.wantClass)
			}
		})
	}
}

func TestPII_TableClassification(t *testing.T) {
	columns := DetectPII([]model.DiscoveredColumn{
		{Name: "user_email"},
		{Name: "phone_number"},
		{Name: "ssn"},
	})

	if got := TableClassification(columns); got != model.DataClassificationRestricted {
		t.Fatalf("TableClassification() = %q, want %q", got, model.DataClassificationRestricted)
	}
}

func TestPII_NoPII_Table(t *testing.T) {
	columns := DetectPII([]model.DiscoveredColumn{
		{Name: "product_id"},
		{Name: "created_at"},
		{Name: "sku"},
	})

	if got := TableClassification(columns); got != model.DataClassificationPublic {
		t.Fatalf("TableClassification() = %q, want %q", got, model.DataClassificationPublic)
	}
}
