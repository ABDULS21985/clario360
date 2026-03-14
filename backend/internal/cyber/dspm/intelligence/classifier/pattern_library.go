package classifier

import (
	"regexp"
	"strings"
	"sync"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// compiledPattern pairs a PIIPattern definition with its compiled regex.
type compiledPattern struct {
	model.PIIPattern
	compiled *regexp.Regexp
}

var (
	patternOnce     sync.Once
	allPatterns     []model.PIIPattern
	compiled        []compiledPattern
	categoryIndex   map[string][]int // category -> indices into allPatterns
	columnNameIndex []compiledPattern // patterns for matching column names
)

func initPatterns() {
	patternOnce.Do(func() {
		allPatterns = buildPatternLibrary()
		compiled = make([]compiledPattern, len(allPatterns))
		categoryIndex = make(map[string][]int)

		for i, p := range allPatterns {
			compiled[i] = compiledPattern{
				PIIPattern: p,
				compiled:   regexp.MustCompile(p.Regex),
			}
			categoryIndex[p.Category] = append(categoryIndex[p.Category], i)
		}

		// Build column name matching patterns (case-insensitive column/field name patterns).
		columnNameIndex = buildColumnNamePatterns()
	})
}

// AllPatterns returns every PII detection pattern in the library.
func AllPatterns() []model.PIIPattern {
	initPatterns()
	out := make([]model.PIIPattern, len(allPatterns))
	copy(out, allPatterns)
	return out
}

// PatternsByCategory returns patterns filtered to the given category.
func PatternsByCategory(category string) []model.PIIPattern {
	initPatterns()
	indices, ok := categoryIndex[category]
	if !ok {
		return nil
	}
	out := make([]model.PIIPattern, len(indices))
	for i, idx := range indices {
		out[i] = allPatterns[idx]
	}
	return out
}

// MatchColumnName returns all patterns whose column-name heuristic matches the given column name.
func MatchColumnName(columnName string) []model.PIIPattern {
	initPatterns()
	lower := strings.ToLower(columnName)
	var matches []model.PIIPattern
	for _, cp := range columnNameIndex {
		if cp.compiled.MatchString(lower) {
			matches = append(matches, cp.PIIPattern)
		}
	}
	return matches
}

// MatchValue returns all patterns whose value regex matches the given string value.
func MatchValue(value string) []model.PIIPattern {
	initPatterns()
	var matches []model.PIIPattern
	for _, cp := range compiled {
		if cp.compiled.MatchString(value) {
			matches = append(matches, cp.PIIPattern)
		}
	}
	return matches
}

// getCompiledPatterns returns the compiled value-matching patterns.
func getCompiledPatterns() []compiledPattern {
	initPatterns()
	return compiled
}

// buildPatternLibrary defines 55+ PII patterns across all categories.
func buildPatternLibrary() []model.PIIPattern {
	return []model.PIIPattern{
		// ===== Financial =====
		{
			Name:               "iban",
			Regex:              `\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"product_code", "serial_number"},
		},
		{
			Name:               "iban_sa",
			Regex:              `\bSA\d{2}\d{2}[A-Z0-9]{18}\b`,
			Weight:             0.95,
			Locale:             "SA",
			Category:           "financial",
			FalsePositiveHints: []string{"product_code"},
		},
		{
			Name:               "iban_de",
			Regex:              `\bDE\d{2}\d{8}\d{10}\b`,
			Weight:             0.95,
			Locale:             "DE",
			Category:           "financial",
			FalsePositiveHints: []string{"product_code"},
		},
		{
			Name:               "iban_gb",
			Regex:              `\bGB\d{2}[A-Z]{4}\d{14}\b`,
			Weight:             0.95,
			Locale:             "GB",
			Category:           "financial",
			FalsePositiveHints: []string{"product_code"},
		},
		{
			Name:               "swift_bic",
			Regex:              `\b[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?\b`,
			Weight:             0.85,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"country_code", "currency_code", "abbreviation"},
		},
		{
			Name:               "tax_id_us",
			Regex:              `\b\d{2}-\d{7}\b`,
			Weight:             0.90,
			Locale:             "US",
			Category:           "financial",
			FalsePositiveHints: []string{"date_range", "version"},
		},
		{
			Name:               "account_number",
			Regex:              `\b\d{8,17}\b`,
			Weight:             0.60,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"timestamp", "row_id", "sequence"},
		},
		{
			Name:               "credit_card",
			Regex:              `\b(?:4\d{3}|5[1-5]\d{2}|3[47]\d{2}|6(?:011|5\d{2}))[- ]?\d{4}[- ]?\d{4}[- ]?\d{1,4}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"phone_number"},
		},
		{
			Name:               "income",
			Regex:              `(?i)\b(?:USD|EUR|GBP|SAR|AED)\s*\d{1,3}(?:,\d{3})*(?:\.\d{2})?\b`,
			Weight:             0.80,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"product_price", "transaction_amount"},
		},
		{
			Name:               "portfolio_id",
			Regex:              `(?i)\b(?:PF|PORT)-\d{6,10}\b`,
			Weight:             0.75,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"project_id"},
		},

		// ===== Health =====
		{
			Name:               "diagnosis_code_icd10",
			Regex:              `\b[A-TV-Z]\d{2}(?:\.\d{1,4})?\b`,
			Weight:             0.85,
			Locale:             "international",
			Category:           "health",
			FalsePositiveHints: []string{"product_code", "category_code"},
		},
		{
			Name:               "medication",
			Regex:              `(?i)\b(?:amoxicillin|metformin|lisinopril|atorvastatin|omeprazole|amlodipine|metoprolol|simvastatin|losartan|gabapentin|hydrochlorothiazide|sertraline|acetaminophen|ibuprofen|aspirin|insulin)\b`,
			Weight:             0.90,
			Locale:             "international",
			Category:           "health",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "lab_result",
			Regex:              `(?i)\b(?:hemoglobin|glucose|cholesterol|creatinine|bilirubin|albumin|platelets|wbc|rbc|hba1c)\s*[:=]?\s*\d+(?:\.\d+)?\s*(?:mg/dL|mmol/L|g/dL|U/L|%|cells/uL)?\b`,
			Weight:             0.90,
			Locale:             "international",
			Category:           "health",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "insurance_id",
			Regex:              `(?i)\b(?:INS|HIC|MBI)[- ]?\d{3}[- ]?\d{3}[- ]?\d{4}\b`,
			Weight:             0.90,
			Locale:             "US",
			Category:           "health",
			FalsePositiveHints: []string{"policy_number"},
		},
		{
			Name:               "blood_type",
			Regex:              `\b(?:A|B|AB|O)[+-]\b`,
			Weight:             0.85,
			Locale:             "international",
			Category:           "health",
			FalsePositiveHints: []string{"grade", "rating"},
		},
		{
			Name:               "medical_record",
			Regex:              `(?i)\b(?:MRN|MR)[- ]?\d{6,10}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "health",
			FalsePositiveHints: []string{"ticket_number"},
		},

		// ===== Identity =====
		{
			Name:               "ssn_us",
			Regex:              `\b(?!000|666|9\d{2})\d{3}-(?!00)\d{2}-(?!0000)\d{4}\b`,
			Weight:             0.95,
			Locale:             "US",
			Category:           "identity",
			FalsePositiveHints: []string{"phone_number", "date"},
		},
		{
			Name:               "ssn_us_no_dash",
			Regex:              `\b(?!000|666|9\d{2})\d{3}(?!00)\d{2}(?!0000)\d{4}\b`,
			Weight:             0.70,
			Locale:             "US",
			Category:           "identity",
			FalsePositiveHints: []string{"phone_number", "account_number"},
		},
		{
			Name:               "passport_us",
			Regex:              `\b[A-Z]\d{8}\b`,
			Weight:             0.80,
			Locale:             "US",
			Category:           "identity",
			FalsePositiveHints: []string{"invoice_number", "tracking_number"},
		},
		{
			Name:               "passport_gb",
			Regex:              `\b\d{9}\b`,
			Weight:             0.50,
			Locale:             "GB",
			Category:           "identity",
			FalsePositiveHints: []string{"phone_number", "reference_number"},
		},
		{
			Name:               "passport_de",
			Regex:              `\b[CFGHJKLMNPRTVWXYZ0-9]{9}\b`,
			Weight:             0.55,
			Locale:             "DE",
			Category:           "identity",
			FalsePositiveHints: []string{"serial_number"},
		},
		{
			Name:               "driver_license_us",
			Regex:              `(?i)\b[A-Z]\d{3}-\d{4}-\d{4}\b`,
			Weight:             0.85,
			Locale:             "US",
			Category:           "identity",
			FalsePositiveHints: []string{"policy_number"},
		},
		{
			Name:               "national_id_sa",
			Regex:              `\b[12]\d{9}\b`,
			Weight:             0.90,
			Locale:             "SA",
			Category:           "identity",
			FalsePositiveHints: []string{"phone_number", "account_number"},
		},
		{
			Name:               "emirates_id_ae",
			Regex:              `\b784-\d{4}-\d{7}-\d\b`,
			Weight:             0.95,
			Locale:             "AE",
			Category:           "identity",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "voter_id",
			Regex:              `(?i)\b(?:VOTER|VID)[- ]?\d{6,12}\b`,
			Weight:             0.85,
			Locale:             "international",
			Category:           "identity",
			FalsePositiveHints: []string{"reference_id"},
		},
		{
			Name:               "birth_certificate",
			Regex:              `(?i)\b(?:BC|BIRTH)[- ]?\d{6,12}\b`,
			Weight:             0.80,
			Locale:             "international",
			Category:           "identity",
			FalsePositiveHints: []string{"reference_id"},
		},

		// ===== Contact =====
		{
			Name:               "email",
			Regex:              `\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b`,
			Weight:             0.90,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{"system_email", "noreply"},
		},
		{
			Name:               "phone_us",
			Regex:              `\b(?:\+1[- ]?)?\(?\d{3}\)?[- ]?\d{3}[- ]?\d{4}\b`,
			Weight:             0.80,
			Locale:             "US",
			Category:           "contact",
			FalsePositiveHints: []string{"fax_number", "order_number"},
		},
		{
			Name:               "phone_international",
			Regex:              `\b\+\d{1,3}[- ]?\d{4,14}\b`,
			Weight:             0.80,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{"reference_number"},
		},
		{
			Name:               "phone_sa",
			Regex:              `\b(?:\+966|00966|0)5\d{8}\b`,
			Weight:             0.85,
			Locale:             "SA",
			Category:           "contact",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "phone_ae",
			Regex:              `\b(?:\+971|00971|0)5\d{8}\b`,
			Weight:             0.85,
			Locale:             "AE",
			Category:           "contact",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "ip_address_v4",
			Regex:              `\b(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\b`,
			Weight:             0.70,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{"version_number", "subnet"},
		},
		{
			Name:               "ip_address_v6",
			Regex:              `\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`,
			Weight:             0.70,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "mac_address",
			Regex:              `\b(?:[0-9a-fA-F]{2}[:\-]){5}[0-9a-fA-F]{2}\b`,
			Weight:             0.75,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "device_id",
			Regex:              `(?i)\b(?:IMEI|MEID|ESN)[:\s]?\d{14,15}\b`,
			Weight:             0.80,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{"serial_number"},
		},
		{
			Name:               "geolocation",
			Regex:              `\b-?(?:90(?:\.0+)?|[1-8]?\d(?:\.\d+)?),\s*-?(?:180(?:\.0+)?|(?:1[0-7]\d|\d{1,2})(?:\.\d+)?)\b`,
			Weight:             0.75,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{"dimension", "resolution"},
		},
		{
			Name:               "gps_coordinates",
			Regex:              `(?i)\b\d{1,3}[°]\s*\d{1,2}[\']\s*\d{1,2}(?:\.\d+)?[\"]\s*[NSEW]\b`,
			Weight:             0.80,
			Locale:             "international",
			Category:           "contact",
			FalsePositiveHints: []string{},
		},

		// ===== Biometric =====
		{
			Name:               "fingerprint_hash",
			Regex:              `(?i)\b(?:FP|FPRINT)[:\-_]?[a-fA-F0-9]{32,64}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "biometric",
			FalsePositiveHints: []string{"file_hash", "checksum"},
		},
		{
			Name:               "facial_recognition_id",
			Regex:              `(?i)\b(?:FACE|FRID|FR)[:\-_]?[a-fA-F0-9]{16,64}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "biometric",
			FalsePositiveHints: []string{"session_id"},
		},
		{
			Name:               "retina_scan",
			Regex:              `(?i)\b(?:RETINA|IRIS)[:\-_]?[a-fA-F0-9]{32,64}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "biometric",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "voice_print",
			Regex:              `(?i)\b(?:VOICE|VPRINT|VP)[:\-_]?[a-fA-F0-9]{16,64}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "biometric",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "dna_sequence",
			Regex:              `\b[ACGT]{20,}\b`,
			Weight:             0.90,
			Locale:             "international",
			Category:           "biometric",
			FalsePositiveHints: []string{"random_string"},
		},

		// ===== Multi-locale =====
		{
			Name:               "eu_vat_number",
			Regex:              `\b(?:AT|BE|BG|CY|CZ|DE|DK|EE|EL|ES|FI|FR|HR|HU|IE|IT|LT|LU|LV|MT|NL|PL|PT|RO|SE|SI|SK)[A-Z0-9]{2,12}\b`,
			Weight:             0.80,
			Locale:             "EU",
			Category:           "financial",
			FalsePositiveHints: []string{"country_code_prefix"},
		},
		{
			Name:               "uk_ni_number",
			Regex:              `\b[A-CEGHJ-PR-TW-Z]{2}\d{6}[A-D]\b`,
			Weight:             0.90,
			Locale:             "GB",
			Category:           "identity",
			FalsePositiveHints: []string{"reference_code"},
		},
		{
			Name:               "date_of_birth",
			Regex:              `\b(?:19|20)\d{2}[-/](?:0[1-9]|1[0-2])[-/](?:0[1-9]|[12]\d|3[01])\b`,
			Weight:             0.75,
			Locale:             "international",
			Category:           "identity",
			FalsePositiveHints: []string{"created_at", "updated_at", "event_date"},
		},
		{
			Name:               "full_name",
			Regex:              `(?i)\b[A-Z][a-z]+\s+(?:[A-Z][a-z]+\s+)?[A-Z][a-z]+\b`,
			Weight:             0.50,
			Locale:             "international",
			Category:           "identity",
			FalsePositiveHints: []string{"company_name", "product_name", "city_name"},
		},
		{
			Name:               "street_address",
			Regex:              `(?i)\b\d{1,5}\s+(?:[A-Za-z]+\s+){1,4}(?:st|street|ave|avenue|blvd|boulevard|dr|drive|rd|road|ln|lane|ct|court|pl|place|way)\b`,
			Weight:             0.70,
			Locale:             "US",
			Category:           "contact",
			FalsePositiveHints: []string{"business_address"},
		},
		{
			Name:               "zip_code_us",
			Regex:              `\b\d{5}(?:-\d{4})?\b`,
			Weight:             0.50,
			Locale:             "US",
			Category:           "contact",
			FalsePositiveHints: []string{"product_code", "sequence"},
		},
		{
			Name:               "postal_code_uk",
			Regex:              `\b[A-Z]{1,2}\d[A-Z\d]?\s*\d[A-Z]{2}\b`,
			Weight:             0.60,
			Locale:             "GB",
			Category:           "contact",
			FalsePositiveHints: []string{"product_code"},
		},
		{
			Name:               "saudi_postal_code",
			Regex:              `\b\d{5}\b`,
			Weight:             0.40,
			Locale:             "SA",
			Category:           "contact",
			FalsePositiveHints: []string{"zip_code", "product_id"},
		},
		{
			Name:               "credit_card_amex",
			Regex:              `\b3[47]\d{2}[- ]?\d{6}[- ]?\d{5}\b`,
			Weight:             0.95,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{},
		},
		{
			Name:               "cvv",
			Regex:              `\b\d{3,4}\b`,
			Weight:             0.30,
			Locale:             "international",
			Category:           "financial",
			FalsePositiveHints: []string{"pin", "code", "quantity"},
		},
		{
			Name:               "iqama_sa",
			Regex:              `\b2\d{9}\b`,
			Weight:             0.85,
			Locale:             "SA",
			Category:           "identity",
			FalsePositiveHints: []string{"phone_number"},
		},
		{
			Name:               "tax_id_sa",
			Regex:              `\b3\d{14}\b`,
			Weight:             0.85,
			Locale:             "SA",
			Category:           "financial",
			FalsePositiveHints: []string{"reference_number"},
		},
		{
			Name:               "passport_sa",
			Regex:              `\b[A-Z]\d{7}\b`,
			Weight:             0.75,
			Locale:             "SA",
			Category:           "identity",
			FalsePositiveHints: []string{"invoice_number"},
		},
	}
}

// columnNamePattern maps column name heuristics to PII pattern metadata.
type columnNamePattern struct {
	regex    string
	name     string
	weight   float64
	category string
}

// buildColumnNamePatterns creates compiled patterns for matching column/field names.
func buildColumnNamePatterns() []compiledPattern {
	defs := []columnNamePattern{
		{`(?i)(?:^|_)ssn(?:$|_)|social_security|social_sec`, "ssn", 0.95, "identity"},
		{`(?i)(?:^|_)national_id(?:$|_)|national_number|nin`, "national_id", 0.90, "identity"},
		{`(?i)passport(?:_num|_number|_no|_id)?`, "passport", 0.85, "identity"},
		{`(?i)driver_?license|driving_?lic|dl_number`, "driver_license", 0.85, "identity"},
		{`(?i)voter_?id|voter_?number|voter_?reg`, "voter_id", 0.80, "identity"},
		{`(?i)birth_?cert(?:ificate)?|bc_number`, "birth_certificate", 0.80, "identity"},
		{`(?i)emirates_?id|eid_number`, "emirates_id", 0.90, "identity"},
		{`(?i)iqama|iqama_?id|residency_?id`, "iqama", 0.85, "identity"},
		{`(?i)(?:^|_)dob(?:$|_)|date_?of_?birth|birth_?date|birthday`, "date_of_birth", 0.80, "identity"},

		{`(?i)(?:^|_)email(?:$|_)|e_?mail(?:_address)?`, "email", 0.90, "contact"},
		{`(?i)(?:^|_)phone(?:$|_)|telephone|mobile|cell(?:_?phone)?|phone_?num`, "phone", 0.85, "contact"},
		{`(?i)ip_?addr(?:ess)?|client_?ip|remote_?ip|src_?ip|dst_?ip`, "ip_address", 0.70, "contact"},
		{`(?i)mac_?addr(?:ess)?|hw_?addr`, "mac_address", 0.75, "contact"},
		{`(?i)device_?id|imei|meid`, "device_id", 0.80, "contact"},
		{`(?i)(?:^|_)lat(?:itude)?(?:$|_)`, "geolocation", 0.70, "contact"},
		{`(?i)(?:^|_)(?:lon|lng|longitude)(?:$|_)`, "geolocation", 0.70, "contact"},
		{`(?i)geo_?loc|geolocation|gps|coordinates`, "geolocation", 0.75, "contact"},
		{`(?i)(?:^|_)address(?:$|_)|street_?addr|mailing_?addr|home_?addr`, "street_address", 0.75, "contact"},
		{`(?i)zip_?code|postal_?code|postcode`, "postal_code", 0.60, "contact"},

		{`(?i)(?:^|_)iban(?:$|_)|iban_?number`, "iban", 0.95, "financial"},
		{`(?i)swift(?:_?code)?|bic(?:_?code)?`, "swift_bic", 0.85, "financial"},
		{`(?i)tax_?id|tin(?:_?number)?|tax_?number|vat_?(?:id|number)`, "tax_id", 0.90, "financial"},
		{`(?i)(?:bank_?)?acct(?:_?(?:num|number))?|bank_?account|account_?no`, "account_number", 0.85, "financial"},
		{`(?i)credit_?card|cc_?(?:num|number)|card_?number|pan(?:$|_)`, "credit_card", 0.95, "financial"},
		{`(?i)(?:^|_)cvv(?:$|_)|cvc|security_?code|card_?code`, "cvv", 0.90, "financial"},
		{`(?i)(?:^|_)income(?:$|_)|salary|wage|compensation|pay_?rate|annual_?pay`, "income", 0.85, "financial"},
		{`(?i)portfolio(?:_?id)?|investment_?id`, "portfolio", 0.75, "financial"},
		{`(?i)routing_?(?:num|number)|sort_?code|aba_?number`, "routing_number", 0.85, "financial"},

		{`(?i)(?:^|_)diagnosis(?:$|_)|icd_?(?:10|9)|diag_?code`, "diagnosis_code", 0.90, "health"},
		{`(?i)medication|drug_?name|prescription|rx_name`, "medication", 0.90, "health"},
		{`(?i)lab_?result|test_?result|lab_?value`, "lab_result", 0.85, "health"},
		{`(?i)insurance_?id|policy_?(?:num|number)|hic_?number|mbi`, "insurance_id", 0.85, "health"},
		{`(?i)blood_?type|blood_?group`, "blood_type", 0.85, "health"},
		{`(?i)(?:medical_?)?record_?(?:num|number|id)|mrn`, "medical_record", 0.90, "health"},
		{`(?i)patient_?id|patient_?name|patient_?num`, "patient", 0.90, "health"},

		{`(?i)fingerprint|fprint|fp_hash`, "fingerprint_hash", 0.95, "biometric"},
		{`(?i)face_?(?:id|rec|scan)|facial_?rec`, "facial_recognition_id", 0.95, "biometric"},
		{`(?i)retina|iris_?scan`, "retina_scan", 0.95, "biometric"},
		{`(?i)voice_?print|voice_?id|vprint`, "voice_print", 0.95, "biometric"},
		{`(?i)dna|dna_?seq|genetic`, "dna_sequence", 0.95, "biometric"},

		{`(?i)first_?name|given_?name|fname`, "person_name", 0.70, "identity"},
		{`(?i)last_?name|surname|family_?name|lname`, "person_name", 0.70, "identity"},
		{`(?i)full_?name|display_?name|person_?name`, "person_name", 0.75, "identity"},
		{`(?i)(?:^|_)gender(?:$|_)|sex(?:$|_)`, "gender", 0.60, "identity"},
		{`(?i)(?:^|_)race(?:$|_)|ethnicity|ethnic_?group`, "ethnicity", 0.75, "identity"},
		{`(?i)religion|religious_?aff`, "religion", 0.80, "identity"},
	}

	result := make([]compiledPattern, len(defs))
	for i, d := range defs {
		result[i] = compiledPattern{
			PIIPattern: model.PIIPattern{
				Name:     d.name,
				Regex:    d.regex,
				Weight:   d.weight,
				Category: d.category,
			},
			compiled: regexp.MustCompile(d.regex),
		}
	}
	return result
}
