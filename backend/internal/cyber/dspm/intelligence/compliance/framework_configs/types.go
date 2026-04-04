package framework_configs

import (
	"strings"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AssetChecker is a function that evaluates whether a single data asset
// satisfies a specific compliance control requirement.
type AssetChecker func(asset *cybermodel.DSPMDataAsset) bool

// ControlMapping pairs a control definition with its asset-level check function.
type ControlMapping struct {
	Definition model.ControlDefinition
	Check      AssetChecker
}

// Helper functions used across framework configs.

// boolPtr safely dereferences a *bool, returning false if nil.
func boolPtr(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

// strPtr safely dereferences a *string, returning empty if nil.
func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// hasRBAC checks if the asset has role-based or attribute-based access control.
func hasRBAC(asset *cybermodel.DSPMDataAsset) bool {
	if asset.AccessControlType == nil {
		return false
	}
	act := strings.ToLower(*asset.AccessControlType)
	return act == "rbac" || act == "abac" || act == "role_based" || act == "attribute_based"
}

// hasAnyAccessControl checks if the asset has any form of access control.
func hasAnyAccessControl(asset *cybermodel.DSPMDataAsset) bool {
	if asset.AccessControlType == nil {
		return false
	}
	act := strings.ToLower(*asset.AccessControlType)
	return act != "" && act != "none"
}

// metadataBool extracts a boolean from metadata, defaulting to false.
func metadataBool(md map[string]interface{}, key string) bool {
	if md == nil {
		return false
	}
	if val, ok := md[key]; ok {
		if b, isBool := val.(bool); isBool {
			return b
		}
	}
	return false
}

// metadataString extracts a string from metadata, defaulting to empty.
func metadataString(md map[string]interface{}, key string) string {
	if md == nil {
		return ""
	}
	if val, ok := md[key]; ok {
		if s, isStr := val.(string); isStr {
			return s
		}
	}
	return ""
}
