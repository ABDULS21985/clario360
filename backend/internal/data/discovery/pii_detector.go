package discovery

import (
	"regexp"
	"strings"

	"github.com/clario360/platform/internal/data/model"
)

type piiPattern struct {
	re             *regexp.Regexp
	piiType        string
	classification model.DataClassification
	reason         string
}

var piiPatterns = []piiPattern{
	{regexp.MustCompile(`(?i)(^|_)(ssn|social_security|national_id|citizen_id|nin|bvn)($|_)`), "national_id", model.DataClassificationRestricted, "column name matches national identifier"},
	{regexp.MustCompile(`(?i)(^|_)(passport)(number|no|num|_id)?($|_)`), "passport", model.DataClassificationRestricted, "column name matches passport"},
	{regexp.MustCompile(`(?i)(^|_)(credit_card|card_number|pan|cc_num)(ber)?($|_)`), "credit_card", model.DataClassificationRestricted, "column name matches credit card"},
	{regexp.MustCompile(`(?i)(^|_)(bank_account|iban|routing_number|sort_code|account_no)($|_)`), "bank_account", model.DataClassificationRestricted, "column name matches bank account"},
	{regexp.MustCompile(`(?i)(^|_)(salary|income|wage|compensation|pay_grade)($|_)`), "financial", model.DataClassificationRestricted, "column name matches financial personal data"},
	{regexp.MustCompile(`(?i)(^|_)(diagnosis|medical|health|patient|prescription|medication)($|_)`), "medical", model.DataClassificationRestricted, "column name matches medical data"},
	{regexp.MustCompile(`(?i)(^|_)(fingerprint|biometric|face_id|iris|retina)($|_)`), "biometric", model.DataClassificationRestricted, "column name matches biometric data"},
	{regexp.MustCompile(`(?i)(^|_)(gender|sex|ethnicity|race|religion|disability|sexual_orientation)($|_)`), "demographic", model.DataClassificationRestricted, "column name matches demographic sensitive data"},
	{regexp.MustCompile(`(?i)(^|_)(password|passwd|pwd|secret|token|api_key|private_key)($|_)`), "credential", model.DataClassificationRestricted, "column name matches credentials"},
	{regexp.MustCompile(`(?i)(^|_)(email|e_mail|email_address|mail)($|_)`), "email", model.DataClassificationConfidential, "column name matches email"},
	{regexp.MustCompile(`(?i)(^|_)(phone|telephone|mobile|cell|fax|sms)(number|no|num)?($|_)`), "phone", model.DataClassificationConfidential, "column name matches phone"},
	{regexp.MustCompile(`(?i)(^|_)(first_name|last_name|full_name|given_name|surname|family_name|display_name)($|_)`), "name", model.DataClassificationConfidential, "column name matches name"},
	{regexp.MustCompile(`(?i)(^|_)(birth_date|dob|date_of_birth|birthday)($|_)`), "dob", model.DataClassificationConfidential, "column name matches birth date"},
	{regexp.MustCompile(`(?i)(^|_)(address|street|city|state|zip|postal|zip_code|postal_code)($|_)`), "address", model.DataClassificationConfidential, "column name matches address"},
	{regexp.MustCompile(`(?i)(^|_)(ip_address|user_agent|device_id|mac_address)($|_)`), "technical_id", model.DataClassificationInternal, "column name matches technical identifier"},
}

var (
	emailRe      = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	phoneRe      = regexp.MustCompile(`^\+?[\d\s\-()]{7,15}$`)
	ssnRe        = regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
	creditCardRe = regexp.MustCompile(`^\d{13,19}$`)
)

func DetectPII(columns []model.DiscoveredColumn) []model.DiscoveredColumn {
	for i := range columns {
		column := &columns[i]
		column.InferredClass = model.DataClassificationPublic
		name := strings.ToLower(strings.TrimSpace(column.Name))

		for _, pattern := range piiPatterns {
			if pattern.re.MatchString(name) {
				column.InferredPII = true
				column.InferredPIIType = pattern.piiType
				column.InferredClass = pattern.classification
				column.DetectionReasons = append(column.DetectionReasons, pattern.reason)
				break
			}
		}

		if !column.InferredPII {
			for _, sample := range column.SampleValues {
				switch {
				case emailRe.MatchString(strings.TrimSpace(sample)):
					column.InferredPII = true
					column.InferredPIIType = "email"
					column.InferredClass = model.DataClassificationConfidential
					column.DetectionReasons = append(column.DetectionReasons, "sample resembles an email address")
				case phoneRe.MatchString(strings.TrimSpace(sample)):
					column.InferredPII = true
					column.InferredPIIType = "phone"
					column.InferredClass = model.DataClassificationConfidential
					column.DetectionReasons = append(column.DetectionReasons, "sample resembles a phone number")
				case creditCardRe.MatchString(strings.TrimSpace(sample)) && passesLuhn(strings.TrimSpace(sample)):
					column.InferredPII = true
					column.InferredPIIType = "credit_card"
					column.InferredClass = model.DataClassificationRestricted
					column.DetectionReasons = append(column.DetectionReasons, "sample matches a Luhn-valid credit card number")
				case ssnRe.MatchString(strings.TrimSpace(sample)):
					column.InferredPII = true
					column.InferredPIIType = "national_id"
					column.InferredClass = model.DataClassificationRestricted
					column.DetectionReasons = append(column.DetectionReasons, "sample matches a national identifier pattern")
				}
				if column.InferredPII {
					break
				}
			}
		}
	}
	return columns
}

func TableClassification(columns []model.DiscoveredColumn) model.DataClassification {
	values := make([]model.DataClassification, 0, len(columns))
	for _, column := range columns {
		values = append(values, column.InferredClass)
	}
	return MaxClassification(values...)
}

func passesLuhn(value string) bool {
	sum := 0
	alternate := false
	for i := len(value) - 1; i >= 0; i-- {
		digit := int(value[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate
	}
	return sum%10 == 0
}
