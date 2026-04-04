package classifier

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// classificationRank maps classification labels to numeric ranks for comparison.
var classificationRank = map[string]int{
	"public":       0,
	"internal":     1,
	"confidential": 2,
	"restricted":   3,
	"top_secret":   4,
}

// rankToClassification maps numeric ranks back to classification labels.
var rankToClassification = map[int]string{
	0: "public",
	1: "internal",
	2: "confidential",
	3: "restricted",
	4: "top_secret",
}

// MLClassifier is the multi-layer classification engine that combines pattern
// matching, content inspection, and statistical analysis to classify data assets.
type MLClassifier struct {
	patterns   []model.PIIPattern
	inspector  *ContentInspector
	confidence *ConfidenceCalculator
	logger     zerolog.Logger
}

// NewMLClassifier creates an MLClassifier with the full pattern library and
// default content inspector (100-row sample) and confidence calculator.
func NewMLClassifier(logger zerolog.Logger) *MLClassifier {
	return &MLClassifier{
		patterns:   AllPatterns(),
		inspector:  NewContentInspector(100),
		confidence: NewConfidenceCalculator(),
		logger:     logger.With().Str("component", "ml_classifier").Logger(),
	}
}

// ClassifyAsset runs all three classification layers on a single asset and
// returns the enhanced classification result.
func (c *MLClassifier) ClassifyAsset(asset *cybermodel.DSPMDataAsset, sampleRows [][]string, columns []string) *model.EnhancedClassification {
	if asset == nil {
		return nil
	}

	result := &model.EnhancedClassification{
		AssetID:   asset.ID,
		AssetName: asset.AssetName,
		PIITypes:  []string{},
		Evidence: model.ClassificationEvidence{
			Explanation: "Multi-layer classification analysis",
		},
	}

	// If no explicit columns provided, extract them from schema_info.
	if len(columns) == 0 {
		columns = extractColumnsFromSchema(asset)
	}

	// Layer 1: Pattern matching on column names.
	patternClass, patternConf, patternMatches, patternPII := c.runPatternLayer(columns)
	result.PatternConfidence = patternConf
	result.Evidence.PatternMatches = patternMatches

	c.logger.Debug().
		Str("asset", asset.AssetName).
		Str("pattern_class", patternClass).
		Float64("pattern_conf", patternConf).
		Int("pattern_matches", len(patternMatches)).
		Msg("layer 1 (pattern) complete")

	// Layer 2: Content inspection on sampled row values.
	var contentClass string
	var contentConf float64
	var contentPII []string

	if len(sampleRows) > 0 && len(columns) > 0 {
		contentResults := c.inspector.Inspect(columns, sampleRows)
		result.Evidence.ContentResults = contentResults
		contentClass, contentConf, contentPII = c.deriveContentClassification(contentResults)

		c.logger.Debug().
			Str("asset", asset.AssetName).
			Str("content_class", contentClass).
			Float64("content_conf", contentConf).
			Int("content_results", len(contentResults)).
			Msg("layer 2 (content) complete")
	}
	result.ContentConfidence = contentConf

	// Layer 3: Statistical analysis of column value distributions.
	statClass, statConf, statAnalyses := c.runStatisticalLayer(columns, sampleRows)
	result.StatisticalConfidence = statConf
	result.Evidence.StatisticalResults = statAnalyses

	c.logger.Debug().
		Str("asset", asset.AssetName).
		Str("stat_class", statClass).
		Float64("stat_conf", statConf).
		Msg("layer 3 (statistical) complete")

	// Combine: final classification = highest classification found.
	finalRank := maxRank(
		classificationRank[patternClass],
		classificationRank[contentClass],
		classificationRank[statClass],
	)
	result.Classification = rankToClassification[finalRank]

	// Determine which method produced the highest classification.
	switch {
	case classificationRank[patternClass] == finalRank && patternConf > 0:
		result.DetectedBy = model.ClassMethodPattern
	case classificationRank[contentClass] == finalRank && contentConf > 0:
		result.DetectedBy = model.ClassMethodContent
	case classificationRank[statClass] == finalRank && statConf > 0:
		result.DetectedBy = model.ClassMethodStatistical
	default:
		result.DetectedBy = model.ClassMethodPattern
	}

	// Merge PII types from all layers, deduplicated.
	result.PIITypes = deduplicateStrings(patternPII, contentPII)

	// Final confidence = weighted average of the three layers.
	result.Confidence = c.confidence.Calculate(patternConf, contentConf, statConf)

	// If confidence is low, flag for human review.
	result.NeedsHumanReview = c.confidence.NeedsHumanReview(result.Confidence)

	// Build the explanation.
	result.Evidence.Explanation = buildExplanation(result)

	c.logger.Info().
		Str("asset", asset.AssetName).
		Str("classification", result.Classification).
		Float64("confidence", result.Confidence).
		Bool("needs_review", result.NeedsHumanReview).
		Int("pii_types", len(result.PIITypes)).
		Msg("classification complete")

	return result
}

