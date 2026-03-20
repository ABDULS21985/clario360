package financial

import (
	"math"
	"strings"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// FineSchedules defines the regulatory fine structure for each supported
// compliance framework based on published regulatory guidelines.
var FineSchedules = map[model.ComplianceFramework]model.RegulatoryFineSchedule{
	model.FrameworkGDPR: {
		Framework:       "gdpr",
		MaxFine:         20_000_000, // EUR 20M
		MaxFineCurrency: "EUR",
		PerViolationMin: 0,
		PerViolationMax: 20_000_000,
		RevenuePercent:  4.0, // 4% of annual global turnover
		Description:     "GDPR Art. 83: Up to EUR 20M or 4% of annual worldwide turnover, whichever is higher",
	},
	model.FrameworkHIPAA: {
		Framework:       "hipaa",
		MaxFine:         2_130_000, // $2.13M per year per violation category
		MaxFineCurrency: "USD",
		PerViolationMin: 50_000,
		PerViolationMax: 1_500_000,
		RevenuePercent:  0,
		Description:     "HIPAA: $50K-$1.5M per violation category, max $2.13M per year per identical provision",
	},
	model.FrameworkPCIDSS: {
		Framework:       "pci_dss",
		MaxFine:         100_000, // per month until remediated
		MaxFineCurrency: "USD",
		PerViolationMin: 5_000,
		PerViolationMax: 100_000,
		RevenuePercent:  0,
		Description:     "PCI DSS: $5K-$100K per month of non-compliance, assessed by card brands through acquiring banks",
	},
	model.FrameworkSaudiPDPL: {
		Framework:       "saudi_pdpl",
		MaxFine:         1_330_000, // SAR 5M ~ $1.33M
		MaxFineCurrency: "SAR",
		PerViolationMin: 0,
		PerViolationMax: 1_330_000,
		RevenuePercent:  0,
		Description:     "Saudi PDPL: Up to SAR 5M (~$1.33M) per violation, with potential imprisonment for certain offenses",
	},
	model.FrameworkSOC2: {
		Framework:       "soc2",
		MaxFine:         100_000, // estimated business impact
		MaxFineCurrency: "USD",
		PerViolationMin: 0,
		PerViolationMax: 100_000,
		RevenuePercent:  0,
		Description:     "SOC 2: No direct regulatory fines; estimated business impact from lost contracts and trust ($100K+)",
	},
	model.FrameworkISO27001: {
		Framework:       "iso27001",
		MaxFine:         250_000, // estimated certification loss impact
		MaxFineCurrency: "USD",
		PerViolationMin: 0,
		PerViolationMax: 250_000,
		RevenuePercent:  0,
		Description:     "ISO 27001: No direct fines; estimated impact from certification loss including contract penalties (~$250K)",
	},
}

// GetFineSchedule returns the fine schedule for a given compliance framework.
// Returns a zero-valued schedule if the framework is not recognized.
func GetFineSchedule(framework model.ComplianceFramework) model.RegulatoryFineSchedule {
	if schedule, ok := FineSchedules[framework]; ok {
		return schedule
	}
	return model.RegulatoryFineSchedule{
		Framework:       string(framework),
		MaxFineCurrency: "USD",
		Description:     "Unknown framework; no fine schedule available",
	}
}

// MaxFineForAsset calculates the maximum potential regulatory fine for an asset
// across all applicable frameworks. For revenue-based fines (e.g., GDPR's 4%),
// the revenue percentage is compared against the fixed maximum and the higher
// value is used.
func MaxFineForAsset(asset *cybermodel.DSPMDataAsset, annualRevenue float64) float64 {
	frameworks := applicableFrameworks(asset)
	if len(frameworks) == 0 {
		return 0
	}

	var maxFine float64
	for _, fw := range frameworks {
		schedule, ok := FineSchedules[fw]
		if !ok {
			continue
		}

		fine := schedule.MaxFine

		// For frameworks with revenue-based fines, calculate the percentage
		// and use the higher of fixed or revenue-based fine.
		if schedule.RevenuePercent > 0 && annualRevenue > 0 {
			revenueFine := annualRevenue * (schedule.RevenuePercent / 100.0)
			fine = math.Max(fine, revenueFine)
		}

		maxFine = math.Max(maxFine, fine)
	}

	return maxFine
}

// applicableFrameworks determines which compliance frameworks apply to an asset
// based on its classification, PII status, and metadata.
func applicableFrameworks(asset *cybermodel.DSPMDataAsset) []model.ComplianceFramework {
	var frameworks []model.ComplianceFramework

	// GDPR applies to assets containing PII.
	if asset.ContainsPII {
		frameworks = append(frameworks, model.FrameworkGDPR)
	}

	// HIPAA applies to healthcare data.
	if isHealthcareData(asset) {
		frameworks = append(frameworks, model.FrameworkHIPAA)
	}

	// PCI DSS applies to payment/cardholder data.
	if isPaymentData(asset) {
		frameworks = append(frameworks, model.FrameworkPCIDSS)
	}

	// Saudi PDPL applies to assets in Saudi Arabia or with Saudi data subjects.
	if isSaudiData(asset) {
		frameworks = append(frameworks, model.FrameworkSaudiPDPL)
	}

	// SOC 2 and ISO 27001 apply broadly to all managed data.
	if asset.DataClassification != "public" {
		frameworks = append(frameworks, model.FrameworkSOC2)
		frameworks = append(frameworks, model.FrameworkISO27001)
	}

	return frameworks
}

// isHealthcareData checks if asset contains healthcare-related data.
func isHealthcareData(asset *cybermodel.DSPMDataAsset) bool {
	for _, pii := range asset.PIITypes {
		lower := strings.ToLower(pii)
		if lower == "health_record" || lower == "medical_record" ||
			lower == "phi" || lower == "health_data" || lower == "hipaa" ||
			lower == "patient_id" || lower == "diagnosis" || lower == "prescription" {
			return true
		}
	}
	if asset.Metadata != nil {
		if industry, ok := asset.Metadata["industry"]; ok {
			if str, isStr := industry.(string); isStr && strings.ToLower(str) == "healthcare" {
				return true
			}
		}
		if regulation, ok := asset.Metadata["regulation"]; ok {
			if str, isStr := regulation.(string); isStr && strings.ToLower(str) == "hipaa" {
				return true
			}
		}
	}
	return false
}

// isPaymentData checks if asset contains payment/cardholder data.
func isPaymentData(asset *cybermodel.DSPMDataAsset) bool {
	for _, pii := range asset.PIITypes {
		lower := strings.ToLower(pii)
		if lower == "credit_card" || lower == "payment_card" ||
			lower == "cardholder" || lower == "pan" || lower == "card_number" ||
			lower == "cvv" || lower == "card_expiry" {
			return true
		}
	}
	if asset.Metadata != nil {
		if regulation, ok := asset.Metadata["regulation"]; ok {
			if str, isStr := regulation.(string); isStr && strings.ToLower(str) == "pci_dss" {
				return true
			}
		}
	}
	return false
}

// isSaudiData checks if the asset is subject to Saudi PDPL.
func isSaudiData(asset *cybermodel.DSPMDataAsset) bool {
	if asset.Metadata == nil {
		return false
	}
	for _, key := range []string{"region", "location", "cloud_region", "country"} {
		if val, ok := asset.Metadata[key]; ok {
			if str, isStr := val.(string); isStr {
				lower := strings.ToLower(str)
				if strings.Contains(lower, "saudi") || strings.Contains(lower, "sa-") ||
					strings.Contains(lower, "me-south") || lower == "sa" ||
					strings.Contains(lower, "riyadh") || strings.Contains(lower, "jeddah") {
					return true
				}
			}
		}
	}
	if regulation, ok := asset.Metadata["regulation"]; ok {
		if str, isStr := regulation.(string); isStr && strings.ToLower(str) == "saudi_pdpl" {
			return true
		}
	}
	return false
}
