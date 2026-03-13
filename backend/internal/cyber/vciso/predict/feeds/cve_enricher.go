package feeds

import "strings"

type CVEEnrichment struct {
	EPSSScore            float64 `json:"epss_score"`
	InKnownExploitedList bool    `json:"in_known_exploited_list"`
	ProofOfConcept       bool    `json:"proof_of_concept"`
	SocialMediaMentions  int     `json:"social_media_mentions"`
	ProductPrevalence    float64 `json:"product_prevalence"`
}

type CVEEnricher struct{}

func NewCVEEnricher() *CVEEnricher {
	return &CVEEnricher{}
}

func (e *CVEEnricher) Enrich(metadata map[string]any, productPrevalence float64) CVEEnrichment {
	return CVEEnrichment{
		EPSSScore:            floatValue(metadata, "epss_score", "epss"),
		InKnownExploitedList: boolValue(metadata, "cisa_kev", "known_exploited", "exploited_in_wild"),
		ProofOfConcept:       boolValue(metadata, "proof_of_concept", "public_exploit_available"),
		SocialMediaMentions:  int(floatValue(metadata, "social_mentions", "social_media_mentions")),
		ProductPrevalence:    productPrevalence,
	}
}

func floatValue(metadata map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case int:
			return float64(typed)
		case string:
			typed = strings.TrimSpace(typed)
			if typed == "true" {
				return 1
			}
			if typed == "false" {
				return 0
			}
		}
	}
	return 0
}

func boolValue(metadata map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			lower := strings.ToLower(strings.TrimSpace(typed))
			return lower == "true" || lower == "yes" || lower == "1"
		case float64:
			return typed > 0
		case int:
			return typed > 0
		}
	}
	return false
}
