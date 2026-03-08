package engine

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type IntentClassifier struct {
	intents []*chatmodel.IntentPattern
}

func NewIntentClassifier() *IntentClassifier {
	return &IntentClassifier{intents: defaultIntentPatterns()}
}

func (c *IntentClassifier) Intents() []*chatmodel.IntentPattern {
	return c.intents
}

func (c *IntentClassifier) Classify(message string) *chatmodel.ClassificationResult {
	normalized := normalizeMessage(message)
	if normalized == "" {
		return &chatmodel.ClassificationResult{
			Intent:      "unknown",
			Confidence:  0,
			MatchMethod: "fallback",
			MatchedRule: "no pattern or keyword matched",
			Entities:    map[string]string{},
		}
	}
	for _, intent := range c.intents {
		for idx, pattern := range intent.Patterns {
			if pattern.MatchString(normalized) {
				return &chatmodel.ClassificationResult{
					Intent:      intent.Intent,
					ToolName:    intent.ToolName,
					Confidence:  0.90,
					MatchMethod: "regex",
					MatchedRule: intent.PatternStrings[idx],
					Entities:    map[string]string{},
				}
			}
		}
	}

	bestScore := 0.0
	bestPattern := (*chatmodel.IntentPattern)(nil)
	matchedKeywords := []string{}
	for _, intent := range c.intents {
		if len(intent.Keywords) == 0 {
			continue
		}
		matches := make([]string, 0, len(intent.Keywords))
		for _, keyword := range intent.Keywords {
			if strings.Contains(normalized, strings.ToLower(keyword)) {
				matches = append(matches, keyword)
			}
		}
		score := float64(len(matches)) / float64(len(intent.Keywords))
		if score > bestScore {
			bestScore = score
			bestPattern = intent
			matchedKeywords = matches
		}
	}
	if bestPattern != nil && bestScore > 0.30 {
		return &chatmodel.ClassificationResult{
			Intent:      bestPattern.Intent,
			ToolName:    bestPattern.ToolName,
			Confidence:  0.50 + bestScore*0.30,
			MatchMethod: "keyword",
			MatchedRule: "keywords: " + strings.Join(matchedKeywords, ", "),
			Entities:    map[string]string{},
		}
	}
	return &chatmodel.ClassificationResult{
		Intent:      "unknown",
		Confidence:  0,
		MatchMethod: "fallback",
		MatchedRule: "no pattern or keyword matched",
		Entities:    map[string]string{},
	}
}

func normalizeMessage(message string) string {
	message = norm.NFC.String(strings.ToLower(strings.TrimSpace(message)))
	if message == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(message))
	lastSpace := false
	for _, r := range message {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case strings.ContainsRune("#@.:/-_", r):
			b.WriteRune(r)
			lastSpace = false
		default:
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

