package analyzer

import "github.com/clario360/platform/internal/lex/model"

type MissingClauseDetector struct {
	required map[model.ContractType][]model.ClauseType
}

func NewMissingClauseDetector() *MissingClauseDetector {
	return &MissingClauseDetector{
		required: map[model.ContractType][]model.ClauseType{
			model.ContractTypeServiceAgreement: {
				model.ClauseTypeTermination,
				model.ClauseTypeLimitationOfLiability,
				model.ClauseTypeConfidentiality,
				model.ClauseTypeGoverningLaw,
				model.ClauseTypeDisputeResolution,
				model.ClauseTypeForceMajeure,
				model.ClauseTypePaymentTerms,
				model.ClauseTypeWarranty,
				model.ClauseTypeSLA,
				model.ClauseTypeIPOwnership,
			},
			model.ContractTypeNDA: {
				model.ClauseTypeConfidentiality,
				model.ClauseTypeTermination,
				model.ClauseTypeGoverningLaw,
				model.ClauseTypeDisputeResolution,
			},
			model.ContractTypeEmployment: {
				model.ClauseTypeTermination,
				model.ClauseTypeNonCompete,
				model.ClauseTypeIPOwnership,
				model.ClauseTypeConfidentiality,
				model.ClauseTypeRepresentations,
			},
			model.ContractTypeVendor: {
				model.ClauseTypeTermination,
				model.ClauseTypeLimitationOfLiability,
				model.ClauseTypeConfidentiality,
				model.ClauseTypeGoverningLaw,
				model.ClauseTypeDataProtection,
				model.ClauseTypeAuditRights,
				model.ClauseTypeInsurance,
				model.ClauseTypeForceMajeure,
			},
			model.ContractTypeLicense: {
				model.ClauseTypeIPOwnership,
				model.ClauseTypeLimitationOfLiability,
				model.ClauseTypeWarranty,
				model.ClauseTypeTermination,
				model.ClauseTypeGoverningLaw,
			},
		},
	}
}

func (d *MissingClauseDetector) Required(contractType model.ContractType) []model.ClauseType {
	return append([]model.ClauseType(nil), d.required[contractType]...)
}

func (d *MissingClauseDetector) Detect(contractType model.ContractType, found map[model.ClauseType]bool) []model.ClauseType {
	required := d.required[contractType]
	if len(required) == 0 {
		return nil
	}
	missing := make([]model.ClauseType, 0, len(required))
	for _, clauseType := range required {
		if !found[clauseType] {
			missing = append(missing, clauseType)
		}
	}
	return missing
}
