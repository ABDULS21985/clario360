package engine

import (
	"sort"
	"strings"
	"unicode"

	"github.com/rs/zerolog"
	"golang.org/x/text/unicode/norm"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

// ---------------------------------------------------------------------------
// Classification result — enriched beyond the original
// ---------------------------------------------------------------------------

// unknownResult returns a zero-confidence classification.
func unknownResult(method, rule string) *chatmodel.ClassificationResult {
	return &chatmodel.ClassificationResult{
		Intent:      "unknown",
		Confidence:  0,
		MatchMethod: method,
		MatchedRule: rule,
		Entities:    map[string]string{},
	}
}

// ---------------------------------------------------------------------------
// Matcher interface — pluggable classification strategies
// ---------------------------------------------------------------------------

// MatchCandidate is produced by a Matcher when it finds a potential match.
type MatchCandidate struct {
	Intent      string
	ToolName    string
	Confidence  float64
	MatchMethod string   // "regex", "keyword", "embedding", etc.
	MatchedRule string   // human-readable explanation of what matched
	Keywords    []string // which keywords fired (empty for regex)
}

// Matcher evaluates a normalised message against one classification
// strategy.  Multiple matchers are chained: highest confidence wins.
type Matcher interface {
	Name() string
	Match(normalised string, intents []*chatmodel.IntentPattern) []MatchCandidate
}

// ---------------------------------------------------------------------------
// Built-in matchers
// ---------------------------------------------------------------------------

// regexMatcher tests compiled patterns.  A hit returns high confidence.
type regexMatcher struct{}

func (m *regexMatcher) Name() string { return "regex" }

func (m *regexMatcher) Match(normalised string, intents []*chatmodel.IntentPattern) []MatchCandidate {
	var candidates []MatchCandidate
	for _, intent := range intents {
		for idx, pattern := range intent.Patterns {
			if pattern.MatchString(normalised) {
				candidates = append(candidates, MatchCandidate{
					Intent:      intent.Intent,
					ToolName:    intent.ToolName,
					Confidence:  0.90,
					MatchMethod: "regex",
					MatchedRule: intent.PatternStrings[idx],
				})
				break // one regex hit per intent is enough
			}
		}
	}
	return candidates
}

// keywordMatcher computes a keyword-overlap score.
type keywordMatcher struct {
	// MinOverlap is the minimum fraction of keywords that must match
	// for the intent to be considered (default 0.30).
	MinOverlap float64

	// BaseConfidence is added to the scaled overlap score.
	// Final confidence = BaseConfidence + overlap * ScaleFactor.
	BaseConfidence float64
	ScaleFactor    float64
}

func (m *keywordMatcher) Name() string { return "keyword" }

func (m *keywordMatcher) defaults() (float64, float64, float64) {
	minOvr := m.MinOverlap
	if minOvr <= 0 {
		minOvr = 0.30
	}
	base := m.BaseConfidence
	if base <= 0 {
		base = 0.50
	}
	scale := m.ScaleFactor
	if scale <= 0 {
		scale = 0.30
	}
	return minOvr, base, scale
}

func (m *keywordMatcher) Match(normalised string, intents []*chatmodel.IntentPattern) []MatchCandidate {
	minOvr, base, scale := m.defaults()

	var candidates []MatchCandidate
	for _, intent := range intents {
		if len(intent.Keywords) == 0 {
			continue
		}

		var matched []string
		for _, kw := range intent.Keywords {
			if strings.Contains(normalised, strings.ToLower(kw)) {
				matched = append(matched, kw)
			}
		}

		overlap := float64(len(matched)) / float64(len(intent.Keywords))
		if overlap <= minOvr {
			continue
		}

		candidates = append(candidates, MatchCandidate{
			Intent:      intent.Intent,
			ToolName:    intent.ToolName,
			Confidence:  clampConfidence(base + overlap*scale),
			MatchMethod: "keyword",
			MatchedRule: "keywords: " + strings.Join(matched, ", "),
			Keywords:    matched,
		})
	}
	return candidates
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const (
	DefaultKeywordMinOverlap     = 0.30
	DefaultKeywordBaseConfidence = 0.50
	DefaultKeywordScaleFactor    = 0.30
	DefaultRegexConfidence       = 0.90
	DefaultAmbiguityGap          = 0.10 // if top-2 are within this gap → ambiguous
)

// ClassifierOption configures the IntentClassifier.
type ClassifierOption func(*IntentClassifier)

// WithIntents replaces the default intent patterns.
func WithIntents(intents []*chatmodel.IntentPattern) ClassifierOption {
	return func(c *IntentClassifier) {
		if len(intents) > 0 {
			c.intents = intents
		}
	}
}

// WithMatchers replaces the built-in matcher chain.
func WithMatchers(matchers ...Matcher) ClassifierOption {
	return func(c *IntentClassifier) {
		if len(matchers) > 0 {
			c.matchers = matchers
		}
	}
}

// WithAmbiguityGap sets the confidence gap threshold for ambiguity
// detection.  When the top two candidates are within this gap, the
// result is flagged as ambiguous.
func WithAmbiguityGap(gap float64) ClassifierOption {
	return func(c *IntentClassifier) {
		if gap > 0 {
			c.ambiguityGap = gap
		}
	}
}

// WithClassifierLogger injects a structured logger.
func WithClassifierLogger(l zerolog.Logger) ClassifierOption {
	return func(c *IntentClassifier) { c.logger = l }
}

// WithEntityExtractor injects a function that extracts entities from the
// normalised message given the winning intent.  The default is a no-op.
type EntityExtractorFunc func(normalised, intent string) map[string]string

func WithEntityExtractor(fn EntityExtractorFunc) ClassifierOption {
	return func(c *IntentClassifier) {
		if fn != nil {
			c.entityExtractor = fn
		}
	}
}

// ---------------------------------------------------------------------------
// IntentClassifier
// ---------------------------------------------------------------------------

// IntentClassifier determines the user's intent from a raw message.
//
// Pipeline:
//  1. Normalise the message (lower, NFC, strip special chars, collapse whitespace)
//  2. Run each Matcher in order, collecting MatchCandidates
//  3. Sort candidates by confidence descending
//  4. Pick the top candidate (if any exceed the minimum threshold)
//  5. Detect ambiguity: if top-2 are within ambiguityGap, flag it
//  6. Run the entity extractor on the winning intent
//  7. Return a ClassificationResult with full audit trail
//
// The classifier is stateless and safe for concurrent use.
type IntentClassifier struct {
	intents         []*chatmodel.IntentPattern
	matchers        []Matcher
	ambiguityGap    float64
	entityExtractor EntityExtractorFunc
	logger          zerolog.Logger
}

func NewIntentClassifier(opts ...ClassifierOption) *IntentClassifier {
	c := &IntentClassifier{
		intents:      defaultIntentPatterns(),
		ambiguityGap: DefaultAmbiguityGap,
		entityExtractor: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		logger: zerolog.Nop(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Default matcher chain if none injected.
	if len(c.matchers) == 0 {
		c.matchers = []Matcher{
			&regexMatcher{},
			&keywordMatcher{
				MinOverlap:     DefaultKeywordMinOverlap,
				BaseConfidence: DefaultKeywordBaseConfidence,
				ScaleFactor:    DefaultKeywordScaleFactor,
			},
		}
	}

	return c
}

// Intents returns the registered intent patterns (useful for introspection).
func (c *IntentClassifier) Intents() []*chatmodel.IntentPattern {
	return c.intents
}

// ===========================================================================
// Classify — primary entry-point
// ===========================================================================

// Classify normalises the message, runs the matcher chain, selects the
// best candidate, and returns a fully populated ClassificationResult.
func (c *IntentClassifier) Classify(message string) *chatmodel.ClassificationResult {
	normalised := normalizeMessage(message)
	if normalised == "" {
		return unknownResult("fallback", "empty message after normalisation")
	}

	// ---- Collect candidates from all matchers --------------------------
	var allCandidates []MatchCandidate
	for _, matcher := range c.matchers {
		candidates := matcher.Match(normalised, c.intents)
		allCandidates = append(allCandidates, candidates...)
	}

	if len(allCandidates) == 0 {
		return unknownResult("fallback", "no pattern or keyword matched")
	}

	// ---- Sort by confidence descending ---------------------------------
	sort.Slice(allCandidates, func(i, j int) bool {
		return allCandidates[i].Confidence > allCandidates[j].Confidence
	})

	winner := allCandidates[0]

	// ---- Detect ambiguity ----------------------------------------------
	ambiguous := false
	var runnerUp *MatchCandidate

	if len(allCandidates) > 1 {
		second := allCandidates[1]
		if second.Intent != winner.Intent &&
			winner.Confidence-second.Confidence < c.ambiguityGap {
			ambiguous = true
			runnerUp = &second
		}
	}

	// ---- Extract entities ----------------------------------------------
	entities := c.entityExtractor(normalised, winner.Intent)

	// ---- Build result --------------------------------------------------
	result := &chatmodel.ClassificationResult{
		Intent:      winner.Intent,
		ToolName:    winner.ToolName,
		Confidence:  winner.Confidence,
		MatchMethod: winner.MatchMethod,
		MatchedRule: winner.MatchedRule,
		Entities:    entities,
	}

	// ---- Log -----------------------------------------------------------
	event := c.logger.Debug().
		Str("intent", winner.Intent).
		Float64("confidence", winner.Confidence).
		Str("method", winner.MatchMethod).
		Str("rule", winner.MatchedRule).
		Bool("ambiguous", ambiguous).
		Int("total_candidates", len(allCandidates))

	if ambiguous && runnerUp != nil {
		event = event.
			Str("runner_up_intent", runnerUp.Intent).
			Float64("runner_up_confidence", runnerUp.Confidence)
	}

	event.Msg("intent classified")

	return result
}

// ===========================================================================
// Normalisation
// ===========================================================================

// normalizeMessage prepares raw user input for classification:
//   - Unicode NFC normalisation
//   - lower-case
//   - strip everything except letters, digits, and a safe symbol set
//   - collapse whitespace
//
// The result is deterministic and locale-independent.
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
		case isSafeSymbol(r):
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

// safeSymbols is the set of non-alphanumeric characters preserved during
// normalisation.  Kept as a const string so it's easy to audit and extend.
const safeSymbols = "#@.:/-_"

func isSafeSymbol(r rune) bool {
	return strings.ContainsRune(safeSymbols, r)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func clampConfidence(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
