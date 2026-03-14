package compliance

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// EEA countries and associated cloud region prefixes.
var eeaRegionPrefixes = []string{
	"eu-", "europe-", "euw", "eun",
}

var eeaCountries = map[string]bool{
	"austria": true, "belgium": true, "bulgaria": true, "croatia": true,
	"cyprus": true, "czech_republic": true, "czechia": true, "denmark": true,
	"estonia": true, "finland": true, "france": true, "germany": true,
	"greece": true, "hungary": true, "iceland": true, "ireland": true,
	"italy": true, "latvia": true, "liechtenstein": true, "lithuania": true,
	"luxembourg": true, "malta": true, "netherlands": true, "norway": true,
	"poland": true, "portugal": true, "romania": true, "slovakia": true,
	"slovenia": true, "spain": true, "sweden": true,
}

// Saudi region indicators.
var saudiRegionPrefixes = []string{
	"sa-", "me-south", "me-central",
}

var saudiRegionNames = map[string]bool{
	"saudi_arabia":  true,
	"saudi-arabia":  true,
	"saudi arabia":  true,
	"sa":            true,
	"riyadh":        true,
	"jeddah":        true,
	"ksa":           true,
}

// residencyRule defines a data residency requirement.
type residencyRule struct {
	regulation     string
	requiredRegion string
	severity       string
	description    string
	assetMatch     func(asset *cybermodel.DSPMDataAsset) bool
	regionMatch    func(region string) bool
}

// ResidencyTracker detects data residency violations by analyzing where
// data assets are physically located against regulatory requirements.
type ResidencyTracker struct {
	rules  []residencyRule
	logger zerolog.Logger
}

// NewResidencyTracker creates a new residency tracker with built-in rules
// for GDPR (EEA) and Saudi PDPL (Saudi Arabia).
func NewResidencyTracker(logger zerolog.Logger) *ResidencyTracker {
	t := &ResidencyTracker{
		logger: logger.With().Str("component", "residency_tracker").Logger(),
	}
	t.rules = t.buildRules()
	return t
}

// Analyze examines a list of data assets and returns all detected data
// residency violations based on the regulatory requirements.
func (r *ResidencyTracker) Analyze(assets []*cybermodel.DSPMDataAsset) []model.ResidencyViolation {
	var violations []model.ResidencyViolation

	for _, asset := range assets {
		region := extractRegion(asset)
		if region == "" {
			// Cannot determine region; skip.
			continue
		}

		for _, rule := range r.rules {
			// Check if this rule applies to this asset.
			if !rule.assetMatch(asset) {
				continue
			}

			// Check if the actual region satisfies the requirement.
			if rule.regionMatch(region) {
				continue // compliant
			}

			violation := model.ResidencyViolation{
				AssetID:        asset.AssetID,
				AssetName:      asset.AssetName,
				Regulation:     rule.regulation,
				RequiredRegion: rule.requiredRegion,
				ActualRegion:   region,
				Severity:       rule.severity,
				Description:    fmt.Sprintf("%s: %s", rule.description, asset.AssetName),
			}
			violations = append(violations, violation)

			r.logger.Warn().
				Str("asset_id", asset.AssetID.String()).
				Str("regulation", rule.regulation).
				Str("required_region", rule.requiredRegion).
				Str("actual_region", region).
				Msg("data residency violation detected")
		}
	}

	r.logger.Info().
		Int("assets_analyzed", len(assets)).
		Int("violations_found", len(violations)).
		Msg("residency analysis complete")

	return violations
}

// buildRules constructs the built-in residency rules.
func (r *ResidencyTracker) buildRules() []residencyRule {
	return []residencyRule{
		{
			regulation:     "GDPR",
			requiredRegion: "EEA (European Economic Area)",
			severity:       "critical",
			description:    "GDPR requires personal data of EU/EEA residents to be stored within the EEA or in countries with adequate data protection",
			assetMatch: func(asset *cybermodel.DSPMDataAsset) bool {
				// GDPR applies to assets containing PII with EU/EEA data subjects.
				if !asset.ContainsPII {
					return false
				}
				// Check if asset is tagged as having EU data subjects.
				return hasEUDataSubjects(asset)
			},
			regionMatch: isEEARegion,
		},
		{
			regulation:     "Saudi PDPL",
			requiredRegion: "Saudi Arabia",
			severity:       "high",
			description:    "Saudi PDPL requires personal data of Saudi residents to be stored within the Kingdom of Saudi Arabia",
			assetMatch: func(asset *cybermodel.DSPMDataAsset) bool {
				if !asset.ContainsPII {
					return false
				}
				return hasSaudiDataSubjects(asset)
			},
			regionMatch: isSaudiRegion,
		},
	}
}

