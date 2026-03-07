package analytics

import (
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/data/model"
)

type MaskingResult struct {
	ColumnsMasked []string `json:"columns_masked"`
	TotalMasked   int      `json:"total_values_masked"`
}

func MaskPIIColumns(rows []map[string]any, dataModel *model.DataModel, userHasPIIPermission bool) []map[string]any {
	masked, _ := ApplyPIIMasking(rows, dataModel, userHasPIIPermission)
	return masked
}

func ApplyPIIMasking(rows []map[string]any, dataModel *model.DataModel, userHasPIIPermission bool) ([]map[string]any, MaskingResult) {
	if userHasPIIPermission || dataModel == nil {
		return rows, MaskingResult{}
	}
	piiFields := make(map[string]string)
	for _, field := range dataModel.SchemaDefinition {
		if field.PIIType != "" {
			piiFields[strings.ToLower(field.Name)] = field.PIIType
		}
	}
	result := MaskingResult{ColumnsMasked: make([]string, 0)}
	columnMasked := make(map[string]struct{})
	output := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		nextRow := make(map[string]any, len(row))
		for key, value := range row {
			piiType, ok := piiFields[strings.ToLower(key)]
			if !ok || value == nil {
				nextRow[key] = value
				continue
			}
			nextRow[key] = maskValue(value, piiType)
			result.TotalMasked++
			if _, seen := columnMasked[key]; !seen {
				columnMasked[key] = struct{}{}
				result.ColumnsMasked = append(result.ColumnsMasked, key)
			}
		}
		output = append(output, nextRow)
	}
	return output, result
}

func maskValue(value any, piiType string) any {
	stringValue := fmt.Sprintf("%v", value)
	switch piiType {
	case "email":
		parts := strings.Split(stringValue, "@")
		if len(parts) != 2 {
			return "***MASKED***"
		}
		local := maskToken(parts[0], 1)
		domainParts := strings.Split(parts[1], ".")
		if len(domainParts) < 2 {
			return local + "@***"
		}
		domainName := maskToken(domainParts[0], 1)
		tld := domainParts[len(domainParts)-1]
		return local + "@" + domainName + "." + tld
	case "phone":
		if len(stringValue) <= 7 {
			return "***MASKED***"
		}
		if len(stringValue) >= 7 {
			return stringValue[:7] + "***-****"
		}
	case "name":
		parts := strings.Fields(stringValue)
		for i, part := range parts {
			parts[i] = maskToken(part, 1)
		}
		return strings.Join(parts, " ")
	case "national_id", "passport":
		if len(stringValue) > 4 {
			return "***-**-" + stringValue[len(stringValue)-4:]
		}
		return "***MASKED***"
	case "credit_card", "bank_account":
		if len(stringValue) > 4 {
			return "****-****-****-" + stringValue[len(stringValue)-4:]
		}
		return "***MASKED***"
	case "address":
		return maskAddress(stringValue)
	case "dob":
		if len(stringValue) >= 4 {
			return stringValue[:4] + "-**-**"
		}
		return "***MASKED***"
	case "financial":
		return "***"
	default:
		return "***MASKED***"
	}
	return "***MASKED***"
}

func maskToken(value string, keep int) string {
	if value == "" {
		return value
	}
	if keep < 1 {
		keep = 1
	}
	runes := []rune(value)
	if len(runes) <= keep {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:keep]) + strings.Repeat("*", len(runes)-keep)
}

func maskAddress(value string) string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		parts[i] = maskToken(part, 1)
	}
	return strings.Join(parts, ", ")
}

