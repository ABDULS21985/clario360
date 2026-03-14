package framework_configs

import (
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// GDPRControls returns the GDPR compliance control mappings.
// Controls are based on the EU General Data Protection Regulation (GDPR)
// with checks derived from Articles 5, 17, 25, 30, 32, 33, and 35.
func GDPRControls() []ControlMapping {
	return []ControlMapping{
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-5.1.f",
				Name:        "Integrity and Confidentiality (Encryption at Rest)",
				Description: "Personal data must be processed with appropriate security including encryption at rest",
				Category:    "Data Protection",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedAtRest)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-5.1.f-transit",
				Name:        "Integrity and Confidentiality (Encryption in Transit)",
				Description: "Personal data must be encrypted during transmission to prevent unauthorized interception",
				Category:    "Data Protection",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedInTransit)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-25",
				Name:        "Data Protection by Design (Access Controls)",
				Description: "Implement role-based access controls to enforce data protection by design and by default",
				Category:    "Access Control",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-30",
				Name:        "Records of Processing Activities (Audit Logging)",
				Description: "Maintain audit logs of data processing activities as required by Art. 30",
				Category:    "Monitoring",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.AuditLogging)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-32",
				Name:        "Security of Processing (Posture Score)",
				Description: "Ensure appropriate technical and organizational security measures with a minimum posture score of 70",
				Category:    "Security Posture",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return asset.PostureScore >= 70
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-32-backup",
				Name:        "Security of Processing (Data Backup)",
				Description: "Ensure ability to restore availability and access to personal data through regular backups",
				Category:    "Business Continuity",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.BackupConfigured)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-35",
				Name:        "Data Protection Impact Assessment",
				Description: "High-risk PII assets must have a completed DPIA documented in metadata",
				Category:    "Risk Assessment",
				Scope:       "high_risk",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "dpia_completed")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-17",
				Name:        "Right to Erasure",
				Description: "Systems handling personal data must support data deletion capabilities (right to be forgotten)",
				Category:    "Data Subject Rights",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "deletion_capable")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-33",
				Name:        "Breach Notification Readiness",
				Description: "Personal data stores must have breach notification procedures documented and tested",
				Category:    "Incident Response",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "breach_notification_ready")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-25-minimization",
				Name:        "Data Minimization",
				Description: "Ensure only necessary personal data is collected and retained for the specified purpose",
				Category:    "Data Protection",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "data_minimization")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "GDPR-25-access-review",
				Name:        "Regular Access Review",
				Description: "Access to personal data must be reviewed at least annually",
				Category:    "Access Control",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.LastAccessReview == nil {
					return false
				}
				// Must have been reviewed within the last 365 days.
				return time.Since(*asset.LastAccessReview) <= 365*24*time.Hour
			},
		},
	}
}
