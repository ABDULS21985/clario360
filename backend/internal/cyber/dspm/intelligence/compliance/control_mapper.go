package compliance

import (
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/compliance/framework_configs"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AssetChecker is a function that evaluates whether a single data asset
// satisfies a specific compliance control requirement.
type AssetChecker = framework_configs.AssetChecker

// ControlMapping pairs a control definition with its asset-level check function.
type ControlMapping = framework_configs.ControlMapping

// BuildControlMappings returns the full set of control mappings for a given
// compliance framework. Each mapping includes the control definition and an
// AssetChecker function that evaluates assets against that control.
func BuildControlMappings(framework model.ComplianceFramework) []ControlMapping {
	switch framework {
	case model.FrameworkGDPR:
		return framework_configs.GDPRControls()
	case model.FrameworkHIPAA:
		return framework_configs.HIPAAControls()
	case model.FrameworkSOC2:
		return framework_configs.SOC2Controls()
	case model.FrameworkPCIDSS:
		return framework_configs.PCIDSSControls()
	case model.FrameworkSaudiPDPL:
		return framework_configs.SaudiPDPLControls()
	case model.FrameworkISO27001:
		return framework_configs.ISO27001Controls()
	default:
		return nil
	}
}

// IsInScope determines whether a given asset falls within the scope of a control
// based on its scope definition.
func IsInScope(asset *cybermodel.DSPMDataAsset, scope string) bool {
	switch scope {
	case "pii":
		return asset.ContainsPII
	case "all":
		return true
	case "high_risk":
		return asset.RiskScore >= 75 && asset.ContainsPII
	case "non_public":
		return asset.DataClassification != "public"
	case "payment":
		return isPaymentAsset(asset)
	case "healthcare":
		return isHealthcareAsset(asset)
	default:
		return true
	}
}

// isPaymentAsset checks if the asset contains payment card data.
func isPaymentAsset(asset *cybermodel.DSPMDataAsset) bool {
	for _, pii := range asset.PIITypes {
		switch pii {
		case "credit_card", "payment_card", "cardholder", "pan", "card_number", "cvv", "card_expiry":
			return true
		}
	}
	return false
}

// isHealthcareAsset checks if the asset contains healthcare data.
func isHealthcareAsset(asset *cybermodel.DSPMDataAsset) bool {
	for _, pii := range asset.PIITypes {
		switch pii {
		case "health_record", "medical_record", "phi", "health_data", "patient_id", "diagnosis", "prescription":
			return true
		}
	}
	return false
}
