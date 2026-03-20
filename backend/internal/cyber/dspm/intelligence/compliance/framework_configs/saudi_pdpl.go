package framework_configs

import (
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// SaudiPDPLControls returns the Saudi PDPL compliance control mappings.
// Controls are based on the Saudi Arabia Personal Data Protection Law
// (Royal Decree M/19, 2021).
func SaudiPDPLControls() []ControlMapping {
	return []ControlMapping{
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-5",
				Name:        "Data Protection (Encryption at Rest)",
				Description: "Personal data must be protected through appropriate technical measures including encryption at rest",
				Category:    "Data Protection",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedAtRest)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-5-transit",
				Name:        "Data Protection (Encryption in Transit)",
				Description: "Personal data must be protected through appropriate technical measures including encryption during transmission",
				Category:    "Data Protection",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedInTransit)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-10",
				Name:        "Purpose Limitation and Data Minimization",
				Description: "Personal data shall be collected for a specific, clear, and legitimate purpose; data minimization must be applied",
				Category:    "Data Governance",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "data_minimization")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-14",
				Name:        "Consent Management",
				Description: "Processing of personal data requires explicit consent from the data subject, which must be verified",
				Category:    "Consent",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "consent_verified")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-24",
				Name:        "Access Control and Authorization",
				Description: "Implement appropriate access controls to protect personal data from unauthorized access",
				Category:    "Access Control",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-29",
				Name:        "Breach Notification",
				Description: "Data breaches involving personal data must be reported to the competent authority; breach procedures must be in place",
				Category:    "Incident Response",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "breach_notification_ready")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-18",
				Name:        "Right to Access and Correction",
				Description: "Data subjects have the right to access their personal data and request corrections",
				Category:    "Data Subject Rights",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "deletion_capable")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-12",
				Name:        "Audit and Monitoring",
				Description: "Processing activities must be logged and monitored to ensure compliance",
				Category:    "Monitoring",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.AuditLogging)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PDPL-24-review",
				Name:        "Periodic Access Review",
				Description: "Access to personal data must be reviewed at least annually",
				Category:    "Access Control",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.LastAccessReview == nil {
					return false
				}
				return time.Since(*asset.LastAccessReview) <= 365*24*time.Hour
			},
		},
	}
}