// ClassifyBatch runs classification on multiple assets. sampleRows are not
// available in batch mode so only pattern and statistical layers run.
func (c *MLClassifier) ClassifyBatch(assets []*cybermodel.DSPMDataAsset) []model.EnhancedClassification {
	results := make([]model.EnhancedClassification, 0, len(assets))
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		r := c.ClassifyAsset(asset, nil, nil)
		if r != nil {
			results = append(results, *r)
		}
	}
	return results
}

// runPatternLayer matches column names against the pattern library.
func (c *MLClassifier) runPatternLayer(columns []string) (classification string, confidence float64, matches []model.PatternMatch, piiTypes []string) {
	if len(columns) == 0 {
		return "internal", 0, nil, nil
	}

	highestRank := 0
	piiSet := make(map[string]bool)

	for _, col := range columns {
		colMatches := MatchColumnName(col)
		for _, pm := range colMatches {
			matches = append(matches, model.PatternMatch{
				PatternName: pm.Name,
				ColumnName:  col,
				Regex:       pm.Regex,
				Locale:      pm.Locale,
				Weight:      pm.Weight,
			})

			rank := piiTypeToRank(pm.Name, pm.Category)
			if rank > highestRank {
				highestRank = rank
			}
			piiSet[pm.Name] = true
		}
	}

	if len(matches) == 0 {
		return "internal", 0, nil, nil
	}

	classification = rankToClassification[highestRank]

	// Confidence based on how many columns matched vs total.
	matchedCols := make(map[string]bool)
	for _, m := range matches {
		matchedCols[m.ColumnName] = true
	}
	confidence = math.Min(float64(len(matchedCols))/float64(len(columns))*2, 1.0)
	// Boost confidence if high-weight patterns matched.
	var maxWeight float64
	for _, m := range matches {
		if m.Weight > maxWeight {
			maxWeight = m.Weight
		}
	}
	confidence = (confidence + maxWeight) / 2

	for t := range piiSet {
		piiTypes = append(piiTypes, t)
	}
	sort.Strings(piiTypes)

	return classification, confidence, matches, piiTypes
}

// deriveContentClassification determines classification from content inspection results.
func (c *MLClassifier) deriveContentClassification(results []model.ContentInspectionResult) (string, float64, []string) {
	if len(results) == 0 {
		return "internal", 0, nil
	}

	highestRank := 0
	var totalConf float64
	piiSet := make(map[string]bool)

	for _, r := range results {
		rank := piiTypeToRank(r.DetectedType, "")
		if rank > highestRank {
			highestRank = rank
		}
		totalConf += r.Confidence
		piiSet[r.DetectedType] = true
	}

	avgConf := totalConf / float64(len(results))
	classification := rankToClassification[highestRank]

	var piiTypes []string
	for t := range piiSet {
		piiTypes = append(piiTypes, t)
	}
	sort.Strings(piiTypes)

	return classification, avgConf, piiTypes
}

// runStatisticalLayer analyzes column value distributions for PII heuristics.
func (c *MLClassifier) runStatisticalLayer(columns []string, sampleRows [][]string) (string, float64, []model.StatisticalAnalysis) {
	if len(columns) == 0 || len(sampleRows) == 0 {
		return "internal", 0, nil
	}

	var analyses []model.StatisticalAnalysis
	highestRank := 0

	for colIdx, colName := range columns {
		values := extractColumnValues(sampleRows, colIdx)
		if len(values) == 0 {
			continue
		}

		analysis := analyzeColumn(colName, values)
		if analysis.InferredType != "" {
			rank := piiTypeToRank(analysis.InferredType, "")
			if rank > highestRank {
				highestRank = rank
			}
		}
		analyses = append(analyses, analysis)
	}

	if len(analyses) == 0 {
		return "internal", 0, nil
	}

	// Aggregate confidence from analyses that found an inferred type.
	var totalConf float64
	var count int
	for _, a := range analyses {
		if a.InferredType != "" {
			totalConf += a.Confidence
			count++
		}
	}

	var avgConf float64
	if count > 0 {
		avgConf = totalConf / float64(count)
	}

	return rankToClassification[highestRank], avgConf, analyses
}

