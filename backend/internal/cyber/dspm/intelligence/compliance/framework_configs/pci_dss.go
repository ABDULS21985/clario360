package framework_configs

import (
	"strings"
	"time"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// PCIDSSControls returns the PCI DSS v4.0 compliance control mappings.
// Controls are based on the Payment Card Industry Data Security Standard v4.0.
func PCIDSSControls() []ControlMapping {
	return []ControlMapping{
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-3.4",
				Name:        "Protect Stored Account Data (Encryption at Rest)",
				Description: "Render PAN unreadable anywhere it is stored using strong cryptography",
				Category:    "Requirement 3 - Protect Stored Account Data",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedAtRest)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-4.2",
				Name:        "Protect Cardholder Data in Transit",
				Description: "Protect cardholder data with strong cryptography during transmission over open, public networks",
				Category:    "Requirement 4 - Encrypt Transmission",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.EncryptedInTransit)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-7.1",
				Name:        "Restrict Access by Business Need",
				Description: "Limit access to system components and cardholder data to only those individuals whose job requires such access",
				Category:    "Requirement 7 - Restrict Access",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return hasRBAC(asset)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-10.2",
				Name:        "Audit Trail Implementation",
				Description: "Implement automated audit trails for all system components to reconstruct events",
				Category:    "Requirement 10 - Track and Monitor Access",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return boolPtr(asset.AuditLogging)
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-3.1",
				Name:        "Data Retention Policy",
				Description: "Keep cardholder data storage to a minimum with documented retention and disposal policies",
				Category:    "Requirement 3 - Protect Stored Account Data",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "retention_policy_defined")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-6.4",
				Name:        "Network Segmentation",
				Description: "Cardholder data environment must not be directly internet-facing without proper segmentation",
				Category:    "Requirement 6 - Secure Systems",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.NetworkExposure == nil {
					return true // Unknown exposure; pass by default
				}
				exposure := strings.ToLower(*asset.NetworkExposure)
				return exposure != "internet_facing" && exposure != "internet-facing" && exposure != "public"
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-12.10",
				Name:        "Incident Response Plan",
				Description: "Implement an incident response plan for immediate response to a system breach",
				Category:    "Requirement 12 - Security Policy",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return metadataBool(asset.Metadata, "incident_response_ready")
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-11.3",
				Name:        "Vulnerability Management",
				Description: "Regularly test security systems and processes through vulnerability scanning and penetration testing",
				Category:    "Requirement 11 - Regular Testing",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				return asset.PostureScore >= 70
			},
		},
		{
			Definition: model.ControlDefinition{
				ControlID:   "PCI-9.4",
				Name:        "Periodic Access Review",
				Description: "Review access to cardholder data at least every 90 days",
				Category:    "Requirement 9 - Physical Access",
				Scope:       "payment",
			},
			Check: func(asset *cybermodel.DSPMDataAsset) bool {
				if asset.LastAccessReview == nil {
					return false
				}
				return time.Since(*asset.LastAccessReview) <= 90*24*time.Hour
			},
		},
	}
}
