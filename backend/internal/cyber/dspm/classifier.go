package dspm

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/clario360/platform/internal/cyber/model"
)

// piiPattern maps a PII type to its column name regex.
type piiPattern struct {
	Name    string
	Pattern *regexp.Regexp
	Weight  int // higher = more sensitive
}

var piiPatterns = []piiPattern{
	{Name: "biometric", Pattern: regexp.MustCompile(`(?i)fingerprint|face_id|iris|biometric`), Weight: 10},
	{Name: "medical", Pattern: regexp.MustCompile(`(?i)diagnosis|treatment|medication|patient`), Weight: 10},
	{Name: "bank", Pattern: regexp.MustCompile(`(?i)bank_account|iban|routing_number|sort_code`), Weight: 9},
	{Name: "credit_card", Pattern: regexp.MustCompile(`(?i)credit_card|card_number|cc_number|pan`), Weight: 9},
	{Name: "ssn", Pattern: regexp.MustCompile(`(?i)ssn|social_security|national_id|nin`), Weight: 9},
	{Name: "salary", Pattern: regexp.MustCompile(`(?i)salary|wage|compensation|income`), Weight: 8},
	{Name: "email", Pattern: regexp.MustCompile(`(?i)email|e_mail|email_address`), Weight: 5},
	{Name: "phone", Pattern: regexp.MustCompile(`(?i)phone|telephone|mobile|cell`), Weight: 5},
	{Name: "name", Pattern: regexp.MustCompile(`(?i)first_name|last_name|full_name|surname|given_name`), Weight: 4},
	{Name: "address", Pattern: regexp.MustCompile(`(?i)address|street|city|state|zip|postal`), Weight: 4},
	{Name: "birth_date", Pattern: regexp.MustCompile(`(?i)birth|dob|date_of_birth`), Weight: 5},
}

// ClassificationResult is the output of PII detection.
type ClassificationResult struct {
	Classification  string
	SensitivityScore float64
	ContainsPII     bool
	PIITypes        []string
	PIIColumnCount  int
}

// DSPMClassifier classifies data assets by PII content.
type DSPMClassifier struct{}

// NewDSPMClassifier creates a DSPMClassifier.
func NewDSPMClassifier() *DSPMClassifier { return &DSPMClassifier{} }

// Classify analyses an asset's schema info (column names) for PII content.
func (c *DSPMClassifier) Classify(asset *model.Asset) *ClassificationResult {
	result := &ClassificationResult{
		Classification: "internal",
		PIITypes:       []string{},
	}

	// Extract column names from asset metadata/schema_info
	columns := extractColumns(asset)
	if len(columns) == 0 {
		result.SensitivityScore = 40 // internal default
		return result
	}

	piiFound := map[string]bool{}
	maxWeight := 0

	for _, col := range columns {
		for _, pattern := range piiPatterns {
			if pattern.Pattern.MatchString(col) {
				if !piiFound[pattern.Name] {
					piiFound[pattern.Name] = true
					result.PIITypes = append(result.PIITypes, pattern.Name)
					result.PIIColumnCount++
					if pattern.Weight > maxWeight {
						maxWeight = pattern.Weight
					}
				}
			}
		}
	}

	result.ContainsPII = len(piiFound) > 0

	// Assign classification based on highest-weight PII found
	switch {
	case maxWeight >= 9: // biometric, medical, bank, credit_card, ssn
		result.Classification = "restricted"
		result.SensitivityScore = 90
	case maxWeight >= 8: // salary
		result.Classification = "confidential"
		result.SensitivityScore = 70
	case maxWeight >= 4 && result.ContainsPII: // names, contact info
		result.Classification = "confidential"
		result.SensitivityScore = 70
	case result.ContainsPII:
		result.Classification = "internal"
		result.SensitivityScore = 50
	default:
		// No PII: check if explicitly tagged as public
		if isPublicAsset(asset) {
			result.Classification = "public"
			result.SensitivityScore = 10
		} else {
			result.Classification = "internal"
			result.SensitivityScore = 40
		}
	}

	return result
}

// extractColumns gets column names from asset metadata or tags.
func extractColumns(asset *model.Asset) []string {
	var columns []string

	if len(asset.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(asset.Metadata, &meta); err == nil {
			// Direct columns list
			if cols, ok := meta["columns"].([]interface{}); ok {
				for _, c := range cols {
					if s, ok := c.(string); ok {
						columns = append(columns, s)
					}
				}
			}
			// schema_info.tables[].columns
			if schema, ok := meta["schema_info"].(map[string]interface{}); ok {
				if tables, ok := schema["tables"].([]interface{}); ok {
					for _, t := range tables {
						if tbl, ok := t.(map[string]interface{}); ok {
							if cols, ok := tbl["columns"].([]interface{}); ok {
								for _, c := range cols {
									if s, ok := c.(string); ok {
										columns = append(columns, s)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Also scan tags for column hints
	for _, tag := range asset.Tags {
		if strings.HasPrefix(tag, "col:") {
			columns = append(columns, strings.TrimPrefix(tag, "col:"))
		}
	}

	return columns
}

func isPublicAsset(asset *model.Asset) bool {
	for _, tag := range asset.Tags {
		if strings.EqualFold(tag, "public") || strings.EqualFold(tag, "open-dataset") {
			return true
		}
	}
	if len(asset.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(asset.Metadata, &meta); err == nil {
			if pub, ok := meta["public"].(bool); ok && pub {
				return true
			}
		}
	}
	return false
}