// analyzeColumn performs statistical analysis on a set of column values.
func analyzeColumn(colName string, values []string) model.StatisticalAnalysis {
	analysis := model.StatisticalAnalysis{
		ColumnName:         colName,
		CharacterClassDist: make(map[string]float64),
	}

	if len(values) == 0 {
		return analysis
	}

	// Cardinality ratio: unique values / total values.
	uniqueSet := make(map[string]bool)
	var totalLen float64
	var lengths []float64
	var nullCount int

	digitCount, alphaCount, specialCount, totalChars := 0, 0, 0, 0

	for _, v := range values {
		if v == "" {
			nullCount++
			continue
		}
		uniqueSet[v] = true
		l := float64(len(v))
		totalLen += l
		lengths = append(lengths, l)

		for _, ch := range v {
			totalChars++
			switch {
			case unicode.IsDigit(ch):
				digitCount++
			case unicode.IsLetter(ch):
				alphaCount++
			default:
				specialCount++
			}
		}
	}

	n := float64(len(values))
	analysis.NullRate = float64(nullCount) / n
	if len(lengths) > 0 {
		analysis.CardinalityRatio = float64(len(uniqueSet)) / float64(len(lengths))
		analysis.AvgValueLength = totalLen / float64(len(lengths))
		analysis.LengthStdDev = stdDev(lengths, analysis.AvgValueLength)

		// Check if fixed length (all non-null values same length).
		firstLen := lengths[0]
		isFixed := true
		for _, l := range lengths[1:] {
			if l != firstLen {
				isFixed = false
				break
			}
		}
		analysis.IsFixedLength = isFixed
	}

	if totalChars > 0 {
		tc := float64(totalChars)
		analysis.CharacterClassDist["digit"] = float64(digitCount) / tc
		analysis.CharacterClassDist["alpha"] = float64(alphaCount) / tc
		analysis.CharacterClassDist["special"] = float64(specialCount) / tc
	}

	// Heuristic inference based on statistical properties.
	analysis.InferredType, analysis.Confidence = inferTypeFromStats(analysis)

	return analysis
}

// inferTypeFromStats uses statistical properties to guess the data type.
func inferTypeFromStats(a model.StatisticalAnalysis) (string, float64) {
	digitRatio := a.CharacterClassDist["digit"]
	specialRatio := a.CharacterClassDist["special"]

	// SSN-like: fixed length 11, high digit ratio, some dashes.
	if a.IsFixedLength && a.AvgValueLength >= 9 && a.AvgValueLength <= 11 &&
		digitRatio > 0.7 && a.CardinalityRatio > 0.9 {
		return "ssn", 0.60
	}

	// Credit card-like: length 15-19, mostly digits, high cardinality.
	if a.AvgValueLength >= 15 && a.AvgValueLength <= 19 &&
		digitRatio > 0.8 && a.CardinalityRatio > 0.95 {
		return "credit_card", 0.55
	}

	// Email-like: contains @ (special chars), moderate length, high cardinality.
	if specialRatio > 0.05 && a.AvgValueLength >= 10 && a.AvgValueLength <= 50 &&
		a.CardinalityRatio > 0.8 {
		return "email", 0.45
	}

	// Phone-like: length 10-15, mostly digits, some special.
	if a.AvgValueLength >= 10 && a.AvgValueLength <= 15 &&
		digitRatio > 0.6 && specialRatio > 0.01 && a.CardinalityRatio > 0.8 {
		return "phone", 0.45
	}

	// IP address-like: fixed-ish length, digits and dots.
	if a.AvgValueLength >= 7 && a.AvgValueLength <= 15 &&
		digitRatio > 0.5 && specialRatio > 0.15 && a.LengthStdDev < 4 {
		return "ip_address", 0.40
	}

	// High-cardinality identifier: could be sensitive.
	if a.CardinalityRatio > 0.98 && a.IsFixedLength && a.AvgValueLength > 5 {
		return "identifier", 0.35
	}

	return "", 0
}

