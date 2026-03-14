package framework_configs

import (
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// SOC2Controls returns the SOC 2 Type II compliance control mappings.
// Controls are based on the AICPA Trust Service Criteria (TSC) 2017.
func SOC2Controls() []ControlMapping {
	return []ControlMapping{
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC6.1",
				Name:        "Logical and Physical Access Controls",
				Description: "The entity implements logical access security software, infrastructure, and architectures over protected information assets",
				Category:    "Common Criteria - Logical Access",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasAnyAccessControl(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC6.3",
				Name:        "Role-Based Access",
				Description: "The entity authorizes, modifies, or removes access to data based on roles and responsibilities",
				Category:    "Common Criteria - Logical Access",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC6.6",
				Name:        "Encryption of Data at Rest",
				Description: "The entity implements controls to prevent or detect unauthorized access to data stored at rest",
				Category:    "Common Criteria - Data Protection",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedAtRest)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC6.7",
				Name:        "Encryption of Data in Transit",
				Description: "The entity restricts the transmission of data to authorized channels protected by encryption",
				Category:    "Common Criteria - Data Protection",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedInTransit)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC7.2",
				Name:        "Monitoring of System Components",
				Description: "The entity monitors system components for anomalies indicative of malicious acts through audit logging",
				Category:    "Common Criteria - System Operations",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.AuditLogging)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-A1.2",
				Name:        "Recovery Planning (Backup)",
				Description: "The entity provides for recovery of data and infrastructure using documented backup procedures",
				Category:    "Availability - Recovery",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.BackupConfigured)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC3.2",
				Name:        "Risk Assessment",
				Description: "The entity assesses and manages risks through periodic evaluation; minimum posture score of 60 required",
				Category:    "Common Criteria - Risk Assessment",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return asset.PostureScore >= 60
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC6.2",
				Name:        "User Access Review",
				Description: "The entity reviews and validates access rights periodically, at least every 180 days",
				Category:    "Common Criteria - Logical Access",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.LastAccessReview == nil {
					return false
				}
				return time.Since(*asset.LastAccessReview) <= 180*24*time.Hour
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "SOC2-CC7.4",
				Name:        "Incident Response Procedures",
				Description: "The entity has documented and tested incident response procedures for security events",
				Category:    "Common Criteria - System Operations",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "incident_response_ready")
			},
		},
	}
}
