package framework_configs

import (
	"strings"
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// HIPAAControls returns the HIPAA compliance control mappings.
// Controls are based on the HIPAA Security Rule (45 CFR Part 164),
// Privacy Rule, and Breach Notification Rule.
func HIPAAControls() []ControlMapping {
	return []ControlMapping{
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.312.a.1",
				Name:        "Access Control",
				Description: "Implement technical policies and procedures to allow access only to authorized persons or software programs",
				Category:    "Security Rule - Access Control",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.312.a.2.iv",
				Name:        "Encryption and Decryption (At Rest)",
				Description: "Implement a mechanism to encrypt and decrypt electronic protected health information at rest",
				Category:    "Security Rule - Access Control",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedAtRest)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.312.e.1",
				Name:        "Transmission Security",
				Description: "Implement technical security measures to guard against unauthorized access to ePHI transmitted over electronic communications",
				Category:    "Security Rule - Transmission Security",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedInTransit)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.312.b",
				Name:        "Audit Controls",
				Description: "Implement hardware, software, and procedural mechanisms to record and examine activity in systems containing ePHI",
				Category:    "Security Rule - Audit Controls",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.AuditLogging)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.308.a.7",
				Name:        "Contingency Plan (Backup)",
				Description: "Establish and implement a data backup plan to create and maintain retrievable exact copies of ePHI",
				Category:    "Security Rule - Contingency Plan",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.BackupConfigured)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.308.a.1",
				Name:        "Security Management Process (Risk Analysis)",
				Description: "Conduct an accurate and thorough assessment of potential risks to ePHI; posture score must be >= 70",
				Category:    "Security Rule - Security Management",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return asset.PostureScore >= 70
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.530.c",
				Name:        "Privacy Rule - Minimum Necessary",
				Description: "Limit the use and disclosure of PHI to the minimum necessary to accomplish the intended purpose",
				Category:    "Privacy Rule",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "data_minimization")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.408",
				Name:        "Breach Notification",
				Description: "Maintain breach notification procedures and readiness for reporting breaches of unsecured PHI",
				Category:    "Breach Notification Rule",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "breach_notification_ready")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.312.d",
				Name:        "Person or Entity Authentication",
				Description: "Implement procedures to verify the identity of persons seeking access to ePHI",
				Category:    "Security Rule - Authentication",
				Scope:       "healthcare",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.AccessControlType == nil {
					return false
				}
				act := strings.ToLower(*asset.AccessControlType)
				return act == "rbac" || act == "abac" || act == "mfa" || act == "role_based"
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "HIPAA-164.308.a.5",
				Name:        "Security Awareness (Access Review)",
				Description: "Periodic review of access rights to ePHI must be performed at least annually",
				Category:    "Security Rule - Security Awareness",
				Scope:       "healthcare",
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