// extractColumnsFromSchema extracts column names from an asset's SchemaInfo.
func extractColumnsFromSchema(asset *cybermodel.DSPMDataAsset) []string {
	if asset.SchemaInfo == nil {
		return nil
	}

	var columns []string

	// Try schema_info.tables[].columns pattern.
	if tables, ok := asset.SchemaInfo["tables"].([]interface{}); ok {
		for _, t := range tables {
			tbl, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			if cols, ok := tbl["columns"].([]interface{}); ok {
				for _, c := range cols {
					switch v := c.(type) {
					case string:
						columns = append(columns, v)
					case map[string]interface{}:
						if name, ok := v["name"].(string); ok {
							columns = append(columns, name)
						}
					}
				}
			}
		}
	}

	// Try direct columns array.
	if cols, ok := asset.SchemaInfo["columns"].([]interface{}); ok {
		for _, c := range cols {
			switch v := c.(type) {
			case string:
				columns = append(columns, v)
			case map[string]interface{}:
				if name, ok := v["name"].(string); ok {
					columns = append(columns, name)
				}
			}
		}
	}

	return columns
}

// extractColumnValues extracts values for a given column index from sample rows.
func extractColumnValues(rows [][]string, colIdx int) []string {
	var values []string
	for _, row := range rows {
		if colIdx < len(row) {
			values = append(values, row[colIdx])
		}
	}
	return values
}

// piiTypeToRank maps a PII type to its classification rank.
func piiTypeToRank(piiType, category string) int {
	// Biometric and health data are highest sensitivity.
	switch category {
	case "biometric":
		return classificationRank["restricted"]
	case "health":
		return classificationRank["restricted"]
	}

	switch piiType {
	case "fingerprint_hash", "facial_recognition_id", "retina_scan",
		"voice_print", "dna_sequence":
		return classificationRank["top_secret"]
	case "ssn", "ssn_us", "national_id", "national_id_sa", "emirates_id",
		"credit_card", "iban", "medical_record", "diagnosis_code",
		"driver_license", "passport":
		return classificationRank["restricted"]
	case "income", "salary", "tax_id", "account_number", "insurance_id",
		"blood_type", "medication", "lab_result", "patient":
		return classificationRank["confidential"]
	case "email", "phone", "date_of_birth", "person_name", "ip_address",
		"geolocation", "street_address", "device_id", "mac_address":
		return classificationRank["confidential"]
	case "postal_code", "gender", "voter_id", "birth_certificate":
		return classificationRank["internal"]
	default:
		return classificationRank["internal"]
	}
}

// maxRank returns the largest of the given ranks.
func maxRank(ranks ...int) int {
	m := 0
	for _, r := range ranks {
		if r > m {
			m = r
		}
	}
	return m
}

// deduplicateStrings merges multiple string slices and removes duplicates.
func deduplicateStrings(slices ...[]string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slices {
		for _, v := range s {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		}
	}
	sort.Strings(result)
	return result
}

// stdDev calculates the standard deviation of a slice of float64 values.
func stdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	var sumSq float64
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

// buildExplanation generates a human-readable explanation of the classification.
func buildExplanation(result *model.EnhancedClassification) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Asset '%s' classified as %s (confidence: %.2f).",
		result.AssetName, strings.ToUpper(result.Classification), result.Confidence))

	if len(result.Evidence.PatternMatches) > 0 {
		parts = append(parts, fmt.Sprintf("Pattern matching found %d column-name matches (confidence: %.2f).",
			len(result.Evidence.PatternMatches), result.PatternConfidence))
	}

	if len(result.Evidence.ContentResults) > 0 {
		parts = append(parts, fmt.Sprintf("Content inspection analyzed %d columns with PII detections (confidence: %.2f).",
			len(result.Evidence.ContentResults), result.ContentConfidence))
	}

	if len(result.Evidence.StatisticalResults) > 0 {
		inferredCount := 0
		for _, s := range result.Evidence.StatisticalResults {
			if s.InferredType != "" {
				inferredCount++
			}
		}
		if inferredCount > 0 {
			parts = append(parts, fmt.Sprintf("Statistical analysis inferred %d potential PII columns (confidence: %.2f).",
				inferredCount, result.StatisticalConfidence))
		}
	}

	if len(result.PIITypes) > 0 {
		parts = append(parts, fmt.Sprintf("Detected PII types: %s.", strings.Join(result.PIITypes, ", ")))
	}

	if result.NeedsHumanReview {
		parts = append(parts, "LOW CONFIDENCE: Human review recommended.")
	}

	return strings.Join(parts, " ")
}