// extractRegion extracts the geographic region from an asset's metadata.
// It checks several common metadata keys.
func extractRegion(asset *cybermodel.DSPMDataAsset) string {
	if asset.Metadata == nil {
		return ""
	}

	for _, key := range []string{"region", "location", "cloud_region", "data_center", "country"} {
		if val, ok := asset.Metadata[key]; ok {
			if str, isStr := val.(string); isStr && str != "" {
				return str
			}
		}
	}
	return ""
}

// isEEARegion checks if a region string indicates an EEA location.
func isEEARegion(region string) bool {
	lower := strings.ToLower(region)

	// Check EEA country names.
	if eeaCountries[lower] {
		return true
	}

	// Check cloud region prefixes.
	for _, prefix := range eeaRegionPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	// Check if the region contains an EEA country name.
	for country := range eeaCountries {
		if strings.Contains(lower, country) {
			return true
		}
	}

	return false
}

// isSaudiRegion checks if a region string indicates a Saudi Arabia location.
func isSaudiRegion(region string) bool {
	lower := strings.ToLower(region)

	if saudiRegionNames[lower] {
		return true
	}

	for _, prefix := range saudiRegionPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	if strings.Contains(lower, "saudi") || strings.Contains(lower, "riyadh") || strings.Contains(lower, "jeddah") {
		return true
	}

	return false
}

// hasEUDataSubjects checks if an asset is tagged as containing EU/EEA data subjects.
func hasEUDataSubjects(asset *cybermodel.DSPMDataAsset) bool {
	if asset.Metadata == nil {
		return false
	}

	// Check explicit data_subjects metadata.
	if val, ok := asset.Metadata["data_subjects"]; ok {
		if str, isStr := val.(string); isStr {
			lower := strings.ToLower(str)
			if strings.Contains(lower, "eu") || strings.Contains(lower, "eea") || strings.Contains(lower, "europe") {
				return true
			}
		}
		if list, isList := val.([]interface{}); isList {
			for _, item := range list {
				if str, isStr := item.(string); isStr {
					lower := strings.ToLower(str)
					if strings.Contains(lower, "eu") || strings.Contains(lower, "eea") || strings.Contains(lower, "europe") {
						return true
					}
				}
			}
		}
	}

	// Check regulation metadata.
	if reg, ok := asset.Metadata["regulation"]; ok {
		if str, isStr := reg.(string); isStr && strings.ToLower(str) == "gdpr" {
			return true
		}
	}

	// If the asset has a region that suggests EU data, assume EU data subjects.
	region := extractRegion(asset)
	if region != "" && isEEARegion(region) {
		return true
	}

	return false
}

// hasSaudiDataSubjects checks if an asset is tagged as containing Saudi data subjects.
func hasSaudiDataSubjects(asset *cybermodel.DSPMDataAsset) bool {
	if asset.Metadata == nil {
		return false
	}

	// Check explicit data_subjects metadata.
	if val, ok := asset.Metadata["data_subjects"]; ok {
		if str, isStr := val.(string); isStr {
			lower := strings.ToLower(str)
			if strings.Contains(lower, "saudi") || strings.Contains(lower, "ksa") {
				return true
			}
		}
		if list, isList := val.([]interface{}); isList {
			for _, item := range list {
				if str, isStr := item.(string); isStr {
					lower := strings.ToLower(str)
					if strings.Contains(lower, "saudi") || strings.Contains(lower, "ksa") {
						return true
					}
				}
			}
		}
	}

	// Check regulation metadata.
	if reg, ok := asset.Metadata["regulation"]; ok {
		if str, isStr := reg.(string); isStr && strings.ToLower(str) == "saudi_pdpl" {
			return true
		}
	}

	return false
}
