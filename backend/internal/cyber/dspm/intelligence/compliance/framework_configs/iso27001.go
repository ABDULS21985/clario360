package framework_configs

import (
	"strings"
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// ISO27001Controls returns the ISO 27001:2022 Annex A compliance control mappings.
// Controls are based on ISO/IEC 27001:2022 Information Security Management System.
func ISO27001Controls() []ControlMapping {
	return []ControlMapping{
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.5.15",
				Name:        "Access Control",
				Description: "Rules to control physical and logical access to information shall be established based on business requirements",
				Category:    "A.5 - Organizational Controls",
				Scope:       "all",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasAnyAccessControl(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.5.18",
				Name:        "Access Rights Management",
				Description: "Access rights shall be provisioned, reviewed, modified, and removed in accordance with policies",
				Category:    "A.5 - Organizational Controls",
				Scope:       "all",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.8.24",
				Name:        "Use of Cryptography (At Rest)",
				Description: "Rules for the effective use of cryptography including encryption at rest shall be defined and implemented",
				Category:    "A.8 - Technological Controls",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedAtRest)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.8.24-transit",
				Name:        "Use of Cryptography (In Transit)",
				Description: "Data shall be encrypted during transmission across networks using strong cryptographic protocols",
				Category:    "A.8 - Technological Controls",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedInTransit)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.8.15",
				Name:        "Logging and Monitoring",
				Description: "Logs that record activities, exceptions, faults, and other relevant events shall be produced and monitored",
				Category:    "A.8 - Technological Controls",
				Scope:       "all",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.AuditLogging)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.8.13",
				Name:        "Information Backup",
				Description: "Backup copies of information shall be maintained and regularly tested in accordance with backup policy",
				Category:    "A.8 - Technological Controls",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.BackupConfigured)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.5.29",
				Name:        "Information Security During Disruption",
				Description: "Information security shall be maintained at an appropriate level during adverse situations; posture >= 60",
				Category:    "A.5 - Organizational Controls",
				Scope:       "all",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return asset.PostureScore >= 60
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.8.20",
				Name:        "Network Security",
				Description: "Networks and network devices shall be secured and managed to protect information in systems",
				Category:    "A.8 - Technological Controls",
				Scope:       "all",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.NetworkExposure == nil {
					return true
				}
				exposure := strings.ToLower(*asset.NetworkExposure)
				// Internet-facing assets must have both encryption and access control.
				if exposure == "internet_facing" || exposure == "internet-facing" || exposure == "public" {
					return boolPtr(asset.EncryptedInTransit) && hasAnyAccessControl(asset)
				}
				return true
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.5.34",
				Name:        "Privacy and Protection of PII",
				Description: "Privacy and protection of personally identifiable information shall be ensured as required by regulations",
				Category:    "A.5 - Organizational Controls",
				Scope:       "pii",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				// PII assets must have encryption and access control.
				return boolPtr(asset.EncryptedAtRest) && hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "A.5.18-review",
				Name:        "Periodic Access Rights Review",
				Description: "Access rights shall be reviewed at regular intervals, at least every 180 days",
				Category:    "A.5 - Organizational Controls",
				Scope:       "non_public",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.LastAccessReview == nil {
					return false
				}
				return time.Since(*asset.LastAccessReview) <= 180*24*time.Hour
			},
		},
	}
}
